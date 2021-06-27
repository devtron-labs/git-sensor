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
	"strings"
)

type WebhookEventService interface {
	SaveWebhookEventJson(webhookEventJson *sql.WebhookEventJson) error
	GetWebhookPrEventDataByGitHostNameAndPrId(gitHostName string, prId string) (*sql.WebhookPRDataEvent, error)
	SaveWebhookPrEventData(webhookPRDataEvent *sql.WebhookPRDataEvent) error
	UpdateWebhookPrEventData(webhookPRDataEvent *sql.WebhookPRDataEvent) error
	HandlePostSavePrWebhook(webhookPRDataEvent *sql.WebhookPRDataEvent) error
}

type WebhookEventServiceImpl struct {
	logger                       *zap.SugaredLogger
	webhookEventRepository       sql.WebhookEventRepository
	materialRepository           sql.MaterialRepository
	nats                         stan.Conn
	gitOperationService			 GitOperationService
}

func NewWebhookEventServiceImpl (
	logger *zap.SugaredLogger, webhookEventRepository sql.WebhookEventRepository, materialRepository sql.MaterialRepository, nats stan.Conn,
	gitOperationService GitOperationService,
) *WebhookEventServiceImpl {
	return &WebhookEventServiceImpl{
		logger:                       logger,
		webhookEventRepository:       webhookEventRepository,
		materialRepository: 		  materialRepository,
		nats: nats,
		gitOperationService: gitOperationService,
	}
}


func (impl WebhookEventServiceImpl) SaveWebhookEventJson(webhookEventJson *sql.WebhookEventJson) error {
	err := impl.webhookEventRepository.SaveJson(webhookEventJson)
	if err != nil {
		impl.logger.Errorw("error in saving webhook event json in db","err", err)
		return err
	}
	return nil
}

func (impl WebhookEventServiceImpl) GetWebhookPrEventDataByGitHostNameAndPrId(gitHostName string, prId string) (*sql.WebhookPRDataEvent, error) {
	impl.logger.Debugw("fetching pr event data for ","gitHostName", gitHostName, "prId", prId)
	webhookEventData ,err := impl.webhookEventRepository.GetPrEventDataByGitHostNameAndPrId(gitHostName, prId)
	if err != nil {
		impl.logger.Errorw("getting error while fetching pr event data ","err", err)
		return nil, err
	}
	return webhookEventData, nil
}

func (impl WebhookEventServiceImpl) SaveWebhookPrEventData(webhookPRDataEvent *sql.WebhookPRDataEvent) error {
	impl.logger.Debug("saving pr event data")
	err := impl.webhookEventRepository.SavePrEventData(webhookPRDataEvent)
	if err != nil {
		impl.logger.Errorw("error while saving pr event data ","err", err)
		return err
	}
	return nil
}

func (impl WebhookEventServiceImpl) UpdateWebhookPrEventData(webhookPRDataEvent *sql.WebhookPRDataEvent) error {
	impl.logger.Debugw("updating pr event data for id : ", webhookPRDataEvent.Id)
	err := impl.webhookEventRepository.UpdatePrEventData(webhookPRDataEvent)
	if err != nil {
		impl.logger.Errorw("error while updating pr event data ","err", err)
		return err
	}
	return nil
}

func (impl WebhookEventServiceImpl) HandlePostSavePrWebhook(webhookPRDataEvent *sql.WebhookPRDataEvent) error {

	impl.logger.Debug("Checking for auto CI in case of PR event")

	// get materials by Urls
	var repoUrls []string
	repoUrls = append(repoUrls, webhookPRDataEvent.RepositoryUrl)
	repoUrls = append(repoUrls, fmt.Sprintf("%s%s", webhookPRDataEvent.RepositoryUrl, ".git"))
	materials, err := impl.materialRepository.FindAllActiveByUrls(repoUrls)

	if err != nil {
		impl.logger.Errorw("error in fetching active materials", "err", err)
		return err
	}

	if len(materials) == 0 {
		impl.logger.Infow("no materials found skipping.")
		return nil
	}

	for _, material := range materials {
		ciPipelineMaterials := material.CiPipelineMaterials
		if len(ciPipelineMaterials) == 0 {
			impl.logger.Infow("no ci pipeline, skipping", "id", material.Id, "url", material.Url)
			continue
		}

		for _, ciPipelineMaterial := range ciPipelineMaterials {

			// ignore if type does not match
			if ciPipelineMaterial.Type != sql.SOURCE_TYPE_PULL_REQUEST {
				continue
			}

			//MatchFilter
			match := impl.MatchFilter(webhookPRDataEvent, ciPipelineMaterial.Value)

			// if does not match, then skip
			if !match {
				continue
			}

			// fetch commit info and update in DB
			commit, err := impl.gitOperationService.FetchAndGetCommitInfo(ciPipelineMaterial.Id, webhookPRDataEvent.SourceBranchHash)
			if err == nil{
				webhookPRDataEvent.AuthorName = commit.Author
				webhookPRDataEvent.LastCommitMessage = commit.Message
				impl.UpdateWebhookPrEventData(webhookPRDataEvent)
			}

			// if open PR, then notify for auto CI
			if webhookPRDataEvent.IsOpen {
				impl.NotifyForAutoCi(impl.BuildNotifyCiObject(ciPipelineMaterial, webhookPRDataEvent))
			}

			// insert mapping into DB
			err = impl.InsertMaterialWebhookMappingIntoDb(ciPipelineMaterial.Id, webhookPRDataEvent.Id)
			if err != nil {
				impl.logger.Error("err in saving mapping", "err", err)
				return err
			}
		}
	}

	return nil
}


func (impl WebhookEventServiceImpl) MatchFilter(webhookPRDataEvent *sql.WebhookPRDataEvent, ciPipelineMaterialJsonValue string) bool {
	pullRequestSourceTypeValue := sql.PullRequestSourceTypeValue{}
	err := json.Unmarshal([]byte(ciPipelineMaterialJsonValue), &pullRequestSourceTypeValue)

	if err != nil {
		impl.logger.Errorw("error in json parsing", "err", err)
		return false
	}

	sourceBranchRegex := pullRequestSourceTypeValue.SourceBranchRegex
	targetBranchRegex := pullRequestSourceTypeValue.TargetBranchRegex

	match := true
	if len(strings.TrimSpace(sourceBranchRegex)) != 0 {
		match, err = regexp.MatchString(sourceBranchRegex, webhookPRDataEvent.SourceBranchName)
	}
	if !match && len(strings.TrimSpace(targetBranchRegex)) != 0 {
		match, err = regexp.MatchString(targetBranchRegex, webhookPRDataEvent.TargetBranchName)
	}

	return match
}


func (impl WebhookEventServiceImpl) BuildNotifyCiObject(ciPipelineMaterial *sql.CiPipelineMaterial, webhookPRDataEvent *sql.WebhookPRDataEvent) *CiPipelineMaterialBean {

	notifyObject := &CiPipelineMaterialBean{
		Id:            ciPipelineMaterial.Id,
		Value:         ciPipelineMaterial.Value,
		GitMaterialId: ciPipelineMaterial.GitMaterialId,
		Type:          ciPipelineMaterial.Type,
		Active:        ciPipelineMaterial.Active,
		GitCommit:     &GitCommit{
			PrData: &PrData{
				Id : webhookPRDataEvent.Id,
				PrTitle : webhookPRDataEvent.PrTitle,
				PrUrl: webhookPRDataEvent.PrUrl,
				SourceBranchName: webhookPRDataEvent.SourceBranchName,
				TargetBranchName: webhookPRDataEvent.TargetBranchName,
				SourceBranchHash: webhookPRDataEvent.SourceBranchHash,
				TargetBranchHash: webhookPRDataEvent.TargetBranchHash,
				AuthorName: webhookPRDataEvent.AuthorName,
				LastCommitMessage: webhookPRDataEvent.LastCommitMessage,
				PrCreatedOn: webhookPRDataEvent.PrCreatedOn,
				PrUpdatedOn: webhookPRDataEvent.PrUpdatedOn,
			},
		},
	}

	return notifyObject

}



func (impl WebhookEventServiceImpl) NotifyForAutoCi(material *CiPipelineMaterialBean) error {
	impl.logger.Warnw("material notification", "material", material)

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


func (impl WebhookEventServiceImpl) InsertMaterialWebhookMappingIntoDb(ciPipelineMaterialId int, webhookPREventDataId int) error {
	exists, err := impl.webhookEventRepository.CiPipelineMaterialPrWebhookMappingExists(ciPipelineMaterialId, webhookPREventDataId)
	if err != nil {
		impl.logger.Error("err in getting mapping", "err", err)
		return err
	}
	if exists {
		return nil
	}

	ciPipelineMaterialPrWebhookMapping := &sql.CiPipelineMaterialPrWebhookMapping{
		CiPipelineMaterialId: ciPipelineMaterialId,
		PrWebhookDataId: webhookPREventDataId,
	}

	err = impl.webhookEventRepository.SaveCiPipelineMaterialPrWebhookMapping(ciPipelineMaterialPrWebhookMapping)

	if err != nil {
		impl.logger.Error("err in saving mapping", "err", err)
		return err
	}

	return nil
}