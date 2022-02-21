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

package git

import (
	"github.com/caarlos0/env"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/internal/middleware"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/devtron-labs/git-sensor/util"
	"github.com/gammazero/workerpool"
	"github.com/nats-io/nats.go"

	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type GitWatcherImpl struct {
	repositoryManager            RepositoryManager
	materialRepo                 sql.MaterialRepository
	cron                         *cron.Cron
	logger                       *zap.SugaredLogger
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository
	pubSubClient                 *internal.PubSubClient
	locker                       *internal.RepositoryLocker
	pollConfig                   *PollConfig
	webhookHandler               WebhookHandler
}

type GitWatcher interface {
	PollAndUpdateGitMaterial(material *sql.GitMaterial) (*sql.GitMaterial, error)
}

type PollConfig struct {
	PollDuration int `env:"POLL_DURATION" envDefault:"2"`
	PollWorker   int `env:"POLL_WORKER" envDefault:"5"`
}

func NewGitWatcherImpl(repositoryManager RepositoryManager,
	materialRepo sql.MaterialRepository,
	logger *zap.SugaredLogger,
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository,
	locker *internal.RepositoryLocker,
	pubSubClient *internal.PubSubClient, webhookHandler WebhookHandler) (*GitWatcherImpl, error) {

	cfg := &PollConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	cronLogger := &CronLoggerImpl{logger: logger}
	cron := cron.New(
		cron.WithChain(
			cron.SkipIfStillRunning(cronLogger),
			cron.Recover(cronLogger)))
	cron.Start()
	watcher := &GitWatcherImpl{
		repositoryManager:            repositoryManager,
		cron:                         cron,
		logger:                       logger,
		ciPipelineMaterialRepository: ciPipelineMaterialRepository,
		materialRepo:                 materialRepo,
		locker:                       locker,
		pubSubClient:                 pubSubClient,
		pollConfig:                   cfg,
		webhookHandler:               webhookHandler,
	}
	logger.Info()
	_, err = cron.AddFunc(fmt.Sprintf("@every %dm", cfg.PollDuration), watcher.Watch)
	if err != nil {
		fmt.Println("error in starting cron")
		return nil, err
	}

	//err = watcher.SubscribePull()
	watcher.SubscribeWebhookEvent()
	return watcher, err
}

func (impl GitWatcherImpl) StopCron() {
	impl.cron.Stop()
}
func (impl GitWatcherImpl) Watch() {
	impl.logger.Infow("starting git watch thread")
	materials, err := impl.materialRepo.FindActive()
	if err != nil {
		impl.logger.Error("error in fetching watchlist", "err", err)
		return
	}
	//impl.Publish(materials)
	middleware.ActiveGitRepoCount.WithLabelValues().Set(float64(len(materials)))
	impl.RunOnWorker(materials)
	impl.logger.Infow("stop git watch thread")
}

func (impl *GitWatcherImpl) RunOnWorker(materials []*sql.GitMaterial) {
	wp := workerpool.New(impl.pollConfig.PollWorker)
	for _, material := range materials {
		if len(material.CiPipelineMaterials) == 0 {
			impl.logger.Infow("no ci pipeline, skipping", "id", material.Id, "url", material.Url)
			continue
		}
		materialMsg := &sql.GitMaterial{Id: material.Id, Url: material.Url}
		wp.SubmitWait(func() {
			_, err := impl.pollAndUpdateGitMaterial(materialMsg)
			if err != nil {
				impl.logger.Errorw("error in polling git material", "material", materialMsg, "err", err)
			}
		})
	}
	wp.StopWait()
}

func (impl GitWatcherImpl) Publish(materials []*sql.GitMaterial) {
	for _, material := range materials {
		if len(material.CiPipelineMaterials) == 0 {
			impl.logger.Infow("no ci pipeline, skipping", "id", material.Id, "url", material.Url)
			continue
		}
		materialMsg := &sql.GitMaterial{Id: material.Id, Url: material.Url}
		b, err := json.Marshal(materialMsg)
		if err != nil {
			impl.logger.Infow("error in serializing", "err", err)
			continue
		}
		impl.logger.Infow("publishing pull msg", "id", material.Id)

		streamInfo, strInfoErr := impl.pubSubClient.JetStrCtxt.StreamInfo(internal.POLL_CI_TOPIC)
		if strInfoErr != nil {
			impl.logger.Errorw("Error while getting stream info", "topic", internal.POLL_CI_TOPIC, "error", strInfoErr)
		}
		if streamInfo == nil {
			//Stream doesn't already exist. Create a new stream from jetStreamContext
			_, addStrError := impl.pubSubClient.JetStrCtxt.AddStream(&nats.StreamConfig{
				Name:     internal.POLL_CI_TOPIC,
				Subjects: []string{internal.POLL_CI_TOPIC + ".*"},
			})
			if addStrError != nil {
				impl.logger.Errorw("Error while creating stream", "topic", internal.POLL_CI_TOPIC, "error", addStrError)
			}
		}

		//Generate random string for passing as Header Id in message
		randString := "MsgHeaderId-" + util.Generate(10)

		_, err = impl.pubSubClient.JetStrCtxt.Publish(internal.POLL_CI_TOPIC, b, nats.MsgId(randString))
		impl.logger.Debugw("published msg", "msg", material)
		if err != nil {
			impl.logger.Infow("error in publishing message", "err", err)
		}
	}
}

//TODO : adhiran : see if we can set durable here
func (impl GitWatcherImpl) SubscribePull() error {
	_, err := impl.pubSubClient.JetStrCtxt.QueueSubscribe(internal.POLL_CI_TOPIC, internal.POLL_CI_TOPIC_GRP, func(msg *nats.Msg) {
		impl.logger.Debugw("received msg", "msg", msg)
		msg.Ack() //ack immediate if lost next min it would get a new message
		material := &sql.GitMaterial{}
		err := json.Unmarshal(msg.Data, material)
		if err != nil {
			impl.logger.Infow("err in reading msg", "err", err)
			return
		}
		msgMetaData, metaErr := msg.Metadata()
		if nil != metaErr {
			impl.logger.Debugw("Error while getting metadata of message", "err", metaErr)
		}
		impl.logger.Debugw("polling for material", "id", material.Id, "url", material.Url, "msg timestamp", msgMetaData.Timestamp)
		_, err = impl.pollAndUpdateGitMaterial(material)
		if err != nil {
			impl.logger.Errorw("err in poling", "material", material, "err", err)
		}
	}, nats.Durable(internal.POLL_CI_TOPIC_DURABLE), nats.DeliverLast(), nats.ManualAck(), nats.BindStream(""))
	return err
}

func (impl GitWatcherImpl) PollAndUpdateGitMaterial(material *sql.GitMaterial) (*sql.GitMaterial, error) {
	//tmp expose remove in future
	return impl.pollAndUpdateGitMaterial(material)
}

func (impl GitWatcherImpl) pollAndUpdateGitMaterial(materialReq *sql.GitMaterial) (*sql.GitMaterial, error) {
	repoLock := impl.locker.LeaseLocker(materialReq.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(materialReq.Id)
	}()
	material, err := impl.materialRepo.FindById(materialReq.Id)
	if err != nil {
		impl.logger.Errorw("error in fetching material ", "material", materialReq, "err", err)
		return nil, err
	}
	err = impl.pollGitMaterialAndNotify(material)
	material.LastFetchTime = time.Now()
	material.FetchStatus = err == nil
	if err != nil {
		material.LastFetchErrorCount = material.LastFetchErrorCount + 1
		material.FetchErrorMessage = err.Error()
	} else {
		material.LastFetchErrorCount = 0
		material.FetchErrorMessage = ""
	}
	err = impl.materialRepo.Update(material)
	if err != nil {
		impl.logger.Errorw("error in updating fetch status", "material", material, "err", err)
	}
	return material, err
}

func (impl GitWatcherImpl) pollGitMaterialAndNotify(material *sql.GitMaterial) error {
	userName, password, err := GetUserNamePassword(material.GitProvider)
	location, err := GetLocationForMaterial(material)
	if err != nil {
		impl.logger.Errorw("error in determining location", "url", material.Url, "err", err)
		return err
	}
	updated, repo, err := impl.repositoryManager.Fetch(userName, password, material.Url, location)
	if err != nil {
		impl.logger.Errorw("error in fetching material details ", "repo", material.Url, "err", err)
		return err
	}
	if !updated {
		return nil
	}
	materials, err := impl.ciPipelineMaterialRepository.FindByGitMaterialId(material.Id)
	if err != nil {
		impl.logger.Errorw("error in calculating head", "err", err, "url", material.Url)
		return err
	}
	var updatedMaterials []*CiPipelineMaterialBean
	var updatedMaterialsModel []*sql.CiPipelineMaterial
	var erroredMaterialsModels []*sql.CiPipelineMaterial
	for _, material := range materials {
		if material.Type != sql.SOURCE_TYPE_BRANCH_FIXED {
			continue
		}
		commits, err := impl.repositoryManager.ChangesSinceByRepository(repo, material.Value, "", "", 15)
		if err != nil {
			material.Errored = true
			material.ErrorMsg = err.Error()
			erroredMaterialsModels = append(erroredMaterialsModels, material)
		} else if len(commits) > 0 {
			latestCommit := commits[0]
			if latestCommit.Commit != material.LastSeenHash {
				//new commit found
				mb := &CiPipelineMaterialBean{
					Id:            material.Id,
					Value:         material.Value,
					GitMaterialId: material.GitMaterialId,
					Type:          material.Type,
					Active:        material.Active,
					GitCommit:     latestCommit,
				}
				updatedMaterials = append(updatedMaterials, mb)

				material.LastSeenHash = latestCommit.Commit
				material.CommitAuthor = latestCommit.Author
				material.CommitDate = latestCommit.Date
				commitJson, _ := json.Marshal(commits)
				material.CommitHistory = string(commitJson)
				material.Errored = false
				material.ErrorMsg = ""
				updatedMaterialsModel = append(updatedMaterialsModel, material)
			}
			middleware.GitMaterialUpdateCounter.WithLabelValues().Inc()
		}
	}
	if len(updatedMaterialsModel) > 0 {
		err = impl.NotifyForMaterialUpdate(updatedMaterials)
		if err != nil {
			impl.logger.Errorw("error in sending notification for materials", "url", material.Url, "update", updatedMaterialsModel)
		}
		err = impl.ciPipelineMaterialRepository.Update(updatedMaterialsModel)
		if err != nil {
			impl.logger.Errorw("error in update db ", "url", material.Url, "update", updatedMaterialsModel)
			impl.logger.Errorw("error in sending notification for materials", "url", material.Url, "update", updatedMaterialsModel)
		}
	}
	if len(erroredMaterialsModels) > 0 {
		err = impl.ciPipelineMaterialRepository.Update(updatedMaterialsModel)
		if err != nil {
			impl.logger.Errorw("error in update db ", "url", material.Url, "update", updatedMaterialsModel)
			impl.logger.Errorw("error in sending notification for materials", "url", material.Url, "update", updatedMaterialsModel)
		}
	}
	return nil
}

func (impl GitWatcherImpl) NotifyForMaterialUpdate(materials []*CiPipelineMaterialBean) error {

	impl.logger.Warnw("material notification", "materials", materials)
	for _, material := range materials {
		mb, err := json.Marshal(material)
		if err != nil {
			impl.logger.Error("err in json marshaling", "err", err)
			continue
		}
		streamInfo, strInfoErr := impl.pubSubClient.JetStrCtxt.StreamInfo(internal.NEW_CI_MATERIAL_TOPIC)
		if strInfoErr != nil {
			impl.logger.Errorw("Error while getting stream info", "topic", internal.NEW_CI_MATERIAL_TOPIC, "error", strInfoErr)
		}
		if streamInfo == nil {
			//Stream doesn't already exist. Create a new stream from jetStreamContext
			_, addStrError := impl.pubSubClient.JetStrCtxt.AddStream(&nats.StreamConfig{
				Name:     internal.NEW_CI_MATERIAL_TOPIC,
				Subjects: []string{internal.NEW_CI_MATERIAL_TOPIC + ".*"},
			})
			if addStrError != nil {
				impl.logger.Errorw("Error while creating stream", "topic", internal.NEW_CI_MATERIAL_TOPIC, "error", addStrError)
			}
		}

		//Generate random string for passing as Header Id in message
		randString := "MsgHeaderId-" + util.Generate(10)

		_, err = impl.pubSubClient.JetStrCtxt.Publish(internal.NEW_CI_MATERIAL_TOPIC, mb, nats.MsgId(randString))
		if err != nil {
			impl.logger.Errorw("error in publishing material modification msg ", "material", material)
		}

	}
	return nil
}

//TODO : adhiran : See if we need to set durable
func (impl GitWatcherImpl) SubscribeWebhookEvent() error {
	_, err := impl.pubSubClient.JetStrCtxt.QueueSubscribe(internal.WEBHOOK_EVENT_TOPIC, internal.WEBHOOK_EVENT_TOPIC_GRP, func(msg *nats.Msg) {
		impl.logger.Debugw("received msg", "msg", msg)
		msg.Ack() //ack immediate if lost next min it would get a new message
		webhookEvent := &WebhookEvent{}
		err := json.Unmarshal(msg.Data, webhookEvent)
		if err != nil {
			impl.logger.Infow("err in reading msg", "err", err)
			return
		}
		impl.webhookHandler.HandleWebhookEvent(webhookEvent)
	}, nats.Durable(internal.WEBHOOK_EVENT_TOPIC_DURABLE), nats.DeliverLast(), nats.ManualAck(), nats.BindStream(""))
	return err
}

/*func (impl GitWatcherImpl) sendRequest(reqByte []byte) {

	url := "http://devtroncd-orchestrator-service-prod:80/webhook/git"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqByte))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("webhook failed" + err.Error())
	}
	defer resp.Body.Close()

//test 3
	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}*/

type CronLoggerImpl struct {
	logger *zap.SugaredLogger
}

func (impl *CronLoggerImpl) Info(msg string, keysAndValues ...interface{}) {
	impl.logger.Infow(msg, keysAndValues...)
}

func (impl *CronLoggerImpl) Error(err error, msg string, keysAndValues ...interface{}) {
	keysAndValues = append([]interface{}{"err", err}, keysAndValues...)
	impl.logger.Errorw(msg, keysAndValues...)
}
