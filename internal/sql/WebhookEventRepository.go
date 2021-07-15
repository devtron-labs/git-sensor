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
	"github.com/devtron-labs/git-sensor/util"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"time"
)

type GitHostWebhookEvent struct {
	tableName   struct{} `sql:"git_host_webhook_event" pg:",discard_unknown_columns"`
	Id          int      `sql:"id,pk"`
	GitHostId   int      `sql:"git_host_id,notnull"`
	Name        string   `sql:"name,notnull"`
	EventTypesCsv string `sql:"event_types_csv,notnull"`
	ActionType  string 	 `sql:"action_type,notnull"`
	IsActive	bool	 `sql:"is_active,notnull"`
	CreatedOn   time.Time `sql:"created_on,notnull"`
	UpdatedOn   time.Time `sql:"updated_on"`

	Selectors   []*GitHostWebhookEventSelectors  `pg:"fk:event_id"`
}

type GitHostWebhookEventSelectors struct {
	tableName   struct{} `sql:"git_host_webhook_event_selectors" pg:",discard_unknown_columns"`
	Id          int      `sql:"id,pk"`
	EventId     int 	 `sql:"event_id,notnull"`
	Name        string   `sql:"name,notnull"`
	Selector    string   `sql:"selector,notnull"`
	ToShow		bool	 `sql:"to_show,notnull"`
	PossibleValues string `sql:"possible_values"`
	IsActive	bool	 `sql:"is_active,notnull"`
	CreatedOn   time.Time `sql:"created_on,notnull"`
	UpdatedOn   time.Time `sql:"updated_on"`
}


type WebhookEventData struct {
	tableName   struct{} `sql:"webhook_event_data"`
	Id          int      `sql:"id,pk"`
	EventId     int 	 `sql:"event_id,notnull"`
	PayloadJson string   `sql:"payload_json,notnull"`
	CreatedOn   time.Time `sql:"created_on,notnull"`
}


type WebhookEventParsedData struct {
	tableName   			struct{} 			`sql:"webhook_event_parsed_data"`
	Id          			int      			`sql:"id,pk"`
	EventId     			int 	 			`sql:"event_id,notnull"`
	UniqueId				string 	 			`sql:"unique_id"`
	EventActionType			string				`sql:"event_action_type,notnull"`
	Data					map[string]string   `sql:"data,notnull"`
	CreatedOn   			time.Time 			`sql:"created_on,notnull"`
	UpdatedOn   			time.Time 			`sql:"updated_on"`
}

type CiPipelineMaterialWebhookDataMapping struct {
	tableName   			struct{} `sql:"ci_pipeline_material_webhook_data_mapping"`
	Id						int 	 `sql:"id,pk"`
	CiPipelineMaterialId    int 	 `sql:"ci_pipeline_material_id"`
	WebhookDataId	        int 	 `sql:"webhook_data_id"`
	ConditionMatched	    bool 	 `sql:"condition_matched,notnull"`
}

type WebhookEventRepository interface {
	GetAllGitHostWebhookEventByGitHostId(gitHostId int)	([]*GitHostWebhookEvent, error)
	SaveWebhookEventData(webhookEventData *WebhookEventData) error
	GetWebhookParsedEventDataByEventIdAndUniqueId(eventId int, uniqueId string) (*WebhookEventParsedData, error)
	SaveWebhookParsedEventData(webhookEventParsedData *WebhookEventParsedData) error
	UpdateWebhookParsedEventData(webhookEventParsedData *WebhookEventParsedData) error
	GetCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialId int, webhookParsedDataId int) (*CiPipelineMaterialWebhookDataMapping, error)
	SaveCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping *CiPipelineMaterialWebhookDataMapping) error
	UpdateCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping *CiPipelineMaterialWebhookDataMapping) error
	GetCiPipelineMaterialWebhookDataMappingForPipelineMaterial(ciPipelineMaterialId int)  ([]*CiPipelineMaterialWebhookDataMapping, error)
	GetWebhookEventParsedDataByIds(ids []int, limit int) ([]*WebhookEventParsedData, error)
	GetWebhookEventParsedDataById(id int) (*WebhookEventParsedData, error)
	GetWebhookEventConfigByEventId(eventId int) (*GitHostWebhookEvent, error)
}

type WebhookEventRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewWebhookEventRepositoryImpl(dbConnection *pg.DB) *WebhookEventRepositoryImpl {
	return &WebhookEventRepositoryImpl{dbConnection: dbConnection}
}


func (impl WebhookEventRepositoryImpl) GetAllGitHostWebhookEventByGitHostId(gitHostId int) ([]*GitHostWebhookEvent, error) {
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


func (impl WebhookEventRepositoryImpl) SaveWebhookEventData(webhookEventData *WebhookEventData) error {
	_, err := impl.dbConnection.Model(webhookEventData).Insert()
	return err
}

func (impl WebhookEventRepositoryImpl) GetWebhookParsedEventDataByEventIdAndUniqueId(eventId int, uniqueId string) (*WebhookEventParsedData, error) {
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

func (impl WebhookEventRepositoryImpl) SaveWebhookParsedEventData(webhookEventParsedData *WebhookEventParsedData) error {
	_, err := impl.dbConnection.Model(webhookEventParsedData).Insert()
	return err
}

func (impl WebhookEventRepositoryImpl) UpdateWebhookParsedEventData(webhookEventParsedData *WebhookEventParsedData) error {
	_, err := impl.dbConnection.Model(webhookEventParsedData).WherePK().Update()
	return err
}

func (impl WebhookEventRepositoryImpl) GetCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialId int, webhookParsedDataId int) (*CiPipelineMaterialWebhookDataMapping, error) {
	var mapping CiPipelineMaterialWebhookDataMapping
	err := impl.dbConnection.Model(&mapping).
		   Where("ci_pipeline_material_id =? ", ciPipelineMaterialId).
		   Where("webhook_data_id =? ", webhookParsedDataId).
		   Select()

	if err != nil {
		if util.IsErrNoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	return &mapping, nil
}

func (impl WebhookEventRepositoryImpl) SaveCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping *CiPipelineMaterialWebhookDataMapping) error {
	_, err := impl.dbConnection.Model(ciPipelineMaterialWebhookDataMapping).Insert()
	return err
}

func (impl WebhookEventRepositoryImpl) UpdateCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping *CiPipelineMaterialWebhookDataMapping) error {
	_, err := impl.dbConnection.Model(ciPipelineMaterialWebhookDataMapping).WherePK().Update()
	return err
}

func (impl WebhookEventRepositoryImpl) GetCiPipelineMaterialWebhookDataMappingForPipelineMaterial(ciPipelineMaterialId int)  ([]*CiPipelineMaterialWebhookDataMapping, error) {
	var pipelineMaterials []*CiPipelineMaterialWebhookDataMapping
	err := impl.dbConnection.Model(&pipelineMaterials).
		Where("ci_pipeline_material_id =? ", ciPipelineMaterialId).
		Select()

	if err != nil {
		if util.IsErrNoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	return pipelineMaterials, nil
}

func (impl WebhookEventRepositoryImpl) GetWebhookEventParsedDataByIds(ids []int, limit int) ([]*WebhookEventParsedData, error) {
	var webhookEventParsedData []*WebhookEventParsedData
	err := impl.dbConnection.Model(&webhookEventParsedData).
		Where("id in (?) ", pg.In(ids)).
		Order("updated_on Desc").
		Limit(limit).
		Select()
	return webhookEventParsedData, err
}

func (impl WebhookEventRepositoryImpl) GetWebhookEventParsedDataById(id int) (*WebhookEventParsedData, error) {
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

func (impl WebhookEventRepositoryImpl) GetWebhookEventConfigByEventId(eventId int) (*GitHostWebhookEvent, error) {
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