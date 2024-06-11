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

package sql

import (
	"github.com/devtron-labs/git-sensor/util"
	"github.com/go-pg/pg"
	"time"
)

type WebhookEventParsedData struct {
	tableName         struct{}          `sql:"webhook_event_parsed_data" pg:",discard_unknown_columns"`
	Id                int               `sql:"id,pk"`
	EventId           int               `sql:"event_id,notnull"`
	PayloadDataId     int               `sql:"payload_data_id"`
	UniqueId          string            `sql:"unique_id"`
	EventActionType   string            `sql:"event_action_type,notnull"`
	Data              map[string]string `sql:"data,notnull"`
	CiEnvVariableData map[string]string `sql:"ci_env_variable_data"`
	CreatedOn         time.Time         `sql:"created_on,notnull"`
	UpdatedOn         time.Time         `sql:"updated_on"`
}

type WebhookEventParsedDataRepository interface {
	GetWebhookParsedEventDataByEventIdAndUniqueId(eventId int, uniqueId string) (*WebhookEventParsedData, error)
	SaveWebhookParsedEventData(webhookEventParsedData *WebhookEventParsedData) error
	UpdateWebhookParsedEventData(webhookEventParsedData *WebhookEventParsedData) error
	GetWebhookEventParsedDataByIds(ids []int, limit int) ([]*WebhookEventParsedData, error)
	GetWebhookEventParsedDataById(id int) (*WebhookEventParsedData, error)
}

type WebhookEventParsedDataRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewWebhookEventParsedDataRepositoryImpl(dbConnection *pg.DB) *WebhookEventParsedDataRepositoryImpl {
	return &WebhookEventParsedDataRepositoryImpl{dbConnection: dbConnection}
}

func (impl WebhookEventParsedDataRepositoryImpl) GetWebhookParsedEventDataByEventIdAndUniqueId(eventId int, uniqueId string) (*WebhookEventParsedData, error) {
	var webhookEventParsedData WebhookEventParsedData
	err := impl.dbConnection.Model(&webhookEventParsedData).Where("event_id =? ", eventId).Where("unique_id =? ", uniqueId).Select()
	if err != nil {
		if util.IsErrNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &webhookEventParsedData, nil
}

func (impl WebhookEventParsedDataRepositoryImpl) SaveWebhookParsedEventData(webhookEventParsedData *WebhookEventParsedData) error {
	_, err := impl.dbConnection.Model(webhookEventParsedData).Insert()
	return err
}

func (impl WebhookEventParsedDataRepositoryImpl) UpdateWebhookParsedEventData(webhookEventParsedData *WebhookEventParsedData) error {
	_, err := impl.dbConnection.Model(webhookEventParsedData).WherePK().Update()
	return err
}

func (impl WebhookEventParsedDataRepositoryImpl) GetWebhookEventParsedDataByIds(ids []int, limit int) ([]*WebhookEventParsedData, error) {
	var webhookEventParsedData []*WebhookEventParsedData
	err := impl.dbConnection.Model(&webhookEventParsedData).
		Where("id in (?) ", pg.In(ids)).
		Order("created_on Desc").
		Limit(limit).
		Select()
	return webhookEventParsedData, err
}

func (impl WebhookEventParsedDataRepositoryImpl) GetWebhookEventParsedDataById(id int) (*WebhookEventParsedData, error) {
	var webhookEventParsedData WebhookEventParsedData
	err := impl.dbConnection.Model(&webhookEventParsedData).
		Where("id = ? ", id).
		Select()

	if err != nil {
		if util.IsErrNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &webhookEventParsedData, nil
}
