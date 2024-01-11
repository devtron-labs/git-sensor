package git

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/git-sensor/util"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"io"
	"log"
	"strings"
	"time"
)

type RepositoryManagerAnalytics interface {
	ChangesSinceByRepositoryForAnalytics(gitCtx GitContext, checkoutPath string, Old string, New string) (*GitChanges, error)
}

type RepositoryManagerAnalyticsImpl struct {
	*RepositoryManagerImpl
}

func NewRepositoryManagerAnalyticsImpl(
	repositoryManagerImpl *RepositoryManagerImpl,
) *RepositoryManagerAnalyticsImpl {
	return &RepositoryManagerAnalyticsImpl{
		RepositoryManagerImpl: repositoryManagerImpl}
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

// from -> old commit
// to -> new commit
func (impl RepositoryManagerAnalyticsImpl) ChangesSinceByRepositoryForAnalytics(gitCtx GitContext, checkoutPath string, Old string, New string) (*GitChanges, error) {
	var err error
	start := time.Now()
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
	if strings.Contains(checkoutPath, "/.git") || impl.configuration.UseGitCli {
		oldHashString := oldHash.String()
		newHashString := newHash.String()
		outputMsg, errorMsg, err := impl.gitManager.FetchDiffStatBetweenCommits(gitCtx, newHashString, oldHashString, checkoutPath)
		if err != nil {
			impl.logger.Errorw("error in fetching fileStat diff between commits ", "errorMsg", errorMsg, "err", err)
			return nil, err
		}
		fileStats, err = getFileStat(outputMsg)
		if err != nil {
			impl.logger.Errorw("can't convert git diff into fileStats ", "err", err)
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

	var commitsCli, commitsGoGit []*Commit
	if impl.configuration.UseGitCli || impl.configuration.AnalyticsDebug {
		commitsCli, err = impl.gitManager.LogMergeBase(gitCtx, checkoutPath, oldHash.String(), newHash.String())
		if err != nil {
			impl.logger.Errorw("error in fetching commits for analytics through CLI: ", "err", err)
			return nil, err
		}
	} else if !impl.configuration.UseGitCli || impl.configuration.AnalyticsDebug {
		commitsGoGit = impl.getCommitDiff(err, repository, newHash, oldHash)

	}
	if impl.configuration.AnalyticsDebug {
		impl.logOldestCommitComparison(commitsGoGit, commitsCli, checkoutPath, Old, New)
	}

	if !impl.configuration.UseGitCli {
		GitChanges.Commits = commitsGoGit
	} else {
		GitChanges.Commits = commitsCli
	}
	return GitChanges, nil
}

func (impl RepositoryManagerAnalyticsImpl) logOldestCommitComparison(commitsGoGit []*Commit, commitsCli []*Commit, checkoutPath string, old string, new string) {
	if len(commitsGoGit) == 0 || len(commitsCli) == 0 {
		return
	}
	if impl.getOldestCommit(commitsGoGit).Hash.Long != impl.getOldestCommit(commitsCli).Hash.Long {
		impl.logger.Infow("oldest commit did not match for analytics flow", "checkoutPath", checkoutPath, "old", old, "new", new)
	}
}

func (impl RepositoryManagerAnalyticsImpl) getOldestCommit(commits []*Commit) *Commit {
	oldest := commits[0]
	for _, commit := range commits {
		if commit.Author.Date.After(oldest.Author.Date) {
			oldest = commit
		}
	}
	return oldest
}

func (impl RepositoryManagerAnalyticsImpl) getCommitDiff(err error, repository *GitRepository, newHash plumbing.Hash, oldHash plumbing.Hash) []*Commit {
	commits, err := computeDiff(repository.Repository, &newHash, &oldHash)
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
	return serializableCommits
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

	ctx, cancel := context.WithTimeout(gitCtx.Context, time.Duration(impl.configuration.GoGitTimeout)*time.Second)
	defer cancel()
	patch, err = oldTree.PatchContext(ctx, newTree)
	if err != nil {
		return nil, err
	}
	return patch, nil
}

func ProcessGitLogOutput(out string) ([]*Commit, error) {
	gitCommits := make([]*Commit, 0)
	if len(out) == 0 {
		return gitCommits, nil
	}
	logOut := out
	logOut = logOut[:len(logOut)-1]      // Remove the last ","
	logOut = fmt.Sprintf("[%s]", logOut) // Add []

	var gitCommitFormattedList []GitCommitFormat
	err := json.Unmarshal([]byte(logOut), &gitCommitFormattedList)
	if err != nil {
		return nil, err
	}

	for _, formattedCommit := range gitCommitFormattedList {

		cm := Commit{
			Hash: &Hash{
				Long: formattedCommit.Commit,
			},
			Author: &Author{
				Name:  formattedCommit.Author.Name,
				Email: formattedCommit.Author.Email,
				Date:  formattedCommit.Author.Date,
			},
			Committer: &Committer{
				Name:  formattedCommit.Commiter.Name,
				Email: formattedCommit.Commiter.Email,
				Date:  formattedCommit.Commiter.Date,
			},
			Tag:     &Tag{},
			Tree:    &Tree{},
			Subject: formattedCommit.Subject,
			Body:    formattedCommit.Body,
		}
		gitCommits = append(gitCommits, &cm)
	}
	return gitCommits, nil
}
