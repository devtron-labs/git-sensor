package git

import (
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
)

type GitManager interface {
	GitManagerBase
	GetCommitStats(commit GitCommit) (FileStats, error)
	GetCommitIterator(repository *GitRepository, iteratorRequest IteratorRequest) (CommitIterator, error)
	GetCommitForHash(checkoutPath, commitHash string) (GitCommit, error)
	GetCommitsForTag(checkoutPath, tag string) (GitCommit, error)
	OpenRepoPlain(checkoutPath string) (*GitRepository, error)
	Init(rootDir string, remoteUrl string, isBare bool) error
}

// GitManagerBase Base methods which will be available to all implementation of the parent interface
type GitManagerBase interface {
	PathMatcher(fileStats *FileStats, gitMaterial *sql.GitMaterial) bool
	Fetch(gitContext *GitContext, rootDir string) (response, errMsg string, err error)
	Checkout(rootDir string, branch string) (response, errMsg string, err error)
	ConfigureSshCommand(rootDir string, sshPrivateKeyPath string) (response, errMsg string, err error)
}
type GitManagerBaseImpl struct {
	logger *zap.SugaredLogger
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

func (impl *GitManagerImpl) OpenNewRepo(location string, url string) (*GitRepository, error) {

	r, err := impl.OpenRepoPlain(location)
	if err != nil {
		err = os.RemoveAll(location)
		if err != nil {
			return r, fmt.Errorf("error in cleaning checkout path: %s", err)
		}
		err = impl.Init(location, url, true)
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

func (impl *GitManagerBaseImpl) Fetch(gitContext *GitContext, rootDir string) (response, errMsg string, err error) {
	impl.logger.Debugw("git fetch ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	output, errMsg, err := impl.runCommandWithCred(cmd, gitContext.Username, gitContext.Password)
	impl.logger.Debugw("fetch output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitManagerBaseImpl) Checkout(rootDir string, branch string) (response, errMsg string, err error) {
	impl.logger.Debugw("git checkout ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "checkout", branch, "--force")
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("checkout output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
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

func (impl *GitManagerBaseImpl) ConfigureSshCommand(rootDir string, sshPrivateKeyPath string) (response, errMsg string, err error) {
	impl.logger.Debugw("configuring ssh command on ", "location", rootDir)
	coreSshCommand := fmt.Sprintf("ssh -i %s -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no", sshPrivateKeyPath)
	cmd := exec.Command("git", "-C", rootDir, "config", "core.sshCommand", coreSshCommand)
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
