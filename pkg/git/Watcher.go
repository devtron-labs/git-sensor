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
	"encoding/json"
	"fmt"
	"github.com/caarlos0/env"
	pubsub "github.com/devtron-labs/common-lib/pubsub-lib"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/internal/middleware"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/gammazero/workerpool"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"time"
)

type GitWatcherImpl struct {
	repositoryManager            RepositoryManager
	materialRepo                 sql.MaterialRepository
	cron                         *cron.Cron
	logger                       *zap.SugaredLogger
	ciPipelineMaterialRepository sql.CiPipelineMaterialRepository
	pubSubClient                 *pubsub.PubSubClientServiceImpl
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
	pubSubClient *pubsub.PubSubClientServiceImpl, webhookHandler WebhookHandler) (*GitWatcherImpl, error) {

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

// Case -> 1. Unique repos (needs regular fetching)
// Case -> 2. Repo with same repo url, and has linked-ci pipeline referencing repo (does not need fetching)
// Case -> 3. Repo with same repo url but no linked-ci (needs fetching only in one repo with same url)

// Proposed approach
// 1. Cron job added for polling the git repos
// 2. Get the list of unique referenced checkout locations and their git providers
// 3. Perform fetch
//     3.1 Fetch
//         3.1.1 Get credentials
//         3.1.2 Get ref checkout location
//         3.1.3 CLI util fetch
//         3.1.4 Update ci pipeline material for all pipelines having same ref checkout location
//     3.2 Update fetch time and fetch status for material which has same checkout location

// (Earlier) Things getting updated in DB (For each git material)
// 1. Ci pipeline material -> Error
// 2. Ci pipeline material -> Error message
// 3. Ci pipeline material -> Last seen hash
// 4. Ci pipeline material -> Author
// 5. Ci pipeline material -> Date
// 6. Ci pipeline material -> Commit history
// 7. Git material -> Last fetch time
// 8. Git material -> Last Fetch status

// Notify
// 1. Topic -> NEW-CI-MATERIAL, Payload -> ci pipeline material bean

// New implementation
func (impl GitWatcherImpl) Watch() {
	impl.logger.Infow("Starting git watcher thread")

	// Get the list of git materials which are referenced by other materials
	refGitMaterials, err := impl.materialRepo.FindAllReferencedGitMaterials()
	if err != nil {
		impl.logger.Errorw("Error getting list of referenced git materials",
			"err", err)
		return
	}
	impl.RunOnWorker(refGitMaterials)
}

func (impl GitWatcherImpl) RunOnWorker(materials []*sql.GitMaterial) {

	// initialized worker pool
	wp := workerpool.New(impl.pollConfig.PollWorker)

	for _, material := range materials {

		nMaterial := &sql.GitMaterial{
			Id:                  material.Id,
			GitProviderId:       material.GitProviderId,
			GitProvider:         material.GitProvider,
			Url:                 material.Url,
			FetchSubmodules:     material.FetchSubmodules,
			Name:                material.Name,
			CheckoutLocation:    material.CheckoutLocation,
			CheckoutStatus:      material.CheckoutStatus,
			CheckoutMsgAny:      material.CheckoutMsgAny,
			Deleted:             material.Deleted,
			LastFetchTime:       material.LastFetchTime,
			LastFetchErrorCount: material.LastFetchErrorCount,
			FetchErrorMessage:   material.FetchErrorMessage,
			RefGitMaterialId:    material.RefGitMaterialId,
			CiPipelineMaterials: material.CiPipelineMaterials,
		}

		wp.Submit(func() {
			_, err := impl.PollAndUpdateGitMaterial(nMaterial)
			if err != nil {
				impl.logger.Errorw("error in polling git material",
					"material", material,
					"err", err)
			}
		})
	}
	wp.StopWait()
}

func (impl GitWatcherImpl) PollAndUpdateGitMaterial(material *sql.GitMaterial) (*sql.GitMaterial, error) {
	repoLock := impl.locker.LeaseLocker(material.Id)
	repoLock.Mutex.Lock()
	defer func() {
		repoLock.Mutex.Unlock()
		impl.locker.ReturnLocker(material.Id)
	}()

	// poll and notify
	err := impl.PollGitMaterialAndNotify(material)

	// update last fetch status, counters and time
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

func (impl GitWatcherImpl) PollGitMaterialAndNotify(material *sql.GitMaterial) error {

	// get credentials and location
	username, password, err := GetUserNamePassword(material.GitProvider)
	location, err := GetLocationForMaterial(material)
	if err != nil {
		impl.logger.Errorw("error in getting credentials/location",
			"materialId", material.Id,
			"authMode", material.GitProvider.AuthMode,
			"url", material.Url,
			"err", err)

		return err
	}

	// fetch changes
	updated, repo, err := impl.repositoryManager.Fetch(username, password, material.Url, location)
	if err != nil {
		impl.logger.Errorw("error in while fetching changes",
			"repo", material.Url,
			"err", err)

		// there might be the case if ssh private key gets flush from disk, so creating and single retrying in this case
		if material.GitProvider.AuthMode != sql.AUTH_MODE_SSH {
			return err

		} else {
			err = impl.repositoryManager.CreateSshFileIfNotExistsAndConfigureSshCommand(location,
				material.GitProvider.Id,
				material.GitProvider.SshPrivateKey)

			if err != nil {
				impl.logger.Errorw("error in creating/configuring ssh private key on disk ",
					"repo", material.Url,
					"gitProviderId", material.GitProvider.Id,
					"err", err)
				return err

			}

			impl.logger.Info("Retrying fetching", "repo", material.Url)
			updated, repo, err = impl.repositoryManager.Fetch(username, password, material.Url, location)
			if err != nil {
				impl.logger.Errorw("error in fetching material details in retry",
					"repo", material.Url,
					"err", err)
				return err
			}
		}
	}

	if !updated {
		impl.logger.Infow("no new updates found",
			"materialId", material.Id,
			"refMaterialId", material.RefGitMaterialId,
			"url", material.Url)
		return nil
	}

	// get all ci pipeline ciMaterials (unique branch) for all git ciMaterials which reference this material
	ciMaterials, err := impl.ciPipelineMaterialRepository.FindAllCiPipelineMaterialsReferencingGivenMaterial(material.Id)
	if err != nil {
		impl.logger.Errorw("error while fetching ci materials",
			"err", err,
			"gitMaterialId", material.Id)
		return err
	}

	for _, ciMaterial := range ciMaterials {

		// Get recent changes
		commits, err := impl.repositoryManager.ChangesSinceByRepository(repo, ciMaterial.Value, "", "", 15)

		if err != nil {
			erroredCiMaterial := &sql.CiPipelineMaterial{
				Errored:  true,
				ErrorMsg: err.Error(),
			}
			// Update errored ci material
			err = impl.ciPipelineMaterialRepository.UpdateErroredCiPipelineMaterialsReferencingGivenGitMaterial(material.Id,
				ciMaterial.Value, erroredCiMaterial)

			if err != nil {
				impl.logger.Errorw("error while updating errored ci pipeline materials",
					"err", err)
			}

		} else if len(commits) > 0 {
			latestCommit := commits[0]
			if latestCommit.Commit != ciMaterial.LastSeenHash {
				//new commit found
				commitJson, _ := json.Marshal(commits)
				toBeUpdatedCiMaterial := &sql.CiPipelineMaterial{
					LastSeenHash:  latestCommit.Commit,
					CommitAuthor:  latestCommit.Author,
					CommitDate:    latestCommit.Date,
					CommitHistory: string(commitJson),
					Errored:       false,
					ErrorMsg:      "",
				}
				// Update in db
				err := impl.ciPipelineMaterialRepository.UpdateCiPipelineMaterialsReferencingGivenGitMaterial(material.Id,
					ciMaterial.Value, toBeUpdatedCiMaterial)

				if err != nil {
					impl.logger.Errorw("error while updating ci pipeline materials",
						"err", err)
				}

				// Notify only if new changes are detected
				_ = impl.NotifyForMaterialUpdate(material.Url, ciMaterial.Value, latestCommit)
			}
			middleware.GitMaterialUpdateCounter.WithLabelValues().Inc()
		}
	}
	return nil
}

func (impl GitWatcherImpl) NotifyForMaterialUpdate(repoUrl string, branch string, commit *GitCommit) error {

	ciMaterials, err := impl.ciPipelineMaterialRepository.FindSimilarCiPipelineMaterials(repoUrl, branch)
	if err != nil {
		impl.logger.Errorw("error while fetching similar ci pipeline materials",
			"repoUrl", repoUrl,
			"branch", branch,
			"err", err)
		return err
	}

	for _, ciMaterial := range ciMaterials {

		event := &CiPipelineMaterialBean{
			Id:            ciMaterial.Id,
			GitMaterialId: ciMaterial.GitMaterialId,
			Active:        ciMaterial.Active,
			GitCommit:     commit,
			Type:          ciMaterial.Type,
			Value:         ciMaterial.Value,
		}

		payload, err := json.Marshal(event)
		if err != nil {
			impl.logger.Error("err in json marshaling",
				"event", event,
				"err", err)

			return err
		}

		err = impl.pubSubClient.Publish(pubsub.NEW_CI_MATERIAL_TOPIC, string(payload))
		if err != nil {
			impl.logger.Errorw("error in publishing material modification msg ",
				"payload", payload,
				"err", err)

			return err
		}
	}
	return nil
}

// Old implementations

//func (impl GitWatcherImpl) Watch() {
//	impl.logger.Infow("starting git watch thread")
//	materials, err := impl.materialRepo.FindActive()
//	if err != nil {
//		impl.logger.Error("error in fetching watchlist", "err", err)
//		return
//	}
//	//impl.Publish(materials)
//	middleware.ActiveGitRepoCount.WithLabelValues().Set(float64(len(materials)))
//	impl.RunOnWorker(materials)
//	impl.logger.Infow("stop git watch thread")
//}

//func (impl *GitWatcherImpl) RunOnWorker(materials []*sql.GitMaterial) {
//	wp := workerpool.New(impl.pollConfig.PollWorker)
//	for _, material := range materials {
//		if len(material.CiPipelineMaterials) == 0 {
//			impl.logger.Infow("no ci pipeline, skipping", "id", material.Id, "url", material.Url)
//			continue
//		}
//		materialMsg := &sql.GitMaterial{Id: material.Id, Url: material.Url}
//		wp.Submit(func() {
//			_, err := impl.pollAndUpdateGitMaterial(materialMsg)
//			if err != nil {
//				impl.logger.Errorw("error in polling git material", "material", materialMsg, "err", err)
//			}
//		})
//	}
//	wp.StopWait()
//}

//func (impl GitWatcherImpl) PollAndUpdateGitMaterial(material *sql.GitMaterial) (*sql.GitMaterial, error) {
//	//tmp expose remove in future
//	return impl.pollAndUpdateGitMaterial(material)
//}

//func (impl GitWatcherImpl) pollAndUpdateGitMaterial(materialReq *sql.GitMaterial) (*sql.GitMaterial, error) {
//	repoLock := impl.locker.LeaseLocker(materialReq.Id)
//	repoLock.Mutex.Lock()
//	defer func() {
//		repoLock.Mutex.Unlock()
//		impl.locker.ReturnLocker(materialReq.Id)
//	}()
//	material, err := impl.materialRepo.FindById(materialReq.Id)
//	if err != nil {
//		impl.logger.Errorw("error in fetching material ", "material", materialReq, "err", err)
//		return nil, err
//	}
//	err = impl.pollGitMaterialAndNotify(material)
//	material.LastFetchTime = time.Now()
//	material.FetchStatus = err == nil
//	if err != nil {
//		material.LastFetchErrorCount = material.LastFetchErrorCount + 1
//		material.FetchErrorMessage = err.Error()
//	} else {
//		material.LastFetchErrorCount = 0
//		material.FetchErrorMessage = ""
//	}
//	err = impl.materialRepo.Update(material)
//	if err != nil {
//		impl.logger.Errorw("error in updating fetch status", "material", material, "err", err)
//	}
//	return material, err
//}
//
//func (impl GitWatcherImpl) pollGitMaterialAndNotify(material *sql.GitMaterial) error {
//	gitProvider := material.GitProvider
//	userName, password, err := GetUserNamePassword(gitProvider)
//	location, err := GetLocationForMaterial(material)
//	if err != nil {
//		impl.logger.Errorw("error in determining location", "url", material.Url, "err", err)
//		return err
//	}
//	updated, repo, err := impl.repositoryManager.Fetch(userName, password, material.Url, location)
//	if err != nil {
//		impl.logger.Errorw("error in fetching material details ", "repo", material.Url, "err", err)
//		// there might be the case if ssh private key gets flush from disk, so creating and single retrying in this case
//		if gitProvider.AuthMode == sql.AUTH_MODE_SSH {
//			err = impl.repositoryManager.CreateSshFileIfNotExistsAndConfigureSshCommand(location, gitProvider.Id, gitProvider.SshPrivateKey)
//			if err != nil {
//				impl.logger.Errorw("error in creating/configuring ssh private key on disk ", "repo", material.Url, "gitProviderId", gitProvider.Id, "err", err)
//				return err
//			} else {
//				impl.logger.Info("Retrying fetching for", "repo", material.Url)
//				updated, repo, err = impl.repositoryManager.Fetch(userName, password, material.Url, location)
//				if err != nil {
//					impl.logger.Errorw("error in fetching material details in retry", "repo", material.Url, "err", err)
//					return err
//				}
//			}
//		} else {
//			return err
//		}
//	}
//	if !updated {
//		return nil
//	}
//	materials, err := impl.ciPipelineMaterialRepository.FindByGitMaterialId(material.Id)
//	if err != nil {
//		impl.logger.Errorw("error in calculating head", "err", err, "url", material.Url)
//		return err
//	}
//	var updatedMaterials []*CiPipelineMaterialBean
//	var updatedMaterialsModel []*sql.CiPipelineMaterial
//	var erroredMaterialsModels []*sql.CiPipelineMaterial
//	for _, material := range materials {
//		if material.Type != sql.SOURCE_TYPE_BRANCH_FIXED {
//			continue
//		}
//		commits, err := impl.repositoryManager.ChangesSinceByRepository(repo, material.Value, "", "", 15)
//		if err != nil {
//			material.Errored = true
//			material.ErrorMsg = err.Error()
//			erroredMaterialsModels = append(erroredMaterialsModels, material)
//		} else if len(commits) > 0 {
//			latestCommit := commits[0]
//			if latestCommit.Commit != material.LastSeenHash {
//				//new commit found
//				mb := &CiPipelineMaterialBean{
//					Id:            material.Id,
//					Value:         material.Value,
//					GitMaterialId: material.GitMaterialId,
//					Type:          material.Type,
//					Active:        material.Active,
//					GitCommit:     latestCommit,
//				}
//				updatedMaterials = append(updatedMaterials, mb)
//
//				material.LastSeenHash = latestCommit.Commit
//				material.CommitAuthor = latestCommit.Author
//				material.CommitDate = latestCommit.Date
//				commitJson, _ := json.Marshal(commits)
//				material.CommitHistory = string(commitJson)
//				material.Errored = false
//				material.ErrorMsg = ""
//				updatedMaterialsModel = append(updatedMaterialsModel, material)
//			}
//			middleware.GitMaterialUpdateCounter.WithLabelValues().Inc()
//		}
//	}
//	if len(updatedMaterialsModel) > 0 {
//		err = impl.NotifyForMaterialUpdate(updatedMaterials)
//		if err != nil {
//			impl.logger.Errorw("error in sending notification for materials", "url", material.Url, "update", updatedMaterialsModel)
//		}
//		err = impl.ciPipelineMaterialRepository.Update(updatedMaterialsModel)
//		if err != nil {
//			impl.logger.Errorw("error in update db ", "url", material.Url, "update", updatedMaterialsModel)
//			impl.logger.Errorw("error in sending notification for materials", "url", material.Url, "update", updatedMaterialsModel)
//		}
//	}
//	if len(erroredMaterialsModels) > 0 {
//		err = impl.ciPipelineMaterialRepository.Update(updatedMaterialsModel)
//		if err != nil {
//			impl.logger.Errorw("error in update db ", "url", material.Url, "update", updatedMaterialsModel)
//			impl.logger.Errorw("error in sending notification for materials", "url", material.Url, "update", updatedMaterialsModel)
//		}
//	}
//	return nil
//}

//func (impl GitWatcherImpl) NotifyForMaterialUpdate(materials []*CiPipelineMaterialBean) error {
//
//	impl.logger.Warnw("material notification", "materials", materials)
//	for _, material := range materials {
//		mb, err := json.Marshal(material)
//		if err != nil {
//			impl.logger.Error("err in json marshaling", "err", err)
//			continue
//		}
//		err = impl.pubSubClient.Publish(pubsub.NEW_CI_MATERIAL_TOPIC, string(mb))
//		if err != nil {
//			impl.logger.Errorw("error in publishing material modification msg ", "material", material)
//		}
//
//	}
//	return nil
//}

func (impl GitWatcherImpl) SubscribeWebhookEvent() error {
	callback := func(msg *pubsub.PubSubMsg) {
		impl.logger.Debugw("received msg", "msg", msg)
		//msg.Ack() //ack immediate if lost next min it would get a new message
		webhookEvent := &WebhookEvent{}
		err := json.Unmarshal([]byte(msg.Data), webhookEvent)
		if err != nil {
			impl.logger.Infow("err in reading msg", "err", err)
			return
		}
		impl.webhookHandler.HandleWebhookEvent(webhookEvent)
	}
	err := impl.pubSubClient.Subscribe(pubsub.WEBHOOK_EVENT_TOPIC, callback)
	return err
}

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
