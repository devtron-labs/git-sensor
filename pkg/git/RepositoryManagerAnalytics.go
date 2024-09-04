/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package git

import (
	"fmt"
	"github.com/devtron-labs/git-sensor/internals"
	"github.com/devtron-labs/git-sensor/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"go.uber.org/zap"
	"io"
	"strings"
	"time"
)

type RepositoryManagerAnalytics interface {
	ChangesSinceByRepositoryForAnalytics(gitCtx GitContext, checkoutPath string, Old string, New string) (*GitChanges, error)
}

type RepositoryManagerAnalyticsImpl struct {
	repoManager   RepositoryManager
	gitManager    GitManager
	configuration *internals.Configuration
	logger        *zap.SugaredLogger
}

func NewRepositoryManagerAnalyticsImpl(repoManager RepositoryManager, gitManager GitManager,
	configuration *internals.Configuration, logger *zap.SugaredLogger) *RepositoryManagerAnalyticsImpl {
	return &RepositoryManagerAnalyticsImpl{
		repoManager:   repoManager,
		gitManager:    gitManager,
		configuration: configuration,
		logger:        logger,
	}
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
			return nil, fmt.Errorf("error in mergebase %s %s - err %w", ancestorStack[lastIndex].Hash.String(), c2.Hash.String(), err)
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

// from -> old commit
// to -> new commit
func (impl RepositoryManagerAnalyticsImpl) ChangesSinceByRepositoryForAnalytics(gitCtx GitContext, checkoutPath string, Old string, New string) (*GitChanges, error) {
	var err error
	start := time.Now()
	useGitCli := impl.configuration.UseGitCli || impl.configuration.UseGitCliAnalytics
	defer func() {
		util.TriggerGitOperationMetrics("changesSinceByRepositoryForAnalytics", start, err)
	}()
	GitChanges := &GitChanges{}
	repository, err := impl.gitManager.OpenRepoPlain(checkoutPath)
	if err != nil {
		return nil, err
	}
	newHash := plumbing.NewHash(New)
	oldHash := plumbing.NewHash(Old)

	var fileStats FileStats
	if strings.Contains(checkoutPath, "/.git") || useGitCli {
		oldHashString := oldHash.String()
		newHashString := newHash.String()
		fileStats, err = impl.gitManager.FetchDiffStatBetweenCommitsWithNumstat(gitCtx, oldHashString, newHashString, checkoutPath)
		if err != nil {
			impl.logger.Errorw("error in fetching fileStat diff between commits and converting git diff into fileStats ", "err", err)
		}
	} else {
		patch, err := impl.getPatchObject(gitCtx, repository.Repository, oldHash, newHash)
		if err != nil {
			impl.logger.Errorw("can't get patch: ", "err", err)
			return nil, err
		}
		fileStats = transformFileStats(patch.Stats())
	}
	GitChanges.FileStats = fileStats
	impl.logger.Debugw("computed files stats", "filestats", fileStats)

	commits, err := impl.computeCommitDiff(gitCtx, checkoutPath, oldHash, newHash, repository)
	if err != nil {
		return nil, err
	}
	GitChanges.Commits = commits
	return GitChanges, nil
}

func (impl RepositoryManagerAnalyticsImpl) computeCommitDiff(gitCtx GitContext, checkoutPath string, oldHash plumbing.Hash, newHash plumbing.Hash, repository *GitRepository) ([]*Commit, error) {
	var commitsCli, commitsGoGit []*Commit
	var err error
	useGitCli := impl.configuration.UseGitCli || impl.configuration.UseGitCliAnalytics
	if useGitCli || impl.configuration.AnalyticsDebug {
		impl.logger.Infow("Computing commit diff using cli ", "checkoutPath", checkoutPath)
		commitsCli, err = impl.gitManager.LogMergeBase(gitCtx, checkoutPath, oldHash.String(), newHash.String())
		if err != nil {
			impl.logger.Errorw("error in fetching commits for analytics through CLI: ", "err", err)
			return nil, err
		}
	}
	if !useGitCli || impl.configuration.AnalyticsDebug {
		impl.logger.Infow("Computing commit diff using go-git ", "checkoutPath", checkoutPath)
		ctx, cancel := gitCtx.WithTimeout(impl.configuration.GoGitTimeout)
		defer cancel()
		commitsGoGit, err = RunWithTimeout(ctx, func() ([]*Commit, error) {
			return impl.getCommitDiff(repository, newHash, oldHash)
		})
		if err != nil {
			impl.logger.Errorw("error in fetching commits for analytics through gogit: ", "err", err)
			return nil, err
		}
	}
	if impl.configuration.AnalyticsDebug {
		impl.logOldestCommitComparison(commitsGoGit, commitsCli, checkoutPath, oldHash.String(), newHash.String())
	}

	if !useGitCli {
		return commitsGoGit, nil
	}
	return commitsCli, nil
}

func (impl RepositoryManagerAnalyticsImpl) logOldestCommitComparison(commitsGoGit []*Commit, commitsCli []*Commit, checkoutPath string, old string, new string) {
	impl.logger.Infow("analysing CLI diff for analytics flow", "checkoutPath", checkoutPath, "old", old, "new", new)
	if len(commitsGoGit) == 0 || len(commitsCli) == 0 {
		return
	}
	oldestHashGoGit := getOldestCommit(commitsGoGit).Hash.Long
	oldestHashCli := getOldestCommit(commitsCli).Hash.Long
	if oldestHashGoGit != oldestHashCli {
		impl.logger.Infow("oldest commit did not match for analytics flow", "checkoutPath", checkoutPath, "old", oldestHashGoGit, "new", oldestHashCli)
	} else {
		impl.logger.Infow("oldest commit matched for analytics flow", "checkoutPath", checkoutPath, "old", oldestHashGoGit, "new", oldestHashCli)
	}
}

func getOldestCommit(commits []*Commit) *Commit {
	oldest := commits[0]
	for _, commit := range commits {
		if oldest.Author.Date.After(commit.Author.Date) {
			oldest = commit
		}
	}
	return oldest
}

func (impl RepositoryManagerAnalyticsImpl) getCommitDiff(repository *GitRepository, newHash plumbing.Hash, oldHash plumbing.Hash) ([]*Commit, error) {
	commits, err := computeDiff(repository.Repository, &newHash, &oldHash)
	if err != nil {
		impl.logger.Errorw("can't get commits: ", "err", err)
		return nil, err
	}
	var serializableCommits []*Commit
	for _, c := range commits {
		t, err := repository.TagObject(c.Hash)
		if err != nil && err != plumbing.ErrObjectNotFound {
			impl.logger.Errorw("can't get tag: ", "err", err)
			return nil, err
		}
		serializableCommits = append(serializableCommits, transform(c, t))
	}
	return serializableCommits, nil
}

func (impl RepositoryManagerAnalyticsImpl) getPatchObject(gitCtx GitContext, repository *git.Repository, oldHash, newHash plumbing.Hash) (*object.Patch, error) {
	patch := &object.Patch{}
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

	ctx, cancel := gitCtx.WithTimeout(impl.configuration.GoGitTimeout)
	defer cancel()
	patch, err = oldTree.PatchContext(ctx, newTree)
	if err != nil {
		return nil, err
	}
	return patch, nil
}

func processGitLogOutputForAnalytics(out string) ([]*Commit, error) {
	gitCommits := make([]*Commit, 0)
	if len(out) == 0 {
		return gitCommits, nil
	}
	gitCommitFormattedList, err := parseFormattedLogOutput(out)
	if err != nil {
		return gitCommits, err
	}
	for _, formattedCommit := range gitCommitFormattedList {
		gitCommits = append(gitCommits, formattedCommit.transformToCommit())
	}
	return gitCommits, nil
}
