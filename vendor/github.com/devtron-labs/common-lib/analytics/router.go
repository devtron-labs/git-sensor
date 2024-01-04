package analytics

import (
	"github.com/devtron-labs/common-lib/analytics/pprof"
	"github.com/devtron-labs/common-lib/analytics/statsViz"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type AnalyticsRouter struct {
	pprofRouter    pprof.PProfRouter
	statsVizRouter statsViz.StatsVizRouter
}

func (r AnalyticsRouter) InitAnalyticsRouter(pprofSubRouter *mux.Router, statvizSubRouter *mux.Router) {
	r.pprofRouter.InitPProfRouter(pprofSubRouter)
	r.statsVizRouter.InitStatsVizRouter(statvizSubRouter)
}

func NewAnalyticsRouter(logger *zap.SugaredLogger) *AnalyticsRouter {
	return &AnalyticsRouter{
		pprofRouter:    pprof.NewPProfRouter(logger),
		statsVizRouter: statsViz.NewStatsVizRouter(logger),
	}
}
