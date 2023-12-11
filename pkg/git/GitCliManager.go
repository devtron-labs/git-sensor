package git

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type CliGitManager interface {
	GitManager
}

type CliGitManagerImpl struct {
	GitManagerBaseImpl
}

func NewCliGitManagerImpl(logger *zap.SugaredLogger) *CliGitManagerImpl {
	return &CliGitManagerImpl{
		GitManagerBaseImpl: GitManagerBaseImpl{logger: logger},
	}
}

const (
	GIT_ASK_PASS                = "/git-ask-pass.sh"
	AUTHENTICATION_FAILED_ERROR = "Authentication failed"
)

func (impl *CliGitManagerImpl) Init(rootDir string, remoteUrl string, isBare bool) error {
	//-----------------

	err := os.MkdirAll(rootDir, 0755)
	if err != nil {
		return err
	}

	err = impl.GitInit(rootDir)
	if err != nil {
		return err
	}
	return impl.GitCreateRemote(rootDir, remoteUrl)

}

func (impl *CliGitManagerImpl) OpenRepoPlain(checkoutPath string) (*GitRepository, error) {

	err := openGitRepo(checkoutPath)
	if err != nil {
		return nil, err
	}
	return &GitRepository{
		rootDir: checkoutPath,
	}, nil
}

func (impl *CliGitManagerImpl) GetCommitsForTag(checkoutPath, tag string) (GitCommit, error) {
	return impl.GitShow(checkoutPath, tag)
}

func (impl *CliGitManagerImpl) GetCommitForHash(checkoutPath, commitHash string) (GitCommit, error) {

	return impl.GitShow(checkoutPath, commitHash)
}
func (impl *CliGitManagerImpl) GetCommitIterator(repository *GitRepository, branchRef string, branch string) (CommitIterator, error) {

	return impl.GetCommits(branchRef, branch, repository.rootDir, repository.commitCount)
}

func openGitRepo(path string) error {
	if _, err := filepath.Abs(path); err != nil {
		return err
	}
	fst := osfs.New(path)
	_, err := fst.Stat(".git")
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}
func (impl *CliGitManagerImpl) GitInit(rootDir string) error {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "init")
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return err
}

func (impl *CliGitManagerImpl) GitCreateRemote(rootDir string, url string) error {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "remote", "add", "origin", url)
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git remote add output", "url", url, "opt", output, "errMsg", errMsg, "error", err)
	return err
}

func (impl *CliGitManagerImpl) GetCommits(branchRef string, branch string, rootDir string, numCommits int) (CommitIterator, error) {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	//cmd := exec.Command("git", "-C", rootDir, "log", branchRef, "-n", string(rune(numCommits)), "--date=iso-strict", GITFORMAT)
	cmd := exec.Command("git", "-C", rootDir, "log", branchRef, "-n", strconv.Itoa(numCommits), "--date=iso-strict", GITFORMAT)
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

func (impl *CliGitManagerImpl) GitShow(rootDir string, hash string) (GitCommit, error) {
	//impl.logger.Debugw("git log --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "show", hash, "--date=iso-strict", GITFORMAT, "-s")
	output, errMsg, err := impl.runCommand(cmd)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	commits, err := impl.processGitLogOutput(output, rootDir)
	if err != nil || len(commits) == 0 {
		return nil, err
	}

	return commits[0], nil
}

func (impl *CliGitManagerImpl) processGitLogOutput(out string, rootDir string) ([]GitCommit, error) {
	if len(out) == 0 {
		return nil, fmt.Errorf("no output to process")
	}
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
			Author:       formattedCommit.Commiter.Name + " <" + formattedCommit.Commiter.Email + ">",
			Date:         formattedCommit.Commiter.Date,
			Message:      formattedCommit.Subject + "\n" + formattedCommit.Body,
			CheckoutPath: rootDir,
		}
		gitCommits = append(gitCommits, &GitCommitCli{
			GitCommitBase: cm,
			impl:          impl,
		})
	}
	return gitCommits, nil
}

func (impl *GitManagerBaseImpl) FetchDiffStatBetweenCommits(gitContext *GitContext, oldHash string, newHash string, rootDir string) (response, errMsg string, err error) {
	impl.logger.Debugw("git diff --numstat", "location", rootDir)
	cmd := exec.Command("git", "-C", rootDir, "diff", "--numstat", oldHash, newHash)
	output, errMsg, err := impl.runCommandWithCred(cmd, gitContext.Username, gitContext.Password)
	impl.logger.Debugw("git diff --stat output", "root", rootDir, "opt", output, "errMsg", errMsg, "error", err)
	return output, errMsg, err
}
