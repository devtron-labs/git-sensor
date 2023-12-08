package git

import (
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"os"
)

type GoGitManager interface {
	GitManager
}
type GoGitManagerImpl struct {
	GitManagerBaseImpl
	logger *zap.SugaredLogger
}

func NewGoGitManagerImpl(logger *zap.SugaredLogger) *GoGitManagerImpl {
	return &GoGitManagerImpl{
		GitManagerBaseImpl: GitManagerBaseImpl{logger: logger},
	}
}

func (impl *GoGitManagerImpl) GetCommitsForTag(checkoutPath, tag string) (GitCommit, error) {

	r, err := impl.OpenRepoPlain(checkoutPath)
	if err != nil {
		return nil, err
	}

	tagRef, err := r.Tag(tag)
	if err != nil {
		impl.logger.Errorw("error in fetching tag", "path", checkoutPath, "tag", tag, "err", err)
		return nil, err
	}
	commit, err := r.CommitObject(plumbing.NewHash(tagRef.Hash().String()))
	if err != nil {
		impl.logger.Errorw("error in fetching tag", "path", checkoutPath, "hash", tagRef, "err", err)
		return nil, err
	}
	cm := GitCommitBase{
		Author:  commit.Author.String(),
		Commit:  commit.Hash.String(),
		Date:    commit.Author.When,
		Message: commit.Message,
	}

	gitCommit := &GitCommitGoGit{
		GitCommitBase: cm,
		cm:            commit,
	}

	return gitCommit, nil
}

func (impl *GoGitManagerImpl) GetCommitForHash(checkoutPath, commitHash string) (GitCommit, error) {
	r, err := impl.OpenRepoPlain(checkoutPath)
	if err != nil {
		return nil, err
	}

	commit, err := r.CommitObject(plumbing.NewHash(commitHash))
	if err != nil {
		impl.logger.Errorw("error in fetching commit", "path", checkoutPath, "hash", commitHash, "err", err)
		return nil, err
	}
	cm := GitCommitBase{
		Author:  commit.Author.String(),
		Commit:  commit.Hash.String(),
		Date:    commit.Author.When,
		Message: commit.Message,
	}

	gitCommit := &GitCommitGoGit{
		GitCommitBase: cm,
		cm:            commit,
	}
	return gitCommit, nil
}

func (impl *GoGitManagerImpl) GetCommitIterator(repository *GitRepository, branchRef string, branch string) (CommitIterator, error) {

	ref, err := repository.Reference(plumbing.ReferenceName(branchRef), true)
	if err != nil && err == plumbing.ErrReferenceNotFound {
		return nil, fmt.Errorf("ref not found %s branch  %s", err, branch)
	} else if err != nil {
		return nil, fmt.Errorf("error in getting reference %s branch  %s", err, branch)
	}
	itr, err := repository.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("error in getting iterator %s branch  %s", err, branch)
	}
	return CommitIteratorGoGit{itr}, nil
}

func (impl *GoGitManagerImpl) OpenRepoPlain(checkoutPath string) (*GitRepository, error) {

	r, err := git.PlainOpen(checkoutPath)
	return &GitRepository{Repository: *r}, err
}

func (impl *GoGitManagerImpl) Init(rootDir string, remoteUrl string, isBare bool) error {
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
