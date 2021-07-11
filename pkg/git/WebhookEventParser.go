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
	ParseEvent(eventId int, selectors []*sql.GitHostWebhookEventSelectors, requestPayloadJson string) (*sql.WebhookEventParsedData, error)
}

type WebhookEventParserImpl struct {
	logger *zap.SugaredLogger
}

func NewWebhookEventParserImpl(logger *zap.SugaredLogger) *WebhookEventParserImpl {
	return &WebhookEventParserImpl{
		logger: logger,
	}
}


const (
	WEBHOOK_SELECTOR_UNIQUE_ID_NAME string = "unique id"
	WEBHOOK_SELECTOR_REPOSITORY_URL_NAME string = "repository url"
	WEBHOOK_SELECTOR_HEADER_NAME string = "header"
	WEBHOOK_SELECTOR_GIT_URL_NAME string = "git url"
	WEBHOOK_SELECTOR_AUTHOR_NAME string = "author"
	WEBHOOK_SELECTOR_DATE_NAME string = "date"
	WEBHOOK_SELECTOR_TARGET_COMMIT_HASH_NAME string = "target commit hash"
	WEBHOOK_SELECTOR_SOURCE_COMMIT_HASH_NAME string = "source commit hash"
	WEBHOOK_SELECTOR_TARGET_BRANCH_NAME_NAME string = "target branch name"
	WEBHOOK_SELECTOR_SOURCE_BRANCH_NAME_NAME string = "source branch name"
)


func (impl WebhookEventParserImpl) ParseEvent(eventId int, selectors []*sql.GitHostWebhookEventSelectors, requestPayloadJson string) (*sql.WebhookEventParsedData, error){

	webhookEventParsedData := &sql.WebhookEventParsedData{
		EventId: eventId,
	}

	additionalData := make(map[string]string)

	// loop in for all selectors
	for _, selector := range selectors {
		name := selector.Name
		selectorValueStr := gjson.Get(requestPayloadJson, selector.Selector).String()
		switch name {
		case WEBHOOK_SELECTOR_UNIQUE_ID_NAME:
			webhookEventParsedData.UniqueId = selectorValueStr
		default:
			additionalData[name] = selectorValueStr
		}
	}

	webhookEventParsedData.Data = additionalData

	return webhookEventParsedData, nil
}
