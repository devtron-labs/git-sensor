package internal

import "github.com/caarlos0/env"

type Configuration struct {
	CommitStatsTimeoutInSec int  `env:"COMMIT_STATS_TIMEOUT_IN_SEC" envDefault:"2"`
	EnableFileStats         bool `env:"ENABLE_FILE_STATS" envDefault:"false"`
	GitHistoryCount         int  `env:"GIT_HISTORY_COUNT" envDefault:"15"`
}

func ParseConfiguration() (*Configuration, error) {
	cfg := &Configuration{}
	err := env.Parse(cfg)
	return cfg, err
}
