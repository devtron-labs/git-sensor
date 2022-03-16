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
	"github.com/gammazero/workerpool"
	//"github.com/devtron-labs/git-sensor/pkg"
	"encoding/json"
	"fmt"
	"github.com/nats-io/stan"
	"go.uber.org/zap"
	"gopkg.in/robfig/cron.v3"
	"time"
)

type GitWatcherImpl struct {
	repositoryManager            RepositoryManager
	materialRepo                 sql.MaterialRepository
	cron                         *cron.Cron
	logger                       *zap.SugaredLogger
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository
	nats                         stan.Conn
	locker                       *internal.RepositoryLocker
	pollConfig                   *PollConfig
	webhookHandler 	 			 WebhookHandler
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
	nats stan.Conn, webhookHandler WebhookHandler) (*GitWatcherImpl, error) {

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
		nats:                         nats,
		pollConfig:                   cfg,
		webhookHandler:     		  webhookHandler,
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
		wp.Submit(func() {
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

		err = impl.nats.Publish(internal.POLL_CI_TOPIC, b)
		impl.logger.Debugw("published msg", "msg", material)
		if err != nil {
			impl.logger.Infow("error in publishing message", "err", err)
		}
	}
}

func (impl GitWatcherImpl) SubscribePull() error {
	aw := (FETCH_TIMEOUT_SEC + 5) * time.Second
	_, err := impl.nats.Subscribe(internal.POLL_CI_TOPIC, func(msg *stan.Msg) {
		impl.logger.Debugw("received msg", "msg", msg)
		msg.Ack() //ack immediate if lost next min it would get a new message
		material := &sql.GitMaterial{}
		err := json.Unmarshal(msg.Data, material)
		if err != nil {
			impl.logger.Infow("err in reading msg", "err", err)
			return
		}
		impl.logger.Debugw("polling for material", "id", material.Id, "url", material.Url, "msg timestamp", msg.Timestamp)
		_, err = impl.pollAndUpdateGitMaterial(material)
		if err != nil {
			impl.logger.Errorw("err in poling", "material", material, "err", err)
		}
	}, stan.SetManualAckMode(), stan.AckWait(aw))
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
	gitProvider := material.GitProvider
	userName, password, err := GetUserNamePassword(gitProvider)
	location, err := GetLocationForMaterial(material)
	if err != nil {
		impl.logger.Errorw("error in determining location", "url", material.Url, "err", err)
		return err
	}
	updated, repo, err := impl.repositoryManager.Fetch(userName, password, material.Url, location)
	if err != nil {
		impl.logger.Errorw("error in fetching material details ", "repo", material.Url, "err", err)
		// there might be the case if ssh private key gets flush from disk, so creating and single retrying in this case
		if gitProvider.AuthMode == sql.AUTH_MODE_SSH {
			err = impl.repositoryManager.CreateSshFileAndConfigureSshCommand(location, gitProvider.Id, gitProvider.SshPrivateKey)
			if err != nil {
				impl.logger.Errorw("error in creating/configuring ssh private key on disk ", "repo", material.Url, "gitProviderId", gitProvider.Id, "err", err)
				return err
			}else{
				impl.logger.Info("Retrying fetching for" , "repo",  material.Url)
				updated, repo, err = impl.repositoryManager.Fetch(userName, password, material.Url, location)
				if err != nil {
					impl.logger.Errorw("error in fetching material details in retry", "repo", material.Url, "err", err)
					return err
				}
			}
		}else{
			return err
		}
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
		err = impl.nats.Publish(internal.NEW_CI_MATERIAL_TOPIC, mb)
		if err != nil {
			impl.logger.Errorw("error in publishing material modification msg ", "material", material)
		}

	}
	return nil
}


func (impl GitWatcherImpl) SubscribeWebhookEvent() error {
	aw := (FETCH_TIMEOUT_SEC + 5) * time.Second
	_, err := impl.nats.Subscribe(internal.WEBHOOK_EVENT_TOPIC, func(msg *stan.Msg) {
		impl.logger.Debugw("received msg", "msg", msg)
		msg.Ack() //ack immediate if lost next min it would get a new message
		webhookEvent := &WebhookEvent{}
		err := json.Unmarshal(msg.Data, webhookEvent)
		if err != nil {
			impl.logger.Infow("err in reading msg", "err", err)
			return
		}
		impl.webhookHandler.HandleWebhookEvent(webhookEvent)
	}, stan.SetManualAckMode(), stan.AckWait(aw))
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
