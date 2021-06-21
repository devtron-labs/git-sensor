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

package internal

import (
	"github.com/caarlos0/env"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan"
	"time"
)

const (
	NEW_CI_MATERIAL_TOPIC = "GIT-SENSOR.NEW-CI-MATERIAL" //{publisher-app-name}-{topic-name}
	POLL_CI_TOPIC ="GIT-SENSOR.PULL"
	WEBHOOK_EVENT_TOPIC = "ORCHESTRATOR.WEBHOOK_EVENT"
)

type PubSubConfig struct {
	NatsServerHost string `env:"NATS_SERVER_HOST" envDefault:"nats://devtron-nats.devtroncd:4222"`
	ClusterId      string `env:"CLUSTER_ID" envDefault:"devtron-stan"`
	ClientId       string `env:"CLIENT_ID" envDefault:"git-sensor"`
}

func NewNatsConnection() (stan.Conn, error) {
	cfg := &PubSubConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	nc, err := nats.Connect(cfg.NatsServerHost, nats.ReconnectWait(10*time.Second), nats.MaxReconnects(100))
	if err != nil {
		return nil, err
	}
	sc, err := stan.Connect(cfg.ClusterId, cfg.ClientId, stan.NatsConn(nc))
	if err != nil {
		return nil, err
	}
	return sc, nil
}