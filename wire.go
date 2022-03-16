//+build wireinject

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

package main

import (
	"github.com/devtron-labs/git-sensor/api"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/internal/logger"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/devtron-labs/git-sensor/pkg"
	"github.com/devtron-labs/git-sensor/pkg/git"
	"github.com/google/wire"
)

func InitializeApp() (*App, error) {

	wire.Build(
		NewApp,
		api.NewMuxRouter,
		logger.NewSugaredLogger,
		api.NewRestHandlerImpl,
		wire.Bind(new(api.RestHandler), new(*api.RestHandlerImpl)),
		pkg.NewRepoManagerImpl,
		wire.Bind(new(pkg.RepoManager), new(*pkg.RepoManagerImpl)),
		sql.NewMaterialRepositoryImpl,
		wire.Bind(new(sql.MaterialRepository), new(*sql.MaterialRepositoryImpl)),
		sql.NewDbConnection,
		sql.GetConfig,
		sql.NewCiPipelineMaterialRepositoryImpl,
		wire.Bind(new(sql.CiPipelineMaterialRepository), new(*sql.CiPipelineMaterialRepositoryImpl)),
		sql.NewGitProviderRepositoryImpl,
		wire.Bind(new(sql.GitProviderRepository), new(*sql.GitProviderRepositoryImpl)),
		git.NewRepositoryManagerImpl,
		wire.Bind(new(git.RepositoryManager), new(*git.RepositoryManagerImpl)),
		git.NewGitWatcherImpl,
		wire.Bind(new(git.GitWatcher), new(*git.GitWatcherImpl)),
		internal.NewRepositoryLocker,
		internal.NewNatsConnection,
		git.NewGitUtil,
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
	)
	return &App{}, nil
}
