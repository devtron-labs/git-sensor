package git

import (
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/devtron-labs/git-sensor/util"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//func (impl *GitUtil) clone(auth transport.AuthMethod, cloneDir string, url string) (*git.Repository, error) {
//	timeoutContext, _ := context.WithTimeout(context.Background(), CLONE_TIMEOUT_SEC*time.Second)
//	impl.logger.Infow("cloning repository ", "url", url, "cloneDir", cloneDir)
//	repo, err := git.PlainCloneContext(timeoutContext, cloneDir, true, &git.CloneOptions{
//		URL:  url,
//		Auth: auth,
//	})
//	if err != nil {
//		impl.logger.Errorw("error in cloning repo ", "url", url, "err", err)
//	} else {
//		impl.logger.Infow("repo cloned", "url", url)
//	}
//	return repo, err
//}

func (impl *GitUtil) OpenNewRepo(location string, url string) (*GitRepository, error) {

	r, err := impl.OpenRepoPlain(location)
	//r := &GitRepository{Repository: *repo}
	if err != nil {
		err = os.RemoveAll(location)
		if err != nil {
			return r, fmt.Errorf("error in cleaning checkout path: %s", err)
		}
		err = impl.Init(location, url, true)
		if err != nil {
			return r, fmt.Errorf("err in git init: %s", err)
		}
		r, err = impl.OpenRepoPlain(location)
		if err != nil {
			return r, fmt.Errorf("err in git init: %s", err)
		}
	}
	return r, nil
}

func (impl *GitUtil) GetCommitsForTag(checkoutPath, tag string) (*GitCommit, error) {

	r, err := impl.OpenRepoPlain(checkoutPath)
	if err != nil {
		return nil, err
	}

	if impl.useCli {
		return impl.GitShow(checkoutPath, tag)
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
	return gitCommit, nil
}

func (impl *GitUtil) GetCommitForHash(checkoutPath, commitHash string) (*GitCommit, error) {
	r, err := impl.OpenRepoPlain(checkoutPath)
	if err != nil {
		return nil, err
	}

	if impl.useCli {
		return impl.GitShow(checkoutPath, commitHash)
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
	return gitCommit, nil
}

func (impl *GitUtil) GetCommitIterator(repository *GitRepository, branchRef string, branch string, useCli bool) (*CommitIterator, error) {

	if useCli {
		return impl.GetCommits(branchRef, branch, repository.rootDir, repository.commitCount)
	}

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
	return &CommitIterator{CommitIter: itr}, nil
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

func (impl *GitUtil) OpenRepoPlain(checkoutPath string) (*GitRepository, error) {

	if impl.useCli {
		err := openGitRepo(checkoutPath)
		if err != nil {
			return nil, err
		}
		return &GitRepository{
			rootDir: checkoutPath,
		}, nil
	}

	r, err := git.PlainOpen(checkoutPath)
	return &GitRepository{Repository: *r}, err
}

func (impl *GitUtil) PathMatcher(fileStats *FileStats, gitMaterial *sql.GitMaterial) bool {
	excluded := false
	var changesInPath []string
	var pathsForFilter []string
	if len(gitMaterial.FilterPattern) == 0 {
		impl.logger.Debugw("no filter configured for this git material", "gitMaterial", gitMaterial)
		return excluded
	}
	for _, path := range gitMaterial.FilterPattern {
		regex := util.GetPathRegex(path)
		pathsForFilter = append(pathsForFilter, regex)
	}
	pathsForFilter = util.ReverseSlice(pathsForFilter)
	impl.logger.Debugw("pathMatcher............", "pathsForFilter", pathsForFilter)
	fileStatBytes, err := json.Marshal(fileStats)
	if err != nil {
		impl.logger.Errorw("marshal error ............", "err", err)
		return false
	}
	var fileChanges []map[string]interface{}
	if err := json.Unmarshal(fileStatBytes, &fileChanges); err != nil {
		impl.logger.Errorw("unmarshal error ............", "err", err)
		return false
	}
	for _, fileChange := range fileChanges {
		path := fileChange["Name"].(string)
		changesInPath = append(changesInPath, path)
	}
	len := len(pathsForFilter)
	for i, filter := range pathsForFilter {
		isExcludeFilter := false
		isMatched := false
		//TODO - handle ! in file name with /!
		const ExcludePathIdentifier = "!"
		if strings.Contains(filter, ExcludePathIdentifier) {
			filter = strings.Replace(filter, ExcludePathIdentifier, "", 1)
			isExcludeFilter = true
		}
		for _, path := range changesInPath {
			match, err := regexp.MatchString(filter, path)
			if err != nil {
				continue
			}
			if match {
				isMatched = true
				break
			}
		}
		if isMatched {
			if isExcludeFilter {
				//if matched for exclude filter
				excluded = true
			} else {
				//if matched for include filter
				excluded = false
			}
			return excluded
		} else if i == len-1 {
			//if it's a last item
			if isExcludeFilter {
				excluded = false
			} else {
				excluded = true
			}
			return excluded
		} else {
			//GO TO THE NEXT FILTER
		}
	}

	return excluded
}

func (impl *GitUtil) Init(rootDir string, remoteUrl string, isBare bool) error {
	//-----------------

	err := os.MkdirAll(rootDir, 0755)
	if err != nil {
		return err
	}

	if impl.useCli {
		err := impl.GitInit(rootDir)
		if err != nil {
			return err
		}
		return impl.GitCreateRemote(rootDir, remoteUrl)
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
