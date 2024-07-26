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
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"time"
)

type GitHostWebhookEvent struct {
	tableName     struct{}  `sql:"git_host_webhook_event" pg:",discard_unknown_columns"`
	Id            int       `sql:"id,pk"`
	GitHostId     int       `sql:"git_host_id,notnull"`
	GitHostName   string    `sql:"git_host_name,notnull"`
	Name          string    `sql:"name,notnull"`
	EventTypesCsv string    `sql:"event_types_csv,notnull"`
	ActionType    string    `sql:"action_type,notnull"`
	IsActive      bool      `sql:"is_active,notnull"`
	CreatedOn     time.Time `sql:"created_on,notnull"`
	UpdatedOn     time.Time `sql:"updated_on"`

	Selectors []*GitHostWebhookEventSelectors `pg:"fk:event_id"`
}

type GitHostWebhookEventSelectors struct {
	tableName            struct{}  `sql:"git_host_webhook_event_selectors" pg:",discard_unknown_columns"`
	Id                   int       `sql:"id,pk"`
	EventId              int       `sql:"event_id,notnull"`
	Name                 string    `sql:"name,notnull"`
	Selector             string    `sql:"selector,notnull"`
	ToShow               bool      `sql:"to_show,notnull"`
	ToShowInCiFilter     bool      `sql:"to_show_in_ci_filter,notnull"`
	ToUseInCiEnvVariable bool      `sql:"to_use_in_ci_env_variable"`
	FixValue             string    `sql:"fix_value"`
	PossibleValues       string    `sql:"possible_values"`
	IsActive             bool      `sql:"is_active,notnull"`
	CreatedOn            time.Time `sql:"created_on,notnull"`
	UpdatedOn            time.Time `sql:"updated_on"`
}

type WebhookEventRepository interface {
	GetAllGitHostWebhookEventByGitHostId(gitHostId int) ([]*GitHostWebhookEvent, error)
	GetAllGitHostWebhookEventByGitHostName(gitHostName string) ([]*GitHostWebhookEvent, error)
	GetWebhookEventConfigByEventId(eventId int) (*GitHostWebhookEvent, error)
	Update(webhookEvent *GitHostWebhookEvent) error
}

type WebhookEventRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewWebhookEventRepositoryImpl(dbConnection *pg.DB) *WebhookEventRepositoryImpl {
	return &WebhookEventRepositoryImpl{dbConnection: dbConnection}
}

func (impl *WebhookEventRepositoryImpl) GetAllGitHostWebhookEventByGitHostId(gitHostId int) ([]*GitHostWebhookEvent, error) {
	var gitHostWebhookEvents []*GitHostWebhookEvent
	err := impl.dbConnection.Model(&gitHostWebhookEvents).
		Column("git_host_webhook_event.*").
		Relation("Selectors", func(q *orm.Query) (query *orm.Query, err error) {
			return q.Where("is_active IS TRUE"), nil
		}).
		Where("git_host_id =? ", gitHostId).
		Where("is_active = TRUE ").
		Select()
	return gitHostWebhookEvents, err
}
func (impl *WebhookEventRepositoryImpl) GetAllGitHostWebhookEventByGitHostName(gitHostName string) ([]*GitHostWebhookEvent, error) {
	var gitHostWebhookEvents []*GitHostWebhookEvent
	err := impl.dbConnection.Model(&gitHostWebhookEvents).
		Column("git_host_webhook_event.*").
		Relation("Selectors", func(q *orm.Query) (query *orm.Query, err error) {
			return q.Where("is_active IS TRUE"), nil
		}).
		Where("git_host_name =? ", gitHostName).
		Where("is_active = TRUE ").
		Select()
	return gitHostWebhookEvents, err
}

func (impl *WebhookEventRepositoryImpl) GetWebhookEventConfigByEventId(eventId int) (*GitHostWebhookEvent, error) {
	gitHostWebhookEvent := &GitHostWebhookEvent{}
	err := impl.dbConnection.Model(gitHostWebhookEvent).
		Column("git_host_webhook_event.*").
		Relation("Selectors", func(q *orm.Query) (query *orm.Query, err error) {
			return q.Where("is_active IS TRUE"), nil
		}).
		Where("id =? ", eventId).
		Where("is_active = TRUE ").
		Select()

	return gitHostWebhookEvent, err
}

func (impl *WebhookEventRepositoryImpl) Update(webhookEvent *GitHostWebhookEvent) error {
	err := impl.dbConnection.Update(webhookEvent)
	if err != nil {
		return err
	}
	return nil
}
