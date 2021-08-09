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

package sql

import (
	"github.com/go-pg/pg"
	"time"
)

type WebhookEventData struct {
	tableName   struct{}  `sql:"webhook_event_data"`
	Id          int       `sql:"id,pk"`
	EventId     int       `sql:"event_id,notnull"`
	PayloadJson string    `sql:"payload_json,notnull"`
	CreatedOn   time.Time `sql:"created_on,notnull"`
}

type WebhookEventDataRepository interface {
	SaveWebhookEventData(webhookEventData *WebhookEventData) error
}

type WebhookEventDataRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewWebhookEventDataRepositoryImpl(dbConnection *pg.DB) *WebhookEventDataRepositoryImpl {
	return &WebhookEventDataRepositoryImpl{dbConnection: dbConnection}
}

func (impl WebhookEventDataRepositoryImpl) SaveWebhookEventData(webhookEventData *WebhookEventData) error {
	_, err := impl.dbConnection.Model(webhookEventData).Insert()
	return err
}
