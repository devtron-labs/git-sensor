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
	"go.uber.org/zap"
	_ "gopkg.in/robfig/cron.v3"
)

type WebhookEventRepoManager interface {
	SaveWebhookEventJson(webhookEventJson *sql.WebhookEventJson) error
	GetWebhookPrEventDataByGitHostNameAndPrId(gitHostName string, prId string) (*sql.WebhookPRDataEvent, error)
	SaveWebhookPrEventData(webhookPRDataEvent *sql.WebhookPRDataEvent) error
	UpdateWebhookPrEventData(webhookPRDataEvent *sql.WebhookPRDataEvent) error
}

type WebhookEventRepoManagerImpl struct {
	logger                       *zap.SugaredLogger
	webhookEventRepository       sql.WebhookEventRepository
}

func NewWebhookEventRepoManagerImpl (
	logger *zap.SugaredLogger, webhookEventRepository sql.WebhookEventRepository,
) *WebhookEventRepoManagerImpl {
	return &WebhookEventRepoManagerImpl{
		logger:                       logger,
		webhookEventRepository:       webhookEventRepository,
	}
}


func (impl WebhookEventRepoManagerImpl) SaveWebhookEventJson(webhookEventJson *sql.WebhookEventJson) error {
	err := impl.webhookEventRepository.SaveJson(webhookEventJson)
	if err != nil {
		impl.logger.Errorw("error in saving webhook event json in db","err", err)
		return err
	}
	return nil
}

func (impl WebhookEventRepoManagerImpl) GetWebhookPrEventDataByGitHostNameAndPrId(gitHostName string, prId string) (*sql.WebhookPRDataEvent, error) {
	impl.logger.Debugw("fetching pr event data for ","gitHostName", gitHostName, "prId", prId)
	webhookEventData ,err := impl.webhookEventRepository.GetPrEventDataByGitHostNameAndPrId(gitHostName, prId)
	if err != nil {
		impl.logger.Errorw("getting error while fetching pr event data ","err", err)
		return nil, err
	}
	return webhookEventData, nil
}

func (impl WebhookEventRepoManagerImpl) SaveWebhookPrEventData(webhookPRDataEvent *sql.WebhookPRDataEvent) error {
	impl.logger.Debug("saving pr event data")
	err := impl.webhookEventRepository.SavePrEventData(webhookPRDataEvent)
	if err != nil {
		impl.logger.Errorw("error while saving pr event data ","err", err)
		return err
	}
	return nil
}

func (impl WebhookEventRepoManagerImpl) UpdateWebhookPrEventData(webhookPRDataEvent *sql.WebhookPRDataEvent) error {
	impl.logger.Debugw("updating pr event data for id : ", webhookPRDataEvent.Id)
	err := impl.webhookEventRepository.UpdatePrEventData(webhookPRDataEvent)
	if err != nil {
		impl.logger.Errorw("error while updating pr event data ","err", err)
		return err
	}
	return nil
}