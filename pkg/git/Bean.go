/*
 * Copyright (c) 2020-2024. Devtron Inc.
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
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"io"
	"time"
	"unicode/utf8"
)

type FetchScmChangesRequest struct {
	PipelineMaterialId int    `json:"pipelineMaterialId"`
	From               string `json:"from"`
	To                 string `json:"to"`
	Count              int    `json:"count"`
	ShowAll            bool   `json:"showAll"`
}

type HeadRequest struct {
	MaterialIds []int `json:"materialIds"`
}

type CiPipelineMaterialBean struct {
	Id                        int
	GitMaterialId             int
	Type                      sql.SourceType
	Value                     string
	Active                    bool
	GitCommit                 *GitCommitBase
	ExtraEnvironmentVariables map[string]string // extra env variables which will be used for CI
}

type GitRepository struct {
	*git.Repository
	rootDir string
}

type CommitIterator interface {
	Next() (GitCommit, error)
}

type CommitCliIterator struct {
	commits []GitCommit
	index   int
}

type CommitGoGitIterator struct {
	object.CommitIter
}

func transformFileStats(stats object.FileStats) FileStats {
	fileStatList := make([]FileStat, 0)
	for _, stat := range stats {
		fileStatList = append(fileStatList, FileStat{
			Name:     stat.Name,
			Addition: stat.Addition,
			Deletion: stat.Deletion,
		})
	}
	return fileStatList
}

func (itr *CommitGoGitIterator) Next() (GitCommit, error) {
	commit, err := itr.CommitIter.Next()
	if err != nil {
		return nil, err
	}
	gitCommit := GitCommitBase{
		Author:  commit.Author.String(),
		Commit:  commit.Hash.String(),
		Date:    commit.Author.When,
		Message: commit.Message,
	}
	return &GitCommitGoGit{
		GitCommitBase: gitCommit,
		Cm:            commit,
	}, err
}

func (itr *CommitCliIterator) Next() (GitCommit, error) {

	if itr.index < len(itr.commits) {
		commit := itr.commits[itr.index]
		itr.index++
		return commit, nil
	}
	return nil, io.EOF
}

type MaterialChangeResp struct {
	Commits        []*GitCommitBase `json:"commits"`
	LastFetchTime  time.Time        `json:"lastFetchTime"`
	IsRepoError    bool             `json:"isRepoError"`
	RepoErrorMsg   string           `json:"repoErrorMsg"`
	IsBranchError  bool             `json:"isBranchError"`
	BranchErrorMsg string           `json:"branchErrorMsg"`
}

type GitCommit interface {
	GetCommit() *GitCommitBase
}

type GitCommitCli struct {
	GitCommitBase
}

type GitCommitGoGit struct {
	GitCommitBase
	Cm *object.Commit
}

func (gitCommit *GitCommitBase) GetCommit() *GitCommitBase {
	return gitCommit
}

type GitCommitBase struct {
	Commit      string
	Author      string
	Date        time.Time
	Message     string
	Changes     []string     `json:",omitempty"`
	FileStats   *FileStats   `json:",omitempty"`
	WebhookData *WebhookData `json:"webhookData"`
	Excluded    bool         `json:",omitempty"`
}

func AppendOldCommitsFromHistory(newCommits []*GitCommitBase, commitHistory string, fetchedCount int) ([]*GitCommitBase, error) {

	oldCommits := make([]*GitCommitBase, 0)
	if len(commitHistory) > 0 {
		err := json.Unmarshal([]byte(commitHistory), &oldCommits)
		if err != nil {
			return newCommits, fmt.Errorf("unmarshalling error %v", err)
		}
	}
	totalCommits := append(newCommits, oldCommits...)
	if len(totalCommits) > fetchedCount {
		totalCommits = totalCommits[:fetchedCount]
	}
	return totalCommits, nil
}

func (gitCommit *GitCommitBase) SetFileStats(stats *FileStats) {
	gitCommit.FileStats = stats
}

func (gitCommit *GitCommitBase) TruncateMessageIfExceedsMaxLength() {
	maxLength := 1024
	if len(gitCommit.Message) > maxLength {
		gitCommit.Message = gitCommit.Message[:maxLength-3] + "..."
	}
}

// IsMessageValidUTF8 checks if a string is valid UTF-8.
func (gitCommit *GitCommitBase) IsMessageValidUTF8() bool {
	return utf8.ValidString(gitCommit.Message)
}

// FixInvalidUTF8Message replaces invalid UTF-8 sequences with the replacement character (U+FFFD).
func (gitCommit *GitCommitBase) FixInvalidUTF8Message() {
	invalidUTF8 := []rune(gitCommit.Message)
	for i, r := range invalidUTF8 {
		if !utf8.ValidRune(r) {
			invalidUTF8[i] = utf8.RuneError
		}
	}
	gitCommit.Message = string(invalidUTF8)
}

type WebhookAndCiData struct {
	ExtraEnvironmentVariables map[string]string `json:"extraEnvironmentVariables"` // extra env variables which will be used for CI
	WebhookData               *WebhookData      `json:"webhookData"`
}

type WebhookData struct {
	Id              int               `json:"id"`
	EventActionType string            `json:"eventActionType"`
	Data            map[string]string `json:"data"`
}

type CommitMetadataRequest struct {
	PipelineMaterialId int    `json:"pipelineMaterialId"`
	GitHash            string `json:"gitHash"`
	GitTag             string `json:"gitTag"`
	BranchName         string `json:"branchName"`
}

type WebhookDataRequest struct {
	Id                   int `json:"id"`
	CiPipelineMaterialId int `json:"ciPipelineMaterialId"`
}

type WebhookEventConfigRequest struct {
	GitHostId   int    `json:"gitHostId"`
	EventId     int    `json:"eventId"`
	GitHostName string `json:"GitHostName"`
}

type RefreshGitMaterialRequest struct {
	GitMaterialId int `json:"gitMaterialId"`
}

type RefreshGitMaterialResponse struct {
	Message       string    `json:"message"`
	ErrorMsg      string    `json:"errorMsg"`
	LastFetchTime time.Time `json:"lastFetchTime"`
}

type WebhookEvent struct {
	PayloadId          int    `json:"payloadId"`
	RequestPayloadJson string `json:"requestPayloadJson"`
	GitHostName        string `json:"gitHostName"`
	GitHostId          int    `json:"gitHostId"`
	EventType          string `json:"eventType"`
}

type WebhookEventResponse struct {
	success bool
}

type WebhookEventConfig struct {
	Id            int       `json:"id"`
	GitHostId     int       `json:"gitHostId"`
	Name          string    `json:"name"`
	EventTypesCsv string    `json:"eventTypesCsv"`
	ActionType    string    `json:"actionType"`
	IsActive      bool      `json:"isActive"`
	CreatedOn     time.Time `json:"createdOn"`
	UpdatedOn     time.Time `json:"updatedOn"`

	Selectors []*WebhookEventSelectors `json:"selectors"`
}

type WebhookEventSelectors struct {
	Id               int       `json:"id"`
	EventId          int       `json:"eventId"`
	Name             string    `json:"name"`
	Selector         string    `json:"selector"`
	ToShow           bool      `json:"toShow"`
	ToShowInCiFilter bool      `json:"toShowInCiFilter"`
	FixValue         string    `json:"fixValue"`
	PossibleValues   string    `json:"possibleValues"`
	IsActive         bool      `json:"isActive"`
	CreatedOn        time.Time `json:"createdOn"`
	UpdatedOn        time.Time `json:"updatedOn"`
}

// key in condition is selectorId
type WebhookSourceTypeValue struct {
	EventId   int            `json:"eventId,omitempty"`
	Condition map[int]string `json:"condition,omitempty"`
}

type WebhookPayloadDataRequest struct {
	CiPipelineMaterialId int    `json:"ciPipelineMaterialId"`
	Limit                int    `json:"limit"`
	Offset               int    `json:"offset"`
	EventTimeSortOrder   string `json:"eventTimeSortOrder"`
}

type WebhookPayloadDataResponse struct {
	Filters       map[string]string                     `json:"filters"`
	RepositoryUrl string                                `json:"repositoryUrl"`
	Payloads      []*WebhookPayloadDataPayloadsResponse `json:"payloads"`
}

type WebhookPayloadDataPayloadsResponse struct {
	ParsedDataId        int       `json:"parsedDataId"`
	EventTime           time.Time `json:"eventTime"`
	MatchedFiltersCount int       `json:"matchedFiltersCount"`
	FailedFiltersCount  int       `json:"failedFiltersCount"`
	MatchedFilters      bool      `json:"matchedFilters"`
}

type WebhookPayloadFilterDataRequest struct {
	CiPipelineMaterialId int `json:"ciPipelineMaterialId"`
	ParsedDataId         int `json:"parsedDataId"`
}

type WebhookPayloadFilterDataResponse struct {
	PayloadId     int                                         `json:"payloadId"`
	SelectorsData []*WebhookPayloadFilterDataSelectorResponse `json:"selectorsData"`
}

type WebhookPayloadFilterDataSelectorResponse struct {
	SelectorName      string `json:"selectorName"`
	SelectorCondition string `json:"selectorCondition"`
	SelectorValue     string `json:"selectorValue"`
	Match             bool   `json:"match"`
}

type GitChanges struct {
	Commits   []*Commit
	FileStats FileStats
}

type FileStatsResult struct {
	FileStats FileStats
	Error     error
}

// FileStat stores the status of changes in content of a file.
type FileStat struct {
	Name     string
	Addition int
	Deletion int
}

// FileStats is a collection of FileStat.
type FileStats []FileStat

type IteratorRequest struct {
	BranchRef      string
	Branch         string
	CommitCount    int
	FromCommitHash string
	ToCommitHash   string
}
