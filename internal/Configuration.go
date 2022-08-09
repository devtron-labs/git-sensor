package internal

import "github.com/caarlos0/env"

type Configuration struct {
	CommitStatsTimeoutInSec int `env:"COMMIT_STATS_TIMEOUT_IN_SEC" envDefault:"2"`
}

func ParseConfiguration() (*Configuration, error) {
	cfg := &Configuration{}
	err := env.Parse(cfg)
	return cfg, err
}
