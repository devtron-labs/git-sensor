package git

import (
	"go.uber.org/zap"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"os"
	"path/filepath"
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
