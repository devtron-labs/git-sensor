package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"go.uber.org/zap"
	"os"
)

type GoGitSDKManagerImpl struct {
	GitManagerBase
	logger *zap.SugaredLogger
}

func NewGoGitSDKManagerImpl(baseManager GitManagerBase, logger *zap.SugaredLogger) *GoGitSDKManagerImpl {
	return &GoGitSDKManagerImpl{
		GitManagerBase: baseManager,
		logger:         logger,
	}
}

func (impl *GoGitSDKManagerImpl) GetCommitsForTag(gitCtx GitContext, checkoutPath, tag string) (GitCommit, error) {

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
		Cm:            commit,
	}

	return gitCommit, nil
}

func (impl *GoGitSDKManagerImpl) GetCommitForHash(gitCtx GitContext, checkoutPath, commitHash string) (GitCommit, error) {
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
		Cm:            commit,
	}
	return gitCommit, nil
}

func (impl *GoGitSDKManagerImpl) GetCommitIterator(gitCtx GitContext, repository *GitRepository, iteratorRequest IteratorRequest) (CommitIterator, error) {

	ref, err := repository.Reference(plumbing.ReferenceName(iteratorRequest.BranchRef), true)
	if err != nil && err == plumbing.ErrReferenceNotFound {
		return nil, fmt.Errorf("ref not found %s branch  %s", err, iteratorRequest.Branch)
	} else if err != nil {
		return nil, fmt.Errorf("error in getting reference %s branch  %s", err, iteratorRequest.Branch)
	}
	itr, err := repository.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, fmt.Errorf("error in getting iterator %s branch  %s", err, iteratorRequest.Branch)
	}
	return &CommitGoGitIterator{itr}, nil
}

func (impl *GoGitSDKManagerImpl) OpenRepoPlain(checkoutPath string) (*GitRepository, error) {

	r, err := git.PlainOpen(checkoutPath)
	if err != nil {
		impl.logger.Errorf("error in OpenRepoPlain go-git %s for path %s", err, checkoutPath)
		return nil, err
	}
	return &GitRepository{Repository: r}, err
}

func (impl *GoGitSDKManagerImpl) Init(gitCtx GitContext, rootDir string, remoteUrl string, isBare bool) error {
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

func (impl *GoGitSDKManagerImpl) GetCommitStats(gitCtx GitContext, commit GitCommit, checkoutPath string) (FileStats, error) {
	if IsShallowCloningEnabled(checkoutPath) {
		return impl.GitManagerBase.GetCommitStatsViaCli(gitCtx, commit, checkoutPath)
	}
	gitCommit := commit.(*GitCommitGoGit)

	stats, err := gitCommit.Cm.Stats()
	if err != nil {
		return nil, err
	}
	return transformFileStats(stats), err
}
