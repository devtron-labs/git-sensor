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

import (
	"encoding/json"
	"fmt"
	pubsub "github.com/devtron-labs/common-lib/pubsub-lib"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"github.com/devtron-labs/git-sensor/internals/util"
	_ "github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"regexp"
	"strings"
	"time"
)

type WebhookEventService interface {
	GetAllGitHostWebhookEventByGitHostId(gitHostId int, gitHostName string) ([]*sql.GitHostWebhookEvent, error)
	GetAllGitHostWebhookEventByGitHostName(gitHostName string) ([]*sql.GitHostWebhookEvent, error)
	GetWebhookParsedEventDataByEventIdAndUniqueId(eventId int, uniqueId string) (*sql.WebhookEventParsedData, error)
	SaveWebhookParsedEventData(webhookEventParsedData *sql.WebhookEventParsedData) error
	UpdateWebhookParsedEventData(webhookEventParsedData *sql.WebhookEventParsedData) error
	MatchCiTriggerConditionAndNotify(event *sql.GitHostWebhookEvent, webhookEventParsedData *sql.WebhookEventParsedData, fullDataMap map[string]string) error
}

type WebhookEventServiceImpl struct {
	logger                                        *zap.SugaredLogger
	webhookEventRepository                        sql.WebhookEventRepository
	webhookEventParsedDataRepository              sql.WebhookEventParsedDataRepository
	webhookEventDataMappingRepository             sql.WebhookEventDataMappingRepository
	webhookEventDataMappingFilterResultRepository sql.WebhookEventDataMappingFilterResultRepository
	materialRepository                            sql.MaterialRepository
	pubSubClient                                  *pubsub.PubSubClientServiceImpl
	webhookEventBeanConverter                     WebhookEventBeanConverter
}

func NewWebhookEventServiceImpl(
	logger *zap.SugaredLogger, webhookEventRepository sql.WebhookEventRepository, webhookEventParsedDataRepository sql.WebhookEventParsedDataRepository,
	webhookEventDataMappingRepository sql.WebhookEventDataMappingRepository, webhookEventDataMappingFilterResultRepository sql.WebhookEventDataMappingFilterResultRepository,
	materialRepository sql.MaterialRepository, pubSubClient *pubsub.PubSubClientServiceImpl, webhookEventBeanConverter WebhookEventBeanConverter,
) *WebhookEventServiceImpl {
	return &WebhookEventServiceImpl{
		logger:                                        logger,
		webhookEventRepository:                        webhookEventRepository,
		webhookEventParsedDataRepository:              webhookEventParsedDataRepository,
		webhookEventDataMappingRepository:             webhookEventDataMappingRepository,
		webhookEventDataMappingFilterResultRepository: webhookEventDataMappingFilterResultRepository,
		materialRepository:                            materialRepository,
		pubSubClient:                                  pubSubClient,
		webhookEventBeanConverter:                     webhookEventBeanConverter,
	}
}

func (impl WebhookEventServiceImpl) GetAllGitHostWebhookEventByGitHostId(gitHostId int, gitHostName string) ([]*sql.GitHostWebhookEvent, error) {
	impl.logger.Debugw("Getting All git host events", "gitHostId", gitHostId)
	events, err := impl.webhookEventRepository.GetAllGitHostWebhookEventByGitHostId(gitHostId)
	if err != nil {
		impl.logger.Errorw("Error in getting all webhook events", "err", err)
		return nil, err
	}
	if events != nil && len(events) > 0 && gitHostName != "" {
		//handle older events to update git_host_name
		for _, event := range events {
			if event.GitHostName == "" {
				event.GitHostName = gitHostName
				err = impl.webhookEventRepository.Update(event)
				if err != nil {
					impl.logger.Errorw("Error in updating webhook event with git_host_name", "git_host_name", gitHostName, "err", err)
				}
			}
		}
	}
	return events, nil
}
func (impl WebhookEventServiceImpl) GetAllGitHostWebhookEventByGitHostName(gitHostName string) ([]*sql.GitHostWebhookEvent, error) {
	impl.logger.Debugw("Getting All git host events", "gitHostName", gitHostName)
	return impl.webhookEventRepository.GetAllGitHostWebhookEventByGitHostName(gitHostName)
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
	repositorySSHUrl := fullDataMap[WEBHOOK_SELECTOR_REPOSITORY_SSH_URL_NAME]

	if len(repositoryUrl) == 0 {
		impl.logger.Warn("repository url is blank. so skipping matching condition")
		return nil
	}
	if len(repositorySSHUrl) == 0 {
		repositoryUrlSplit := strings.Split(repositoryUrl, "/")
		if len(repositoryUrlSplit) > 3 {
			workspaceName := repositoryUrlSplit[3]
			projectName := repositoryUrlSplit[4]
			repositorySSHUrl = fmt.Sprintf("git@bitbucket.org:%s/%s.git", workspaceName, projectName)
		}
	}
	// get materials by Urls
	var repoUrls []string
	repoUrls = append(repoUrls, repositoryUrl)
	repoUrls = append(repoUrls, fmt.Sprintf("%s%s", repositoryUrl, ".git"))
	repoUrls = append(repoUrls, repositorySSHUrl)

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
			filterResults, overallMatch, err := impl.MatchFilter(event, fullDataMap, ciPipelineMaterial.Value)
			if err != nil {
				impl.logger.Errorw("err in matching filter", "err", err)
				return err
			}
			impl.logger.Debug("Matched : ", overallMatch)

			// insert/update mapping into DB
			err = impl.HandleMaterialWebhookMappingIntoDb(ciPipelineMaterial.Id, webhookEventParsedData.Id, overallMatch, filterResults)
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
			if overallMatch {
				impl.NotifyForAutoCi(impl.BuildNotifyCiObject(ciPipelineMaterial, webhookEventParsedData, filterResults))
			}
		}
	}

	return nil
}

func (impl WebhookEventServiceImpl) MatchFilter(event *sql.GitHostWebhookEvent, fullDataMap map[string]string, ciPipelineMaterialJsonValue string) ([]*sql.CiPipelineMaterialWebhookDataMappingFilterResult, bool, error) {
	webhookSourceTypeValue := WebhookSourceTypeValue{}
	err := json.Unmarshal([]byte(ciPipelineMaterialJsonValue), &webhookSourceTypeValue)

	if err != nil {
		impl.logger.Errorw("error in json parsing", "err", err, "ciPipelineMaterialJsonValue", ciPipelineMaterialJsonValue)
		return nil, false, err
	}

	// match event Id
	if event.Id != webhookSourceTypeValue.EventId {
		return nil, false, nil
	}

	// match condition

	// if no condition found then assume it matched
	condition := webhookSourceTypeValue.Condition
	if len(condition) == 0 {
		return nil, true, nil
	}

	overallMatch := true
	var filterResults []*sql.CiPipelineMaterialWebhookDataMappingFilterResult

	// loop in all selectors and match condition
	for _, selector := range event.Selectors {

		// if selector is not active, then don't consider it
		if !selector.IsActive {
			continue
		}

		selectorId := selector.Id
		actualValue := fullDataMap[selector.Name]

		conditionRegexValue := condition[selectorId]

		filterResult := &sql.CiPipelineMaterialWebhookDataMappingFilterResult{
			SelectorName:      selector.Name,
			SelectorCondition: conditionRegexValue,
			SelectorValue:     actualValue,
			ConditionMatched:  true,
			IsActive:          true,
		}

		if len(conditionRegexValue) != 0 {
			match, err := regexp.MatchString(conditionRegexValue, actualValue)
			if err != nil || !match {
				filterResult.ConditionMatched = false
			} else {
				matchedGroups := impl.GetRegexGroupData(conditionRegexValue, actualValue)
				filterResult.MatchedGroups = matchedGroups
			}
			if overallMatch && !filterResult.ConditionMatched {
				overallMatch = false
			}
		}

		if _, ok := condition[selectorId]; ok {
			filterResults = append(filterResults, filterResult)
		}

	}

	return filterResults, overallMatch, nil
}

func (impl WebhookEventServiceImpl) GetRegexGroupData(regex string, val string) map[string]string {
	matchedGroups := make(map[string]string)
	r := regexp.MustCompile(regex)
	matches := r.FindStringSubmatch(val)
	subexpNames := r.SubexpNames()
	for i, name := range subexpNames {
		if len(name) != 0 && len(matches) > i {
			matchedGroups[name] = matches[i]
		}
	}
	return matchedGroups
}

func (impl WebhookEventServiceImpl) BuildNotifyCiObject(ciPipelineMaterial *sql.CiPipelineMaterial, webhookEventParsedData *sql.WebhookEventParsedData, filterResults []*sql.CiPipelineMaterialWebhookDataMappingFilterResult) *CiPipelineMaterialBean {
	extraEnvironmentVariables := util.BuildExtraEnvironmentVariablesForCi(filterResults, webhookEventParsedData.CiEnvVariableData)
	notifyObject := &CiPipelineMaterialBean{
		Id:            ciPipelineMaterial.Id,
		Value:         ciPipelineMaterial.Value,
		GitMaterialId: ciPipelineMaterial.GitMaterialId,
		Type:          ciPipelineMaterial.Type,
		Active:        ciPipelineMaterial.Active,
		GitCommit: &GitCommitBase{
			WebhookData: impl.webhookEventBeanConverter.ConvertFromWebhookParsedDataSqlBean(webhookEventParsedData),
		},
		ExtraEnvironmentVariables: extraEnvironmentVariables,
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

	err = impl.pubSubClient.Publish(pubsub.NEW_CI_MATERIAL_TOPIC, string(mb))
	if err != nil {
		impl.logger.Errorw("error in publishing material modification msg ", "material", material)
	}

	return err
}

func (impl WebhookEventServiceImpl) HandleMaterialWebhookMappingIntoDb(ciPipelineMaterialId int, webhookParsedDataId int, conditionMatched bool, filterResults []*sql.CiPipelineMaterialWebhookDataMappingFilterResult) error {
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
		IsActive:             true,
		UpdatedOn:            time.Now(),
	}

	isNewMapping := mapping == nil

	if isNewMapping {
		// insert into DB
		impl.logger.Debug("Saving mapping into DB")
		ciPipelineMaterialWebhookDataMapping.CreatedOn = time.Now()
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

	return impl.HandleMaterialWebhookMappingFilterResultIntoDb(filterResults, ciPipelineMaterialWebhookDataMapping.Id, isNewMapping)
}

func (impl WebhookEventServiceImpl) HandleMaterialWebhookMappingFilterResultIntoDb(filterResults []*sql.CiPipelineMaterialWebhookDataMappingFilterResult, webhookDataMappingId int, isNewMapping bool) error {
	impl.logger.Debug("Handling Material webhook mapping filter results into DB")

	// if not new mapping, then inactivate old
	if !isNewMapping {
		err := impl.webhookEventDataMappingFilterResultRepository.InactivateForMappingId(webhookDataMappingId)
		if err != nil {
			impl.logger.Errorw("err in inactivating ci-pipeline vs webhook data mapping filter results", "err", err)
			return err
		}
	}

	if len(filterResults) == 0 {
		return nil
	}

	for _, filterResult := range filterResults {
		filterResult.WebhookDataMappingId = webhookDataMappingId
		filterResult.CreatedOn = time.Now()
	}

	// insert into DB
	err := impl.webhookEventDataMappingFilterResultRepository.SaveAll(filterResults)
	if err != nil {
		impl.logger.Errorw("err in saving ci-pipeline vs webhook data mapping filter results", "err", err)
		return err
	}

	return nil
}
