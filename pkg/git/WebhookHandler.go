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
	"github.com/devtron-labs/git-sensor/internals/sql"
	"go.uber.org/zap"
	"strings"
	"time"
)

type WebhookHandler interface {
	HandleWebhookEvent(webhookEvent *WebhookEvent) error
}

type WebhookHandlerImpl struct {
	logger              *zap.SugaredLogger
	webhookEventService WebhookEventService
	webhookEventParser  WebhookEventParser
}

func NewWebhookHandlerImpl(logger *zap.SugaredLogger, webhookEventService WebhookEventService, webhookEventParser WebhookEventParser) *WebhookHandlerImpl {
	return &WebhookHandlerImpl{
		logger:              logger,
		webhookEventService: webhookEventService,
		webhookEventParser:  webhookEventParser,
	}
}

func (impl WebhookHandlerImpl) HandleWebhookEvent(webhookEvent *WebhookEvent) error {
	impl.logger.Debug("Webhook event came")

	gitHostId := webhookEvent.GitHostId
	gitHostName := webhookEvent.GitHostName
	eventType := webhookEvent.EventType
	payloadJson := webhookEvent.RequestPayloadJson
	payloadId := webhookEvent.PayloadId

	impl.logger.Debugw("webhook event request data", "gitHostId", gitHostId, "eventType", eventType)

	var events []*sql.GitHostWebhookEvent
	var err error
	if gitHostName != "" {
		// get all configured events from database for given git host Name
		events, err = impl.webhookEventService.GetAllGitHostWebhookEventByGitHostName(gitHostName)
		if err != nil {
			impl.logger.Errorw("error in getting webhook events from db", "err", err, "gitHostId", gitHostId)
			return err
		}
	}

	if len(events) == 0 {
		// retry with gitHostId and get all configured events from database for given git host Id
		events, err = impl.webhookEventService.GetAllGitHostWebhookEventByGitHostId(gitHostId, gitHostName)
		if err != nil {
			impl.logger.Errorw("error in getting webhook events from db", "err", err, "gitHostId", gitHostId)
			return err
		}
		if len(events) == 0 {
			impl.logger.Warnw("webhook events not found for given gitHostId ", "gitHostId", gitHostId)
			return nil
		}

	}

	// operate for all matching event (match for eventType)
	impl.logger.Debug("Checking for event matching")
	for _, event := range events {
		if len(event.EventTypesCsv) > 0 {
			eventTypes := strings.Split(event.EventTypesCsv, ",")
			if !contains(eventTypes, eventType) {
				continue
			}
		}

		eventId := event.Id

		// parse event data using selectors
		webhookEventParsedData, fullDataMap, err := impl.webhookEventParser.ParseEvent(event.Selectors, payloadJson)
		if err != nil {
			impl.logger.Errorw("error in parsing webhook event data", "err", err)
			return err
		}

		// set event details in webhook data (eventId and merged/non-merged etc..)
		webhookEventParsedData.EventId = eventId
		webhookEventParsedData.EventActionType = event.ActionType
		webhookEventParsedData.PayloadDataId = payloadId

		// fetch webhook parsed data from DB if unique id is not blank
		webhookParsedEventGetData, err := impl.webhookEventService.GetWebhookParsedEventDataByEventIdAndUniqueId(eventId, webhookEventParsedData.UniqueId)
		if err != nil {
			impl.logger.Errorw("error in getting parsed webhook event data", "err", err)
			return err
		}

		// save or update in DB
		if webhookParsedEventGetData != nil {
			webhookEventParsedData.Id = webhookParsedEventGetData.Id
			webhookEventParsedData.CreatedOn = webhookParsedEventGetData.CreatedOn
			webhookEventParsedData.UpdatedOn = time.Now()
			impl.webhookEventService.UpdateWebhookParsedEventData(webhookEventParsedData)
		} else {
			webhookEventParsedData.CreatedOn = time.Now()
			impl.webhookEventService.SaveWebhookParsedEventData(webhookEventParsedData)
		}

		// match ci trigger condition and notify
		err = impl.webhookEventService.MatchCiTriggerConditionAndNotify(event, webhookEventParsedData, fullDataMap)
		if err != nil {
			impl.logger.Errorw("error in matching ci trigger condition for webhook after db save", "err", err)
			return err
		}

	}

	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
