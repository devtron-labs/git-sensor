package git

import (
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
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

const GIT_AKS_PASS = "/git-ask-pass.sh"

func (impl *GitUtil) fetch(rootDir string, username string, password string) (response, errMsg string, err error) {
	impl.logger.Debugw("git fetch ", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "fetch", "origin", "--tags", "--force")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_ASKPASS=%s", GIT_AKS_PASS),
		fmt.Sprintf("GIT_USERNAME=%s", username), // ignored
		fmt.Sprintf("GIT_PASSWORD=%s", password), // this value is used
	)
	cmd.Env = append(cmd.Env, "HOME=/dev/null")
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		exErr, ok := err.(*exec.ExitError)
		if !ok {
			return "", "", err
		}
		errOutput := string(exErr.Stderr)
		return "", errOutput, err
	}
	// Trims off a single newline for user convenience
	output := string(outBytes)
	output = strings.TrimSpace(output)
	impl.logger.Debugw("fetch output", "root", rootDir, "opt", output)
	return output, "", nil
}

func (impl *GitUtil) Init(rootDir string, remoteUrl string) error {

	//-----------------

	err := os.MkdirAll(rootDir, 0755)
	if err != nil {
		return err
	}
	repo, err := git.PlainInit(rootDir, true)
	if err != nil {
		return err
	}
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: git.DefaultRemoteName,
		URLs: []string{remoteUrl},
	})
	return err
}
