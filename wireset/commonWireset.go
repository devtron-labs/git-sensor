/*
 * Copyright (c) 2024. Devtron Inc.
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

package wireset

import (
	"github.com/devtron-labs/common-lib/monitoring"
	pubsub "github.com/devtron-labs/common-lib/pubsub-lib"
	"github.com/devtron-labs/git-sensor/api"
	"github.com/devtron-labs/git-sensor/app"
	"github.com/devtron-labs/git-sensor/internals"
	"github.com/devtron-labs/git-sensor/internals/logger"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"github.com/devtron-labs/git-sensor/pkg"
	"github.com/devtron-labs/git-sensor/pkg/git"
	"github.com/google/wire"
)

var CommonWireSet = wire.NewSet(app.NewApp,
	api.NewMuxRouter,
	internals.ParseConfiguration,
	logger.NewSugaredLogger,
	api.NewRestHandlerImpl,
	wire.Bind(new(api.RestHandler), new(*api.RestHandlerImpl)),
	api.NewGrpcHandlerImpl,
	sql.NewMaterialRepositoryImpl,
	wire.Bind(new(sql.MaterialRepository), new(*sql.MaterialRepositoryImpl)),
	sql.NewDbConnection,
	sql.GetConfig,
	sql.NewCiPipelineMaterialRepositoryImpl,
	wire.Bind(new(sql.CiPipelineMaterialRepository), new(*sql.CiPipelineMaterialRepositoryImpl)),
	sql.NewGitProviderRepositoryImpl,
	wire.Bind(new(sql.GitProviderRepository), new(*sql.GitProviderRepositoryImpl)),
	git.NewGitManagerImpl,
	wire.Bind(new(git.GitManager), new(*git.GitManagerImpl)),
	git.NewRepositoryManagerAnalyticsImpl,
	wire.Bind(new(git.RepositoryManagerAnalytics), new(*git.RepositoryManagerAnalyticsImpl)),
	pkg.NewRepoManagerImpl,
	wire.Bind(new(pkg.RepoManager), new(*pkg.RepoManagerImpl)),
	git.NewGitWatcherImpl,
	wire.Bind(new(git.GitWatcher), new(*git.GitWatcherImpl)),
	internals.NewRepositoryLocker,
	//internal.NewNatsConnection,
	pubsub.NewPubSubClientServiceImpl,
	sql.NewWebhookEventRepositoryImpl,
	wire.Bind(new(sql.WebhookEventRepository), new(*sql.WebhookEventRepositoryImpl)),
	sql.NewWebhookEventParsedDataRepositoryImpl,
	wire.Bind(new(sql.WebhookEventParsedDataRepository), new(*sql.WebhookEventParsedDataRepositoryImpl)),
	sql.NewWebhookEventDataMappingRepositoryImpl,
	wire.Bind(new(sql.WebhookEventDataMappingRepository), new(*sql.WebhookEventDataMappingRepositoryImpl)),
	sql.NewWebhookEventDataMappingFilterResultRepositoryImpl,
	wire.Bind(new(sql.WebhookEventDataMappingFilterResultRepository), new(*sql.WebhookEventDataMappingFilterResultRepositoryImpl)),
	git.NewWebhookEventBeanConverterImpl,
	wire.Bind(new(git.WebhookEventBeanConverter), new(*git.WebhookEventBeanConverterImpl)),
	git.NewWebhookEventServiceImpl,
	wire.Bind(new(git.WebhookEventService), new(*git.WebhookEventServiceImpl)),
	git.NewWebhookEventParserImpl,
	wire.Bind(new(git.WebhookEventParser), new(*git.WebhookEventParserImpl)),
	git.NewWebhookHandlerImpl,
	wire.Bind(new(git.WebhookHandler), new(*git.WebhookHandlerImpl)),
	monitoring.NewMonitoringRouter,
)
