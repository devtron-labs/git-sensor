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

type WebhookEventJson struct {
	tableName   struct{} `sql:"webhook_event_json"`
	Id          int      `sql:"id,pk"`
	GitHostName string   `sql:"git_host_name,notnull"`
	PayloadJson string   `sql:"payload_json,notnull"`
	CreatedOn   time.Time `sql:"created_on,notnull"`
}

type WebhookPRDataEvent struct {
	tableName   			struct{} `sql:"webhook_event_pr_data"`
	Id          			int      `sql:"id,pk"`
	GitHostName 			string   `sql:"git_host_name,notnull"`
	PrId        			string   `sql:"pr_id,notnull"`
	PrTitle        			string   `sql:"pr_title,notnull"`
	PrUrl        			string   `sql:"pr_url,notnull"`
	SourceBranchName        string   `sql:"source_branch_name,notnull"`
	SourceBranchHash        string   `sql:"source_branch_hash,notnull"`
	TargetBranchName        string   `sql:"target_branch_name,notnull"`
	TargetBranchHash        string   `sql:"target_branch_hash,notnull"`
	RepositoryUrl	        string   `sql:"repository_url,notnull"`
	AuthorName		        string   `sql:"author_name,notnull"`
	IsOpen					bool	 `sql:"is_open,notnull"`
	ActualState				string	 `sql:"actual_state,notnull"`
	LastCommitMessage		string	 `sql:"last_commit_message"`
	PrCreatedOn   			time.Time `sql:"pr_created_on,notnull"`
	PrUpdatedOn   			time.Time `sql:"pr_updated_on"`
	CreatedOn   			time.Time `sql:"created_on,notnull"`
	UpdatedOn   			time.Time `sql:"updated_on"`
}

type CiPipelineMaterialPrWebhookMapping struct {
	tableName   			struct{} `sql:"ci_pipeline_material_pr_webhook_mapping"`
	Id						int 	 `sql:"id,pk"`
	CiPipelineMaterialId    int 	 `sql:"ci_pipeline_material_id"`
	PrWebhookDataId         int 	 `sql:"pr_webhook_data_id"`
}

type WebhookEventRepository interface {
	SaveJson(webhookEventJson *WebhookEventJson) error
	GetPrEventDataByGitHostNameAndPrId(gitHostName string, prId string) (*WebhookPRDataEvent, error)
	GetPrEventDataById(id int) (*WebhookPRDataEvent, error)
	SavePrEventData(webhookPRDataEvent *WebhookPRDataEvent) error
	UpdatePrEventData(webhookPRDataEvent *WebhookPRDataEvent) error
	CiPipelineMaterialPrWebhookMappingExists(ciPipelineMaterialId int, prWebhookDataId int) (bool, error)
	SaveCiPipelineMaterialPrWebhookMapping(ciPipelineMaterialPrWebhookMapping *CiPipelineMaterialPrWebhookMapping) error
	GetCiPipelineMaterialPrWebhookMapping(ciPipelineMaterialId int)  ([]*CiPipelineMaterialPrWebhookMapping, error)
	GetOpenPrEventDataByIds(ids []int, limit int) ([]*WebhookPRDataEvent, error)
}

type WebhookEventRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewWebhookEventRepositoryImpl(dbConnection *pg.DB) *WebhookEventRepositoryImpl {
	return &WebhookEventRepositoryImpl{dbConnection: dbConnection}
}

func (impl WebhookEventRepositoryImpl) SaveJson(webhookEventJson *WebhookEventJson) error {
	_, err := impl.dbConnection.Model(webhookEventJson).Insert()
	return err
}

func (impl WebhookEventRepositoryImpl) GetPrEventDataByGitHostNameAndPrId(gitHostName string, prId string) (*WebhookPRDataEvent, error) {
	var webhookPRDataEvent WebhookPRDataEvent
	err := impl.dbConnection.Model(&webhookPRDataEvent).Where("git_host_name =? ", gitHostName).Where("pr_id =? ", prId).Select()
	return &webhookPRDataEvent, err
}

func (impl WebhookEventRepositoryImpl) GetPrEventDataById(id int) (*WebhookPRDataEvent, error) {
	var webhookPRDataEvent WebhookPRDataEvent
	err := impl.dbConnection.Model(&webhookPRDataEvent).Where("id =? ", id).Select()
	return &webhookPRDataEvent, err
}

func (impl WebhookEventRepositoryImpl) SavePrEventData(webhookPRDataEvent *WebhookPRDataEvent) error {
	_, err := impl.dbConnection.Model(webhookPRDataEvent).Insert()
	return err
}

func (impl WebhookEventRepositoryImpl) UpdatePrEventData(webhookPRDataEvent *WebhookPRDataEvent) error {
	_, err := impl.dbConnection.Model(webhookPRDataEvent).WherePK().Update()
	return err
}

func (impl WebhookEventRepositoryImpl) CiPipelineMaterialPrWebhookMappingExists(ciPipelineMaterialId int, prWebhookDataId int) (bool, error) {
	mapping := &CiPipelineMaterialPrWebhookMapping{}
	exists, err := impl.dbConnection.
		Model(mapping).
		Where("ci_pipeline_material_id = ?", ciPipelineMaterialId).
		Where("pr_webhook_data_id = ?", prWebhookDataId).
		Exists()
	return exists, err
}

func (impl WebhookEventRepositoryImpl) SaveCiPipelineMaterialPrWebhookMapping(ciPipelineMaterialPrWebhookMapping *CiPipelineMaterialPrWebhookMapping) error {
	_, err := impl.dbConnection.Model(ciPipelineMaterialPrWebhookMapping).Insert()
	return err
}

func (impl WebhookEventRepositoryImpl) GetCiPipelineMaterialPrWebhookMapping(ciPipelineMaterialId int)  ([]*CiPipelineMaterialPrWebhookMapping, error) {
	var pipelineMaterials []*CiPipelineMaterialPrWebhookMapping
	err := impl.dbConnection.Model(&pipelineMaterials).
		Where("ci_pipeline_material_id =? ", ciPipelineMaterialId).
		Column("pr_webhook_data_id").
		Select()
	return pipelineMaterials, err
}

func (impl WebhookEventRepositoryImpl) GetOpenPrEventDataByIds(ids []int, limit int) ([]*WebhookPRDataEvent, error) {
	var webhookPrEventData []*WebhookPRDataEvent
	err := impl.dbConnection.Model(&webhookPrEventData).
		Where("id in (?) ", pg.In(ids)).
		Where("is_open =? ", true).
		Order("pr_updated_on Desc").
		Limit(limit).
		Select()
	return webhookPrEventData, err
}