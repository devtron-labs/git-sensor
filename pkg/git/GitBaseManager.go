package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/devtron-labs/git-sensor/util"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type GitManager interface {
	GitManagerBase
	// GetCommitStats retrieves the stats for the given commit vs its parent
	GetCommitStats(gitCtx GitContext, commit GitCommit) (FileStats, error)
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
	// FetchDiffStatBetweenCommits returns the file stats reponse on executing git action
	FetchDiffStatBetweenCommits(gitCtx GitContext, oldHash string, newHash string, rootDir string) (response, errMsg string, err error)
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
	// LogMergeBase get the commit diff between using a merge base strategy
	LogMergeBase(gitCtx GitContext, rootDir, from string, to string) ([]*Commit, error)
}
type GitManagerBaseImpl struct {
	logger            *zap.SugaredLogger
	conf              *internal.Configuration
	commandTimeoutMap map[string]time.Duration
}

func NewGitManagerBaseImpl(logger *zap.SugaredLogger, config *internal.Configuration) *GitManagerBaseImpl {

	commandTimeoutMap, err := parseCmdTimeoutJson(config)
	if err != nil {
		logger.Errorw("error in parsing config", "config", config, "err", err)
	}

	return &GitManagerBaseImpl{logger: logger, conf: config, commandTimeoutMap: commandTimeoutMap}
}

func parseCmdTimeoutJson(config *internal.Configuration) (map[string]time.Duration, error) {
	commandTimeoutMap := make(map[string]time.Duration)
	var err error
	if config.CliCmdTimeoutJson != "" {
		err = json.Unmarshal([]byte(config.CliCmdTimeoutJson), &commandTimeoutMap)
	}
	return commandTimeoutMap, err
}

type GitManagerImpl struct {
	GitManager
}

func NewGitManagerImpl(configuration *internal.Configuration,
	cliGitManager GitCliManager,
	goGitManager GoGitSDKManager) GitManagerImpl {

	if configuration.UseGitCli {
		return GitManagerImpl{cliGitManager}
	}
	return GitManagerImpl{goGitManager}
}

func (impl *GitManagerImpl) OpenNewRepo(gitCtx GitContext, location string, url string) (*GitRepository, error) {

	r, err := impl.OpenRepoPlain(location)
	if err != nil {
		err = os.RemoveAll(location)
		if err != nil {
			return r, fmt.Errorf("error in cleaning checkout path: %s", err)
		}
		err = impl.Init(gitCtx, location, url, true)
		if err != nil {
			return r, fmt.Errorf("err in git init: %s", err)
		}
		r, err = impl.OpenRepoPlain(location)
		if err != nil {
			return r, fmt.Errorf("err in git init: %s", err)
		}
	}
	return r, nil
}

func (impl *GitManagerBaseImpl) Fetch(gitCtx GitContext, rootDir string) (response, errMsg string, err error) {
	impl.logger.Debugw("git fetch ", "location", rootDir)
	cmd, cancel := impl.CreateCmdWithContext(gitCtx, "git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	defer cancel()
	output, errMsg, err := impl.runCommandWithCred(cmd, gitCtx.Username, gitCtx.Password)
	impl.logger.Debugw("fetch output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitManagerBaseImpl) Checkout(gitCtx GitContext, rootDir, branch string) (response, errMsg string, err error) {
	impl.logger.Debugw("git checkout ", "location", rootDir)
	cmd, cancel := impl.CreateCmdWithContext(gitCtx, "git", "-C", rootDir, "checkout", branch, "--force")
	defer cancel()
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("checkout output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitManagerBaseImpl) LogMergeBase(gitCtx GitContext, rootDir, from string, to string) ([]*Commit, error) {
	cmdArgs := []string{"-C", rootDir, "log", from + "^..." + to, "--date=iso-strict", GITFORMAT}
	impl.logger.Debugw("git", cmdArgs)
	cmd, cancel := impl.CreateCmdWithContext(gitCtx, "git", cmdArgs...)
	defer cancel()
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	if err != nil {
		return nil, err
	}
	commits, err := ProcessGitLogOutput(output)
	if err != nil {
		return nil, err
	}
	return commits, nil
}

func (impl *GitManagerBaseImpl) runCommandWithCred(cmd *exec.Cmd, userName, password string) (response, errMsg string, err error) {
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_ASKPASS=%s", GIT_ASK_PASS),
		fmt.Sprintf("GIT_USERNAME=%s", userName),
		fmt.Sprintf("GIT_PASSWORD=%s", password),
	)
	return impl.runCommand(cmd)
}

func (impl *GitManagerBaseImpl) runCommand(cmd *exec.Cmd) (response, errMsg string, err error) {
	cmd.Env = append(cmd.Env, "HOME=/dev/null")
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		impl.logger.Errorw("error in git cli operation", "msg", string(outBytes), "err", err)
		exErr, ok := err.(*exec.ExitError)
		if !ok {
			return "", string(outBytes), err
		}
		if strings.Contains(string(outBytes), AUTHENTICATION_FAILED_ERROR) {
			impl.logger.Errorw("authentication failed", "msg", string(outBytes), "err", err.Error())
			return "", "authentication failed", errors.New("authentication failed")
		}
		errOutput := string(exErr.Stderr)
		return "", errOutput, err
	}
	output := string(outBytes)
	output = strings.TrimSpace(output)
	return output, "", nil
}

func (impl *GitManagerBaseImpl) ConfigureSshCommand(gitCtx GitContext, rootDir string, sshPrivateKeyPath string) (response, errMsg string, err error) {
	impl.logger.Debugw("configuring ssh command on ", "location", rootDir)
	coreSshCommand := fmt.Sprintf("ssh -i %s -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no", sshPrivateKeyPath)
	cmd, cancel := impl.CreateCmdWithContext(gitCtx, "git", "-C", rootDir, "config", "core.sshCommand", coreSshCommand)
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

func (impl *GitManagerBaseImpl) FetchDiffStatBetweenCommits(gitCtx GitContext, oldHash string, newHash string, rootDir string) (response, errMsg string, err error) {
	impl.logger.Debugw("git", "-C", rootDir, "diff", "--numstat", oldHash, newHash)

	if newHash == "" {
		newHash = oldHash
		oldHash = oldHash + "^"
	}
	cmd, cancel := impl.CreateCmdWithContext(gitCtx, "git", "-C", rootDir, "diff", "--numstat", oldHash, newHash)
	defer cancel()

	output, errMsg, err := impl.runCommandWithCred(cmd, gitCtx.Username, gitCtx.Password)
	impl.logger.Debugw("root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitManagerBaseImpl) CreateCmdWithContext(ctx GitContext, name string, arg ...string) (*exec.Cmd, context.CancelFunc) {
	newCtx := ctx.Context
	cancel := func() {}

	//TODO: how to make it generic, currently works because the
	// git command is placed at index 2 for current implementations
	timeout := impl.getCommandTimeout(arg[2])

	if timeout > 0 {
		newCtx, cancel = context.WithTimeout(ctx.Context, timeout*time.Second)
	}
	cmd := exec.CommandContext(newCtx, name, arg...)
	return cmd, cancel
}

func (impl *GitManagerBaseImpl) getCommandTimeout(command string) time.Duration {
	timeout := time.Duration(impl.conf.CliCmdTimeoutGlobal)
	if cmdTimeout, ok := impl.commandTimeoutMap[command]; ok && cmdTimeout > 0 {
		timeout = cmdTimeout
	}
	return timeout
}
