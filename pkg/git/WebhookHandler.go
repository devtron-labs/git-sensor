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
	"strings"
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

	gitHostId := webhookEvent.GitHostId
	eventType := webhookEvent.EventType
	payloadJson := webhookEvent.RequestPayloadJson

	// get all configured events from database for given git host Id
	events, err := impl.webhookEventService.GetAllGitHostWebhookEventByGitHostId(gitHostId)
	if err != nil {
		impl.logger.Errorw("error in getting webhook events from db","err", err, "gitHostId", gitHostId)
		return err
	}

	if len(events) == 0{
		impl.logger.Warnw("webhook events not found for given gitHostId ","gitHostId", gitHostId)
		return nil
	}

	// operate for all matching event (match for eventType)
	for _, event := range events {
		eventTypes := strings.Split(event.EventTypesCsv, ",")
		if !contains(eventTypes, eventType){
			continue
		}

		eventId := event.Id

		// store in audit json table
		webhookEventData := &sql.WebhookEventData{
			EventId: eventId,
			PayloadJson: payloadJson,
			CreatedOn: time.Now(),
		}
		err := impl.webhookEventService.SaveWebhookEventData(webhookEventData)
		if err != nil{
			impl.logger.Errorw("error in saving webhook event data in db","err", err)
			return err
		}

		// parse event data using selectors
		webhookEventParsedData, err := impl.webhookEventParser.ParseEvent(eventId, event.Selectors, payloadJson)
		if err != nil{
			impl.logger.Errorw("error in parsing webhook event data","err", err)
			return err
		}

		// fetch webhook parsed data from DB if unique id is not blank
		webhookParsedEventGetData, err := impl.webhookEventService.GetWebhookParsedEventDataByEventIdAndUniqueId(eventId, webhookEventParsedData.UniqueId)
		if err != nil{
			impl.logger.Errorw("error in getting parsed webhook event data","err", err)
			return err
		}

		// save or update in DB
		if webhookParsedEventGetData != nil {
			webhookEventParsedData.Id = webhookParsedEventGetData.Id
			webhookEventParsedData.CreatedOn = webhookParsedEventGetData.CreatedOn
			webhookEventParsedData.UpdatedOn = time.Now()
			impl.webhookEventService.UpdateWebhookParsedEventData(webhookEventParsedData)
		}else{
			webhookEventParsedData.CreatedOn = time.Now()
			impl.webhookEventService.SaveWebhookParsedEventData(webhookEventParsedData)
		}

		// match ci trigger condition and notify
		err = impl.webhookEventService.MatchCiTriggerConditionAndNotify(event, webhookEventParsedData)
		if err != nil{
			impl.logger.Errorw("error in matching ci trigger condition for webhook after db save","err", err)
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