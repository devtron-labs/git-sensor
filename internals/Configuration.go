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

package internals

import "github.com/caarlos0/env"

type Configuration struct {
	CommitStatsTimeoutInSec int    `env:"COMMIT_STATS_TIMEOUT_IN_SEC" envDefault:"2"`
	EnableFileStats         bool   `env:"ENABLE_FILE_STATS" envDefault:"false"`
	GitHistoryCount         int    `env:"GIT_HISTORY_COUNT" envDefault:"15"`
	MinLimit                int    `env:"MIN_LIMIT_FOR_PVC" envDefault:"1"` // in MB
	UseGitCli               bool   `env:"USE_GIT_CLI" envDefault:"false"`
	UseGitCliAnalytics      bool   `env:"USE_GIT_CLI_ANALYTICS" envDefault:"false"` // This flag is used to compute commitDiff using git-cli only for analytics
	AnalyticsDebug          bool   `env:"ANALYTICS_DEBUG" envDefault:"false"`
	CliCmdTimeoutGlobal     int    `env:"CLI_CMD_TIMEOUT_GLOBAL_SECONDS" envDefault:"0"`
	CliCmdTimeoutJson       string `env:"CLI_CMD_TIMEOUT_JSON" envDefault:""`
	GoGitTimeout            int    `env:"GOGIT_TIMEOUT_SECONDS" envDefault:"10" `
}

func ParseConfiguration() (*Configuration, error) {
	cfg := &Configuration{}
	err := env.Parse(cfg)
	return cfg, err
}
