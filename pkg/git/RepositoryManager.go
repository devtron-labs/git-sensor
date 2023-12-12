/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package git

import (
	"errors"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/util"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"strings"
	"time"

	"github.com/devtron-labs/git-sensor/internal/middleware"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"go.uber.org/zap"
)

type RepositoryManager interface {
	Fetch(gitContext *GitContext, url string, location string) (updated bool, repo *GitRepository, err error)
	Add(gitProviderId int, location, url string, gitContext *GitContext, authMode sql.AuthMode, sshPrivateKeyContent string) error
	Clean(cloneDir string) error
	ChangesSince(checkoutPath string, branch string, from string, to string, count int) ([]*GitCommitBase, error)
	ChangesSinceByRepository(repository *GitRepository, branch string, from string, to string, count int, checkoutPath string) ([]*GitCommitBase, error)
	GetCommitMetadata(checkoutPath, commitHash string) (*GitCommitBase, error)
	GetCommitForTag(checkoutPath, tag string) (*GitCommitBase, error)
	CreateSshFileIfNotExistsAndConfigureSshCommand(location string, gitProviderId int, sshPrivateKeyContent string) error
}

type RepositoryManagerImpl struct {
	logger        *zap.SugaredLogger
	gitManager    GitManagerImpl
	configuration *internal.Configuration
}

func NewRepositoryManagerImpl(
	logger *zap.SugaredLogger,
	configuration *internal.Configuration,
	gitManager GitManagerImpl,
) *RepositoryManagerImpl {
	return &RepositoryManagerImpl{logger: logger, configuration: configuration, gitManager: gitManager}
}

func (impl RepositoryManagerImpl) IsSpaceAvailableOnDisk() bool {
	var statFs unix.Statfs_t
	err := unix.Statfs(GIT_BASE_DIR, &statFs)
	if err != nil {
		return false
	}
	availableSpace := int64(statFs.Bavail) * int64(statFs.Bsize)
	return availableSpace > int64(impl.configuration.MinLimit)*1024*1024
}

func (impl RepositoryManagerImpl) Add(gitProviderId int, location string, url string, gitContext *GitContext, authMode sql.AuthMode, sshPrivateKeyContent string) error {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("add", start, err)
	}()
	err = os.RemoveAll(location)
	if err != nil {
		impl.logger.Errorw("error in cleaning checkout path", "err", err)
		return err
	}
	if !impl.IsSpaceAvailableOnDisk() {
		err = errors.New("git-sensor PVC - disk full, please increase space")
		return err
	}
	err = impl.gitManager.Init(location, url, true)
	if err != nil {
		impl.logger.Errorw("err in git init", "err", err)
		return err
	}

	// check ssh
	if authMode == sql.AUTH_MODE_SSH {
		err = impl.CreateSshFileIfNotExistsAndConfigureSshCommand(location, gitProviderId, sshPrivateKeyContent)
		if err != nil {
			return err
		}
	}

	opt, errorMsg, err := impl.gitManager.Fetch(gitContext, location)
	if err != nil {
		impl.logger.Errorw("error in cloning repo", "errorMsg", errorMsg, "err", err)
		return err
	}
	impl.logger.Debugw("opt msg", "opt", opt)
	return nil
}

func (impl RepositoryManagerImpl) Clean(dir string) error {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("clean", start, err)
	}()
	err = os.RemoveAll(dir)
	return err
}

func (impl RepositoryManagerImpl) Fetch(gitContext *GitContext, url string, location string) (updated bool, repo *GitRepository, err error) {
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("fetch", start, err)
	}()
	middleware.GitMaterialPollCounter.WithLabelValues().Inc()
	if !impl.IsSpaceAvailableOnDisk() {
		err = errors.New("git-sensor PVC - disk full, please increase space")
		return false, nil, err
	}
	r, err := impl.gitManager.OpenNewRepo(location, url)
	if err != nil {
		return false, r, err
	}
	res, errorMsg, err := impl.gitManager.Fetch(gitContext, location)

	if err == nil && len(res) > 0 {
		impl.logger.Infow("repository updated", "location", url)
		//updated
		middleware.GitPullDuration.WithLabelValues("true", "true").Observe(time.Since(start).Seconds())
		return true, r, nil
	} else if err == nil && len(res) == 0 {
		impl.logger.Debugw("no update for ", "path", url)
		middleware.GitPullDuration.WithLabelValues("true", "false").Observe(time.Since(start).Seconds())
		return false, r, nil
	} else {
		impl.logger.Errorw("error in updating repository", "err", err, "location", url, "error msg", errorMsg)
		middleware.GitPullDuration.WithLabelValues("false", "false").Observe(time.Since(start).Seconds())
		return false, r, err
	}

}

func (impl RepositoryManagerImpl) GetCommitForTag(checkoutPath, tag string) (*GitCommitBase, error) {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("getCommitForTag", start, err)
	}()
	tag = strings.TrimSpace(tag)
	commit, err := impl.gitManager.GetCommitsForTag(checkoutPath, tag)
	if err != nil {
		return nil, err
	}
	return commit.GetCommit(), nil
}

func (impl RepositoryManagerImpl) GetCommitMetadata(checkoutPath, commitHash string) (*GitCommitBase, error) {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("getCommitMetadata", start, err)
	}()
	gitCommit, err := impl.gitManager.GetCommitForHash(checkoutPath, commitHash)
	if err != nil {
		return nil, err
	}

	return gitCommit.GetCommit(), nil
}

// from -> old commit
// to -> new commit
func (impl RepositoryManagerImpl) ChangesSinceByRepository(repository *GitRepository, branch string, from string, to string, count int, checkoutPath string) ([]*GitCommitBase, error) {
	// fix for azure devops (manual trigger webhook bases pipeline) :
	// branch name comes as 'refs/heads/master', we need to extract actual branch name out of it.
	// https://stackoverflow.com/questions/59956206/how-to-get-a-branch-name-with-a-slash-in-azure-devops

	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("changesSinceByRepository", start, err)
	}()
	branch, branchRef := GetBranchReference(branch)
	itr, err := impl.gitManager.GetCommitIterator(repository, IteratorRequest{
		BranchRef:      branchRef,
		Branch:         branch,
		CommitCount:    count,
		FromCommitHash: from,
		ToCommitHash:   to,
	})
	if err != nil {
		return nil, err
	}
	var gitCommits []*GitCommitBase
	itrCounter := 0
	commitToFind := len(to) == 0 //no commit mentioned
	breakLoop := false
	for {
		if breakLoop {
			break
		}
		//TODO: move code out of this dummy function after removing defer inside loop
		func() {
			if itrCounter > 1000 || len(gitCommits) == count {
				breakLoop = true
				return
			}
			commit, err := itr.Next()
			if err == io.EOF {
				breakLoop = true
				return
			}
			if err != nil {
				impl.logger.Errorw("error in  iterating", "branch", branch, "err", err)
				breakLoop = true
				return
			}
			if !commitToFind && strings.Contains(commit.GetCommit().Commit, to) {
				commitToFind = true
			}
			if !commitToFind {
				return
			}
			if commit.GetCommit().Commit == from && len(from) > 0 {
				//found end
				breakLoop = true
				return
			}

			gitCommit := commit.GetCommit()
			gitCommit.TruncateMessageIfExceedsMaxLength()
			if !gitCommit.IsMessageValidUTF8() {
				gitCommit.FixInvalidUTF8Message()
			}
			impl.logger.Debugw("commit dto for repo ", "repo", repository, commit)
			gitCommits = append(gitCommits, gitCommit)
			itrCounter = itrCounter + 1
			if impl.configuration.EnableFileStats {
				defer func() {
					if err := recover(); err != nil {
						impl.logger.Error("file stats function panicked for commit", "err", err, "commit", commit)
					}
				}()
				//TODO: implement below Stats() function using git CLI as it panics in some cases, remove defer function after using git CLI

				stats, err := impl.gitManager.GetCommitStats(commit)
				if err != nil {
					impl.logger.Errorw("error in  fetching stats", "err", err)
				}
				gitCommit.SetFileStats(&stats)
			}
		}()
	}
	return gitCommits, err
}

func (impl RepositoryManagerImpl) ChangesSince(checkoutPath string, branch string, from string, to string, count int) ([]*GitCommitBase, error) {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("changesSince", start, err)
	}()
	if count == 0 {
		count = impl.configuration.GitHistoryCount
	}
	r, err := impl.gitManager.OpenRepoPlain(checkoutPath)
	if err != nil {
		return nil, err
	}
	///---------------------
	return impl.ChangesSinceByRepository(r, branch, from, to, count, checkoutPath)
	///----------------------

}

func (impl RepositoryManagerImpl) CreateSshFileIfNotExistsAndConfigureSshCommand(location string, gitProviderId int, sshPrivateKeyContent string) error {
	// add private key
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("createSshFileIfNotExistsAndConfigureSshCommand", start, err)
	}()
	sshPrivateKeyPath, err := GetOrCreateSshPrivateKeyOnDisk(gitProviderId, sshPrivateKeyContent)
	if err != nil {
		impl.logger.Errorw("error in creating ssh private key", "err", err)
		return err
	}

	//git config core.sshCommand
	_, errorMsg, err := impl.gitManager.ConfigureSshCommand(location, sshPrivateKeyPath)
	if err != nil {
		impl.logger.Errorw("error in configuring ssh command while adding repo", "errorMsg", errorMsg, "err", err)
		return err
	}

	return nil
}
