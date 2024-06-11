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

package middleware

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"net/http"
	"strconv"
	"time"
)

var constLabels = map[string]string{"app": "git-sensor"}

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "http_duration_seconds",
		Help:        "Duration of HTTP requests.",
		ConstLabels: constLabels,
	}, []string{"path", "method", "status"})
)

var responseCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:        "http_response_total",
		Help:        "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
		ConstLabels: constLabels,
	},
	[]string{"path", "method", "status"})

var requestCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:        "http_requests_total",
		Help:        "How many HTTP requests processed, partitioned by status code, method and HTTP path.",
		ConstLabels: constLabels,
	},
	[]string{"path", "method"})

var currentRequestGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name:        "http_requests_current",
	Help:        "no of request being served currently",
	ConstLabels: constLabels,
}, []string{"path", "method"})

// prometheusMiddleware implements mux.MiddlewareFunc.
func PrometheusMiddleware(next http.Handler) http.Handler {
	//	prometheus.MustRegister(requestCounter)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		method := r.Method
		requestCounter.WithLabelValues(path, method).Inc()
		g := currentRequestGauge.WithLabelValues(path, method)
		g.Inc()
		defer g.Dec()
		d := newDelegator(w, nil)
		next.ServeHTTP(d, r)
		httpDuration.WithLabelValues(path, method, strconv.Itoa(d.Status())).Observe(time.Since(start).Seconds())
		responseCounter.WithLabelValues(path, method, strconv.Itoa(d.Status())).Inc()
	})
}

var ActiveGitRepoCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name:        "active_git_repo_count",
	Help:        "no of active git repository ",
	ConstLabels: constLabels,
}, []string{})

var GitMaterialUpdateCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:        "total_material_update",
		Help:        "no of update received in given repo and branch",
		ConstLabels: constLabels,
	},
	[]string{})

var GitPullDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:        "git_pull_duration_seconds",
	Help:        "Duration of git pull request ",
	ConstLabels: constLabels,
}, []string{"status", "updated"})

var GitOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "git_operation_duration_seconds",
	Help: "Duration of git operation request",
}, []string{"method", "status"})

var GitMaterialPollCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:        "git_pull",
		Help:        "no of update received in given repo and branch",
		ConstLabels: constLabels,
	},
	[]string{})

var PanicCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:        "panic",
		Help:        "panic in the app",
		ConstLabels: constLabels,
	},
	[]string{})

var CommitStatParsingErrorCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:        "commit_stats_parsing_errors_counter",
		Help:        "total number of parsing errors encountered while processing git commit stats.",
		ConstLabels: constLabels,
	},
	[]string{})
