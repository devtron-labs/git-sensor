package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type GitContext struct {
	context.Context // Embedding original Go context
	Username        string
	Password        string
	CloningMode     string
}

type GitUtil struct {
	logger *zap.SugaredLogger
	useCli bool
}

func NewGitUtil(logger *zap.SugaredLogger) *GitUtil {
	val := os.Getenv("USE_CLI")
	useCLI, _ := strconv.ParseBool(val)
	return &GitUtil{
		logger: logger,
		useCli: useCLI,
	}
}

const (
	GIT_ASK_PASS                = "/git-ask-pass.sh"
	AUTHENTICATION_FAILED_ERROR = "Authentication failed"
)

func (impl *GitUtil) Fetch(gitContext *GitContext, rootDir string) (response, errMsg string, err error) {
	impl.logger.Debugw("git fetch ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	output, errMsg, err := impl.runCommandWithCred(cmd, gitContext.Username, gitContext.Password)
	impl.logger.Debugw("fetch output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitUtil) Checkout(rootDir string, branch string) (response, errMsg string, err error) {
	impl.logger.Debugw("git checkout ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "checkout", branch, "--force")
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("checkout output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitUtil) runCommandWithCred(cmd *exec.Cmd, userName, password string) (response, errMsg string, err error) {
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_ASKPASS=%s", GIT_ASK_PASS),
		fmt.Sprintf("GIT_USERNAME=%s", userName),
		fmt.Sprintf("GIT_PASSWORD=%s", password),
	)
	return impl.runCommand(cmd)
}

func (impl *GitUtil) runCommand(cmd *exec.Cmd) (response, errMsg string, err error) {
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

func (impl *GitUtil) ConfigureSshCommand(rootDir string, sshPrivateKeyPath string) (response, errMsg string, err error) {
	impl.logger.Debugw("configuring ssh command on ", "location", rootDir)
	coreSshCommand := fmt.Sprintf("ssh -i %s -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no", sshPrivateKeyPath)
	cmd := exec.Command("git", "-C", rootDir, "config", "core.sshCommand", coreSshCommand)
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("configure ssh command output ", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitUtil) FetchDiffStatBetweenCommits(gitContext *GitContext, oldHash string, newHash string, rootDir string) (response, errMsg string, err error) {
	impl.logger.Debugw("git diff --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "diff", "--numstat", oldHash, newHash)
	output, errMsg, err := impl.runCommandWithCred(cmd, gitContext.Username, gitContext.Password)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}

func (impl *GitUtil) GitInit(rootDir string) error {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "init")
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return err
}

func (impl *GitUtil) GitCreateRemote(rootDir string, url string) error {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "remote", "add", "origin", url)
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git remote add output", "url", url, "opt", output, "errMsg", errMsg, "error", err)
	return err
}

func (impl *GitUtil) GetCommits(branchRef string, branch string, rootDir string, numCommits int) (*CommitIterator, error) {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "log", branchRef, "-n", string(rune(numCommits)), "--date=iso-strict", GITFORMAT)
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	commits, err := impl.processGitLogOutput(output, rootDir)
	if err != nil {
		return nil, err
	}
	return &CommitIterator{
		useCLI:  true,
		commits: commits,
	}, nil
}

func (impl *GitUtil) GitShow(rootDir string, hash string) (*GitCommit, error) {
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

func (impl *GitUtil) processGitLogOutput(out string, rootDir string) ([]*GitCommit, error) {
	logOut := out
	logOut = logOut[:len(logOut)-1]      // Remove the last ","
	logOut = fmt.Sprintf("[%s]", logOut) // Add []

	var gitCommitFormattedList []GitCommitFormat
	err := json.Unmarshal([]byte(logOut), &gitCommitFormattedList)
	if err != nil {
		return nil, err
	}

	gitCommits := make([]*GitCommit, 0)
	for _, formattedCommit := range gitCommitFormattedList {
		gitCommits = append(gitCommits, &GitCommit{
			Commit:       formattedCommit.Commit,
			Author:       formattedCommit.Commiter.Name + formattedCommit.Commiter.Email,
			Date:         formattedCommit.Commiter.Date,
			Message:      formattedCommit.Subject + formattedCommit.Body,
			useCLI:       true,
			gitUtil:      impl,
			CheckoutPath: rootDir,
		})
	}
	return gitCommits, nil
}
