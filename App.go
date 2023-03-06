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
	"fmt"
	pubsub "github.com/devtron-labs/common-lib/pubsub-lib"
	"github.com/devtron-labs/git-sensor/api"
	"github.com/devtron-labs/git-sensor/internal/middleware"
	"github.com/devtron-labs/git-sensor/pkg/git"
	pb "github.com/devtron-labs/protos/git-sensor"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

type App struct {
	Logger             *zap.SugaredLogger
	watcher            *git.GitWatcherImpl
	server             *grpc.Server
	db                 *pg.DB
	pubSubClient       *pubsub.PubSubClientServiceImpl
	GrpcControllerImpl *api.GrpcControllerImpl
}

func NewApp(Logger *zap.SugaredLogger, impl *git.GitWatcherImpl, db *pg.DB, pubSubClient *pubsub.PubSubClientServiceImpl, GrpcControllerImpl *api.GrpcControllerImpl) *App {
	return &App{
		Logger:             Logger,
		watcher:            impl,
		db:                 db,
		pubSubClient:       pubSubClient,
		GrpcControllerImpl: GrpcControllerImpl,
	}
}

type PanicLogger struct {
	Logger *zap.SugaredLogger
}

func (impl *PanicLogger) Println(param ...interface{}) {
	impl.Logger.Errorw("PANIC", "err", param)
	middleware.PanicCounter.WithLabelValues().Inc()
}

func (app *App) Start() {
	port := 8080 //TODO: extract from environment variable
	app.Logger.Infow("starting server on ", "port", port)
	err := app.initGrpcServer(port)
	//app.MuxRouter.Init()
	//authEnforcer := casbin2.Create()
	//
	//h := handlers.RecoveryHandler(handlers.RecoveryLogger(&PanicLogger{Logger: app.Logger}))(app.MuxRouter.Router)
	//
	//server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: h}
	//app.MuxRouter.Router.Use(middleware.PrometheusMiddleware)
	//app.server = server
	//err := server.ListenAndServe()
	//
	if err != nil {
		app.Logger.Errorw("error in startup", "err", err)
		os.Exit(2)
	}
}

func (app *App) initGrpcServer(port int) error {

	//listen on the port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to start server %v", err)
		return err
	}

	// create a new gRPC server
	app.server = grpc.NewServer()

	// register GitSensor service
	pb.RegisterGitSensorServiceServer(app.server, app.GrpcControllerImpl)

	// start listening on address
	if err = app.server.Serve(lis); err != nil {
		log.Fatalf("failed to start: %v", err)
		return err
	}

	log.Printf("server started at %v", lis.Addr())
	return nil
}

// Stop stops the server and cleans resources. Called during shutdown
func (app *App) Stop() {
	app.Logger.Infow("orchestrator shutdown initiating")

	app.Logger.Infow("stopping cron")
	app.watcher.StopCron()

	app.Logger.Infow("gracefully stopping GitSensor")
	app.server.GracefulStop()

	app.Logger.Infow("closing db connection")
	err := app.db.Close()
	if err != nil {
		app.Logger.Errorw("error in closing db connection", "err", err)
	}

	app.Logger.Infow("housekeeping done. exiting now")
}
