/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package git

import (
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type WebhookEventParser interface {
	ParseEvent(webhookEvent *WebhookEvent) (*sql.WebhookPRDataEvent, error)
}

type WebhookEventParserImpl struct {
	logger *zap.SugaredLogger
}

func NewWebhookEventParserImpl(logger *zap.SugaredLogger) *WebhookEventParserImpl {
	return &WebhookEventParserImpl{
		logger: logger,
	}
}

func (impl WebhookEventParserImpl) ParseEvent(webhookEvent *WebhookEvent) (*sql.WebhookPRDataEvent, error){

	eventPayloadJson := webhookEvent.RequestPayloadJson

	if webhookEvent.GitHostType == GitHostType(GIT_HOST_NAME_GITHUB){
		return impl.ParseGithubEvent(eventPayloadJson)
	}else if webhookEvent.GitHostType == GitHostType(GIT_HOST_NAME_BITBUCKET_CLOUD) {
		return impl.ParseBitbucketCloudEvent(eventPayloadJson)
	}

	return nil, nil
}


func (impl WebhookEventParserImpl) ParseGithubEvent(eventPayloadJson string) (*sql.WebhookPRDataEvent, error){

	prId := gjson.Get(eventPayloadJson, "pull_request.id").String()
	prUrl := gjson.Get(eventPayloadJson, "pull_request.html_url").String()
	prTitle := gjson.Get(eventPayloadJson, "pull_request.title").String()
	sourceBranchName := gjson.Get(eventPayloadJson, "pull_request.head.ref").String()
	sourceBranchHash := gjson.Get(eventPayloadJson, "pull_request.head.sha").String()
	targetBranchName := gjson.Get(eventPayloadJson, "pull_request.base.ref").String()
	targetBranchHash := gjson.Get(eventPayloadJson, "pull_request.base.sha").String()
	prCreatedOn := gjson.Get(eventPayloadJson, "pull_request.created_at").Time()
	prUpdatedOn := gjson.Get(eventPayloadJson, "pull_request.updated_at").Time()
	isOpen := gjson.Get(eventPayloadJson, "pull_request.state").String() == "open"
	currentState := gjson.Get(eventPayloadJson, "action").String()
	repositoryUrl := gjson.Get(eventPayloadJson, "repository.html_url").String()
	authorName := gjson.Get(eventPayloadJson, "sender.login").String()

	webhookEventData := &sql.WebhookPRDataEvent{
		GitHostName: GIT_HOST_NAME_GITHUB,
		PrId: prId,
		PrUrl: prUrl,
		PrTitle: prTitle,
		SourceBranchName: sourceBranchName,
		SourceBranchHash: sourceBranchHash,
		TargetBranchName: targetBranchName,
		TargetBranchHash: targetBranchHash,
		PrCreatedOn: prCreatedOn,
		PrUpdatedOn: prUpdatedOn,
		IsOpen: isOpen,
		ActualState: currentState,
		RepositoryUrl: repositoryUrl,
		AuthorName: authorName,
	}

	return webhookEventData, nil
}


func (impl WebhookEventParserImpl) ParseBitbucketCloudEvent(eventPayloadJson string) (*sql.WebhookPRDataEvent, error){

	// parse payload
	prId := gjson.Get(eventPayloadJson, "pullrequest.id").String()
	prUrl := gjson.Get(eventPayloadJson, "pullrequest.links.html.href").String()
	prTitle := gjson.Get(eventPayloadJson, "pullrequest.title").String()
	sourceBranchName := gjson.Get(eventPayloadJson, "pullrequest.source.branch.name").String()
	sourceBranchHash := gjson.Get(eventPayloadJson, "pullrequest.source.commit.hash").String()
	targetBranchName := gjson.Get(eventPayloadJson, "pullrequest.destination.branch.name").String()
	targetBranchHash := gjson.Get(eventPayloadJson, "pullrequest.destination.commit.hash").String()
	prCreatedOn := gjson.Get(eventPayloadJson, "pullrequest.created_on").Time()
	prUpdatedOn := gjson.Get(eventPayloadJson, "pullrequest.updated_on").Time()
	isOpen := gjson.Get(eventPayloadJson, "pullrequest.state").String() == "OPEN"
	currentState := gjson.Get(eventPayloadJson, "pullrequest.state").String()
	repositoryUrl := gjson.Get(eventPayloadJson, "repository.links.html.href").String()
	authorName := gjson.Get(eventPayloadJson, "actor.display_name").String()

	webhookEventData := &sql.WebhookPRDataEvent{
		GitHostName: GIT_HOST_NAME_BITBUCKET_CLOUD,
		PrId: prId,
		PrUrl: prUrl,
		PrTitle: prTitle,
		SourceBranchName: sourceBranchName,
		SourceBranchHash: sourceBranchHash,
		TargetBranchName: targetBranchName,
		TargetBranchHash: targetBranchHash,
		PrCreatedOn: prCreatedOn,
		PrUpdatedOn: prUpdatedOn,
		IsOpen: isOpen,
		ActualState: currentState,
		RepositoryUrl: repositoryUrl,
		AuthorName: authorName,
	}

	return webhookEventData, nil

}
