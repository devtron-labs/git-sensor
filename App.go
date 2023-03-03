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
	"context"
	"fmt"
	pb "github.com/devtron-labs/git-sensor/protos"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	pubsub "github.com/devtron-labs/common-lib/pubsub-lib"
	"github.com/devtron-labs/git-sensor/api"
	"github.com/devtron-labs/git-sensor/internal/middleware"
	"github.com/devtron-labs/git-sensor/pkg/git"
	"github.com/go-pg/pg"
	"github.com/gorilla/handlers"
	"go.uber.org/zap"
)

type App struct {
	MuxRouter          *api.MuxRouter
	Logger             *zap.SugaredLogger
	watcher            *git.GitWatcherImpl
	server             *http.Server
	db                 *pg.DB
	pubSubClient       *pubsub.PubSubClientServiceImpl
	GrpcControllerImpl *api.GrpcControllerImpl
}

func NewApp(MuxRouter *api.MuxRouter, Logger *zap.SugaredLogger, impl *git.GitWatcherImpl, db *pg.DB, pubSubClient *pubsub.PubSubClientServiceImpl, GrpcControllerImpl *api.GrpcControllerImpl) *App {
	return &App{
		MuxRouter:          MuxRouter,
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
	app.MuxRouter.Init()
	//authEnforcer := casbin2.Create()

	h := handlers.RecoveryHandler(handlers.RecoveryLogger(&PanicLogger{Logger: app.Logger}))(app.MuxRouter.Router)

	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: h}
	app.MuxRouter.Router.Use(middleware.PrometheusMiddleware)
	app.server = server
	err := server.ListenAndServe()

	if err != nil {
		app.Logger.Errorw("error in startup", "err", err)
		os.Exit(2)
	}

	app.initGrpcServer()
}

func (app *App) initGrpcServer() {

	//listen on the port
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start server %v", err)
	}
	// create a new gRPC server
	grpcServer := grpc.NewServer()
	// register the greet service
	pb.RegisterGitServiceServer(grpcServer, app.GrpcControllerImpl)
	log.Printf("Server started at %v", lis.Addr())
	//list is the port, the grpc server needs to start there
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}

func (app *App) Stop() {
	app.Logger.Infow("orchestrator shutdown initiating")
	timeoutContext, _ := context.WithTimeout(context.Background(), 5*time.Second)
	app.Logger.Infow("stopping cron")
	app.watcher.StopCron()
	app.Logger.Infow("closing router")
	err := app.server.Shutdown(timeoutContext)
	if err != nil {
		app.Logger.Errorw("error in mux router shutdown", "err", err)
	}
	app.Logger.Infow("closing db connection")
	err = app.db.Close()
	if err != nil {
		app.Logger.Errorw("error in closing db connection", "err", err)
	}

	app.Logger.Infow("housekeeping done. exiting now")
}
