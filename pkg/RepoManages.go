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

package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devtron-labs/git-sensor/bean"
	"github.com/devtron-labs/git-sensor/internals"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"github.com/devtron-labs/git-sensor/internals/util"
	"github.com/devtron-labs/git-sensor/pkg/git"
	_ "github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"strings"
)

type RepoManager interface {
	GetHeadForPipelineMaterials(ids []int) ([]*git.CiPipelineMaterialBean, error)
	FetchChanges(pipelineMaterialId int, from string, to string, count int, showAll bool) (*git.MaterialChangeResp, error) //limit
	GetCommitMetadata(gitCtx git.GitContext, pipelineMaterialId int, gitHash string) (*git.GitCommitBase, error)
	GetLatestCommitForBranch(gitCtx git.GitContext, pipelineMaterialId int, branchName string) (*git.GitCommitBase, error)
	GetCommitMetadataForPipelineMaterial(gitCtx git.GitContext, pipelineMaterialId int, gitHash string) (*git.GitCommitBase, error)
	SaveGitProvider(provider *sql.GitProvider) (*sql.GitProvider, error)
	AddRepo(gitCtx git.GitContext, material []*sql.GitMaterial) ([]*sql.GitMaterial, error)
	UpdateRepo(gitCtx git.GitContext, material *sql.GitMaterial) (*sql.GitMaterial, error)
	SavePipelineMaterial(gitCtx git.GitContext, material []*sql.CiPipelineMaterial) ([]*sql.CiPipelineMaterial, error)
	ReloadAllRepo(gitCtx git.GitContext, req *bean.ReloadAllMaterialQuery) (err error)
	ResetRepo(gitCtx git.GitContext, materialId int) error
	GetReleaseChanges(gitCtx git.GitContext, request *ReleaseChangesRequest) (*git.GitChanges, error)
	GetCommitInfoForTag(gitCtx git.GitContext, request *git.CommitMetadataRequest) (*git.GitCommitBase, error)
	RefreshGitMaterial(req *git.RefreshGitMaterialRequest) (*git.RefreshGitMaterialResponse, error)

	GetWebhookAndCiDataById(id int, ciPipelineMaterialId int) (*git.WebhookAndCiData, error)
	GetAllWebhookEventConfigForHost(req *git.WebhookEventConfigRequest) ([]*git.WebhookEventConfig, error)
	GetWebhookEventConfig(eventId int) (*git.WebhookEventConfig, error)
	GetWebhookPayloadDataForPipelineMaterialId(request *git.WebhookPayloadDataRequest) (*git.WebhookPayloadDataResponse, error)
	GetWebhookPayloadFilterDataForPipelineMaterialId(request *git.WebhookPayloadFilterDataRequest) (*git.WebhookPayloadFilterDataResponse, error)
}

type RepoManagerImpl struct {
	logger                                        *zap.SugaredLogger
	materialRepository                            sql.MaterialRepository
	repositoryManager                             git.RepositoryManager
	repositoryManagerAnalytics                    git.RepositoryManagerAnalytics
	gitProviderRepository                         sql.GitProviderRepository
	ciPipelineMaterialRepository                  sql.CiPipelineMaterialRepository
	locker                                        *internals.RepositoryLocker
	gitWatcher                                    git.GitWatcher
	webhookEventRepository                        sql.WebhookEventRepository
	webhookEventParsedDataRepository              sql.WebhookEventParsedDataRepository
	webhookEventDataMappingRepository             sql.WebhookEventDataMappingRepository
	webhookEventDataMappingFilterResultRepository sql.WebhookEventDataMappingFilterResultRepository
	webhookEventBeanConverter                     git.WebhookEventBeanConverter
	configuration                                 *internals.Configuration
	gitManager                                    git.GitManager
}

func NewRepoManagerImpl(
	logger *zap.SugaredLogger,
	materialRepository sql.MaterialRepository,
	repositoryManager git.RepositoryManager,
	repositoryManagerAnalytics git.RepositoryManagerAnalytics,
	gitProviderRepository sql.GitProviderRepository,
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository,
	locker *internals.RepositoryLocker,
	gitWatcher git.GitWatcher, webhookEventRepository sql.WebhookEventRepository,
	webhookEventParsedDataRepository sql.WebhookEventParsedDataRepository,
	webhookEventDataMappingRepository sql.WebhookEventDataMappingRepository,
	webhookEventDataMappingFilterResultRepository sql.WebhookEventDataMappingFilterResultRepository,
	webhookEventBeanConverter git.WebhookEventBeanConverter,
	configuration *internals.Configuration,
	gitManager git.GitManager,
) *RepoManagerImpl {
	return &RepoManagerImpl{
		logger:                            logger,
		materialRepository:                materialRepository,
		repositoryManager:                 repositoryManager,
		repositoryManagerAnalytics:        repositoryManagerAnalytics,
		gitProviderRepository:             gitProviderRepository,
		ciPipelineMaterialRepository:      ciPipelineMaterialRepository,
		locker:                            locker,
		gitWatcher:                        gitWatcher,
		webhookEventRepository:            webhookEventRepository,
		webhookEventParsedDataRepository:  webhookEventParsedDataRepository,
		webhookEventDataMappingRepository: webhookEventDataMappingRepository,
		webhookEventDataMappingFilterResultRepository: webhookEventDataMappingFilterResultRepository,
		webhookEventBeanConverter:                     webhookEventBeanConverter,
		configuration:                                 configuration,
		gitManager:                                    gitManager,
	}
}

func (impl RepoManagerImpl) SavePipelineMaterial(gitCtx git.GitContext, materials []*sql.CiPipelineMaterial) ([]*sql.CiPipelineMaterial, error) {
	var old []*sql.CiPipelineMaterial
	var newMaterial []*sql.CiPipelineMaterial
	for _, material := range materials {
		exists, err := impl.ciPipelineMaterialRepository.Exists(material.Id)
		if err != nil {
			return materials, err
		}
		if exists {
			old = append(old, material)
		} else {
			newMaterial = append(newMaterial, material)
		}
	}
	if len(old) > 0 {
		err := impl.InactivateWebhookDataMappingForPipelineMaterials(old)
		if err != nil {
			return nil, err
		}
		err = impl.ciPipelineMaterialRepository.Update(old)
		if err != nil {
			return nil, err
		}
	}
	if len(newMaterial) > 0 {
		_, err := impl.ciPipelineMaterialRepository.Save(newMaterial)
		if err != nil {
			return nil, err
		}
	}

	var oldNotDeleted []*sql.CiPipelineMaterial
	for _, material := range old {
		if material.Active {
			oldNotDeleted = append(oldNotDeleted, material)
		}
	}
	err := impl.updatePipelineMaterialCommit(gitCtx, append(newMaterial, oldNotDeleted...))
	if err != nil {
		return nil, err
	}
	return materials, nil
}

func (impl RepoManagerImpl) InactivateWebhookDataMappingForPipelineMaterials(oldMaterials []*sql.CiPipelineMaterial) error {
	var ciPipelineMaterialIdsWebhookMappingDeactivate []int
	for _, oldMaterial := range oldMaterials {
		if oldMaterial.Type != sql.SOURCE_TYPE_WEBHOOK {
			continue
		}

		materialFromDb, err := impl.ciPipelineMaterialRepository.FindById(oldMaterial.Id)
		if err != nil {
			impl.logger.Errorw("error in getting pipeline material from DB", "id", oldMaterial.Id, "err", err)
			return err
		}

		if materialFromDb.Value != oldMaterial.Value {
			ciPipelineMaterialIdsWebhookMappingDeactivate = append(ciPipelineMaterialIdsWebhookMappingDeactivate, oldMaterial.Id)
		}
	}

	if len(ciPipelineMaterialIdsWebhookMappingDeactivate) > 0 {
		impl.logger.Debugw("Inactivating webhook data mapping for ids ", "ids", ciPipelineMaterialIdsWebhookMappingDeactivate)
		err := impl.webhookEventDataMappingRepository.InactivateWebhookDataMappingForPipelineMaterials(ciPipelineMaterialIdsWebhookMappingDeactivate)
		if err != nil {
			impl.logger.Errorw("error in inactivating webhook data mapping for ", "ids", ciPipelineMaterialIdsWebhookMappingDeactivate, "err", err)
			return err
		}
	}

	return nil
}

func (impl RepoManagerImpl) updatePipelineMaterialCommit(gitCtx git.GitContext, materials []*sql.CiPipelineMaterial) error {
	var materialCommits []*sql.CiPipelineMaterial
	for _, pipelineMaterial := range materials {

		//some points are missing so fetch
		pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(pipelineMaterial.Id)
		if err != nil {
			impl.logger.Errorw("material not found", "material", pipelineMaterial)
			return err
		}

		material, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
		if err != nil {
			impl.logger.Errorw("error in fetching material", "err", err)
			continue
		}

		gitCtx = gitCtx.WithCredentials(material.GitProvider.UserName, material.GitProvider.Password).
			WithTLSData(material.GitProvider.CaCert, material.GitProvider.TlsKey, material.GitProvider.TlsCert, material.GitProvider.EnableTLSVerification)

		fetchCount := impl.configuration.GitHistoryCount
		var repository *git.GitRepository
		commits, err := impl.repositoryManager.ChangesSinceByRepository(gitCtx, repository, pipelineMaterial.Value, "", "", fetchCount, material.CheckoutLocation, true)
		//commits, err := impl.FetchChanges(pipelineMaterial.Id, "", "", 0)
		if gitCtx.Err() != nil {
			impl.logger.Errorw("context error in getting commits", "err", gitCtx.Err())
			return gitCtx.Err()
		} else if err == nil {
			impl.logger.Infow("commits found", "commit", commits)
			b, err := json.Marshal(commits)
			if err == nil {
				pipelineMaterial.CommitHistory = string(b)
				if len(commits) > 0 {
					latestCommit := commits[0]
					pipelineMaterial.LastSeenHash = latestCommit.Commit
					pipelineMaterial.CommitAuthor = latestCommit.Author
					pipelineMaterial.CommitDate = latestCommit.Date
					pipelineMaterial.CommitMessage = latestCommit.Message
				}
				pipelineMaterial.Errored = false
				pipelineMaterial.ErrorMsg = ""
			} else {
				pipelineMaterial.Errored = true
				pipelineMaterial.ErrorMsg = err.Error()
				pipelineMaterial.LastSeenHash = ""
			}
		} else {
			pipelineMaterial.Errored = true
			pipelineMaterial.ErrorMsg = err.Error()
			pipelineMaterial.LastSeenHash = ""
		}
		materialCommits = append(materialCommits, pipelineMaterial)
	}
	err := impl.ciPipelineMaterialRepository.Update(materialCommits)
	return err
}

func (impl RepoManagerImpl) SaveGitProvider(provider *sql.GitProvider) (*sql.GitProvider, error) {
	exists, err := impl.gitProviderRepository.Exists(provider.Id)
	if err != nil {
		return provider, err
	}
	if exists {
		err = impl.gitProviderRepository.Update(provider)
	} else {
		err = impl.gitProviderRepository.Save(provider)
	}

	if (err == nil) && (provider.AuthMode == sql.AUTH_MODE_SSH) {
		err = git.CreateOrUpdateSshPrivateKeyOnDisk(provider.Id, provider.SshPrivateKey)
		if err != nil {
			impl.logger.Errorw("error in creating/updating ssh private key ", "err", err)
		}
	}

	return provider, err
}

// handle update
func (impl RepoManagerImpl) AddRepo(gitCtx git.GitContext, materials []*sql.GitMaterial) ([]*sql.GitMaterial, error) {
	for _, material := range materials {
		_, err := impl.addRepo(gitCtx, material)
		if err != nil {
			impl.logger.Errorw("error in saving material ", "material", material, "err", err)
			return materials, err
		}
	}
	return materials, nil
}

func (impl RepoManagerImpl) UpdateRepo(gitCtx git.GitContext, material *sql.GitMaterial) (*sql.GitMaterial, error) {
	existingMaterial, err := impl.materialRepository.FindById(material.Id)
	if err != nil {
		impl.logger.Errorw("error in fetching material", err)
		return nil, err
	}
	existingMaterial.Name = material.Name
	existingMaterial.Url = material.Url
	existingMaterial.GitProviderId = material.GitProviderId
	existingMaterial.Deleted = material.Deleted
	existingMaterial.CheckoutStatus = false
	existingMaterial.FetchSubmodules = material.FetchSubmodules
	existingMaterial.FilterPattern = material.FilterPattern
	err = impl.materialRepository.Update(existingMaterial)
	if err != nil {
		impl.logger.Errorw("error in updating material ", "material", material, "err", err)
		return nil, err
	}

	repoLock := impl.locker.LeaseLocker(material.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(material.Id)
	}()

	err = impl.repositoryManager.Clean(existingMaterial.CheckoutLocation)
	if err != nil {
		impl.logger.Errorw("error in refreshing material ", "err", err)
		return nil, err
	}

	if !existingMaterial.Deleted {
		err = impl.checkoutUpdatedRepo(gitCtx, material.Id)
		if err != nil {
			impl.logger.Errorw("error in checking out updated repo", "err", err)
			return nil, err
		}
	}
	return existingMaterial, nil
}

func (impl RepoManagerImpl) checkoutUpdatedRepo(gitCtx git.GitContext, materialId int) error {
	material, err := impl.materialRepository.FindById(materialId)
	if err != nil {
		impl.logger.Errorw("error in fetching material", "id", materialId, "err", err)
		return err
	}
	_, err = impl.checkoutMaterial(gitCtx, material)
	if err != nil {
		impl.logger.Errorw("error in repo refresh", "id", material, "err", err)
		return err
	}
	return nil
}

func (impl RepoManagerImpl) addRepo(gitCtx git.GitContext, material *sql.GitMaterial) (*sql.GitMaterial, error) {
	err := impl.materialRepository.Save(material)
	if err != nil {
		impl.logger.Errorw("error in saving material ", "material", material, "err", err)
		return material, err
	}
	return impl.checkoutRepo(gitCtx, material)
}

func (impl RepoManagerImpl) checkoutRepo(gitCtx git.GitContext, material *sql.GitMaterial) (*sql.GitMaterial, error) {
	repoLock := impl.locker.LeaseLocker(material.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(material.Id)
	}()
	return impl.checkoutMaterial(gitCtx, material)
}

func (impl RepoManagerImpl) checkoutMaterial(gitCtx git.GitContext, material *sql.GitMaterial) (*sql.GitMaterial, error) {
	impl.logger.Infow("checking out material", "id", material.Id)
	gitProvider, err := impl.gitProviderRepository.GetById(material.GitProviderId)
	if err != nil {
		return material, err
	}
	userName, password, err := git.GetUserNamePassword(gitProvider)
	if err != nil {
		return material, nil
	}

	gitCtx = gitCtx.WithCredentials(userName, password).
		WithTLSData(gitProvider.CaCert, gitProvider.TlsKey, gitProvider.TlsCert, gitProvider.EnableTLSVerification)

	checkoutPath, _, _, err := impl.repositoryManager.GetCheckoutLocationFromGitUrl(material, gitCtx.CloningMode)
	if err != nil {
		return material, err
	}

	checkoutLocationForFetching := impl.repositoryManager.GetCheckoutLocation(gitCtx, material, gitProvider.Url, checkoutPath)

	err = impl.repositoryManager.Add(gitCtx, material.GitProviderId, checkoutPath, material.Url, gitProvider.AuthMode, gitProvider.SshPrivateKey)
	if gitCtx.Err() != nil {
		impl.logger.Errorw("context error in git checkout", "err", gitCtx.Err())
		return material, gitCtx.Err()
	} else if err == nil {
		material.CheckoutLocation = checkoutLocationForFetching
		material.CheckoutStatus = true
	} else {
		material.CheckoutStatus = false
		material.CheckoutMsgAny = err.Error()
		material.FetchErrorMessage = err.Error()
	}
	err = impl.materialRepository.Update(material)
	if err != nil {
		impl.logger.Errorw("error in updating material repo", "err", err, "material", material)
		return nil, err
	}
	ciPipelineMaterial, err := impl.ciPipelineMaterialRepository.FindByGitMaterialId(material.Id)
	if err != nil {
		impl.logger.Errorw("unable to load material", "err", err)
		return nil, err
	}
	err = impl.updatePipelineMaterialCommit(gitCtx, ciPipelineMaterial)
	if err != nil {
		impl.logger.Errorw("error in updating pipeline material", "err", err)
	}
	return material, nil
}

func (impl RepoManagerImpl) ReloadAllRepo(gitCtx git.GitContext, req *bean.ReloadAllMaterialQuery) (err error) {
	var materials []*sql.GitMaterial
	if req != nil {
		materials, err = impl.materialRepository.FindInRage(req.Start, req.End)
		if err != nil {
			impl.logger.Errorw(bean.GetReloadAllLog("error in getting reload materials"), "err", err)
			return err
		}
	} else {
		materials, err = impl.materialRepository.FindAll()
		if err != nil {
			impl.logger.Errorw(bean.GetReloadAllLog("error in getting reload materials"), "err", err)
			return err
		}
	}

	for _, material := range materials {
		impl.logger.Infow(bean.GetReloadAllLog("performing material checkout for"), "materialId", material.Id)
		_, err = impl.checkoutRepo(gitCtx, material)
		if gitCtx.Err() != nil {
			impl.logger.Errorw(bean.GetReloadAllLog("error in material checkout"), "materialId", material.Id, "err", gitCtx.Err())
			return gitCtx.Err()
		} else if err != nil {
			impl.logger.Errorw(bean.GetReloadAllLog("context error in material while checkout"), "materialId", material.Id, "err", err)
			// skipping for other materials to be processed
		} else {
			impl.logger.Infow(bean.GetReloadAllLog("successfully checked out for material"), "materialId", material.Id)
		}
	}
	return nil
}

func (impl RepoManagerImpl) ResetRepo(gitCtx git.GitContext, materialId int) error {
	material, err := impl.materialRepository.FindById(materialId)
	if err != nil {
		impl.logger.Errorw("error in fetching material", "id", materialId, "err", err)
		return err
	}
	_, err = impl.checkoutRepo(gitCtx, material)
	if err != nil {
		impl.logger.Errorw("error in repo refresh", "id", material, "err", err)
		return err
	}
	return nil
}

func (impl RepoManagerImpl) GetHeadForPipelineMaterials(ids []int) (materialBeans []*git.CiPipelineMaterialBean, err error) {
	materials, err := impl.ciPipelineMaterialRepository.FindByIds(ids)
	for _, material := range materials {
		materialBean := impl.materialTOMaterialBeanConverter(material)
		materialBeans = append(materialBeans, materialBean)
	}
	return materialBeans, err
}

func (impl RepoManagerImpl) materialTOMaterialBeanConverter(material *sql.CiPipelineMaterial) *git.CiPipelineMaterialBean {
	materialBean := &git.CiPipelineMaterialBean{
		Id:            material.Id,
		Type:          material.Type,
		GitMaterialId: material.GitMaterialId,
		Value:         material.Value,
		Active:        material.Active,
		GitCommit: &git.GitCommitBase{
			Commit:  material.LastSeenHash,
			Author:  material.CommitAuthor,
			Date:    material.CommitDate,
			Message: material.CommitMessage,
		},
	}
	return materialBean
}

func (impl RepoManagerImpl) FetchChanges(pipelineMaterialId int, from string, to string, count int, showAll bool) (*git.MaterialChangeResp, error) {
	pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(pipelineMaterialId)
	if err != nil {
		return nil, err
	}
	gitMaterial, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
	if err != nil {
		return nil, err
	}

	pipelineMaterialType := pipelineMaterial.Type

	if pipelineMaterialType == sql.SOURCE_TYPE_BRANCH_FIXED {
		return impl.FetchGitCommitsForBranchFixPipeline(pipelineMaterial, gitMaterial, showAll)
	} else if pipelineMaterialType == sql.SOURCE_TYPE_WEBHOOK {
		return impl.FetchGitCommitsForWebhookTypePipeline(pipelineMaterial, gitMaterial)
	} else {
		err = errors.New("unknown pipelineMaterial Type")
	}

	return nil, err
}

func (impl RepoManagerImpl) FetchGitCommitsForBranchFixPipeline(pipelineMaterial *sql.CiPipelineMaterial, gitMaterial *sql.GitMaterial, showAll bool) (*git.MaterialChangeResp, error) {
	response := &git.MaterialChangeResp{}
	response.LastFetchTime = gitMaterial.LastFetchTime
	if pipelineMaterial.Errored {
		impl.logger.Infow("errored material ", "id", pipelineMaterial.Id, "errMsg", pipelineMaterial.ErrorMsg)
		if !gitMaterial.CheckoutStatus {
			response.IsRepoError = true
			response.RepoErrorMsg = gitMaterial.FetchErrorMessage
		} else {
			response.IsBranchError = true
			response.BranchErrorMsg = pipelineMaterial.ErrorMsg
		}

		return response, nil
	}
	commits := make([]*git.GitCommitBase, 0)
	err := json.Unmarshal([]byte(pipelineMaterial.CommitHistory), &commits)
	if err != nil {
		return nil, err
	}
	if len(gitMaterial.FilterPattern) == 0 {
		response.Commits = commits
		return response, nil
	}

	filterCommits := make([]*git.GitCommitBase, 0)
	for _, commit := range commits {
		excluded := impl.gitManager.PathMatcher(commit.FileStats, gitMaterial)
		impl.logger.Debugw("include exclude result", "excluded", excluded)
		if showAll {
			commit.Excluded = excluded
			filterCommits = append(filterCommits, commit)
		} else if excluded == false {
			filterCommits = append(filterCommits, commit)
		} else {
			impl.logger.Debugw("this item neither included nor excluded, skip this")
		}
	}
	response.Commits = filterCommits
	return response, nil
}

func (impl RepoManagerImpl) FetchGitCommitsForWebhookTypePipeline(pipelineMaterial *sql.CiPipelineMaterial, gitMaterial *sql.GitMaterial) (*git.MaterialChangeResp, error) {
	response := &git.MaterialChangeResp{}
	response.LastFetchTime = gitMaterial.LastFetchTime
	if pipelineMaterial.Errored && !gitMaterial.CheckoutStatus {
		response.IsRepoError = true
		response.RepoErrorMsg = gitMaterial.FetchErrorMessage
		return response, nil
	}

	pipelineMaterialId := pipelineMaterial.Id
	matchedWebhookMappings, err := impl.webhookEventDataMappingRepository.GetMatchedCiPipelineMaterialWebhookDataMappingForPipelineMaterial(pipelineMaterialId)
	if err != nil {
		impl.logger.Errorw("error in getting webhook mapping for pipelineId ", "id", pipelineMaterialId, "errMsg", err)
		return nil, err
	}

	if len(matchedWebhookMappings) == 0 {
		impl.logger.Infow("no webhook mapping for ci pipeline, pipelineId", "pipelineId", pipelineMaterialId)
		return response, nil
	}

	var webhookDataIds []int
	for _, webhookMapping := range matchedWebhookMappings {
		webhookDataIds = append(webhookDataIds, webhookMapping.WebhookDataId)
	}

	impl.logger.Debugw("webhookDataIds :", webhookDataIds)

	if len(webhookDataIds) == 0 {
		impl.logger.Debugw("webhook data Ids are null skipping")
		return response, nil
	}

	webhookEventDataArr, err := impl.webhookEventParsedDataRepository.GetWebhookEventParsedDataByIds(webhookDataIds, 15)
	if err != nil {
		impl.logger.Errorw("error in getting webhook data for ids ", "ids", webhookDataIds, "errMsg", err)
		return nil, err
	}

	if len(webhookEventDataArr) == 0 {
		impl.logger.Infow("no webhooks data found for ci pipeline, pipelineId", "pipelineId", pipelineMaterialId)
		return response, nil
	}

	var commits []*git.GitCommitBase
	for _, webhookEventData := range webhookEventDataArr {
		gitCommit := &git.GitCommitBase{
			WebhookData: impl.webhookEventBeanConverter.ConvertFromWebhookParsedDataSqlBean(webhookEventData),
		}
		commits = append(commits, gitCommit)
	}
	response.Commits = commits
	return response, nil
}

func (impl RepoManagerImpl) GetCommitInfoForTag(gitCtx git.GitContext, request *git.CommitMetadataRequest) (*git.GitCommitBase, error) {
	pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(request.PipelineMaterialId)
	if err != nil {
		return nil, err
	}
	gitMaterial, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
	if err != nil {
		return nil, err
	}
	if !gitMaterial.CheckoutStatus {
		return nil, fmt.Errorf("checkout not succeed please checkout first %s", gitMaterial.Url)
	}
	//refresh repo. and notify all pending
	//lock inside watcher itself
	_, err = impl.gitWatcher.PollAndUpdateGitMaterial(gitMaterial)
	if err != nil {
		impl.logger.Infow("error in refreshing repo", "req", request, "err", err)
		return nil, err
	}
	//lock for getting commit
	repoLock := impl.locker.LeaseLocker(gitMaterial.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(gitMaterial.Id)
	}()
	commit, err := impl.repositoryManager.GetCommitForTag(gitCtx, gitMaterial.CheckoutLocation, request.GitTag)
	return commit, err
}

func (impl RepoManagerImpl) GetCommitMetadata(gitCtx git.GitContext, pipelineMaterialId int, gitHash string) (*git.GitCommitBase, error) {
	pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(pipelineMaterialId)
	if err != nil {
		return nil, err
	}
	gitMaterial, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
	if err != nil {
		return nil, err
	}
	if !gitMaterial.CheckoutStatus {
		return nil, fmt.Errorf("checkout not succeed please checkout first %s", gitMaterial.Url)
	}
	repoLock := impl.locker.LeaseLocker(gitMaterial.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(gitMaterial.Id)
	}()
	commit, err := impl.repositoryManager.GetCommitMetadata(gitCtx, gitMaterial.CheckoutLocation, gitHash)
	return commit, err
}

func (impl RepoManagerImpl) GetLatestCommitForBranch(gitCtx git.GitContext, pipelineMaterialId int, branchName string) (*git.GitCommitBase, error) {
	pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(pipelineMaterialId)

	if err != nil {
		impl.logger.Errorw("error in getting pipeline material ", "pipelineMaterialId", pipelineMaterialId, "err", err)
		return nil, err
	}

	gitMaterial, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
	if err != nil {
		impl.logger.Errorw("error in getting material ", "gitMaterialId", pipelineMaterial.GitMaterialId, "err", err)
		return nil, err
	}

	if !gitMaterial.CheckoutStatus {
		return nil, fmt.Errorf("checkout not succeed please checkout first %s", gitMaterial.Url)
	}

	repoLock := impl.locker.LeaseLocker(gitMaterial.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(gitMaterial.Id)
	}()

	userName, password, err := git.GetUserNamePassword(gitMaterial.GitProvider)

	gitCtx = gitCtx.WithCredentials(userName, password).
		WithTLSData(gitMaterial.GitProvider.CaCert, gitMaterial.GitProvider.TlsKey, gitMaterial.GitProvider.TlsCert, gitMaterial.GitProvider.EnableTLSVerification)
	updated, repo, err := impl.repositoryManager.Fetch(gitCtx, gitMaterial.Url, gitMaterial.CheckoutLocation)
	if !updated {
		impl.logger.Warn("repository is up to date")
	}
	if err == nil {
		gitMaterial.CheckoutStatus = true
	} else {
		gitMaterial.CheckoutStatus = false
		gitMaterial.CheckoutMsgAny = err.Error()
		gitMaterial.FetchErrorMessage = err.Error()

		impl.logger.Errorw("error in fetching the repository ", "gitMaterial", gitMaterial, "err", err)
		return nil, err
	}

	err = impl.materialRepository.Update(gitMaterial)
	if err != nil {
		impl.logger.Errorw("error in updating material repo", "err", err, "material", gitMaterial)
		return nil, err
	}
	ciPipelineMaterial, err := impl.ciPipelineMaterialRepository.FindByGitMaterialId(gitMaterial.Id)
	if err != nil {
		impl.logger.Errorw("unable to load material", "err", err, "ciPipelineMaterial", ciPipelineMaterial)
		return nil, err
	}
	err = impl.updatePipelineMaterialCommit(gitCtx, ciPipelineMaterial)
	if err != nil {
		impl.logger.Errorw("error in updating pipeline material", "err", err)
		return nil, err
	}

	commits, err := impl.repositoryManager.ChangesSinceByRepository(gitCtx, repo, branchName, "", "", 1, gitMaterial.CheckoutLocation, false)

	if commits == nil {
		return nil, err
	} else {
		return commits[0], err
	}
}

func (impl RepoManagerImpl) GetCommitMetadataForPipelineMaterial(gitCtx git.GitContext, pipelineMaterialId int, gitHash string) (*git.GitCommitBase, error) {
	// fetch ciPipelineMaterial
	pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(pipelineMaterialId)
	if err != nil {
		impl.logger.Errorw("error while fetching ciPipelineMaterial", "pipelineMaterialId", pipelineMaterialId, "err", err)
		return nil, err
	}

	// validations
	if pipelineMaterial.Type == sql.SOURCE_TYPE_WEBHOOK {
		return nil, errors.New("fetching commit info is not supported for webhook based ci pipeline material")
	}
	branchName := pipelineMaterial.Value
	if len(branchName) == 0 {
		impl.logger.Errorw("branch name is empty", "pipelineMaterialId", pipelineMaterialId)
		return nil, errors.New("branch name is empty in ci pipeline material")
	}

	// fetch gitMaterial
	gitMaterialId := pipelineMaterial.GitMaterialId
	gitMaterial, err := impl.materialRepository.FindById(gitMaterialId)
	if err != nil {
		impl.logger.Errorw("error while fetching gitMaterial", "gitMaterialId", gitMaterialId, "err", err)
		return nil, err
	}

	gitCtx = gitCtx.WithCredentials(gitMaterial.GitProvider.UserName, gitMaterial.GitProvider.Password).
		WithTLSData(gitMaterial.GitProvider.CaCert, gitMaterial.GitProvider.TlsKey, gitMaterial.GitProvider.TlsCert, gitMaterial.GitProvider.EnableTLSVerification) // validate checkout status of gitMaterial
	if !gitMaterial.CheckoutStatus {
		impl.logger.Errorw("checkout not success", "gitMaterialId", gitMaterialId)
		return nil, fmt.Errorf("checkout not succeed please checkout first %s", gitMaterial.Url)
	}

	// lock-unlock
	repoLock := impl.locker.LeaseLocker(gitMaterial.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(gitMaterial.Id)
	}()
	var repository *git.GitRepository
	commits, err := impl.repositoryManager.ChangesSinceByRepository(gitCtx, repository, branchName, "", gitHash, 1, gitMaterial.CheckoutLocation, true)
	if err != nil {
		if strings.Contains(err.Error(), git.NO_COMMIT_CUSTOM_ERROR_MESSAGE) {
			impl.logger.Warnw("No commit found for given hash", "hash", gitHash, "branchName", branchName)
			return nil, nil
		}
		impl.logger.Errorw("error while fetching commit info", "pipelineMaterialId", pipelineMaterialId, "gitHash", gitHash, "err", err)
		return nil, err
	}

	if len(commits) > 1 {
		impl.logger.Errorw("more than one commits found", "commitHash", gitHash, "pipelineMaterialId", pipelineMaterialId, "branch", branchName)
		return nil, fmt.Errorf("more than one commit found for commit hash %s in branch %s", gitHash, branchName)
	}

	if len(commits) == 0 {
		impl.logger.Errorw("no commits found", "commitHash", gitHash, "pipelineMaterialId", pipelineMaterialId, "branch", branchName)
		return nil, nil
	}
	commit := commits[0]
	excluded := impl.gitManager.PathMatcher(commit.FileStats, gitMaterial)
	commit.Excluded = excluded
	return commits[0], err
}

func (impl RepoManagerImpl) GetReleaseChanges(gitCtx git.GitContext, request *ReleaseChangesRequest) (*git.GitChanges, error) {
	pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(request.PipelineMaterialId)
	if err != nil {
		return nil, err
	}
	gitMaterial, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
	if err != nil {
		return nil, err
	}
	if !gitMaterial.CheckoutStatus {
		return nil, fmt.Errorf("checkout not succeed please checkout first %s", gitMaterial.Url)
	}
	repoLock := impl.locker.LeaseLocker(gitMaterial.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(gitMaterial.Id)
	}()

	gitCtx = gitCtx.WithCredentials(gitMaterial.GitProvider.UserName, gitMaterial.GitProvider.Password).
		WithTLSData(gitMaterial.GitProvider.CaCert, gitMaterial.GitProvider.TlsKey, gitMaterial.GitProvider.TlsCert, gitMaterial.GitProvider.EnableTLSVerification)

	gitChanges, err := impl.repositoryManagerAnalytics.ChangesSinceByRepositoryForAnalytics(gitCtx, gitMaterial.CheckoutLocation, request.OldCommit, request.NewCommit)
	if err != nil {
		impl.logger.Errorw("error in computing changes", "req", request, "err", err)
	} else {
		impl.logger.Infow("commits found for ", "req", request, "commits", len(gitChanges.Commits))
	}

	return gitChanges, err
}

type ReleaseChangesRequest struct {
	PipelineMaterialId int    `json:"pipelineMaterialId"`
	OldCommit          string `json:"oldCommit"`
	NewCommit          string `json:"newCommit"`
}

func (impl RepoManagerImpl) RefreshGitMaterial(req *git.RefreshGitMaterialRequest) (*git.RefreshGitMaterialResponse, error) {
	material := &sql.GitMaterial{Id: req.GitMaterialId}
	res := &git.RefreshGitMaterialResponse{}
	//refresh repo. and notify all pipeline for changes
	//lock inside watcher itself
	material, err := impl.gitWatcher.PollAndUpdateGitMaterial(material)
	if err != nil {
		res.ErrorMsg = err.Error()
	} else if material.LastFetchErrorCount > 0 {
		res.ErrorMsg = material.FetchErrorMessage
		res.LastFetchTime = material.LastFetchTime
	} else {
		res.Message = "successfully refreshed material"
		res.LastFetchTime = material.LastFetchTime
	}
	return res, err
}

func (impl RepoManagerImpl) GetWebhookAndCiDataById(id int, ciPipelineMaterialId int) (*git.WebhookAndCiData, error) {

	impl.logger.Debugw("Getting webhook data ", "id", id)

	webhookDataFromDb, err := impl.webhookEventParsedDataRepository.GetWebhookEventParsedDataById(id)

	if err != nil {
		impl.logger.Errorw("error in getting webhook data for Id ", "Id", id, "err", err)
		return nil, err
	}

	filterData, err := impl.webhookEventDataMappingRepository.GetWebhookPayloadFilterDataForPipelineMaterialId(ciPipelineMaterialId, id)
	if err != nil {
		impl.logger.Errorw("error in getting webhook payload filter data for webhookParsedId ", "Id", id, "err", err)
		return nil, err
	}

	webhookData := impl.webhookEventBeanConverter.ConvertFromWebhookParsedDataSqlBean(webhookDataFromDb)

	webhookAndCiData := &git.WebhookAndCiData{
		WebhookData: webhookData,
	}

	if filterData != nil {
		webhookAndCiData.ExtraEnvironmentVariables = util.BuildExtraEnvironmentVariablesForCi(filterData.FilterResults, webhookDataFromDb.CiEnvVariableData)
	}

	return webhookAndCiData, nil
}

func (impl RepoManagerImpl) GetAllWebhookEventConfigForHost(req *git.WebhookEventConfigRequest) ([]*git.WebhookEventConfig, error) {
	gitHostId := req.GitHostId
	gitHostName := req.GitHostName
	var webhookEventsFromDb []*sql.GitHostWebhookEvent
	var err error
	impl.logger.Debugw("Getting All webhook event config ", "gitHostId", gitHostId)
	if gitHostName != "" {
		webhookEventsFromDb, err = impl.webhookEventRepository.GetAllGitHostWebhookEventByGitHostName(gitHostName)
		if err != nil {
			impl.logger.Errorw("error in getting webhook events", "gitHostName", gitHostName, "err", err)
			return nil, err
		}
		if webhookEventsFromDb == nil || len(webhookEventsFromDb) == 0 {
			webhookEventsFromDb, err = impl.webhookEventRepository.GetAllGitHostWebhookEventByGitHostId(gitHostId)
			if err != nil {
				impl.logger.Errorw("error in getting webhook events", "gitHostId", gitHostId, "err", err)
				return nil, err
			}
		}
	} else {
		webhookEventsFromDb, err = impl.webhookEventRepository.GetAllGitHostWebhookEventByGitHostId(gitHostId)
		if err != nil {
			impl.logger.Errorw("error in getting webhook events", "gitHostId", gitHostId, "err", err)
			return nil, err
		}
	}

	// build events
	return impl.convertSqlBeansToWebhookEventConfig(webhookEventsFromDb)

}

func (impl RepoManagerImpl) convertSqlBeansToWebhookEventConfig(webhookEventsFromDb []*sql.GitHostWebhookEvent) ([]*git.WebhookEventConfig, error) {
	var webhookEvents []*git.WebhookEventConfig
	for _, webhookEventFromDb := range webhookEventsFromDb {
		webhookEvent := impl.webhookEventBeanConverter.ConvertFromWebhookEventSqlBean(webhookEventFromDb)
		webhookEvents = append(webhookEvents, webhookEvent)
	}

	return webhookEvents, nil
}

func (impl RepoManagerImpl) GetWebhookEventConfig(eventId int) (*git.WebhookEventConfig, error) {

	impl.logger.Debugw("Getting webhook event config ", "eventId", eventId)

	webhookEventFromDb, err := impl.webhookEventRepository.GetWebhookEventConfigByEventId(eventId)

	if err != nil {
		impl.logger.Errorw("error in getting webhook event ", "eventId", eventId, "err", err)
		return nil, err
	}

	webhookEvent := impl.webhookEventBeanConverter.ConvertFromWebhookEventSqlBean(webhookEventFromDb)

	return webhookEvent, nil
}

func (impl RepoManagerImpl) GetWebhookPayloadDataForPipelineMaterialId(request *git.WebhookPayloadDataRequest) (*git.WebhookPayloadDataResponse, error) {
	impl.logger.Debugw("Getting webhook payload data ", "request", request)

	pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(request.CiPipelineMaterialId)
	if err != nil {
		impl.logger.Errorw("error in getting ci pipeline material", "err", err)
		return nil, err
	}

	if pipelineMaterial.Type != sql.SOURCE_TYPE_WEBHOOK {
		error := "pipeline material is not of webhook type"
		impl.logger.Error(error)
		return nil, errors.New(error)
	}

	gitMaterial, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
	if err != nil {
		impl.logger.Errorw("error in getting git material", "err", err)
		return nil, err
	}

	webhookSourceTypeValue := &git.WebhookSourceTypeValue{}
	err = json.Unmarshal([]byte(pipelineMaterial.Value), &webhookSourceTypeValue)
	if err != nil {
		impl.logger.Errorw("error in json parsing", "err", err, "ciPipelineMaterialJsonValue", pipelineMaterial.Value)
		return nil, err
	}

	eventConfig, err := impl.GetWebhookEventConfig(webhookSourceTypeValue.EventId)
	if err != nil {
		impl.logger.Errorw("error in getting webhook event config ", "err", err)
		return nil, err
	}

	mappings, err := impl.webhookEventDataMappingRepository.GetWebhookPayloadDataForPipelineMaterialId(request.CiPipelineMaterialId, request.Limit, request.Offset, request.EventTimeSortOrder)
	if err != nil {
		impl.logger.Errorw("error in getting webhook payload data ", "err", err)
		return nil, err
	}

	// build filters
	filters := make(map[string]string)
	for _, selector := range eventConfig.Selectors {
		if condition, ok := webhookSourceTypeValue.Condition[selector.Id]; ok {
			filters[selector.Name] = condition
		}
	}

	// build payloads
	var webhookPayloadDataPayloadResponses []*git.WebhookPayloadDataPayloadsResponse
	for _, mapping := range mappings {

		if len(mapping.FilterResults) == 0 {
			continue
		}

		matchedFiltersCount := 0
		failedFiltersCount := 0

		for _, filterResult := range mapping.FilterResults {
			if filterResult.ConditionMatched {
				matchedFiltersCount = matchedFiltersCount + 1
			} else {
				failedFiltersCount = failedFiltersCount + 1
			}
		}

		webhookPayloadDataPayloadResponse := &git.WebhookPayloadDataPayloadsResponse{
			ParsedDataId:        mapping.WebhookDataId,
			EventTime:           mapping.UpdatedOn,
			MatchedFiltersCount: matchedFiltersCount,
			FailedFiltersCount:  failedFiltersCount,
			MatchedFilters:      mapping.ConditionMatched,
		}

		webhookPayloadDataPayloadResponses = append(webhookPayloadDataPayloadResponses, webhookPayloadDataPayloadResponse)
	}

	webhookPayloadDataResponse := &git.WebhookPayloadDataResponse{
		RepositoryUrl: gitMaterial.Url,
		Filters:       filters,
		Payloads:      webhookPayloadDataPayloadResponses,
	}

	return webhookPayloadDataResponse, nil
}

func (impl RepoManagerImpl) GetWebhookPayloadFilterDataForPipelineMaterialId(request *git.WebhookPayloadFilterDataRequest) (*git.WebhookPayloadFilterDataResponse, error) {
	impl.logger.Debugw("Getting webhook payload filter data ", "request", request)

	mapping, err := impl.webhookEventDataMappingRepository.GetWebhookPayloadFilterDataForPipelineMaterialId(request.CiPipelineMaterialId, request.ParsedDataId)
	if err != nil {
		impl.logger.Errorw("error in getting webhook filter payload data ", "err", err)
		return nil, err
	}

	parsedData, err := impl.webhookEventParsedDataRepository.GetWebhookEventParsedDataById(request.ParsedDataId)
	if err != nil {
		impl.logger.Errorw("error in getting parsed webhook data ", "err", err)
		return nil, err
	}

	filterResults := mapping.FilterResults

	// build payload
	var webhookPayloadFilterDataSelectorResponses []*git.WebhookPayloadFilterDataSelectorResponse
	if len(filterResults) > 0 {
		for _, filterResult := range filterResults {
			webhookPayloadFilterDataSelectorResponse := &git.WebhookPayloadFilterDataSelectorResponse{
				SelectorName:      filterResult.SelectorName,
				SelectorCondition: filterResult.SelectorCondition,
				SelectorValue:     filterResult.SelectorValue,
				Match:             filterResult.ConditionMatched,
			}
			webhookPayloadFilterDataSelectorResponses = append(webhookPayloadFilterDataSelectorResponses, webhookPayloadFilterDataSelectorResponse)
		}
	}

	webhookPayloadFilterDataResponse := &git.WebhookPayloadFilterDataResponse{
		PayloadId:     parsedData.PayloadDataId,
		SelectorsData: webhookPayloadFilterDataSelectorResponses,
	}

	return webhookPayloadFilterDataResponse, nil
}
