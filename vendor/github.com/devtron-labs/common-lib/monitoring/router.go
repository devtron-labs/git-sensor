package monitoring

import (
	"github.com/devtron-labs/common-lib/monitoring/pprof"
	"github.com/devtron-labs/common-lib/monitoring/statsViz"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type MonitoringRouter struct {
	pprofRouter    pprof.PProfRouter
	statsVizRouter statsViz.StatsVizRouter
}

func (r MonitoringRouter) InitMonitoringRouter(pprofSubRouter *mux.Router, statvizSubRouter *mux.Router) {
	r.pprofRouter.InitPProfRouter(pprofSubRouter)
	r.statsVizRouter.InitStatsVizRouter(statvizSubRouter)
}

func NewMonitoringRouter(logger *zap.SugaredLogger) *MonitoringRouter {
	return &MonitoringRouter{
		pprofRouter:    pprof.NewPProfRouter(logger),
		statsVizRouter: statsViz.NewStatsVizRouter(logger),
	}
}
