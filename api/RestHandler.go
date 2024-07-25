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

package api

import (
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/git-sensor/bean"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"github.com/devtron-labs/git-sensor/pkg"
	"github.com/devtron-labs/git-sensor/pkg/git"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
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
	GetCommitMetadataForPipelineMaterial(w http.ResponseWriter, r *http.Request)
	ReloadAllMaterial(w http.ResponseWriter, r *http.Request)
	ReloadMaterial(w http.ResponseWriter, r *http.Request)
	ReloadMaterials(w http.ResponseWriter, r *http.Request)
	GetChangesInRelease(w http.ResponseWriter, r *http.Request)
	GetCommitInfoForTag(w http.ResponseWriter, r *http.Request)
	RefreshGitMaterial(w http.ResponseWriter, r *http.Request)
	GetWebhookData(w http.ResponseWriter, r *http.Request)
	GetAllWebhookEventConfigForHost(w http.ResponseWriter, r *http.Request)
	GetWebhookEventConfig(w http.ResponseWriter, r *http.Request)
	GetWebhookPayloadDataForPipelineMaterialId(w http.ResponseWriter, r *http.Request)
	GetWebhookPayloadFilterDataForPipelineMaterialId(w http.ResponseWriter, r *http.Request)
}

func NewRestHandlerImpl(repositoryManager pkg.RepoManager, logger *zap.SugaredLogger) *RestHandlerImpl {
	return &RestHandlerImpl{repositoryManager: repositoryManager, logger: logger}
}

type RestHandlerImpl struct {
	repositoryManager pkg.RepoManager
	logger            *zap.SugaredLogger
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
	cloningMode := git.CloningModeFull
	if Repo != nil && len(Repo) > 0 {
		cloningMode = Repo[0].CloningMode
	}
	gitCtx := git.BuildGitContext(r.Context()).WithCloningMode(cloningMode)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("add repo request ", "req", Repo)
	res, err := handler.repositoryManager.AddRepo(gitCtx, Repo)
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
	gitCtx := git.BuildGitContext(r.Context()).WithCloningMode(Repo.CloningMode)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("update repo request ", "req", Repo)
	res, err := handler.repositoryManager.UpdateRepo(gitCtx, Repo)
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
	gitCtx := git.BuildGitContext(r.Context())

	handler.logger.Infow("update pipelineMaterial request ", "req", material)
	res, err := handler.repositoryManager.SavePipelineMaterial(gitCtx, material)
	if err != nil {
		handler.logger.Errorw("error in saving pipeline material", "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, nil, res, http.StatusOK)
	}
}

func (handler RestHandlerImpl) ReloadAllMaterial(w http.ResponseWriter, r *http.Request) {
	handler.logger.Infow(bean.GetReloadAllLog("received reload all pipelineMaterial request"))
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	req := &bean.ReloadAllMaterialQuery{}
	err := decoder.Decode(req, r.URL.Query())
	if err != nil {
		handler.logger.Errorw(bean.GetReloadAllLog("invalid query params, ReloadAllMaterial"), "err", err, "query", r.URL.Query())
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	if req.Start < 0 || req.End < 0 || req.Start > req.End {
		handler.logger.Errorw(bean.GetReloadAllLog("invalid request, ReloadAllMaterial"), "query", r.URL.Query())
		handler.writeJsonResp(w, fmt.Errorf("invalid query params"), nil, http.StatusBadRequest)
		return
	}
	gitCtx := git.BuildGitContext(r.Context())
	err = handler.repositoryManager.ReloadAllRepo(gitCtx, req)
	if err != nil {
		handler.logger.Errorw(bean.GetReloadAllLog("error in request, ReloadAllMaterial"), "query", r.URL.Query(), "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	handler.logger.Infow(bean.GetReloadAllLog("Reloaded materials successfully!"), "query", r.URL.Query())
	handler.writeJsonResp(w, nil, "Reloaded materials successfully!", http.StatusOK)
}

func (handler RestHandlerImpl) ReloadMaterials(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var request ReloadMaterialsDto
	err := decoder.Decode(&request)

	//materialId, err := strconv.Atoi(vars["materialId"])
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	for _, materialReq := range request.ReloadMaterial {
		handler.logger.Infow("reload all pipelineMaterial request", "id", materialReq.GitmaterialId)
		gitCtx := git.BuildGitContext(r.Context()).WithCloningMode(materialReq.CloningMode)
		err = handler.repositoryManager.ResetRepo(gitCtx, materialReq.GitmaterialId)
		if err != nil {
			handler.logger.Errorw("error in reloading pipeline material", "err", err)
			//handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
		}
	}
	//TODO: handle in such a way that it can propagate which material weren't able to reload
	handler.writeJsonResp(w, nil, Response{
		Code:   http.StatusOK,
		Status: http.StatusText(http.StatusOK),
	}, http.StatusOK)

}

func (handler RestHandlerImpl) ReloadMaterial(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gitCtx := git.BuildGitContext(r.Context())
	materialId, err := strconv.Atoi(vars["materialId"])
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("reload all pipelineMaterial request", "id", materialId)
	err = handler.repositoryManager.ResetRepo(gitCtx, materialId)
	if err != nil {
		handler.logger.Errorw("error in reloading pipeline material", "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, nil, "reloaded", http.StatusOK)
	}
}

// -------------
func (handler RestHandlerImpl) FetchChanges(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	material := &git.FetchScmChangesRequest{}
	err := decoder.Decode(material)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("fetch git materials ", "req", material)
	commits, err := handler.repositoryManager.FetchChanges(material.PipelineMaterialId, material.From, material.To, material.Count, material.ShowAll)
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
	gitCtx := git.BuildGitContext(r.Context())
	handler.logger.Infow("commit detail request", "req", material)
	var commits *git.GitCommitBase
	if len(material.GitTag) > 0 {
		commits, err = handler.repositoryManager.GetCommitInfoForTag(gitCtx, material)
	} else if len(material.BranchName) > 0 {
		commits, err = handler.repositoryManager.GetLatestCommitForBranch(gitCtx, material.PipelineMaterialId, material.BranchName)
	} else {
		commits, err = handler.repositoryManager.GetCommitMetadata(gitCtx, material.PipelineMaterialId, material.GitHash)
	}
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, commits, http.StatusOK)
	}
}

func (handler RestHandlerImpl) GetCommitMetadataForPipelineMaterial(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	material := &git.CommitMetadataRequest{}
	err := decoder.Decode(material)
	if err != nil {
		handler.logger.Errorw("err", "material", material, "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	gitCtx := git.BuildGitContext(r.Context())

	handler.logger.Infow("commit detail request for pipeline material", "req", material)
	commit, err := handler.repositoryManager.GetCommitMetadataForPipelineMaterial(gitCtx, material.PipelineMaterialId, material.GitHash)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, commit, http.StatusOK)
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
	gitCtx := git.BuildGitContext(r.Context())

	commits, err := handler.repositoryManager.GetCommitInfoForTag(gitCtx, material)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
	} else {
		handler.writeJsonResp(w, err, commits, http.StatusOK)
	}

}

func (handler RestHandlerImpl) GetChangesInRelease(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	gitCtx := git.BuildGitContext(r.Context())
	request := &pkg.ReleaseChangesRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Error(err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("commit detail request", "req", request)
	commits, err := handler.repositoryManager.GetReleaseChanges(gitCtx, request)
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
	handler.logger.Debug("GetWebhookData API call")
	decoder := json.NewDecoder(r.Body)
	request := &git.WebhookDataRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Errorw("error in decoding request of GetWebhookData ", "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Debugw("webhook data request ", "req", request)
	webhookData, err := handler.repositoryManager.GetWebhookAndCiDataById(request.Id, request.CiPipelineMaterialId)

	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, err, webhookData, http.StatusOK)
	}
}

func (handler RestHandlerImpl) GetAllWebhookEventConfigForHost(w http.ResponseWriter, r *http.Request) {
	handler.logger.Debug("GetAllWebhookEventConfigForHost API call")
	decoder := json.NewDecoder(r.Body)
	request := &git.WebhookEventConfigRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Errorw("error in decoding request of GetAllWebhookEventConfigForHost ", "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("webhook event config request ", "req", request)

	var webhookEventConfigArr []*git.WebhookEventConfig
	webhookEventConfigArr, err = handler.repositoryManager.GetAllWebhookEventConfigForHost(request)

	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, err, webhookEventConfigArr, http.StatusOK)
	}
}

func (handler RestHandlerImpl) GetWebhookEventConfig(w http.ResponseWriter, r *http.Request) {
	handler.logger.Debug("GetWebhookEventConfig API call")
	decoder := json.NewDecoder(r.Body)
	request := &git.WebhookEventConfigRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Errorw("error in decoding request of GetWebhookEventConfig ", "err", err)
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

func (handler RestHandlerImpl) GetWebhookPayloadDataForPipelineMaterialId(w http.ResponseWriter, r *http.Request) {
	handler.logger.Debug("GetWebhookPayloadDataForPipelineMaterialId API call")
	decoder := json.NewDecoder(r.Body)
	request := &git.WebhookPayloadDataRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Errorw("error in decoding request of GetWebhookPayloadDataForPipelineMaterialId ", "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("webhook payload data request ", "req", request)

	data, err := handler.repositoryManager.GetWebhookPayloadDataForPipelineMaterialId(request)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, err, data, http.StatusOK)
	}

}

func (handler RestHandlerImpl) GetWebhookPayloadFilterDataForPipelineMaterialId(w http.ResponseWriter, r *http.Request) {
	handler.logger.Debug("GetWebhookPayloadFilterDataForPipelineMaterialId API call")
	decoder := json.NewDecoder(r.Body)
	request := &git.WebhookPayloadFilterDataRequest{}
	err := decoder.Decode(request)
	if err != nil {
		handler.logger.Errorw("error in decoding request of GetWebhookPayloadFilterDataForPipelineMaterialId ", "err", err)
		handler.writeJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Infow("webhook payload filter data request ", "req", request)

	data, err := handler.repositoryManager.GetWebhookPayloadFilterDataForPipelineMaterialId(request)
	if err != nil {
		handler.writeJsonResp(w, err, nil, http.StatusInternalServerError)
	} else {
		handler.writeJsonResp(w, err, data, http.StatusOK)
	}

}
