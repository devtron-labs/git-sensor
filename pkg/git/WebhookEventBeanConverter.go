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

import "github.com/devtron-labs/git-sensor/internals/sql"

type WebhookEventBeanConverter interface {
	ConvertFromWebhookParsedDataSqlBean(sqlBean *sql.WebhookEventParsedData) *WebhookData
	ConvertFromWebhookEventSqlBean(sqlBean *sql.GitHostWebhookEvent) *WebhookEventConfig
}

type WebhookEventBeanConverterImpl struct {
}

func NewWebhookEventBeanConverterImpl() *WebhookEventBeanConverterImpl {
	return &WebhookEventBeanConverterImpl{}
}

func (impl WebhookEventBeanConverterImpl) ConvertFromWebhookParsedDataSqlBean(sqlBean *sql.WebhookEventParsedData) *WebhookData {
	return &WebhookData{
		Id:              sqlBean.Id,
		EventActionType: sqlBean.EventActionType,
		Data:            sqlBean.Data,
	}
}

func (impl WebhookEventBeanConverterImpl) ConvertFromWebhookEventSqlBean(webhookEventFromDb *sql.GitHostWebhookEvent) *WebhookEventConfig {

	webhookEvent := &WebhookEventConfig{
		Id:            webhookEventFromDb.Id,
		GitHostId:     webhookEventFromDb.GitHostId,
		Name:          webhookEventFromDb.Name,
		EventTypesCsv: webhookEventFromDb.EventTypesCsv,
		ActionType:    webhookEventFromDb.ActionType,
		IsActive:      webhookEventFromDb.IsActive,
		CreatedOn:     webhookEventFromDb.CreatedOn,
		UpdatedOn:     webhookEventFromDb.UpdatedOn,
	}

	// build selectors
	var webhookEventSelectors []*WebhookEventSelectors
	for _, selectorFromDb := range webhookEventFromDb.Selectors {
		selector := &WebhookEventSelectors{
			Id:               selectorFromDb.Id,
			EventId:          selectorFromDb.EventId,
			Name:             selectorFromDb.Name,
			Selector:         selectorFromDb.Selector,
			ToShow:           selectorFromDb.ToShow,
			ToShowInCiFilter: selectorFromDb.ToShowInCiFilter,
			FixValue:         selectorFromDb.FixValue,
			PossibleValues:   selectorFromDb.PossibleValues,
			IsActive:         selectorFromDb.IsActive,
			CreatedOn:        selectorFromDb.CreatedOn,
			UpdatedOn:        selectorFromDb.UpdatedOn,
		}
		webhookEventSelectors = append(webhookEventSelectors, selector)
	}

	webhookEvent.Selectors = webhookEventSelectors

	return webhookEvent

}
