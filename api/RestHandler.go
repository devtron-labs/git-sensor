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

package api

import (
	"encoding/json"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/devtron-labs/git-sensor/pkg"
	"github.com/devtron-labs/git-sensor/pkg/git"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type RestHandler interface {
	SaveGitProvider(w http.ResponseWriter, r *http.Request)
	AddRepo(w http.ResponseWriter, r *http.Request)
	UpdateRepo(w http.ResponseWriter, r *http.Request)
	SavePipelineMaterial(w http.ResponseWriter, r *http.Request)
	FetchChanges(w http.ResponseWriter, r *http.Request)
	GetHeadForPipelineMaterials(w http.ResponseWriter, r *http.Request)
	GetCommitMetadata(w http.ResponseWriter, r *http.Request)
	ReloadAllMaterial(w http.ResponseWriter, r *http.Request)
	ReloadMaterial(w http.ResponseWriter, r *http.Request)
	GetChangesInRelease(w http.ResponseWriter, r *http.Request)
	GetCommitInfoForTag(w http.ResponseWriter, r *http.Request)
	RefreshGitMaterial(w http.ResponseWriter, r *http.Request)
	GetWebhookData(w http.ResponseWriter, r *http.Request)
	GetAllWebhookEventConfigForHost(w http.ResponseWriter, r *http.Request)
	GetWebhookEventConfig(w http.ResponseWriter, r *http.Request)
}

func NewRestHandlerImpl(repositoryManager pkg.RepoManager, logger *zap.SugaredLogger, gitOperationService git.GitOperationService) *RestHandlerImpl {
	return &RestHandlerImpl{repositoryManager: repositoryManager, logger: logger, gitOperationService: gitOperationService}
}

type RestHandlerImpl struct {
	repositoryManager pkg.RepoManager
	logger            *zap.SugaredLogger
	gitOperationService git.GitOperationService
}

type Response struct {
	Code   int         `json:"code,omitempty"`
	Status string      `json:"status,omitempty"`
	Result interface{} `json:"result,omitempty"`
	Errors []*ApiError `json:"errors,omitempty"`
}
type ApiError struct {
	HttpStatusCode    int         `json:"-"`
	Code              string      `json:"code,omitempty"`
	InternalMessage   string      `json:"internalMessage,omitempty"`
	UserMessage       interface{} `json:"userMessage,omitempty"`
	UserDetailMessage string      `json:"userDetailMessage,omitempty"`
}

func (impl RestHandlerImpl) writeJsonResp(w http.ResponseWriter, err error, respBody interface{}, status int) {
	response := Response{}
	response.Code = status
	response.Status = http.StatusText(status)
	if err == nil {
		response.Result = respBody
	} else {
		apiErr := &ApiError{}
		apiErr.Code = "000" // 000=unknown
		apiErr.InternalMessage = err.Error()
		apiErr.UserMessage = respBody
		response.Errors = []*ApiError{apiErr}

	}
	b, err := json.Marshal(response)
	if err != nil {
		impl.logger.Error("error in marshaling err object", err)
		status = 500
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(b)
}

func (handler RestHandlerImpl) SaveGitProvider(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	gitProvider := &sql.GitProvider{}
	err := decoder.Decode(gitProvider)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("update gitProvider request ", "req", gitProvider)
	res, err := handler.repositoryManager.SaveGitProvider(gitProvider)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, res, http.StatusOK)
	}
}

func (handler RestHandlerImpl) AddRepo(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var Repo []*sql.GitMaterial
	err := decoder.Decode(&Repo)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("add repo request ", "req", Repo)
	res, err := handler.repositoryManager.AddRepo(Repo)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, res, http.StatusOK)
	}
}

func (handler RestHandlerImpl) UpdateRepo(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var Repo *sql.GitMaterial
	err := decoder.Decode(&Repo)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("update repo request ", "req", Repo)
	res, err := handler.repositoryManager.UpdateRepo(Repo)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, res, http.StatusOK)
	}
}

func (handler RestHandlerImpl) SavePipelineMaterial(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var material []*sql.CiPipelineMaterial
	err := decoder.Decode(&material)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("update pipelineMaterial request ", "req", material)
	res, err := handler.repositoryManager.SavePipelineMaterial(material)
	if err != nil {
		handler.logger.Errorw("error in saving pipeline material", "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, nil, res, http.StatusOK)
	}
}

func (handler RestHandlerImpl) ReloadAllMaterial(w http.ResponseWriter, r *http.Request) {
	handler.logger.Infow("reload all pipelineMaterial request")
	handler.repositoryManager.ReloadAllRepo()
	handler.writeJsonResp(w, nil, "reloaded se logs for detail", http.StatusOK)
}

func (handler RestHandlerImpl) ReloadMaterial(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	materialId, err := strconv.Atoi(vars["materialId"])
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("reload all pipelineMaterial request", "id", materialId)
	err = handler.repositoryManager.ResetRepo(materialId)
	if err != nil {
		handler.logger.Errorw("error in reloading pipeline material", "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, nil, "reloaded", http.StatusOK)
	}
}

//-------------
func (handler RestHandlerImpl) FetchChanges(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	material := &git.FetchScmChangesRequest{}
	err := decoder.Decode(material)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("update pipelineMaterial request ", "req", material)
	commits, err := handler.repositoryManager.FetchChanges(material.PipelineMaterialId, material.From, material.To, material.Count)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, commits, http.StatusOK)
	}
}

func (handler RestHandlerImpl) GetHeadForPipelineMaterials(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	material := &git.HeadRequest{}
	err := decoder.Decode(material)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("update pipelineMaterial request ", "req", material)
	commits, err := handler.repositoryManager.GetHeadForPipelineMaterials(material.MaterialIds)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, commits, http.StatusOK)
	}
}

func (handler RestHandlerImpl) GetCommitMetadata(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	material := &git.CommitMetadataRequest{}
	err := decoder.Decode(material)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("commit detail request", "req", material)
	var commits *git.GitCommit
	if len(material.GitTag) > 0 {
		commits, err = handler.repositoryManager.GetCommitInfoForTag(material)
	} else if len(material.BranchName) > 0 {
		commits, err = handler.gitOperationService.GetLatestCommitForBranch(material.PipelineMaterialId, material.BranchName)
	}else{
		commits, err = handler.repositoryManager.GetCommitMetadata(material.PipelineMaterialId, material.GitHash)
	}
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, commits, http.StatusOK)
	}
}

func (handler RestHandlerImpl) GetCommitInfoForTag(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	material := &git.CommitMetadataRequest{}
	err := decoder.Decode(material)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("tag detail request", "req", material)
	commits, err := handler.repositoryManager.GetCommitInfoForTag(material)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, commits, http.StatusOK)
	}

}

func (handler RestHandlerImpl) GetChangesInRelease(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := &pkg.ReleaseChangesRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("commit detail request", "req", request)
	commits, err := handler.repositoryManager.GetReleaseChanges(request)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, commits, http.StatusOK)
	}
}

func (handler RestHandlerImpl) RefreshGitMaterial(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := &git.RefreshGitMaterialRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("commit detail request", "req", request)
	resp, err := handler.repositoryManager.RefreshGitMaterial(request)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, err, resp, http.StatusOK)
	}
}


func (handler RestHandlerImpl) GetWebhookData(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := &git.WebhookDataRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("webhook data request ", "req", request)
	webhookData, err := handler.repositoryManager.GetWebhookDataById(request.Id)

	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, err, webhookData, http.StatusOK)
	}
}


func (handler RestHandlerImpl) GetAllWebhookEventConfigForHost(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := &git.WebhookEventConfigRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("webhook event config request ", "req", request)

	webhookEventConfigArr, err := handler.repositoryManager.GetAllWebhookEventConfigForHost(request.GitHostId)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, err, webhookEventConfigArr, http.StatusOK)
	}
}


func (handler RestHandlerImpl) GetWebhookEventConfig(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := &git.WebhookEventConfigRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("webhook event config request ", "req", request)

	webhookEventConfig, err := handler.repositoryManager.GetWebhookEventConfig(request.EventId)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, err, webhookEventConfig, http.StatusOK)
	}

}


