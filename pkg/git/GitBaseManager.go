package git

import (
	"encoding/json"
	"errors"
	"fmt"
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
	GetCommitIterator(repository *GitRepository, branchRef string, branch string) (CommitIterator, error)
	GetCommitForHash(checkoutPath, commitHash string) (GitCommit, error)
	GetCommitsForTag(checkoutPath, tag string) (GitCommit, error)
	OpenRepoPlain(checkoutPath string) (*GitRepository, error)
	Init(rootDir string, remoteUrl string, isBare bool) error
}

type GitManagerBase interface {
	PathMatcher(fileStats *FileStats, gitMaterial *sql.GitMaterial) bool
	Fetch(gitContext *GitContext, rootDir string) (response, errMsg string, err error)
	Checkout(rootDir string, branch string) (response, errMsg string, err error)
	ConfigureSshCommand(rootDir string, sshPrivateKeyPath string) (response, errMsg string, err error)
}
type GitManagerBaseImpl struct {
	logger *zap.SugaredLogger
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

func (impl *GitManagerBaseImpl) FetchDiffStatBetweenCommits(gitContext *GitContext, oldHash string, newHash string, rootDir string) (response, errMsg string, err error) {
	impl.logger.Debugw("git diff --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "diff", "--numstat", oldHash, newHash)
	output, errMsg, err := impl.runCommandWithCred(cmd, gitContext.Username, gitContext.Password)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitManagerBaseImpl) GitInit(rootDir string) error {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "init")
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return err
}

func (impl *GitManagerBaseImpl) GitCreateRemote(rootDir string, url string) error {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "remote", "add", "origin", url)
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git remote add output", "url", url, "opt", output, "errMsg", errMsg, "error", err)
	return err
}

func (impl *GitManagerBaseImpl) GetCommits(branchRef string, branch string, rootDir string, numCommits int) (CommitIterator, error) {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "log", branchRef, "-n", string(rune(numCommits)), "--date=iso-strict", GITFORMAT)
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	commits, err := impl.processGitLogOutput(output, rootDir)
	if err != nil {
		return nil, err
	}
	return CommitIteratorCli{
		commits: commits,
	}, nil
}

func (impl *GitManagerBaseImpl) GitShow(rootDir string, hash string) (GitCommit, error) {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "show", hash, GITFORMAT)
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	commits, err := impl.processGitLogOutput(output, rootDir)
	if err != nil {
		return nil, err
	}
	return commits[0], nil
}

func (impl *GitManagerBaseImpl) processGitLogOutput(out string, rootDir string) ([]GitCommit, error) {
	logOut := out
	logOut = logOut[:len(logOut)-1]      // Remove the last ","
	logOut = fmt.Sprintf("[%s]", logOut) // Add []

	var gitCommitFormattedList []GitCommitFormat
	err := json.Unmarshal([]byte(logOut), &gitCommitFormattedList)
	if err != nil {
		return nil, err
	}

	gitCommits := make([]GitCommit, 0)
	for _, formattedCommit := range gitCommitFormattedList {

		cm := GitCommitBase{
			Commit:       formattedCommit.Commit,
			Author:       formattedCommit.Commiter.Name + formattedCommit.Commiter.Email,
			Date:         formattedCommit.Commiter.Date,
			Message:      formattedCommit.Subject + formattedCommit.Body,
			CheckoutPath: rootDir,
		}
		gitCommits = append(gitCommits, &GitCommitCli{
			GitCommitBase: cm,
			impl:          impl,
		})
	}
	return gitCommits, nil
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
