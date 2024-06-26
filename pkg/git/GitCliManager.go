/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package git

import (
	"errors"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type GitCliManagerImpl struct {
	GitManagerBase
	logger *zap.SugaredLogger
}

func NewGitCliManagerImpl(baseManager GitManagerBase, logger *zap.SugaredLogger) *GitCliManagerImpl {

	return &GitCliManagerImpl{
		GitManagerBase: baseManager,
		logger:         logger,
	}
}

const (
	GIT_ASK_PASS                = "/git-ask-pass.sh"
	AUTHENTICATION_FAILED_ERROR = "Authentication failed"
	LOCK_REF_MESSAGE            = "cannot lock ref"
)

func (impl *GitCliManagerImpl) Init(gitCtx GitContext, rootDir string, remoteUrl string, isBare bool) error {
	//-----------------

	err := os.MkdirAll(rootDir, 0755)
	if err != nil {
		return err
	}

	err = impl.GitInit(gitCtx, rootDir)
	if err != nil {
		return err
	}
	return impl.GitCreateRemote(gitCtx, rootDir, remoteUrl)

}

func (impl *GitCliManagerImpl) OpenRepoPlain(checkoutPath string) (*GitRepository, error) {

	err := openGitRepo(checkoutPath)
	if err != nil {
		return nil, err
	}
	return &GitRepository{
		rootDir: checkoutPath,
	}, nil
}

func (impl *GitCliManagerImpl) GetCommitsForTag(gitCtx GitContext, checkoutPath, tag string) (GitCommit, error) {
	return impl.GitShow(gitCtx, checkoutPath, tag)
}

func (impl *GitCliManagerImpl) GetCommitForHash(gitCtx GitContext, checkoutPath, commitHash string) (GitCommit, error) {

	return impl.GitShow(gitCtx, checkoutPath, commitHash)
}
func (impl *GitCliManagerImpl) GetCommitIterator(gitCtx GitContext, repository *GitRepository, iteratorRequest IteratorRequest) (CommitIterator, error) {

	commits, err := impl.GetCommits(gitCtx, iteratorRequest.BranchRef, iteratorRequest.Branch, repository.rootDir, iteratorRequest.CommitCount, iteratorRequest.FromCommitHash, iteratorRequest.ToCommitHash)
	if err != nil {
		impl.logger.Errorw("error in fetching commits for", "err", err, "path", repository.rootDir)
		return nil, err
	}
	return &CommitCliIterator{
		commits: commits,
	}, nil
}

func openGitRepo(path string) error {
	if _, err := filepath.Abs(path); err != nil {
		return err
	}
	fst := osfs.New(path)
	_, err := fst.Stat(".git")
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}
func (impl *GitCliManagerImpl) GitInit(gitCtx GitContext, rootDir string) error {
	impl.logger.Debugw("git", "-C", rootDir, "init")
	output, errMsg, err := impl.GitManagerBase.ExecuteCustomCommand(gitCtx, "git", "-C", rootDir, "init")
	impl.logger.Debugw("root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return err
}

func (impl *GitCliManagerImpl) GitCreateRemote(gitCtx GitContext, rootDir string, url string) error {
	impl.logger.Debugw("git", "-C", rootDir, "remote", "add", "origin", url)
	output, errMsg, err := impl.GitManagerBase.ExecuteCustomCommand(gitCtx, "git", "-C", rootDir, "remote", "add", "origin", url)
	impl.logger.Debugw("url", url, "opt", output, "errMsg", errMsg, "error", err)
	return err
}

func (impl *GitCliManagerImpl) GetCommits(gitCtx GitContext, branchRef string, branch string, rootDir string, numCommits int, from string, to string) ([]GitCommit, error) {
	baseCmdArgs := []string{"-C", rootDir, "log"}
	rangeCmdArgs := []string{branchRef}
	extraCmdArgs := []string{"-n", strconv.Itoa(numCommits), "--date=iso-strict", GITFORMAT}
	cmdArgs := impl.getCommandForLogRange(branchRef, from, to, rangeCmdArgs, baseCmdArgs, extraCmdArgs)
	impl.logger.Debugw("git", cmdArgs)
	output, errMsg, err := impl.GitManagerBase.ExecuteCustomCommand(gitCtx, "git", cmdArgs...)
	impl.logger.Debugw("root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	if err != nil {
		if strings.Contains(output, NO_COMMIT_GIT_ERROR_MESSAGE) {
			return nil, errors.New(NO_COMMIT_CUSTOM_ERROR_MESSAGE)
		}
		return nil, err
	}
	commits, err := impl.processGitLogOutput(output)
	if err != nil {
		return nil, err
	}
	return commits, nil
}

func (impl *GitCliManagerImpl) getCommandForLogRange(branchRef string, from string, to string, rangeCmdArgs []string, baseCmdArgs []string, extraCmdArgs []string) []string {
	if from != "" && to != "" {
		rangeCmdArgs = []string{from + "^.." + to}
	} else if from != "" {
		rangeCmdArgs = []string{from + "^.." + branchRef}
	} else if to != "" {
		rangeCmdArgs = []string{to}
	}
	return append(baseCmdArgs, append(rangeCmdArgs, extraCmdArgs...)...)
}

func (impl *GitCliManagerImpl) GitShow(gitCtx GitContext, rootDir string, hash string) (GitCommit, error) {
	impl.logger.Debugw("git", "-C", rootDir, "show", hash, "--date=iso-strict", GITFORMAT, "-s")
	output, errMsg, err := impl.GitManagerBase.ExecuteCustomCommand(gitCtx, "git", "-C", rootDir, "show", hash, "--date=iso-strict", GITFORMAT, "-s")
	impl.logger.Debugw("root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	if err != nil {
		return nil, err
	}
	commits, err := impl.processGitLogOutput(output)
	if err != nil || len(commits) == 0 {
		return nil, err
	}

	return commits[0], nil
}

func (impl *GitCliManagerImpl) GetCommitStats(gitCtx GitContext, commit GitCommit, checkoutPath string) (FileStats, error) {
	gitCommit := commit.GetCommit()
	return impl.FetchDiffStatBetweenCommitsNameOnly(gitCtx, gitCommit.Commit, "", checkoutPath)
}

func (impl *GitCliManagerImpl) processGitLogOutput(out string) ([]GitCommit, error) {

	gitCommits := make([]GitCommit, 0)
	if len(out) == 0 {
		return gitCommits, nil
	}
	gitCommitFormattedList, err := parseFormattedLogOutput(out)
	if err != nil {
		return gitCommits, err
	}

	for _, formattedCommit := range gitCommitFormattedList {

		subject := strings.TrimSpace(formattedCommit.Subject)
		body := strings.TrimSpace(formattedCommit.Body)
		message := subject
		if len(body) > 0 {
			message = strings.Join([]string{subject, body}, "\n")
		}

		cm := GitCommitBase{
			Commit:  formattedCommit.Commit,
			Author:  formattedCommit.Commiter.Name + " <" + formattedCommit.Commiter.Email + ">",
			Date:    formattedCommit.Commiter.Date,
			Message: message,
		}
		gitCommits = append(gitCommits, &GitCommitCli{
			GitCommitBase: cm,
		})
	}
	return gitCommits, nil
}
