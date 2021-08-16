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
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/nats-io/stan"
	"go.uber.org/zap"
	_ "gopkg.in/robfig/cron.v3"
	"regexp"
	"time"
)

type WebhookEventService interface {
	GetAllGitHostWebhookEventByGitHostId(gitHostId int) ([]*sql.GitHostWebhookEvent, error)
	GetWebhookParsedEventDataByEventIdAndUniqueId(eventId int, uniqueId string) (*sql.WebhookEventParsedData, error)
	SaveWebhookParsedEventData(webhookEventParsedData *sql.WebhookEventParsedData) error
	UpdateWebhookParsedEventData(webhookEventParsedData *sql.WebhookEventParsedData) error
	MatchCiTriggerConditionAndNotify(event *sql.GitHostWebhookEvent, webhookEventParsedData *sql.WebhookEventParsedData, fullDataMap map[string]string) error
}

type WebhookEventServiceImpl struct {
	logger                            *zap.SugaredLogger
	webhookEventRepository            sql.WebhookEventRepository
	webhookEventParsedDataRepository  sql.WebhookEventParsedDataRepository
	webhookEventDataMappingRepository sql.WebhookEventDataMappingRepository
	materialRepository                sql.MaterialRepository
	nats                              stan.Conn
	webhookEventBeanConverter         WebhookEventBeanConverter
}

func NewWebhookEventServiceImpl(
	logger *zap.SugaredLogger, webhookEventRepository sql.WebhookEventRepository, webhookEventParsedDataRepository sql.WebhookEventParsedDataRepository,
	webhookEventDataMappingRepository sql.WebhookEventDataMappingRepository, materialRepository sql.MaterialRepository, nats stan.Conn, webhookEventBeanConverter WebhookEventBeanConverter,
) *WebhookEventServiceImpl {
	return &WebhookEventServiceImpl{
		logger:                            logger,
		webhookEventRepository:            webhookEventRepository,
		webhookEventParsedDataRepository:  webhookEventParsedDataRepository,
		webhookEventDataMappingRepository: webhookEventDataMappingRepository,
		materialRepository:                materialRepository,
		nats:                              nats,
		webhookEventBeanConverter:         webhookEventBeanConverter,
	}
}

func (impl WebhookEventServiceImpl) GetAllGitHostWebhookEventByGitHostId(gitHostId int) ([]*sql.GitHostWebhookEvent, error) {
	impl.logger.Debugw("Getting All git host events", "gitHostId", gitHostId)
	return impl.webhookEventRepository.GetAllGitHostWebhookEventByGitHostId(gitHostId)
}

func (impl WebhookEventServiceImpl) GetWebhookParsedEventDataByEventIdAndUniqueId(eventId int, uniqueId string) (*sql.WebhookEventParsedData, error) {
	impl.logger.Debugw("fetching webhook event parsed data for ", "eventId", eventId, "uniqueId", uniqueId)

	if len(uniqueId) == 0 {
		return nil, nil
	}

	webhookEventParsedData, err := impl.webhookEventParsedDataRepository.GetWebhookParsedEventDataByEventIdAndUniqueId(eventId, uniqueId)
	if err != nil {
		impl.logger.Errorw("getting error while fetching webhook event parsed data ", "err", err)
		return nil, err
	}

	return webhookEventParsedData, nil
}

func (impl WebhookEventServiceImpl) SaveWebhookParsedEventData(webhookEventParsedData *sql.WebhookEventParsedData) error {
	impl.logger.Debug("saving webhook parsed event data")
	err := impl.webhookEventParsedDataRepository.SaveWebhookParsedEventData(webhookEventParsedData)
	if err != nil {
		impl.logger.Errorw("error while saving webhook parsed event data ", "err", err)
		return err
	}
	return nil
}

func (impl WebhookEventServiceImpl) UpdateWebhookParsedEventData(webhookEventParsedData *sql.WebhookEventParsedData) error {
	impl.logger.Debugw("updating webhook parsed event data for id : ", webhookEventParsedData.Id)
	err := impl.webhookEventParsedDataRepository.UpdateWebhookParsedEventData(webhookEventParsedData)
	if err != nil {
		impl.logger.Errorw("error while updating webhook parsed event data ", "err", err)
		return err
	}
	return nil
}

func (impl WebhookEventServiceImpl) MatchCiTriggerConditionAndNotify(event *sql.GitHostWebhookEvent, webhookEventParsedData *sql.WebhookEventParsedData, fullDataMap map[string]string) error {

	impl.logger.Debug("matching CI trigger condition")

	repositoryUrl := fullDataMap[WEBHOOK_SELECTOR_REPOSITORY_URL_NAME]

	if len(repositoryUrl) == 0 {
		impl.logger.Warn("repository url is blank. so skipping matching condition")
		return nil
	}

	// get materials by Urls
	var repoUrls []string
	repoUrls = append(repoUrls, repositoryUrl)
	repoUrls = append(repoUrls, fmt.Sprintf("%s%s", repositoryUrl, ".git"))

	impl.logger.Debug("getting CI materials for URLs : ", repoUrls)
	materials, err := impl.materialRepository.FindAllActiveByUrls(repoUrls)

	if err != nil {
		impl.logger.Errorw("error in fetching active materials", "err", err)
		return err
	}

	if len(materials) == 0 {
		impl.logger.Info("no materials found skipping.")
		return nil
	}

	for _, material := range materials {
		impl.logger.Debug("matching material with Id ", material.Id)

		ciPipelineMaterials := material.CiPipelineMaterials
		if len(ciPipelineMaterials) == 0 {
			impl.logger.Infow("no ci pipeline, skipping", "id", material.Id, "url", material.Url)
			continue
		}

		for _, ciPipelineMaterial := range ciPipelineMaterials {

			impl.logger.Debug("matching ciPipelineMaterial with Id ", ciPipelineMaterial.Id)

			// ignore if type does not match
			if ciPipelineMaterial.Type != sql.SOURCE_TYPE_WEBHOOK {
				continue
			}

			//MatchFilter
			impl.logger.Debug("Matching filter")
			match := impl.MatchFilter(event, fullDataMap, ciPipelineMaterial.Value)
			impl.logger.Debug("Matched : ", match)

			// insert/update mapping into DB
			err = impl.HandleMaterialWebhookMappingIntoDb(ciPipelineMaterial.Id, webhookEventParsedData.Id, match)
			if err != nil {
				impl.logger.Errorw("err in handling mapping", "err", err)
				return err
			}

			// update material with last fetch time
			impl.logger.Debug("Updating material with last fetch time")
			material.LastFetchTime = time.Now()
			err = impl.materialRepository.Update(material)
			if err != nil {
				impl.logger.Errorw("error in updating material with last fetch time", "material", material, "err", err)
			}

			// if condition is match, then notify for CI
			if match {
				impl.NotifyForAutoCi(impl.BuildNotifyCiObject(ciPipelineMaterial, webhookEventParsedData))
			}
		}
	}

	return nil
}

func (impl WebhookEventServiceImpl) MatchFilter(event *sql.GitHostWebhookEvent, fullDataMap map[string]string, ciPipelineMaterialJsonValue string) bool {
	webhookSourceTypeValue := WebhookSourceTypeValue{}
	err := json.Unmarshal([]byte(ciPipelineMaterialJsonValue), &webhookSourceTypeValue)

	if err != nil {
		impl.logger.Errorw("error in json parsing", "err", err, "ciPipelineMaterialJsonValue", ciPipelineMaterialJsonValue)
		return false
	}

	// match event Id
	if event.Id != webhookSourceTypeValue.EventId {
		return false
	}

	// match condition

	// if no condition found then assume it matched
	condition := webhookSourceTypeValue.Condition
	if len(condition) == 0 {
		return true
	}

	match := true

	// loop in all selectors and match condition
	for _, selector := range event.Selectors {

		// if selector is not active, then don't consider it
		if !selector.IsActive {
			continue
		}

		selectorId := selector.Id
		conditionRegexValue := condition[selectorId]
		if len(conditionRegexValue) == 0 {
			continue
		}

		actualValue := fullDataMap[selector.Name]

		match, err = regexp.MatchString(conditionRegexValue, actualValue)
		if err != nil {
			impl.logger.Errorw("err in matching regex condition", "err", err)
			return false
		}

		if !match {
			break
		}

	}

	return match
}

func (impl WebhookEventServiceImpl) BuildNotifyCiObject(ciPipelineMaterial *sql.CiPipelineMaterial, webhookEventParsedData *sql.WebhookEventParsedData) *CiPipelineMaterialBean {

	notifyObject := &CiPipelineMaterialBean{
		Id:            ciPipelineMaterial.Id,
		Value:         ciPipelineMaterial.Value,
		GitMaterialId: ciPipelineMaterial.GitMaterialId,
		Type:          ciPipelineMaterial.Type,
		Active:        ciPipelineMaterial.Active,
		GitCommit: &GitCommit{
			WebhookData: impl.webhookEventBeanConverter.ConvertFromWebhookParsedDataSqlBean(webhookEventParsedData),
		},
	}
	return notifyObject
}

func (impl WebhookEventServiceImpl) NotifyForAutoCi(material *CiPipelineMaterialBean) error {
	impl.logger.Debugw("Notifying for Auto CI", "request", material)

	mb, err := json.Marshal(material)
	if err != nil {
		impl.logger.Error("err in json marshaling", "err", err)
		return err
	}

	err = impl.nats.Publish(internal.NEW_CI_MATERIAL_TOPIC, mb)
	if err != nil {
		impl.logger.Errorw("error in publishing material modification msg ", "material", material)
	}

	return nil
}

func (impl WebhookEventServiceImpl) HandleMaterialWebhookMappingIntoDb(ciPipelineMaterialId int, webhookParsedDataId int, conditionMatched bool) error {
	impl.logger.Debug("Handling Material webhook mapping into DB")

	mapping, err := impl.webhookEventDataMappingRepository.GetCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialId, webhookParsedDataId)
	if err != nil {
		impl.logger.Errorw("err in getting ci-pipeline vs webhook data mapping", "err", err)
		return err
	}

	ciPipelineMaterialWebhookDataMapping := &sql.CiPipelineMaterialWebhookDataMapping{
		CiPipelineMaterialId: ciPipelineMaterialId,
		WebhookDataId:        webhookParsedDataId,
		ConditionMatched:     conditionMatched,
	}

	if mapping == nil {
		// insert into DB
		impl.logger.Debug("Saving mapping into DB")
		err = impl.webhookEventDataMappingRepository.SaveCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping)
	} else {
		// update DB
		impl.logger.Debug("Updating mapping into DB")
		ciPipelineMaterialWebhookDataMapping.Id = mapping.Id
		err = impl.webhookEventDataMappingRepository.UpdateCiPipelineMaterialWebhookDataMapping(ciPipelineMaterialWebhookDataMapping)
	}

	if err != nil {
		impl.logger.Errorw("err in saving ci-pipeline vs webhook data mapping", "err", err)
		return err
	}

	return nil
}
