package internal

import (
	"github.com/caarlos0/env"
)

type Configuration struct {
	CommitStatsTimeoutInSec int    `env:"COMMIT_STATS_TIMEOUT_IN_SEC" envDefault:"2"`
	EnableFileStats         bool   `env:"ENABLE_FILE_STATS" envDefault:"false"`
	GitHistoryCount         int    `env:"GIT_HISTORY_COUNT" envDefault:"15"`
	MinLimit                int    `env:"MIN_LIMIT_FOR_PVC" envDefault:"1"` // in MB
	UseGitCli               bool   `env:"USE_GIT_CLI" envDefault:"false"`
	CliCmdTimeoutGlobal     int    `env:"CLI_CMD_TIMEOUT_GLOBAL" envDefault:"0"`
	CliCmdTimeoutJson       string `env:"CLI_CMD_TIMEOUT_JSON" envDefault:"0"`
}

func ParseConfiguration() (*Configuration, error) {
	cfg := &Configuration{}
	err := env.Parse(cfg)
	return cfg, err
}
