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
	"go.uber.org/zap"
	"time"
)

type WebhookHandler interface {
	HandleWebhookEvent(webhookEvent *WebhookEvent) error
}

type WebhookHandlerImpl struct {
	logger *zap.SugaredLogger
	webhookEventService WebhookEventService
	webhookEventParser WebhookEventParser
}

func NewWebhookHandlerImpl(logger *zap.SugaredLogger, webhookEventService WebhookEventService, webhookEventParser WebhookEventParser) *WebhookHandlerImpl {
	return &WebhookHandlerImpl{
		logger: logger,
		webhookEventService: webhookEventService,
		webhookEventParser: webhookEventParser,
	}
}

func (impl WebhookHandlerImpl) HandleWebhookEvent(webhookEvent *WebhookEvent) error{

	gitHostNameStr := string(webhookEvent.GitHostType)

	// store in webhook_event_json table
	webhookEventJson := &sql.WebhookEventJson{
		GitHostName: gitHostNameStr,
		PayloadJson: webhookEvent.RequestPayloadJson,
		CreatedOn: time.Now(),
	}

	err := impl.webhookEventService.SaveWebhookEventJson(webhookEventJson)
	if err != nil{
		impl.logger.Errorw("error in saving webhook event json in db","err", err)
		return err
	}

	// parse event data
	webhookEventData, err := impl.webhookEventParser.ParseEvent(webhookEvent)

	if webhookEventData == nil {
		impl.logger.Errorw("event parsed data is null. unsupported git host type", "gitHostNameStr", gitHostNameStr)
		return err
	}

	// get current pull request data from DB
	webhookEventGetData, err := impl.webhookEventService.GetWebhookPrEventDataByGitHostNameAndPrId(gitHostNameStr, webhookEventData.PrId)

	// save or update in DB
	if webhookEventGetData != nil {
		webhookEventData.Id = webhookEventGetData.Id
		webhookEventData.CreatedOn = webhookEventGetData.CreatedOn
		webhookEventData.UpdatedOn = time.Now()
		impl.webhookEventService.UpdateWebhookPrEventData(webhookEventData)
	}else{
		webhookEventData.CreatedOn = time.Now()
		impl.webhookEventService.SaveWebhookPrEventData(webhookEventData)
	}

	// handle post save event
	err = impl.webhookEventService.HandlePostSavePrWebhook(webhookEventData)
	if err != nil{
		impl.logger.Errorw("error in handling pr webhook after db save","err", err)
		return err
	}

	return nil
}
