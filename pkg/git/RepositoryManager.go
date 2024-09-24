/*
 * Copyright (c) 2020-2024. Devtron Inc.
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
	Fetch(gitCtx GitContext, url string, location string) (updated bool, repo *GitRepository, errMsg string, err error)
	// Add adds and initializes a new git repo , cleans the directory if not empty and fetches latest commits
	Add(gitCtx GitContext, gitProviderId int, location, url string, authMode sql.AuthMode, sshPrivateKeyContent string) (errMsg string, err error)
	InitRepoAndGetSshPrivateKeyPath(gitCtx GitContext, gitProviderId int, location, url string, authMode sql.AuthMode, sshPrivateKeyContent string) (string, string, error)
	FetchRepo(gitCtx GitContext, location string) (errMsg string, err error)
	GetCheckoutLocationFromGitUrl(material *sql.GitMaterial, cloningMode string) (location string, httpMatched bool, shMatched bool, err error)
	GetCheckoutLocation(gitCtx GitContext, material *sql.GitMaterial, url, checkoutPath string) string
	TrimLastGitCommit(gitCommits []*GitCommitBase, count int) []*GitCommitBase
	// Clean cleans a directory
	Clean(cloneDir string) error
	// ChangesSinceByRepository returns the latest commits list for the given range and count for an existing repo
	ChangesSinceByRepository(gitCtx GitContext, repository *GitRepository, branch string, from string, to string, count int, checkoutPath string, openNewGitRepo bool) (gitCommits []*GitCommitBase, errMsg string, err error)
	// GetCommitMetadata retrieves the commit metadata for given hash
	GetCommitMetadata(gitCtx GitContext, checkoutPath, commitHash string) (*GitCommitBase, error)
	// GetCommitForTag retrieves the commit metadata for given tag
	GetCommitForTag(gitCtx GitContext, checkoutPath, tag string) (*GitCommitBase, error)
	// CreateSshFileIfNotExistsAndConfigureSshCommand creates ssh file with creds and configures it at the location
	CreateSshFileIfNotExistsAndConfigureSshCommand(gitCtx GitContext, location string, gitProviderId int, sshPrivateKeyContent string) (privateKeyPath string, errMsg string, err error)
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

func (impl *RepositoryManagerImpl) GetCheckoutLocationFromGitUrl(material *sql.GitMaterial, cloningMode string) (location string, httpMatched bool, shMatched bool, err error) {
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

func (impl *RepositoryManagerImpl) GetCheckoutLocation(gitCtx GitContext, material *sql.GitMaterial, url, checkoutPath string) string {
	return checkoutPath
}

func (impl *RepositoryManagerImpl) Add(gitCtx GitContext, gitProviderId int, location, url string, authMode sql.AuthMode, sshPrivateKeyContent string) (string, error) {
	_, errMsg, err := impl.InitRepoAndGetSshPrivateKeyPath(gitCtx, gitProviderId, location, url, authMode, sshPrivateKeyContent)
	if err != nil {
		return errMsg, err
	}
	return impl.FetchRepo(gitCtx, location)
}

func (impl *RepositoryManagerImpl) InitRepoAndGetSshPrivateKeyPath(gitCtx GitContext, gitProviderId int, location, url string, authMode sql.AuthMode, sshPrivateKeyContent string) (string, string, error) {
	var err error
	var errMsg string
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("add", start, err)
	}()
	errMsg, err = impl.CleanupAndInitRepo(gitCtx, location, url)
	if err != nil {
		return "", errMsg, err
	}
	var sshPrivateKeyPath string
	// check ssh
	if authMode == sql.AUTH_MODE_SSH {
		sshPrivateKeyPath, errMsg, err = impl.CreateSshFileIfNotExistsAndConfigureSshCommand(gitCtx, location, gitProviderId, sshPrivateKeyContent)
		if err != nil {
			impl.logger.Errorw("error while creating ssh file for shallow clone", "checkoutPath", location, "sshPrivateKeyPath", sshPrivateKeyPath, "err", err)
			return "", errMsg, err
		}
	}

	return sshPrivateKeyPath, "", nil
}

func (impl *RepositoryManagerImpl) CleanupAndInitRepo(gitCtx GitContext, location string, url string) (string, error) {
	err := os.RemoveAll(location)
	if err != nil {
		impl.logger.Errorw("error in cleaning checkout path", "err", err)
		return "", err
	}
	if !impl.IsSpaceAvailableOnDisk() {
		err = errors.New("git-sensor PVC - disk full, please increase space")
		return "", err
	}
	errMsg, err := impl.gitManager.Init(gitCtx, location, url, true)
	if err != nil {
		impl.logger.Errorw("err in git init", "err", err)
		return errMsg, err
	}
	return "", nil
}

func (impl *RepositoryManagerImpl) FetchRepo(gitCtx GitContext, location string) (string, error) {
	opt, errMsg, err := impl.gitManager.Fetch(gitCtx, location)
	if err != nil {
		impl.logger.Errorw("error in fetching repo", "errorMsg", errMsg, "err", err)
		return errMsg, err
	}
	impl.logger.Debugw("opt msg", "opt", opt)
	return "", nil
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

func (impl *RepositoryManagerImpl) Fetch(gitCtx GitContext, url string, location string) (updated bool, repo *GitRepository, errMsg string, err error) {
	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("fetch", start, err)
	}()
	middleware.GitMaterialPollCounter.WithLabelValues().Inc()
	if !impl.IsSpaceAvailableOnDisk() {
		err = errors.New("git-sensor PVC - disk full, please increase space")
		return false, nil, "", err
	}
	r, errMsg, err := impl.openNewRepo(gitCtx, location, url)
	if err != nil {
		return false, r, errMsg, err
	}
	res, errMsg, err := impl.gitManager.Fetch(gitCtx, location)

	if err == nil && len(res) > 0 {
		impl.logger.Infow("repository updated", "location", url)
		//updated
		middleware.GitPullDuration.WithLabelValues("true", "true").Observe(time.Since(start).Seconds())
		return true, r, "", nil
	} else if err == nil && len(res) == 0 {
		impl.logger.Debugw("no update for ", "path", url)
		middleware.GitPullDuration.WithLabelValues("true", "false").Observe(time.Since(start).Seconds())
		return false, r, "", nil
	} else {
		impl.logger.Errorw("error in updating repository", "err", err, "location", url, "error msg", errMsg)
		middleware.GitPullDuration.WithLabelValues("false", "false").Observe(time.Since(start).Seconds())
		return false, r, errMsg, err
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
func (impl *RepositoryManagerImpl) ChangesSinceByRepository(gitCtx GitContext, repository *GitRepository, branch string, from string, to string, count int, checkoutPath string, openNewGitRepo bool) (gitCommits []*GitCommitBase, errMsg string, err error) {
	// fix for azure devops (manual trigger webhook bases pipeline) :
	// branch name comes as 'refs/heads/master', we need to extract actual branch name out of it.
	// https://stackoverflow.com/questions/59956206/how-to-get-a-branch-name-with-a-slash-in-azure-devops

	if count == 0 {
		count = impl.configuration.GitHistoryCount
	}

	if openNewGitRepo {
		repository, err = impl.gitManager.OpenRepoPlain(checkoutPath)
		if err != nil {
			return nil, "", err
		}
	}

	start := time.Now()
	defer func() {
		util.TriggerGitOperationMetrics("changesSinceByRepository", start, err)
	}()
	branch, branchRef := GetBranchReference(branch)
	itr, cliOutput, errMsg, err := impl.gitManager.GetCommitIterator(gitCtx, repository, IteratorRequest{
		BranchRef:      branchRef,
		Branch:         branch,
		CommitCount:    count,
		FromCommitHash: from,
		ToCommitHash:   to,
	})
	if err != nil {
		impl.logger.Errorw("error in getting iterator", "branch", branch, "cliOutput", cliOutput, "errMsg", errMsg, "err", err)
		return nil, errMsg, err
	}
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
			if !commitToFind && strings.HasPrefix(commit.GetCommit().Commit, to) {
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

				statsCheckoutPath := repository.rootDir
				if len(statsCheckoutPath) == 0 {
					//TODO: needed this in case of go-git mode where we are executing cli command
					statsCheckoutPath = checkoutPath
				}
				stats, err := impl.gitManager.GetCommitStats(gitCtx, commit, statsCheckoutPath)
				if err != nil {
					impl.logger.Errorw("error in  fetching stats", "err", err)
				}
				gitCommit.SetFileStats(&stats)
			}
		}()
	}
	return gitCommits, "", err
}

func (impl *RepositoryManagerImpl) TrimLastGitCommit(gitCommits []*GitCommitBase, count int) []*GitCommitBase {
	if len(gitCommits) > count {
		gitCommits = gitCommits[:len(gitCommits)-1]
	}
	return gitCommits
}

func (impl *RepositoryManagerImpl) CreateSshFileIfNotExistsAndConfigureSshCommand(gitCtx GitContext, location string, gitProviderId int, sshPrivateKeyContent string) (string, string, error) {
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
		return sshPrivateKeyPath, "", err
	}

	//git config core.sshCommand
	_, errorMsg, err := impl.gitManager.ConfigureSshCommand(gitCtx, location, sshPrivateKeyPath)
	if err != nil {
		impl.logger.Errorw("error in configuring ssh command while adding repo", "errorMsg", errorMsg, "err", err)
		return sshPrivateKeyPath, errorMsg, err
	}

	return sshPrivateKeyPath, "", nil
}

func (impl *RepositoryManagerImpl) openNewRepo(gitCtx GitContext, location string, url string) (*GitRepository, string, error) {

	var errMsg string
	r, err := impl.gitManager.OpenRepoPlain(location)
	if err != nil {
		err = os.RemoveAll(location)
		if err != nil {
			impl.logger.Errorw("error in cleaning checkout path: %s", "err", err)
			return r, "", err
		}
		errMsg, err = impl.gitManager.Init(gitCtx, location, url, true)
		if err != nil {
			impl.logger.Errorw("err in git init: %s", "err", err)
			return r, errMsg, err
		}
		r, err = impl.gitManager.OpenRepoPlain(location)
		if err != nil {
			impl.logger.Errorw("err in git init: %s", "err", err)
			return r, "", err
		}
	}
	return r, "", nil
}
