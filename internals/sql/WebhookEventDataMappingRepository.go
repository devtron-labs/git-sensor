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
	"github.com/go-pg/pg/orm"
	"time"
)

type CiPipelineMaterialWebhookDataMapping struct {
	tableName            struct{}  `sql:"ci_pipeline_material_webhook_data_mapping" pg:",discard_unknown_columns"`
	Id                   int       `sql:"id,pk"`
	CiPipelineMaterialId int       `sql:"ci_pipeline_material_id"`
	WebhookDataId        int       `sql:"webhook_data_id"`
	ConditionMatched     bool      `sql:"condition_matched,notnull"`
	IsActive             bool      `sql:"is_active,notnull"`
	CreatedOn            time.Time `sql:"created_on"`
	UpdatedOn            time.Time `sql:"updated_on"`

	FilterResults []*CiPipelineMaterialWebhookDataMappingFilterResult `pg:"fk:webhook_data_mapping_id"`
}

type WebhookEventDataMappingRepository interface {
	GetCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialId int, webhookParsedDataId int) (*CiPipelineMaterialWebhookDataMapping, error)
	SaveCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping *CiPipelineMaterialWebhookDataMapping) error
	UpdateCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping *CiPipelineMaterialWebhookDataMapping) error
	GetMatchedCiPipelineMaterialWebhookDataMappingForPipelineMaterial(ciPipelineMaterialId int) ([]*CiPipelineMaterialWebhookDataMapping, error)
	InactivateWebhookDataMappingForPipelineMaterials(ciPipelineMaterialIds []int) error
	GetWebhookPayloadDataForPipelineMaterialId(ciPipelineMaterialId int, limit int, offset int, eventTimeSortOrder string) ([]*CiPipelineMaterialWebhookDataMapping, error)
	GetWebhookPayloadFilterDataForPipelineMaterialId(ciPipelineMaterialId int, webhookParsedDataId int) (*CiPipelineMaterialWebhookDataMapping, error)
}

type WebhookEventDataMappingRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewWebhookEventDataMappingRepositoryImpl(dbConnection *pg.DB) *WebhookEventDataMappingRepositoryImpl {
	return &WebhookEventDataMappingRepositoryImpl{dbConnection: dbConnection}
}

func (impl WebhookEventDataMappingRepositoryImpl) GetCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialId int, webhookParsedDataId int) (*CiPipelineMaterialWebhookDataMapping, error) {
	var mapping CiPipelineMaterialWebhookDataMapping
	err := impl.dbConnection.Model(&mapping).
		Where("ci_pipeline_material_id =? ", ciPipelineMaterialId).
		Where("webhook_data_id =? ", webhookParsedDataId).
		Where("is_active = TRUE ").
		Select()

	if err != nil {
		if util.IsErrNoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	return &mapping, nil
}

func (impl WebhookEventDataMappingRepositoryImpl) SaveCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping *CiPipelineMaterialWebhookDataMapping) error {
	_, err := impl.dbConnection.Model(ciPipelineMaterialWebhookDataMapping).Insert()
	return err
}

func (impl WebhookEventDataMappingRepositoryImpl) UpdateCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping *CiPipelineMaterialWebhookDataMapping) error {
	_, err := impl.dbConnection.Model(ciPipelineMaterialWebhookDataMapping).WherePK().Update()
	return err
}

func (impl WebhookEventDataMappingRepositoryImpl) GetMatchedCiPipelineMaterialWebhookDataMappingForPipelineMaterial(ciPipelineMaterialId int) ([]*CiPipelineMaterialWebhookDataMapping, error) {
	var pipelineMaterials []*CiPipelineMaterialWebhookDataMapping
	err := impl.dbConnection.Model(&pipelineMaterials).
		Where("ci_pipeline_material_id =? ", ciPipelineMaterialId).
		Where("is_active = TRUE ").
		Where("condition_matched = TRUE ").
		Select()

	if err != nil {
		if util.IsErrNoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	return pipelineMaterials, nil
}

func (impl WebhookEventDataMappingRepositoryImpl) InactivateWebhookDataMappingForPipelineMaterials(ciPipelineMaterialIds []int) error {
	_, err := impl.dbConnection.Model(&CiPipelineMaterialWebhookDataMapping{}).
		Set("is_active = FALSE ").
		Where("ci_pipeline_material_id in (?) ", pg.In(ciPipelineMaterialIds)).
		Update()
	return err
}

func (impl WebhookEventDataMappingRepositoryImpl) GetWebhookPayloadDataForPipelineMaterialId(ciPipelineMaterialId int, limit int, offset int, eventTimeSortOrder string) ([]*CiPipelineMaterialWebhookDataMapping, error) {

	sortOrder := "DESC"
	if eventTimeSortOrder == "ASC" {
		sortOrder = "ASC"
	}

	var mappings []*CiPipelineMaterialWebhookDataMapping
	err := impl.dbConnection.Model(&mappings).
		Column("ci_pipeline_material_webhook_data_mapping.*").
		Relation("FilterResults", func(q *orm.Query) (query *orm.Query, err error) {
			return q.Where("is_active IS TRUE"), nil
		}).
		Where("ci_pipeline_material_id =? ", ciPipelineMaterialId).
		Where("is_active = TRUE ").
		Limit(limit).
		Offset(offset).
		Order("updated_on " + sortOrder).
		Select()

	if err != nil {
		if util.IsErrNoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	return mappings, nil
}

func (impl WebhookEventDataMappingRepositoryImpl) GetWebhookPayloadFilterDataForPipelineMaterialId(ciPipelineMaterialId int, webhookParsedDataId int) (*CiPipelineMaterialWebhookDataMapping, error) {
	var mapping CiPipelineMaterialWebhookDataMapping
	err := impl.dbConnection.Model(&mapping).
		Column("ci_pipeline_material_webhook_data_mapping.*").
		Relation("FilterResults", func(q *orm.Query) (query *orm.Query, err error) {
			return q.Where("is_active IS TRUE"), nil
		}).
		Where("ci_pipeline_material_id =? ", ciPipelineMaterialId).
		Where("webhook_data_id =? ", webhookParsedDataId).
		Where("is_active = TRUE ").
		Select()

	if err != nil {
		if util.IsErrNoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	return &mapping, nil
}
