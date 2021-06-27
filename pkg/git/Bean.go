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
	"github.com/devtron-labs/git-sensor/internal/sql"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"time"
)

const (
	GIT_HOST_NAME_GITHUB string = "Github"
	GIT_HOST_NAME_BITBUCKET_CLOUD string = "Bitbucket Cloud"
)

type FetchScmChangesRequest struct {
	PipelineMaterialId int    `json:"pipelineMaterialId"`
	From               string `json:"from"`
	To                 string `json:"to"`
	Count              int    `json:"count"`
}
type HeadRequest struct {
	MaterialIds []int `json:"materialIds"`
}

type CiPipelineMaterialBean struct {
	Id            int
	GitMaterialId int
	Type          sql.SourceType
	Value         string
	Active        bool
	GitCommit     *GitCommit
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
	Commit    string
	Author    string
	Date      time.Time
	Message   string
	Changes   []string          `json:",omitempty"`
	FileStats *object.FileStats `json:",omitempty"`
	PrData	  *PrData			`json:"prData"`
}

type PrData struct {
	Id					int 	`json:"id"`
	PrTitle        		string  `json:"prTitle"`
	PrUrl        		string	`json:"prUrl"`
	SourceBranchName    string	`json:"sourceBranchName"`
	SourceBranchHash    string	`json:"sourceBranchHash"`
	TargetBranchName    string	`json:"targetBranchName"`
	TargetBranchHash    string	`json:"targetBranchHash"`
	AuthorName		    string	`json:"authorName"`
	LastCommitMessage	string	`json:"lastCommitMessage"`
	PrCreatedOn   		time.Time `json:"prCreatedOn"`
	PrUpdatedOn   		time.Time `json:"prUpdatedOn"`
}


type CommitMetadataRequest struct {
	PipelineMaterialId int    `json:"pipelineMaterialId"`
	GitHash            string `json:"gitHash"`
	GitTag             string `json:"gitTag"`
}

type LatestCommitMetadataRequest struct {
	PipelineMaterialId int    `json:"pipelineMaterialId"`
	BranchName         string `json:"branchName"`
}

type PrDataRequest struct {
	Id int `json:"id"`
}

type RefreshGitMaterialRequest struct {
	GitMaterialId int `json:"gitMaterialId"`
}

type RefreshGitMaterialResponse struct {
	Message       string    `json:"message"`
	ErrorMsg      string    `json:"errorMsg"`
	LastFetchTime time.Time `json:"lastFetchTime"`
}

type GitHostType string

type WebhookEventType string

type WebhookEvent struct {
	RequestPayloadJson string `json:"requestPayloadJson"`
	GitHostType GitHostType `json:"gitHostType"`
	WebhookEventType WebhookEventType `json:"webhookEventType"`
}

type WebhookEventResponse struct {
	success bool
}
