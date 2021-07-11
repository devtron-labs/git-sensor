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
	"fmt"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"go.uber.org/zap"
	_ "gopkg.in/robfig/cron.v3"
	"gopkg.in/src-d/go-git.v4"
)

type GitOperationService interface {
	GetLatestCommitForBranch(pipelineMaterialId int, branchName string) (*GitCommit, error)
}

type GitOperationServiceImpl struct {
	logger                       *zap.SugaredLogger
	materialRepository           sql.MaterialRepository
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository
	locker                       *internal.RepositoryLocker
	gitRepositoryManager         RepositoryManager
}

func NewGitOperationServiceImpl (
	logger *zap.SugaredLogger, materialRepository sql.MaterialRepository,
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository, locker *internal.RepositoryLocker, gitRepositoryManager RepositoryManager,
) *GitOperationServiceImpl {
	return &GitOperationServiceImpl{
		logger:                       logger,
		materialRepository: 		  materialRepository,
		ciPipelineMaterialRepository: ciPipelineMaterialRepository,
		locker: locker,
		gitRepositoryManager: gitRepositoryManager,
	}
}


func (impl GitOperationServiceImpl) GetLatestCommitForBranch(pipelineMaterialId int, branchName string) (*GitCommit, error) {
	repo, err := impl.FetchRepository(pipelineMaterialId)

	if err != nil {
		impl.logger.Errorw("error in fetching the repository " ,"err", err)
		return nil, err
	}

	commit, err := impl.gitRepositoryManager.GetLatestCommitForBranch(repo, branchName)
	return commit, err
}


func (impl GitOperationServiceImpl) FetchRepository(pipelineMaterialId int) (repo *git.Repository, err error) {
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

	userName, password, err := GetUserNamePassword(gitMaterial.GitProvider)

	updated, repo, err := impl.gitRepositoryManager.Fetch(userName, password, gitMaterial.Url, gitMaterial.CheckoutLocation)

	if err != nil {
		impl.logger.Errorw("error in fetching the repository " ,"err", err)
		return nil, err
	}

	if !updated{
		impl.logger.Warn("repository is up to date")
	}

	return repo, err
}