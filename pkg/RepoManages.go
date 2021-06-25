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

package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/devtron-labs/git-sensor/pkg/git"
	"go.uber.org/zap"
	_ "gopkg.in/robfig/cron.v3"
)

type RepoManager interface {
	GetHeadForPipelineMaterials(ids []int) ([]*git.CiPipelineMaterialBean, error)
	FetchChanges(pipelineMaterialId int, from string, to string, count int) (*git.MaterialChangeResp, error) //limit
	GetCommitMetadata(pipelineMaterialId int, gitHash string) (*git.GitCommit, error)

	SaveGitProvider(provider *sql.GitProvider) (*sql.GitProvider, error)
	AddRepo(material []*sql.GitMaterial) ([]*sql.GitMaterial, error)
	UpdateRepo(material *sql.GitMaterial) (*sql.GitMaterial, error)
	SavePipelineMaterial(material []*sql.CiPipelineMaterial) ([]*sql.CiPipelineMaterial, error)
	ReloadAllRepo()
	ResetRepo(materialId int) error
	GetReleaseChanges(request *ReleaseChangesRequest) (*git.GitChanges, error)
	GetCommitInfoForTag(request *git.CommitMetadataRequest) (*git.GitCommit, error)
	RefreshGitMaterial(req *git.RefreshGitMaterialRequest) (*git.RefreshGitMaterialResponse, error)
}

type RepoManagerImpl struct {
	logger                       *zap.SugaredLogger
	materialRepository           sql.MaterialRepository
	repositoryManager            git.RepositoryManager
	gitProviderRepository        sql.GitProviderRepository
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository
	locker                       *internal.RepositoryLocker
	gitWatcher                   git.GitWatcher
	webhookEventRepository		 sql.WebhookEventRepository
}

func NewRepoManagerImpl(
	logger *zap.SugaredLogger,
	materialRepository sql.MaterialRepository,
	repositoryManager git.RepositoryManager,
	gitProviderRepository sql.GitProviderRepository,
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository,
	locker *internal.RepositoryLocker,
	gitWatcher git.GitWatcher, webhookEventRepository sql.WebhookEventRepository,
) *RepoManagerImpl {
	return &RepoManagerImpl{
		logger:                       logger,
		materialRepository:           materialRepository,
		repositoryManager:            repositoryManager,
		gitProviderRepository:        gitProviderRepository,
		ciPipelineMaterialRepository: ciPipelineMaterialRepository,
		locker:                       locker,
		gitWatcher:                   gitWatcher,
		webhookEventRepository: 	  webhookEventRepository,
	}
}

func (impl RepoManagerImpl) SavePipelineMaterial(materials []*sql.CiPipelineMaterial) ([]*sql.CiPipelineMaterial, error) {
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
		err := impl.ciPipelineMaterialRepository.Update(old)
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
	err := impl.updatePipelineMaterialCommit(append(newMaterial, oldNotDeleted...))
	if err != nil {
		return nil, err
	}
	return materials, nil
}

func (impl RepoManagerImpl) updatePipelineMaterialCommit(materials []*sql.CiPipelineMaterial) error {
	var materialCommits []*sql.CiPipelineMaterial
	for _, pipelineMaterial := range materials {

		//some points are missing so fetch
		pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(pipelineMaterial.Id)
		if err != nil {
			impl.logger.Errorw("material not found", "material", pipelineMaterial)
			return err
		}

		if pipelineMaterial.Type == sql.SOURCE_TYPE_TAG_REGEX {
			commits := make([]*git.GitCommit, 0)
			b, err := json.Marshal(commits)
			if err != nil {
				pipelineMaterial.Errored = true
				pipelineMaterial.ErrorMsg = err.Error()
			} else {
				pipelineMaterial.CommitHistory = string(b)
				pipelineMaterial.Errored = false
				pipelineMaterial.ErrorMsg = ""
			}
			materialCommits = append(materialCommits, pipelineMaterial)
			continue
		}
		material, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
		if err != nil {
			impl.logger.Errorw("error in fetching material", "err", err)
			continue
		}
		commits, err := impl.repositoryManager.ChangesSince(material.CheckoutLocation, pipelineMaterial.Value, "", "", 0)
		//commits, err := impl.FetchChanges(pipelineMaterial.Id, "", "", 0)
		if err == nil {
			impl.logger.Infow("commits found", "commit", commits)
			b, err := json.Marshal(commits)
			if err == nil {
				pipelineMaterial.CommitHistory = string(b)
				if len(commits) > 0 {
					latestCommit := commits[0]
					pipelineMaterial.LastSeenHash = latestCommit.Commit
					pipelineMaterial.CommitAuthor = latestCommit.Author
					pipelineMaterial.CommitDate = latestCommit.Date
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
	return provider, err
}

//handle update
func (impl RepoManagerImpl) AddRepo(materials []*sql.GitMaterial) ([]*sql.GitMaterial, error) {
	for _, material := range materials {
		_, err := impl.addRepo(material)
		if err != nil {
			impl.logger.Errorw("error in saving material ", "material", material, "err", err)
			return materials, err
		}
	}
	return materials, nil
}

func (impl RepoManagerImpl) UpdateRepo(material *sql.GitMaterial) (*sql.GitMaterial, error) {
	existingMaterial, err := impl.materialRepository.FindById(material.Id)
	if err != nil {
		impl.logger.Errorw("err", err)
		return nil, err
	}
	existingMaterial.Name = material.Name
	existingMaterial.Url = material.Url
	existingMaterial.GitProviderId = material.GitProviderId
	existingMaterial.Deleted = material.Deleted
	existingMaterial.CheckoutStatus = false

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
		impl.logger.Errorw("err", err)
		return nil, err
	}

	if !existingMaterial.Deleted {
		err = impl.checkoutUpdatedRepo(material.Id)
		if err != nil {
			impl.logger.Errorw("err", err)
			return nil, err
		}
	}
	return existingMaterial, nil
}

func (impl RepoManagerImpl) checkoutUpdatedRepo(materialId int) error {
	material, err := impl.materialRepository.FindById(materialId)
	if err != nil {
		impl.logger.Errorw("error in fetching material", "id", materialId, "err", err)
		return err
	}
	_, err = impl.checkoutMaterial(material)
	if err != nil {
		impl.logger.Errorw("error in repo refresh", "id", material, "err", err)
		return err
	}
	return nil
}

func (impl RepoManagerImpl) addRepo(material *sql.GitMaterial) (*sql.GitMaterial, error) {
	err := impl.materialRepository.Save(material)
	if err != nil {
		impl.logger.Errorw("error in saving material ", "material", material, "err", err)
		return material, err
	}
	return impl.checkoutRepo(material)
}

func (impl RepoManagerImpl) checkoutRepo(material *sql.GitMaterial) (*sql.GitMaterial, error) {
	repoLock := impl.locker.LeaseLocker(material.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(material.Id)
	}()
	return impl.checkoutMaterial(material)
}

func (impl RepoManagerImpl) checkoutMaterial(material *sql.GitMaterial) (*sql.GitMaterial, error) {
	impl.logger.Infow("checking out material", "id", material.Id)
	gitProvider, err := impl.gitProviderRepository.GetById(material.GitProviderId)
	if err != nil {
		return material, err
	}
	userName, password, err := git.GetUserNamePassword(gitProvider)
	if err != nil {
		return material, nil
	}
	checkoutPath, err := git.GetLocationForMaterial(material)
	if err != nil {
		return material, err
	}
	err = impl.repositoryManager.Add(checkoutPath, material.Url, userName, password)
	if err == nil {
		material.CheckoutLocation = checkoutPath
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
	err = impl.updatePipelineMaterialCommit(ciPipelineMaterial)
	if err != nil {
		impl.logger.Errorw("error in updating pipeline material", "err", err)
	}
	return material, nil
}

func (impl RepoManagerImpl) ReloadAllRepo() {
	materials, err := impl.materialRepository.FindAll()
	if err != nil {
		impl.logger.Errorw("error in reloading materials")
	}
	for _, material := range materials {
		if _, err := impl.checkoutRepo(material); err != nil {
			impl.logger.Errorw("error in checkout", "material", material, "err", err)
		}

	}
}
func (impl RepoManagerImpl) ResetRepo(materialId int) error {
	material, err := impl.materialRepository.FindById(materialId)
	if err != nil {
		impl.logger.Errorw("error in fetching material", "id", materialId, "err", err)
		return err
	}
	_, err = impl.checkoutRepo(material)
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
		GitCommit: &git.GitCommit{
			Commit: material.LastSeenHash,
			Author: material.CommitAuthor,
			Date:   material.CommitDate,
		},
	}
	return materialBean
}

func (impl RepoManagerImpl) FetchChanges(pipelineMaterialId int, from string, to string, count int) (*git.MaterialChangeResp, error) {
	pipelineMaterial, err := impl.ciPipelineMaterialRepository.FindById(pipelineMaterialId)
	if err != nil {
		return nil, err
	}
	gitMaterial, err := impl.materialRepository.FindById(pipelineMaterial.GitMaterialId)
	if err != nil {
		return nil, err
	}

	if  pipelineMaterial.Type != sql.SOURCE_TYPE_PULL_REQUEST {
		return impl.FetchGitCommitsForOtherThanPrType(pipelineMaterial, gitMaterial)
	}else{
		return impl.FetchGitCommitsForPrType(pipelineMaterial)
	}

	return nil, nil
}


func (impl RepoManagerImpl) FetchGitCommitsForOtherThanPrType(pipelineMaterial *sql.CiPipelineMaterial, gitMaterial *sql.GitMaterial) (*git.MaterialChangeResp, error) {
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
	commits := make([]*git.GitCommit, 0)
	err := json.Unmarshal([]byte(pipelineMaterial.CommitHistory), &commits)
	if err != nil {
		return nil, err
	}
	response.Commits = commits
	return response, nil
}


func (impl RepoManagerImpl) FetchGitCommitsForPrType(pipelineMaterial *sql.CiPipelineMaterial) (*git.MaterialChangeResp, error) {
	response := &git.MaterialChangeResp{}

	pipelineMaterialId := pipelineMaterial.Id

	webhookMappings, err := impl.webhookEventRepository.GetCiPipelineMaterialPrWebhookMapping(pipelineMaterialId)
	if err != nil {
		impl.logger.Errorw("error in getting webhook mapping for pipelineId ", "id", pipelineMaterialId, "errMsg", err)
		return nil, err
	}

	if len(webhookMappings) == 0 {
		impl.logger.Infow("no webhook mapping for ci pipeline, pipelineId", "pipelineId", pipelineMaterialId)
		return nil, err
	}

	var prWebhookEventIds []int
	for _, webhookMapping := range webhookMappings {
		prWebhookEventIds = append(prWebhookEventIds, webhookMapping.PrWebhookDataId)
	}

	prWebhookEvents, err := impl.webhookEventRepository.GetOpenPrEventDataByIds(prWebhookEventIds, 15)
	if err != nil {
		impl.logger.Errorw("error in getting webhook for ids ", "ids", prWebhookEventIds, "errMsg", err)
		return nil, err
	}

	if len(prWebhookEvents) == 0 {
		impl.logger.Infow("no webhooks for ci pipeline, pipelineId", "pipelineId", pipelineMaterialId)
		return nil, err
	}

	commits := make([]*git.GitCommit, 0)
	for _, webhookPRDataEvent := range prWebhookEvents {
		gitCommit := &git.GitCommit{
			PrData: &git.PrData{
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
		}
		commits = append(commits, gitCommit)
	}
	response.Commits = commits
	return response, nil
}


func (impl RepoManagerImpl) GetCommitInfoForTag(request *git.CommitMetadataRequest) (*git.GitCommit, error) {
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
	commit, err := impl.repositoryManager.GetCommitForTag(gitMaterial.CheckoutLocation, request.GitTag)
	return commit, err
}

func (impl RepoManagerImpl) GetCommitMetadata(pipelineMaterialId int, gitHash string) (*git.GitCommit, error) {
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
	commit, err := impl.repositoryManager.GetCommitMetadata(gitMaterial.CheckoutLocation, gitHash)
	return commit, err
}

func (impl RepoManagerImpl) GetReleaseChanges(request *ReleaseChangesRequest) (*git.GitChanges, error) {
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
	gitChanges, err := impl.repositoryManager.ChangesSinceByRepositoryForAnalytics(gitMaterial.CheckoutLocation, pipelineMaterial.Value, request.OldCommit, request.NewCommit)
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