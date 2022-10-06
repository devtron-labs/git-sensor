/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package git

import (
	"context"
	"fmt"
	"github.com/devtron-labs/git-sensor/internal"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/devtron-labs/git-sensor/internal/middleware"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

type RepositoryManager interface {
	Fetch(userName, password string, url string, location string) (updated bool, repo *git.Repository, err error)
	Add(gitProviderId int, location, url string, userName, password string, authMode sql.AuthMode, sshPrivateKeyContent string) error
	Clean(cloneDir string) error
	ChangesSince(checkoutPath string, branch string, from string, to string, count int) ([]*GitCommit, error)
	ChangesSinceByRepository(repository *git.Repository, branch string, from string, to string, count int) ([]*GitCommit, error)
	GetCommitMetadata(checkoutPath, commitHash string) (*GitCommit, error)
	ChangesSinceByRepositoryForAnalytics(checkoutPath string, branch string, Old string, New string) (*GitChanges, error)
	GetCommitForTag(checkoutPath, tag string) (*GitCommit, error)
	CreateSshFileIfNotExistsAndConfigureSshCommand(location string, gitProviderId int, sshPrivateKeyContent string) error
}

type RepositoryManagerImpl struct {
	logger        *zap.SugaredLogger
	gitUtil       *GitUtil
	configuration *internal.Configuration
}

func NewRepositoryManagerImpl(logger *zap.SugaredLogger, gitUtil *GitUtil, configuration *internal.Configuration) *RepositoryManagerImpl {
	return &RepositoryManagerImpl{logger: logger, gitUtil: gitUtil, configuration: configuration}
}

func (impl RepositoryManagerImpl) Add(gitProviderId int, location string, url string, userName, password string, authMode sql.AuthMode, sshPrivateKeyContent string) error {
	err := os.RemoveAll(location)
	if err != nil {
		impl.logger.Errorw("error in cleaning checkout path", "err", err)
		return err
	}
	err = impl.gitUtil.Init(location, url, true)
	if err != nil {
		impl.logger.Errorw("err in git init", "err", err)
		return err
	}

	// check ssh
	if authMode == sql.AUTH_MODE_SSH {
		err = impl.CreateSshFileIfNotExistsAndConfigureSshCommand(location, gitProviderId, sshPrivateKeyContent)
		if err != nil {
			return err
		}
	}

	opt, errorMsg, err := impl.gitUtil.Fetch(location, userName, password)
	if err != nil {
		impl.logger.Errorw("error in cloning repo", "errorMsg", errorMsg, "err", err)
		return err
	}
	impl.logger.Debugw("opt msg", "opt", opt)
	return nil
}

func (impl RepositoryManagerImpl) Clean(dir string) error {
	err := os.RemoveAll(dir)
	return err
}

func (impl RepositoryManagerImpl) clone(auth transport.AuthMethod, cloneDir string, url string) (*git.Repository, error) {
	timeoutContext, _ := context.WithTimeout(context.Background(), CLONE_TIMEOUT_SEC*time.Second)
	impl.logger.Infow("cloning repository ", "url", url, "cloneDir", cloneDir)
	repo, err := git.PlainCloneContext(timeoutContext, cloneDir, true, &git.CloneOptions{
		URL:  url,
		Auth: auth,
	})
	if err != nil {
		impl.logger.Errorw("error in cloning repo ", "url", url, "err", err)
	} else {
		impl.logger.Infow("repo cloned", "url", url)
	}
	return repo, err
}

func (impl RepositoryManagerImpl) Fetch(userName, password string, url string, location string) (updated bool, repo *git.Repository, err error) {
	start := time.Now()
	middleware.GitMaterialPollCounter.WithLabelValues().Inc()
	r, err := git.PlainOpen(location)
	if err != nil {
		return false, nil, err
	}
	res, errorMsg, err := impl.gitUtil.Fetch(location, userName, password)
	if err == nil && len(res) > 0 {
		impl.logger.Infow("repository updated", "location", url)
		//updated
		middleware.GitPullDuration.WithLabelValues("true", "true").Observe(time.Since(start).Seconds())
		return true, r, nil
	} else if err == nil && len(res) == 0 {
		impl.logger.Debugw("no update for ", "path", url)
		middleware.GitPullDuration.WithLabelValues("true", "false").Observe(time.Since(start).Seconds())
		return false, r, nil
	} else {
		impl.logger.Errorw("error in updating repository", "err", err, "location", url, "error msg", errorMsg)
		middleware.GitPullDuration.WithLabelValues("false", "false").Observe(time.Since(start).Seconds())
		return false, r, err
	}

}

func (impl RepositoryManagerImpl) GetCommitForTag(checkoutPath, tag string) (*GitCommit, error) {
	tag = strings.TrimSpace(tag)
	r, err := git.PlainOpen(checkoutPath)
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
	gitCommit := &GitCommit{
		Author:  commit.Author.String(),
		Commit:  commit.Hash.String(),
		Date:    commit.Author.When,
		Message: commit.Message,
	}
	fs, err := impl.getStats(commit)
	if err != nil {
		impl.logger.Errorw("error in getting fs", "path", checkoutPath, "err", err)
		return nil, err
	}
	for _, f := range fs {
		gitCommit.Changes = append(gitCommit.Changes, f.Name)
	}
	return gitCommit, nil
}

func (impl RepositoryManagerImpl) GetCommitMetadata(checkoutPath, commitHash string) (*GitCommit, error) {
	r, err := git.PlainOpen(checkoutPath)
	if err != nil {
		return nil, err
	}
	commit, err := r.CommitObject(plumbing.NewHash(commitHash))
	if err != nil {
		impl.logger.Errorw("error in fetching commit", "path", checkoutPath, "hash", commitHash, "err", err)
		return nil, err
	}
	gitCommit := &GitCommit{
		Author:  commit.Author.String(),
		Commit:  commit.Hash.String(),
		Date:    commit.Author.When,
		Message: commit.Message,
	}
	fs, err := impl.getStats(commit)
	if err != nil {
		impl.logger.Errorw("error in getting fs", "path", checkoutPath, "err", err)
		return nil, err
	}
	for _, f := range fs {
		gitCommit.Changes = append(gitCommit.Changes, f.Name)
	}
	return gitCommit, nil
}

//from -> old commit
//to -> new commit
//
func (impl RepositoryManagerImpl) ChangesSinceByRepository(repository *git.Repository, branch string, from string, to string, count int) ([]*GitCommit, error) {
	// fix for azure devops (manual trigger webhook bases pipeline) :
	// branch name comes as 'refs/heads/master', we need to extract actual branch name out of it.
	// https://stackoverflow.com/questions/59956206/how-to-get-a-branch-name-with-a-slash-in-azure-devops
	if strings.HasPrefix(branch, "refs/heads/") {
		branch = strings.ReplaceAll(branch, "refs/heads/", "")
	}

	branchRef := fmt.Sprintf("refs/remotes/origin/%s", branch)
	ref, err := repository.Reference(plumbing.ReferenceName(branchRef), true)
	if err != nil && err == plumbing.ErrReferenceNotFound {
		impl.logger.Errorw("ref not found", "branch", branch, "err", err)
		return nil, fmt.Errorf("branch %s not found in the repository ", branch)
	} else if err != nil {
		impl.logger.Errorw("error in getting reference", "branch", branch, "err", err)
		return nil, err
	}
	itr, err := repository.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		impl.logger.Errorw("error in getting iterator", "branch", branch, "err", err)
		return nil, err
	}
	var gitCommits []*GitCommit
	itrCounter := 0
	commitToFind := len(to) == 0 //no commit mentioned
	for {
		if itrCounter > 1000 || len(gitCommits) == count {
			break
		}
		commit, err := itr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			impl.logger.Errorw("error in  iterating", "branch", branch, "err", err)
			break
		}
		if !commitToFind && commit.Hash.String() == to {
			commitToFind = true
		}
		if !commitToFind {
			continue
		}
		if commit.Hash.String() == from && len(from) > 0 {
			//found end
			break
		}
		gitCommit := &GitCommit{
			Author:  commit.Author.String(),
			Commit:  commit.Hash.String(),
			Date:    commit.Author.When,
			Message: commit.Message,
		}
		fs, err := impl.getStats(commit)
		if err != nil {
			impl.logger.Errorw("error in getting fs", "branch", branch, "err", err)
			break
		}
		for _, f := range fs {
			gitCommit.Changes = append(gitCommit.Changes, f.Name)
		}
		gitCommits = append(gitCommits, gitCommit)
		itrCounter = itrCounter + 1
	}
	return gitCommits, err
}

// this function gives file stats in timed manner.
// if timed-out, return empty result without error
func (impl RepositoryManagerImpl) getStats(commit *object.Commit) (object.FileStats, error) {
	result := make(chan FileStatsResult, 1)
	go func() {
		result <- impl.getUntimedFileStats(commit)
	}()

	select {
	case <-time.After(time.Duration(impl.configuration.CommitStatsTimeoutInSec) * time.Second):
		impl.logger.Errorw("Timeout occurred for getting file stats", "commit", commit.Hash.String())
		return nil, nil
	case result := <-result:
		fmt.Println("stats for " + commit.Hash.String())
		fmt.Println(result.FileStats)
		fmt.Println(result.Error)
		return result.FileStats, result.Error
	}
}

// this function gives file stats in untimed manner. There is no timeout for this
// avoid calling this method directly until and unless timeout is not needed
func (impl RepositoryManagerImpl) getUntimedFileStats(commit *object.Commit) FileStatsResult {
	defer func() {
		if err := recover(); err != nil {
			// sometimes the Patch generation will fail due to a known bug in
			// sergi's go-diff: https://github.com/sergi/go-diff/issues/89.
			impl.logger.Errorw("panic error in commit getStats", "commitHash", commit.Hash.String(), "err", err)
			return
		}
	}()

	fs, err := commit.Stats()
	return FileStatsResult{
		FileStats: fs,
		Error:     err,
	}
}

func (impl RepositoryManagerImpl) ChangesSince(checkoutPath string, branch string, from string, to string, count int) ([]*GitCommit, error) {
	if count == 0 {
		count = 15
	}
	r, err := git.PlainOpen(checkoutPath)
	if err != nil {
		return nil, err
	}
	///---------------------
	return impl.ChangesSinceByRepository(r, branch, from, to, count)
	///----------------------

}

type GitChanges struct {
	Commits   []*Commit
	FileStats object.FileStats
}

type FileStatsResult struct {
	FileStats object.FileStats
	Error     error
}

//from -> old commit
//to -> new commit
func (impl RepositoryManagerImpl) ChangesSinceByRepositoryForAnalytics(checkoutPath string, branch string, Old string, New string) (*GitChanges, error) {
	GitChanges := &GitChanges{}
	repository, err := git.PlainOpen(checkoutPath)
	if err != nil {
		return nil, err
	}
	newHash := plumbing.NewHash(New)
	oldHash := plumbing.NewHash(Old)
	old, err := repository.CommitObject(newHash)
	if err != nil {
		return nil, err
	}
	new, err := repository.CommitObject(oldHash)
	if err != nil {
		return nil, err
	}
	oldTree, err := old.Tree()
	if err != nil {
		return nil, err
	}
	newTree, err := new.Tree()
	if err != nil {
		return nil, err
	}
	patch, err := oldTree.Patch(newTree)
	if err != nil {
		impl.logger.Errorw("can'tget patch: ", "err", err)
		return nil, err
	}
	commits, err := computeDiff(repository, &newHash, &oldHash)
	if err != nil {
		impl.logger.Errorw("can't get commits: ", "err", err)
	}
	var serializableCommits []*Commit
	for _, c := range commits {
		t, err := repository.TagObject(c.Hash)
		if err != nil && err != plumbing.ErrObjectNotFound {
			impl.logger.Errorw("can't get tag: ", "err", err)
		}
		serializableCommits = append(serializableCommits, transform(c, t))
	}
	GitChanges.Commits = serializableCommits
	fileStats := patch.Stats()
	impl.logger.Debugw("computed files stats", "filestats", fileStats)
	GitChanges.FileStats = fileStats
	return GitChanges, nil
}

func (impl RepositoryManagerImpl) CreateSshFileIfNotExistsAndConfigureSshCommand(location string, gitProviderId int, sshPrivateKeyContent string) error {
	// add private key
	sshPrivateKeyPath, err := GetOrCreateSshPrivateKeyOnDisk(gitProviderId, sshPrivateKeyContent)
	if err != nil {
		impl.logger.Errorw("error in creating ssh private key", "err", err)
		return err
	}

	//git config core.sshCommand
	_, errorMsg, err := impl.gitUtil.ConfigureSshCommand(location, sshPrivateKeyPath)
	if err != nil {
		impl.logger.Errorw("error in configuring ssh command while adding repo", "errorMsg", errorMsg, "err", err)
		return err
	}

	return nil
}

func computeDiff(r *git.Repository, newHash *plumbing.Hash, oldHash *plumbing.Hash) ([]*object.Commit, error) {
	processed := make(map[string]*object.Commit, 0)
	//t := time.Now()
	h := newHash  //plumbing.NewHash(newHash)
	h2 := oldHash //plumbing.NewHash(oldHash)
	c1, err := r.CommitObject(*h)
	if err != nil {
		return nil, fmt.Errorf("not found commit %s", h.String())
	}
	c2, err := r.CommitObject(*h2)
	if err != nil {
		return nil, fmt.Errorf("not found commit %s", h2.String())
	}

	var parents, ancestorStack []*object.Commit
	ps := c1.Parents()
	for {
		n, err := ps.Next()
		if err == io.EOF {
			break
		}
		if n.Hash.String() != c2.Hash.String() {
			parents = append(parents, n)
		}
	}
	ancestorStack = append(ancestorStack, parents...)
	processed[c1.Hash.String()] = c1

	for len(ancestorStack) > 0 {
		lastIndex := len(ancestorStack) - 1
		//dont process already processed in this algorithm path is not important
		if _, ok := processed[ancestorStack[lastIndex].Hash.String()]; ok {
			ancestorStack = ancestorStack[:lastIndex]
			continue
		}
		//if this is old commit provided for processing then ignore it
		if ancestorStack[lastIndex].Hash.String() == c2.Hash.String() {
			ancestorStack = ancestorStack[:lastIndex]
			continue
		}
		m, err := ancestorStack[lastIndex].MergeBase(c2)
		//fmt.Printf("mergebase between %s and %s is %s length %d\n", ancestorStack[lastIndex].Hash.String(), c2.Hash.String(), m[0].Hash.String(), len(m))
		if err != nil {
			log.Fatal("Error in mergebase " + ancestorStack[lastIndex].Hash.String() + " " + c2.Hash.String())
		}
		// if commit being analyzed is itself merge commit then dont process as it is common in both old and new
		if in(ancestorStack[lastIndex], m) {
			ancestorStack = ancestorStack[:lastIndex]
			continue
		}
		d, p := getDiffTillBranchingOrDest(ancestorStack[lastIndex], m)
		//fmt.Printf("length of diff %d\n", len(d))
		for _, v := range d {
			processed[v.Hash.String()] = v
		}
		curNodes := make(map[string]bool, 0)
		for _, v := range ancestorStack {
			curNodes[v.Hash.String()] = true
		}
		processed[ancestorStack[lastIndex].Hash.String()] = ancestorStack[lastIndex]
		ancestorStack = ancestorStack[:lastIndex]
		for _, v := range p {
			if ok2, _ := curNodes[v.Hash.String()]; !ok2 {
				ancestorStack = append(ancestorStack, v)
			}
		}
	}
	var commits []*object.Commit
	for _, d := range processed {
		commits = append(commits, d)
	}
	return commits, nil
}

func getDiffTillBranchingOrDest(src *object.Commit, dst []*object.Commit) (diff, parents []*object.Commit) {
	if in(src, dst) {
		return
	}
	new := src
	for {
		ps := new.Parents()
		parents = make([]*object.Commit, 0)
		for {
			n, err := ps.Next()
			if err == io.EOF {
				break
			}
			parents = append(parents, n)
		}
		if len(parents) > 1 || len(parents) == 0 {
			return
		}
		if in(parents[0], dst) {
			parents = nil
			return
		} else {
			//fmt.Printf("added %s when child is %s and merge base is %s", parents[0].Hash.String(), src.Hash.String(), dst[0].Hash.String())
			diff = append(diff, parents[0])
		}
		new = parents[0]
	}
}

func in(obj *object.Commit, list []*object.Commit) bool {
	for _, v := range list {
		if v.Hash.String() == obj.Hash.String() {
			return true
		}
	}
	return false
}

func transform(src *object.Commit, tag *object.Tag) (dst *Commit) {
	if src == nil {
		return nil
	}
	dst = &Commit{
		Hash: &Hash{
			Long:  src.Hash.String(),
			Short: src.Hash.String()[:8],
		},
		Tree: &Tree{
			Long:  src.TreeHash.String(),
			Short: src.TreeHash.String()[:8],
		},
		Author: &Author{
			Name:  src.Author.Name,
			Email: src.Author.Email,
			Date:  src.Author.When,
		},
		Committer: &Committer{
			Name:  src.Committer.Name,
			Email: src.Committer.Email,
			Date:  src.Committer.When,
		},
		Subject: src.Message,
		Body:    "",
	}
	if tag != nil {
		dst.Tag = &Tag{
			Name: tag.Name,
			Date: tag.Tagger.When,
		}
	}
	return
}
