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
	"github.com/avdkp/go-git/plumbing/object"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"time"
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
	GitCommit                 *GitCommit
	ExtraEnvironmentVariables map[string]string // extra env variables which will be used for CI
}

type MaterialChangeResp struct {
	Commits        []*GitCommit `json:"commits"`
	LastFetchTime  time.Time    `json:"lastFetchTime"`
	IsRepoError    bool         `json:"isRepoError"`
	RepoErrorMsg   string       `json:"repoErrorMsg"`
	IsBranchError  bool         `json:"isBranchError"`
	BranchErrorMsg string       `json:"branchErrorMsg"`
}

type GitCommit struct {
	Commit      string
	Author      string
	Date        time.Time
	Message     string
	Changes     []string          `json:",omitempty"`
	FileStats   *object.FileStats `json:",omitempty"`
	WebhookData *WebhookData      `json:"webhookData"`
	Excluded    bool              `json:",omitempty"`
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
	GitHostId int `json:"gitHostId"`
	EventId   int `json:"eventId"`
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
