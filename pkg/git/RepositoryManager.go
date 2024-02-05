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
	"fmt"
	"github.com/devtron-labs/git-sensor/internals"
	"github.com/devtron-labs/git-sensor/util"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/devtron-labs/git-sensor/internals/middleware"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"go.uber.org/zap"
)

type RepositoryManager interface {
	// Fetch Fetches latest commit for  repo. Creates a new repo if it doesn't already exist
	// and returns the reference to the repo
	Fetch(gitCtx GitContext, url string, location string) (updated bool, repo *GitRepository, err error)
	// Add adds and initializes a new git repo , cleans the directory if not empty and fetches latest commits
	Add(gitCtx GitContext, gitProviderId int, location, url string, authMode sql.AuthMode, sshPrivateKeyContent string) error
	GetSshPrivateKeyPath(gitCtx GitContext, gitProviderId int, location, url string, authMode sql.AuthMode, sshPrivateKeyContent string) (string, error)
	FetchRepo(gitCtx GitContext, location string) error
	GetLocationForMaterial(material *sql.GitMaterial, cloningMode string) (location string, httpMatched bool, shMatched bool, err error)
	GetCheckoutPathAndLocation(gitCtx GitContext, material *sql.GitMaterial, url string) (string, string, error)
	TrimLastGitCommit(gitCommits []*GitCommitBase, count int) []*GitCommitBase
	// Clean cleans a directory
	Clean(cloneDir string) error
	// ChangesSince given the checkput path, retrieves the latest commits for the gt repo existing on the path
	ChangesSince(gitCtx GitContext, checkoutPath string, branch string, from string, to string, count int) ([]*GitCommitBase, error)
	// ChangesSinceByRepository returns the latest commits list for the given range and count for an existing repo
	ChangesSinceByRepository(gitCtx GitContext, repository *GitRepository, branch string, from string, to string, count int, checkoutPath string) ([]*GitCommitBase, error)
	// GetCommitMetadata retrieves the commit metadata for given hash
	GetCommitMetadata(gitCtx GitContext, checkoutPath, commitHash string) (*GitCommitBase, error)
	// GetCommitForTag retrieves the commit metadata for given tag
	GetCommitForTag(gitCtx GitContext, checkoutPath, tag string) (*GitCommitBase, error)
	// CreateSshFileIfNotExistsAndConfigureSshCommand creates ssh file with creds and configures it at the location
	CreateSshFileIfNotExistsAndConfigureSshCommand(gitCtx GitContext, location string, gitProviderId int, sshPrivateKeyContent string) (string, error)
}

type RepositoryManagerImpl struct {
	logger        *zap.SugaredLogger
	gitManager    GitManager
	configuration *internals.Configuration
}

func NewRepositoryManagerImpl(
	logger *zap.SugaredLogger,
	configuration *internals.Configuration,
	gitManager GitManager,
) *RepositoryManagerImpl {
	return &RepositoryManagerImpl{logger: logger, configuration: configuration, gitManager: gitManager}
}

func (impl *RepositoryManagerImpl) IsSpaceAvailableOnDisk() bool {
	var statFs unix.Statfs_t
	err := unix.Statfs(GIT_BASE_DIR, &statFs)
	if err != nil {
		return false
	}
	availableSpace := int64(statFs.Bavail) * int64(statFs.Bsize)
	return availableSpace > int64(impl.configuration.MinLimit)*1024*1024
}

func (impl *RepositoryManagerImpl) GetLocationForMaterial(material *sql.GitMaterial, cloningMode string) (location string, httpMatched bool, shMatched bool, err error) {
	//gitRegex := `/(?:git|ssh|https?|git@[-\w.]+):(\/\/)?(.*?)(\.git)(\/?|\#[-\d\w._]+?)$/`
	httpsRegex := `^https.*`
	httpsMatched, err := regexp.MatchString(httpsRegex, material.Url)
	if httpsMatched {
		locationWithoutProtocol := strings.ReplaceAll(material.Url, "https://", "")
		checkoutPath := path.Join(GIT_BASE_DIR, strconv.Itoa(material.Id), locationWithoutProtocol)
		return checkoutPath, httpsMatched, false, nil
	}

	sshRegex := `^git@.*`
	sshMatched, err := regexp.MatchString(sshRegex, material.Url)
	if sshMatched {
		checkoutPath := path.Join(GIT_BASE_DIR, strconv.Itoa(material.Id), material.Url)
		return checkoutPath, httpsMatched, sshMatched, nil
	}

	return "", httpsMatched, sshMatched, fmt.Errorf("unsupported format url %s", material.Url)
}

func (impl *RepositoryManagerImpl) GetCheckoutPathAndLocation(gitCtx GitContext, material *sql.GitMaterial, url string) (string, string, error) {
	var checkoutPath string
	var checkoutLocationForFetching string
	checkoutPath, _, _, err := impl.GetLocationForMaterial(material, gitCtx.CloningMode)
	if err != nil {
		return checkoutPath, checkoutLocationForFetching, err
	}
	checkoutLocationForFetching = checkoutPath
	return checkoutPath, checkoutLocationForFetching, nil
}

func (impl *RepositoryManagerImpl) Add(gitCtx GitContext, gitProviderId int, location, url string, authMode sql.AuthMode, sshPrivateKeyContent string) error {
	_, err := impl.GetSshPrivateKeyPath(gitCtx, gitProviderId, location, url, authMode, sshPrivateKeyContent)
	if err != nil {
		return err
	}
	return impl.FetchRepo(gitCtx, location)
}

func (impl *RepositoryManagerImpl) GetSshPrivateKeyPath(gitCtx GitContext, gitProviderId int, location, url string, authMode sql.AuthMode, sshPrivateKeyContent string) (string, error) {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("add", start, err)
	}()
	err = os.RemoveAll(location)
	if err != nil {
		impl.logger.Errorw("error in cleaning checkout path", "err", err)
		return "", err
	}
	if !impl.IsSpaceAvailableOnDisk() {
		err = errors.New("git-sensor PVC - disk full, please increase space")
		return "", err
	}
	err = impl.gitManager.Init(gitCtx, location, url, true)
	if err != nil {
		impl.logger.Errorw("err in git init", "err", err)
		return "", err
	}
	var sshPrivateKeyPath string
	// check ssh
	if authMode == sql.AUTH_MODE_SSH {
		sshPrivateKeyPath, err = impl.CreateSshFileIfNotExistsAndConfigureSshCommand(gitCtx, location, gitProviderId, sshPrivateKeyContent)
		if err != nil {
			impl.logger.Errorw("error while creating ssh file for shallow clone", "checkoutPath", location, "sshPrivateKeyPath", sshPrivateKeyPath, "err", err)
			return "", err
		}
	}

	return sshPrivateKeyPath, nil
}

func (impl *RepositoryManagerImpl) FetchRepo(gitCtx GitContext, location string) error {
	opt, errorMsg, err := impl.gitManager.Fetch(gitCtx, location)
	if err != nil {
		impl.logger.Errorw("error in fetching repo", "errorMsg", errorMsg, "err", err)
		return err
	}
	impl.logger.Debugw("opt msg", "opt", opt)
	return nil
}

func (impl *RepositoryManagerImpl) Clean(dir string) error {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("clean", start, err)
	}()
	err = os.RemoveAll(dir)
	return err
}

func (impl *RepositoryManagerImpl) Fetch(gitCtx GitContext, url string, location string) (updated bool, repo *GitRepository, err error) {
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("fetch", start, err)
	}()
	middleware.GitMaterialPollCounter.WithLabelValues().Inc()
	if !impl.IsSpaceAvailableOnDisk() {
		err = errors.New("git-sensor PVC - disk full, please increase space")
		return false, nil, err
	}
	r, err := impl.openNewRepo(gitCtx, location, url)
	if err != nil {
		return false, r, err
	}
	res, errorMsg, err := impl.gitManager.Fetch(gitCtx, location)

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

func (impl *RepositoryManagerImpl) GetCommitForTag(gitCtx GitContext, checkoutPath, tag string) (*GitCommitBase, error) {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("getCommitForTag", start, err)
	}()
	tag = strings.TrimSpace(tag)
	commit, err := impl.gitManager.GetCommitsForTag(gitCtx, checkoutPath, tag)
	if err != nil {
		return nil, err
	}
	return commit.GetCommit(), nil
}

func (impl *RepositoryManagerImpl) GetCommitMetadata(gitCtx GitContext, checkoutPath, commitHash string) (*GitCommitBase, error) {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("getCommitMetadata", start, err)
	}()
	gitCommit, err := impl.gitManager.GetCommitForHash(gitCtx, checkoutPath, commitHash)
	if err != nil {
		return nil, err
	}

	return gitCommit.GetCommit(), nil
}

// from -> old commit
// to -> new commit
func (impl *RepositoryManagerImpl) ChangesSinceByRepository(gitCtx GitContext, repository *GitRepository, branch string, from string, to string, count int, checkoutPath string) ([]*GitCommitBase, error) {
	// fix for azure devops (manual trigger webhook bases pipeline) :
	// branch name comes as 'refs/heads/master', we need to extract actual branch name out of it.
	// https://stackoverflow.com/questions/59956206/how-to-get-a-branch-name-with-a-slash-in-azure-devops

	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("changesSinceByRepository", start, err)
	}()
	branch, branchRef := GetBranchReference(branch)
	itr, err := impl.gitManager.GetCommitIterator(gitCtx, repository, IteratorRequest{
		BranchRef:      branchRef,
		Branch:         branch,
		CommitCount:    count,
		FromCommitHash: from,
		ToCommitHash:   to,
	})
	if err != nil {
		impl.logger.Errorw("error in getting iterator", "branch", branch, "err", err)
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
						impl.logger.Error("file stats function panicked for commit", "err", err, "commit", commit, "count", count)
					}
				}()
				//TODO: implement below Stats() function using git CLI as it panics in some cases, remove defer function after using git CLI

				stats, err := impl.gitManager.GetCommitStats(gitCtx, commit, repository.rootDir)
				if err != nil {
					impl.logger.Errorw("error in  fetching stats", "err", err)
				}
				gitCommit.SetFileStats(&stats)
			}
		}()
	}
	return gitCommits, err
}

func (impl *RepositoryManagerImpl) TrimLastGitCommit(gitCommits []*GitCommitBase, count int) []*GitCommitBase {
	if len(gitCommits) > count {
		gitCommits = gitCommits[:len(gitCommits)-1]
	}
	return gitCommits
}

func (impl *RepositoryManagerImpl) ChangesSince(gitCtx GitContext, checkoutPath string, branch string, from string, to string, count int) ([]*GitCommitBase, error) {
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
	return impl.ChangesSinceByRepository(gitCtx, r, branch, from, to, count, checkoutPath)
	///----------------------

}

func (impl *RepositoryManagerImpl) CreateSshFileIfNotExistsAndConfigureSshCommand(gitCtx GitContext, location string, gitProviderId int, sshPrivateKeyContent string) (string, error) {
	// add private key
	var err error
	var sshPrivateKeyPath string
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("createSshFileIfNotExistsAndConfigureSshCommand", start, err)
	}()
	sshPrivateKeyPath, err = GetOrCreateSshPrivateKeyOnDisk(gitProviderId, sshPrivateKeyContent)
	if err != nil {
		impl.logger.Errorw("error in creating ssh private key", "err", err)
		return sshPrivateKeyPath, err
	}

	//git config core.sshCommand
	_, errorMsg, err := impl.gitManager.ConfigureSshCommand(gitCtx, location, sshPrivateKeyPath)
	if err != nil {
		impl.logger.Errorw("error in configuring ssh command while adding repo", "errorMsg", errorMsg, "err", err)
		return sshPrivateKeyPath, err
	}

	return sshPrivateKeyPath, nil
}

func (impl *RepositoryManagerImpl) openNewRepo(gitCtx GitContext, location string, url string) (*GitRepository, error) {

	r, err := impl.gitManager.OpenRepoPlain(location)
	if err != nil {
		err = os.RemoveAll(location)
		if err != nil {
			impl.logger.Errorw("error in cleaning checkout path: %s", err)
			return r, err
		}
		err = impl.gitManager.Init(gitCtx, location, url, true)
		if err != nil {
			impl.logger.Errorw("err in git init: %s", err)
			return r, err
		}
		r, err = impl.gitManager.OpenRepoPlain(location)
		if err != nil {
			impl.logger.Errorw("err in git init: %s", err)
			return r, err
		}
	}
	return r, nil
}
