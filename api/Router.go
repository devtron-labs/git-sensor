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
	"github.com/devtron-labs/common-lib/middlewares"
	"github.com/devtron-labs/common-lib/monitoring"
	"github.com/devtron-labs/git-sensor/util"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
)

type MuxRouter struct {
	logger           *zap.SugaredLogger
	Router           *mux.Router
	restHandler      RestHandler
	monitoringRouter *monitoring.MonitoringRouter
}

func NewMuxRouter(logger *zap.SugaredLogger, restHandler RestHandler, monitoringRouter *monitoring.MonitoringRouter) *MuxRouter {
	return &MuxRouter{logger: logger, Router: mux.NewRouter(), restHandler: restHandler, monitoringRouter: monitoringRouter}
}

func (r MuxRouter) Init() {
	pProfListenerRouter := r.Router.PathPrefix("/gitsensor/debug/pprof/").Subrouter()
	statsVizRouter := r.Router.PathPrefix("/gitsensor").Subrouter()

	r.monitoringRouter.InitMonitoringRouter(pProfListenerRouter, statsVizRouter, "/gitsensor")
	r.Router.StrictSlash(true)
	r.Router.Handle("/metrics", promhttp.Handler())
	r.Router.Path("/health").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		response := Response{}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(200)
		response.Code = 200
		response.Result = struct {
			Status    string `json:"status"`
			GitCommit string `json:"gitCommit"`
			BuildTime string `json:"buildTime"`
		}{"OK", util.GitCommit, util.BuildTime}
		b, err := json.Marshal(response)
		if err != nil {
			b = []byte("OK")
			r.logger.Errorw("Unexpected error in apiError", "err", err)
		}
		_, _ = writer.Write(b)
	})
	r.Router.Use(middlewares.Recovery)
	r.Router.Path("/git-provider").HandlerFunc(r.restHandler.SaveGitProvider).Methods("POST")
	r.Router.Path("/git-repo").HandlerFunc(r.restHandler.AddRepo).Methods("POST")
	r.Router.Path("/git-repo").HandlerFunc(r.restHandler.UpdateRepo).Methods("PUT")
	r.Router.Path("/git-pipeline-material").HandlerFunc(r.restHandler.SavePipelineMaterial).Methods("POST")
	r.Router.Path("/git-changes").HandlerFunc(r.restHandler.FetchChanges).Methods("POST")
	r.Router.Path("/git-head").HandlerFunc(r.restHandler.GetHeadForPipelineMaterials).Methods("POST")
	r.Router.Path("/commit-metadata").HandlerFunc(r.restHandler.GetCommitMetadata).Methods("POST")
	r.Router.Path("/pipeline-material-commit-metadata").HandlerFunc(r.restHandler.GetCommitMetadataForPipelineMaterial).Methods("GET")
	r.Router.Path("/tag-commit-metadata").HandlerFunc(r.restHandler.GetCommitInfoForTag).Methods("POST")
	r.Router.Path("/git-repo/refresh").HandlerFunc(r.restHandler.RefreshGitMaterial).Methods("POST")

	r.Router.Path("/admin/reload-all").HandlerFunc(r.restHandler.ReloadAllMaterial).Methods("POST")
	r.Router.Path("/admin/reload/{materialId}").HandlerFunc(r.restHandler.ReloadMaterial).Methods("POST")
	r.Router.Path("/admin/reload-multi/materials").HandlerFunc(r.restHandler.ReloadMaterials).Methods("POST")

	r.Router.Path("/release/changes").HandlerFunc(r.restHandler.GetChangesInRelease).Methods("POST")

	r.Router.Path("/webhook/data").HandlerFunc(r.restHandler.GetWebhookData).Methods("GET")
	r.Router.Path("/webhook/host/events").HandlerFunc(r.restHandler.GetAllWebhookEventConfigForHost).Methods("GET")
	r.Router.Path("/webhook/host/event").HandlerFunc(r.restHandler.GetWebhookEventConfig).Methods("GET")
	r.Router.Path("/webhook/ci-pipeline-material/payload-data").HandlerFunc(r.restHandler.GetWebhookPayloadDataForPipelineMaterialId).Methods("GET")
	r.Router.Path("/webhook/ci-pipeline-material/payload-filter-data").HandlerFunc(r.restHandler.GetWebhookPayloadFilterDataForPipelineMaterialId).Methods("GET")
}
