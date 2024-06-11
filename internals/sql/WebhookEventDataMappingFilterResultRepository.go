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
	"time"
)

type CiPipelineMaterialWebhookDataMappingFilterResult struct {
	tableName            struct{}          `sql:"ci_pipeline_material_webhook_data_mapping_filter_result" pg:",discard_unknown_columns"`
	Id                   int               `sql:"id,pk"`
	WebhookDataMappingId int               `sql:"webhook_data_mapping_id,notnull"`
	SelectorName         string            `sql:"selector_name,notnull"`
	SelectorCondition    string            `sql:"selector_condition"`
	SelectorValue        string            `sql:"selector_value"`
	ConditionMatched     bool              `sql:"condition_matched,notnull"`
	MatchedGroups        map[string]string `sql:"matched_groups"`
	IsActive             bool              `sql:"is_active,notnull"`
	CreatedOn            time.Time         `sql:"created_on,notnull"`
}

type WebhookEventDataMappingFilterResultRepository interface {
	SaveAll(results []*CiPipelineMaterialWebhookDataMappingFilterResult) error
	InactivateForMappingId(webhookDataMappingId int) error
}

type WebhookEventDataMappingFilterResultRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewWebhookEventDataMappingFilterResultRepositoryImpl(dbConnection *pg.DB) *WebhookEventDataMappingFilterResultRepositoryImpl {
	return &WebhookEventDataMappingFilterResultRepositoryImpl{dbConnection: dbConnection}
}

func (impl WebhookEventDataMappingFilterResultRepositoryImpl) SaveAll(results []*CiPipelineMaterialWebhookDataMappingFilterResult) error {
	_, err := impl.dbConnection.Model(&results).Insert()
	return err
}

func (impl WebhookEventDataMappingFilterResultRepositoryImpl) InactivateForMappingId(webhookDataMappingId int) error {
	_, err := impl.dbConnection.Model(&CiPipelineMaterialWebhookDataMappingFilterResult{}).
		Set("is_active = FALSE").
		Where("webhook_data_mapping_id = ?  ", webhookDataMappingId).
		Update()
	return err
}
