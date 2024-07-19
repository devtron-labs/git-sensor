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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	commonLibGitManager "github.com/devtron-labs/common-lib/git-manager"
	"github.com/devtron-labs/git-sensor/internals"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"github.com/devtron-labs/git-sensor/util"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type GitManager interface {
	GitManagerBase
	// GetCommitStats retrieves the stats for the given commit vs its parent
	GetCommitStats(gitCtx GitContext, commit GitCommit, checkoutPath string) (FileStats, error)
	// GetCommitIterator returns an iterator for the provided git repo and iterator request describing the commits to fetch
	GetCommitIterator(gitCtx GitContext, repository *GitRepository, iteratorRequest IteratorRequest) (CommitIterator, error)
	// GetCommitForHash retrieves the commit reference for given tag
	GetCommitForHash(gitCtx GitContext, checkoutPath, commitHash string) (GitCommit, error)
	// GetCommitsForTag retrieves the commit reference for given tag
	GetCommitsForTag(gitCtx GitContext, checkoutPath, tag string) (GitCommit, error)
	// OpenRepoPlain opens a new git repo at the given path
	OpenRepoPlain(checkoutPath string) (*GitRepository, error)
	// Init initializes a git repo
	Init(gitCtx GitContext, rootDir string, remoteUrl string, isBare bool) error
}

// GitManagerBase Base methods which will be available to all implementation of the parent interface
type GitManagerBase interface {
	// PathMatcher matches paths of files changes with defined regex expression
	PathMatcher(fileStats *FileStats, gitMaterial *sql.GitMaterial) bool
	// Fetch executes git fetch
	Fetch(gitCtx GitContext, rootDir string) (response, errMsg string, err error)
	// Checkout executes git checkout
	Checkout(gitCtx GitContext, rootDir, branch string) (response, errMsg string, err error)
	// ConfigureSshCommand configures ssh in git repo
	ConfigureSshCommand(gitCtx GitContext, rootDir string, sshPrivateKeyPath string) (response, errMsg string, err error)
	//  FetchDiffStatBetweenCommitsNameOnly returns the list of files changed in reponse on executing git action
	FetchDiffStatBetweenCommitsNameOnly(gitCtx GitContext, oldHash string, newHash string, rootDir string) (FileStats, error)
	//  FetchDiffStatBetweenCommitsWithNumstat returns the file stats reponse on executing git action
	FetchDiffStatBetweenCommitsWithNumstat(gitCtx GitContext, oldHash string, newHash string, rootDir string) (FileStats, error)
	// LogMergeBase get the commit diff between using a merge base strategy
	LogMergeBase(gitCtx GitContext, rootDir, from string, to string) ([]*Commit, error)
	ExecuteCustomCommand(gitContext GitContext, name string, arg ...string) (response, errMsg string, err error)
}
type GitManagerBaseImpl struct {
	logger            *zap.SugaredLogger
	conf              *internals.Configuration
	commandTimeoutMap map[string]int
}

func NewGitManagerBaseImpl(logger *zap.SugaredLogger, config *internals.Configuration) *GitManagerBaseImpl {

	commandTimeoutMap, err := parseCmdTimeoutJson(config)
	if err != nil {
		logger.Errorw("error in parsing config", "config", config, "err", err)
	}

	return &GitManagerBaseImpl{logger: logger, conf: config, commandTimeoutMap: commandTimeoutMap}
}

type GitManagerImpl struct {
	GitManager
}

func NewGitManagerImpl(logger *zap.SugaredLogger, configuration *internals.Configuration) *GitManagerImpl {

	baseImpl := NewGitManagerBaseImpl(logger, configuration)
	if configuration.UseGitCli {
		return &GitManagerImpl{
			GitManager: NewGitCliManagerImpl(baseImpl, logger),
		}

	}
	return &GitManagerImpl{
		GitManager: NewGoGitSDKManagerImpl(baseImpl, logger),
	}
}

func parseCmdTimeoutJson(config *internals.Configuration) (map[string]int, error) {
	commandTimeoutMap := make(map[string]int)
	var err error
	if config.CliCmdTimeoutJson != "" {
		err = json.Unmarshal([]byte(config.CliCmdTimeoutJson), &commandTimeoutMap)
	}
	return commandTimeoutMap, err
}

func (impl *GitManagerBaseImpl) Fetch(gitCtx GitContext, rootDir string) (response, errMsg string, err error) {
	impl.logger.Debugw("git fetch ", "location", rootDir)
	cmd, cancel := impl.createCmdWithContext(gitCtx, "git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	defer cancel()
	tlsPathInfo, err := commonLibGitManager.CreateFilesForTlsData(commonLibGitManager.BuildTlsData(gitCtx.TLSKey, gitCtx.TLSCertificate, gitCtx.CACert, gitCtx.TLSVerificationEnabled), TLS_FILES_DIR)
	if err != nil {
		//making it non-blocking
		impl.logger.Errorw("error encountered in createFilesForTlsData", "err", err)
	}
	defer commonLibGitManager.DeleteTlsFiles(tlsPathInfo)
	output, errMsg, err := impl.runCommandWithCred(cmd, gitCtx.Username, gitCtx.Password, tlsPathInfo)
	if strings.Contains(output, LOCK_REF_MESSAGE) {
		impl.logger.Info("error in fetch, pruning local refs and retrying", "rootDir", rootDir)
		// running git remote prune origin and retrying fetch. gitHub issue - https://github.com/devtron-labs/devtron/issues/4605
		pruneCmd, pruneCmdCancel := impl.createCmdWithContext(gitCtx, "git", "-C", rootDir, "remote", "prune", "origin")
		pruneOutput, pruneMsg, pruneErr := impl.runCommandWithCred(pruneCmd, gitCtx.Username, gitCtx.Password, tlsPathInfo)
		defer pruneCmdCancel()
		if pruneErr != nil {
			impl.logger.Errorw("error in pruning local refs that do not exist at remote")
			return pruneOutput, pruneMsg, pruneErr
		}

		retryFetchCmd, retryFetchCancel := impl.createCmdWithContext(gitCtx, "git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
		defer retryFetchCancel()

		output, errMsg, err = impl.runCommandWithCred(retryFetchCmd, gitCtx.Username, gitCtx.Password, tlsPathInfo)
	}
	impl.logger.Debugw("fetch output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitManagerBaseImpl) Checkout(gitCtx GitContext, rootDir, branch string) (response, errMsg string, err error) {
	impl.logger.Debugw("git checkout ", "location", rootDir)
	cmd, cancel := impl.createCmdWithContext(gitCtx, "git", "-C", rootDir, "checkout", branch, "--force")
	defer cancel()
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("checkout output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitManagerBaseImpl) LogMergeBase(gitCtx GitContext, rootDir, from string, to string) ([]*Commit, error) {

	//this is a safe check to handle empty `to` hash given to request
	// go-git implementation breaks for invalid `to` hashes
	var toCommitHash string
	if len(to) != 0 {
		toCommitHash = to + "^"
	}
	cmdArgs := []string{"-C", rootDir, "log", from + "..." + toCommitHash, "--date=iso-strict", GITFORMAT}
	impl.logger.Debugw("git", cmdArgs)
	cmd, cancel := impl.createCmdWithContext(gitCtx, "git", cmdArgs...)
	defer cancel()
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	if err != nil {
		return nil, err
	}
	commits, err := processGitLogOutputForAnalytics(output)
	if err != nil {
		impl.logger.Errorw("error in parsing log output", "err", err, "output", output)
		return nil, err
	}
	return commits, nil
}

func (impl *GitManagerBaseImpl) runCommandWithCred(cmd *exec.Cmd, userName, password string, tlsPathInfo *commonLibGitManager.TlsPathInfo) (response, errMsg string, err error) {
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_ASKPASS=%s", GIT_ASK_PASS),
		fmt.Sprintf("GIT_USERNAME=%s", userName),
		fmt.Sprintf("GIT_PASSWORD=%s", password),
	)
	if tlsPathInfo != nil {
		if tlsPathInfo.TlsKeyPath != "" && tlsPathInfo.TlsCertPath != "" {
			cmd.Env = append(cmd.Env,
				fmt.Sprintf("GIT_SSL_KEY=%s", tlsPathInfo.TlsKeyPath),
				fmt.Sprintf("GIT_SSL_CERT=%s", tlsPathInfo.TlsCertPath))
		}
		if tlsPathInfo.CaCertPath != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_SSL_CAINFO=%s", tlsPathInfo.CaCertPath))
		}
	}
	return impl.runCommand(cmd)
}

func (impl *GitManagerBaseImpl) runCommand(cmd *exec.Cmd) (response, errMsg string, err error) {
	cmd.Env = append(cmd.Env, "HOME=/dev/null")
	outBytes, err := cmd.CombinedOutput()
	output := string(outBytes)
	output = strings.TrimSpace(output)
	if err != nil {
		impl.logger.Errorw("error in git cli operation", "msg", string(outBytes), "err", err)
		exErr, ok := err.(*exec.ExitError)
		if !ok {
			return output, string(outBytes), err
		}
		if strings.Contains(output, AUTHENTICATION_FAILED_ERROR) {
			impl.logger.Errorw("authentication failed", "msg", string(outBytes), "err", err.Error())
			return output, "authentication failed", errors.New("authentication failed")
		}
		errOutput := string(exErr.Stderr)
		return output, errOutput, err
	}
	return output, "", nil
}

func (impl *GitManagerBaseImpl) ConfigureSshCommand(gitCtx GitContext, rootDir string, sshPrivateKeyPath string) (response, errMsg string, err error) {
	impl.logger.Debugw("configuring ssh command on ", "location", rootDir)
	coreSshCommand := fmt.Sprintf("ssh -i %s -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no", sshPrivateKeyPath)
	cmd, cancel := impl.createCmdWithContext(gitCtx, "git", "-C", rootDir, "config", "core.sshCommand", coreSshCommand)
	defer cancel()
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("configure ssh command output ", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitManagerBaseImpl) PathMatcher(fileStats *FileStats, gitMaterial *sql.GitMaterial) bool {
	excluded := false
	var changesInPath []string
	var pathsForFilter []string
	if len(gitMaterial.FilterPattern) == 0 {
		impl.logger.Debugw("no filter configured for this git material", "gitMaterial", gitMaterial)
		return excluded
	}
	for _, path := range gitMaterial.FilterPattern {
		regex := util.GetPathRegex(path)
		pathsForFilter = append(pathsForFilter, regex)
	}
	pathsForFilter = util.ReverseSlice(pathsForFilter)
	impl.logger.Debugw("pathMatcher............", "pathsForFilter", pathsForFilter)
	fileStatBytes, err := json.Marshal(fileStats)
	if err != nil {
		impl.logger.Errorw("marshal error ............", "err", err)
		return false
	}
	var fileChanges []map[string]interface{}
	if err := json.Unmarshal(fileStatBytes, &fileChanges); err != nil {
		impl.logger.Errorw("unmarshal error ............", "err", err)
		return false
	}
	for _, fileChange := range fileChanges {
		path := fileChange["Name"].(string)
		changesInPath = append(changesInPath, path)
	}
	len := len(pathsForFilter)
	for i, filter := range pathsForFilter {
		isExcludeFilter := false
		isMatched := false
		//TODO - handle ! in file name with /!
		const ExcludePathIdentifier = "!"
		if strings.Contains(filter, ExcludePathIdentifier) {
			filter = strings.Replace(filter, ExcludePathIdentifier, "", 1)
			isExcludeFilter = true
		}
		for _, path := range changesInPath {
			match, err := regexp.MatchString(filter, path)
			if err != nil {
				continue
			}
			if match {
				isMatched = true
				break
			}
		}
		if isMatched {
			if isExcludeFilter {
				//if matched for exclude filter
				excluded = true
			} else {
				//if matched for include filter
				excluded = false
			}
			return excluded
		} else if i == len-1 {
			//if it's a last item
			if isExcludeFilter {
				excluded = false
			} else {
				excluded = true
			}
			return excluded
		} else {
			//GO TO THE NEXT FILTER
		}
	}

	return excluded
}

func GetBranchReference(branch string) (string, string) {
	if strings.HasPrefix(branch, "refs/heads/") {
		branch = strings.ReplaceAll(branch, "refs/heads/", "")
	}

	branchRef := fmt.Sprintf("refs/remotes/origin/%s", branch)
	return branch, branchRef
}

func (impl *GitManagerBaseImpl) FetchDiffStatBetweenCommitsNameOnly(gitCtx GitContext, oldHash string, newHash string, rootDir string) (FileStats, error) {
	impl.logger.Debugw("git", "-C", rootDir, "diff", "--name-only", oldHash, newHash)

	if newHash == "" {
		newHash = oldHash
		oldHash = oldHash + "^"
	}
	cmd, cancel := impl.createCmdWithContext(gitCtx, "git", "-C", rootDir, "diff", "--name-only", oldHash, newHash)

	tlsPathInfo, err := commonLibGitManager.CreateFilesForTlsData(commonLibGitManager.BuildTlsData(gitCtx.TLSKey, gitCtx.TLSCertificate, gitCtx.CACert, gitCtx.TLSVerificationEnabled), TLS_FILES_DIR)
	if err != nil {
		//making it non-blocking
		impl.logger.Errorw("error encountered in createFilesForTlsData", "err", err)
	}

	defer func() {
		cancel()
		commonLibGitManager.DeleteTlsFiles(tlsPathInfo)
	}()

	output, errMsg, err := impl.runCommandWithCred(cmd, gitCtx.Username, gitCtx.Password, tlsPathInfo)
	impl.logger.Debugw("root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	if err != nil || len(errMsg) > 0 {
		impl.logger.Errorw("error in fetching fileStat diff btw commits: ", "oldHash", oldHash, "newHash", newHash, "checkoutPath", rootDir, "errorMsg", errMsg, "err", err)
		return nil, err
	}
	return processFileStatOutputNameOnly(output)
}

func (impl *GitManagerBaseImpl) FetchDiffStatBetweenCommitsWithNumstat(gitCtx GitContext, oldHash string, newHash string, rootDir string) (FileStats, error) {
	impl.logger.Debugw("git", "-C", rootDir, "diff", "--numstat", oldHash, newHash)

	if newHash == "" {
		newHash = oldHash
		oldHash = oldHash + "^"
	}
	cmd, cancel := impl.createCmdWithContext(gitCtx, "git", "-C", rootDir, "diff", "--numstat", oldHash, newHash)
	defer cancel()
	tlsPathInfo, err := commonLibGitManager.CreateFilesForTlsData(commonLibGitManager.BuildTlsData(gitCtx.TLSKey, gitCtx.TLSCertificate, gitCtx.CACert, gitCtx.TLSVerificationEnabled), TLS_FILES_DIR)
	if err != nil {
		//making it non-blocking
		impl.logger.Errorw("error encountered in createFilesForTlsData", "err", err)
	}
	defer commonLibGitManager.DeleteTlsFiles(tlsPathInfo)
	output, errMsg, err := impl.runCommandWithCred(cmd, gitCtx.Username, gitCtx.Password, tlsPathInfo)
	impl.logger.Debugw("root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	if err != nil || len(errMsg) > 0 {
		impl.logger.Errorw("error in fetching fileStat diff btw commits: ", "oldHash", oldHash, "newHash", newHash, "checkoutPath", rootDir, "errorMsg", errMsg, "err", err)
		return nil, err
	}
	return processFileStatOutputWithNumstat(output)
}

func (impl *GitManagerBaseImpl) createCmdWithContext(ctx GitContext, name string, arg ...string) (*exec.Cmd, context.CancelFunc) {
	newCtx := ctx
	cancel := func() {}

	//TODO: how to make it generic, currently works because the
	// git command is placed at index 2 for current implementations
	timeout := impl.getCommandTimeout(arg[2])

	if timeout > 0 {

		newCtx, cancel = ctx.WithTimeout(timeout) //context.WithTimeout(ctx.Context, timeout*time.Second)
	}
	cmd := exec.CommandContext(newCtx, name, arg...)
	return cmd, cancel
}

func (impl *GitManagerBaseImpl) getCommandTimeout(command string) int {
	timeout := impl.conf.CliCmdTimeoutGlobal
	if cmdTimeout, ok := impl.commandTimeoutMap[command]; ok {
		timeout = cmdTimeout
	}
	return timeout
}

func (impl *GitManagerBaseImpl) ExecuteCustomCommand(gitContext GitContext, name string, arg ...string) (response, errMsg string, err error) {
	cmd, cancel := impl.createCmdWithContext(gitContext, name, arg...)
	defer cancel()
	tlsPathInfo, err := commonLibGitManager.CreateFilesForTlsData(commonLibGitManager.BuildTlsData(gitContext.TLSKey, gitContext.TLSCertificate, gitContext.CACert, gitContext.TLSVerificationEnabled), TLS_FILES_DIR)
	if err != nil {
		//making it non-blocking
		impl.logger.Errorw("error encountered in createFilesForTlsData", "err", err)
	}
	defer commonLibGitManager.DeleteTlsFiles(tlsPathInfo)
	output, errMsg, err := impl.runCommandWithCred(cmd, gitContext.Username, gitContext.Password, tlsPathInfo)
	return output, errMsg, err
}
