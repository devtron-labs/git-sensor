package git

import (
	"fmt"
	"github.com/avdkp/go-git"
	"github.com/avdkp/go-git/config"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strings"
)

type GitUtil struct {
	logger *zap.SugaredLogger
}

func NewGitUtil(logger *zap.SugaredLogger) *GitUtil {
	return &GitUtil{
		logger: logger,
	}
}

const GIT_ASK_PASS = "/git-ask-pass.sh"

func (impl *GitUtil) Fetch(rootDir string, username string, password string) (response, errMsg string, err error) {
	impl.logger.Debugw("git fetch ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	output, errMsg, err := impl.runCommandWithCred(cmd, username, password)
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
		errOutput := string(exErr.Stderr)
		return "", errOutput, err
	}
	output := string(outBytes)
	output = strings.TrimSpace(output)
	return output, "", nil
}

func (impl *GitUtil) Init(rootDir string, remoteUrl string, isBare bool) error {
	//-----------------

	err := os.MkdirAll(rootDir, 0755)
	if err != nil {
		return err
	}
	repo, err := git.PlainInit(rootDir, isBare)
	if err != nil {
		return err
	}
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: git.DefaultRemoteName,
		URLs: []string{remoteUrl},
	})
	return err
}

func (impl *GitUtil) ConfigureSshCommand(rootDir string, sshPrivateKeyPath string) (response, errMsg string, err error) {
	impl.logger.Debugw("configuring ssh command on ", "location", rootDir)
	coreSshCommand := fmt.Sprintf("ssh -i %s -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no", sshPrivateKeyPath)
	cmd := exec.Command("git", "-C", rootDir, "config", "core.sshCommand", coreSshCommand)
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("configure ssh command output ", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}
