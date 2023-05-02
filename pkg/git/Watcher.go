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
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"strings"
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
	PathMatcher(fileStats *object.FileStats, gitMaterial *sql.GitMaterial) bool
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
			err = impl.repositoryManager.CreateSshFileIfNotExistsAndConfigureSshCommand(location, gitProvider.Id, gitProvider.SshPrivateKey)
			if err != nil {
				impl.logger.Errorw("error in creating/configuring ssh private key on disk ", "repo", material.Url, "gitProviderId", gitProvider.Id, "err", err)
				return err
			} else {
				impl.logger.Info("Retrying fetching for", "repo", material.Url)
				updated, repo, err = impl.repositoryManager.Fetch(userName, password, material.Url, location)
				if err != nil {
					impl.logger.Errorw("error in fetching material details in retry", "repo", material.Url, "err", err)
					return err
				}
			}
		} else {
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
		err = impl.NotifyForMaterialUpdate(updatedMaterials, material)
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

func (impl GitWatcherImpl) NotifyForMaterialUpdate(materials []*CiPipelineMaterialBean, gitMaterial *sql.GitMaterial) error {

	impl.logger.Warnw("material notification", "materials", materials)
	for _, material := range materials {
		excluded := impl.PathMatcher(material.GitCommit.FileStats, gitMaterial)
		if excluded {
			impl.logger.Infow("skip this auto trigger", "exclude", excluded)
			continue
		}
		mb, err := json.Marshal(material)
		if err != nil {
			impl.logger.Errorw("err in json marshaling", "err", err)
			continue
		}
		err = impl.pubSubClient.Publish(pubsub.NEW_CI_MATERIAL_TOPIC, string(mb))
		if err != nil {
			impl.logger.Errorw("error in publishing material modification msg ", "material", material)
		}

	}
	return nil
}

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

/**
- if include is present in filters, that means paths outside of the include by default marked by excluded
- if exclude is present in filters, that means paths outside of the exclude by default marked as included.
- if include and exclude both present, and path is outside of scope of both, marked as excluded
- if include and exclude both present, and path is inside include and outside of exclude, marked as included.
- if include and exclude both present, and path is inside exclude and outside of include, marked as excluded.
- if include and exclude both present, and path is inside both, marked it as included
*/

func (impl GitWatcherImpl) PathMatcher(fileStats *object.FileStats, gitMaterial *sql.GitMaterial) bool {
	excluded := false
	isExcluded := 0
	isIncluded := 0
	var paths []string
	var includedPaths []string
	var excludedPaths []string
	for _, path := range gitMaterial.FilterPattern {
		if strings.Contains(path, "!") {
			excludedPaths = append(excludedPaths, strings.ReplaceAll(path, "!", ""))
		} else {
			includedPaths = append(includedPaths, path)
		}
	}
	impl.logger.Infow("pathMatcher............", "includedPaths", includedPaths, "excludedPaths", excludedPaths)
	fileStatBytes, err := json.Marshal(fileStats)
	if err != nil {
		impl.logger.Infow("testing marshal error ............", "err", err)
		return false
	}
	var fileChanges []map[string]interface{}
	if err := json.Unmarshal(fileStatBytes, &fileChanges); err != nil {
		impl.logger.Infow("testing unmarshal error ............", "err", err)
		return false
	}
	for _, fileChange := range fileChanges {
		path := fileChange["Name"].(string)
		paths = append(paths, path)
	}
	impl.logger.Infow("pathMatcher. ............", "changes in paths", paths)
	//TODO read file stat
	outsideOfIncludedScope := 0
	for _, path := range paths {
		included := false
		for _, includedPath := range includedPaths {
			if strings.Contains(path, includedPath) {
				included = true
				break
			}
		}
		if included {
			isIncluded = 1
		} else {
			outsideOfIncludedScope = outsideOfIncludedScope + 1
		}
	}

	outsideOfExcludedScope := 0
	//if changes detected in included path, check if falls under excluded paths
	for _, path := range paths {
		excluded := false
		for _, excludedPath := range excludedPaths {
			if strings.Contains(path, excludedPath) {
				excluded = true
				break
			}
		}
		if excluded {
			// if found changes under excluded, return hideMaterials
			isExcluded = 1
		} else {
			outsideOfExcludedScope = outsideOfExcludedScope + 1
		}
	}

	/**
	- if include is present in filters, that means paths outside of the include by default marked as excluded
	- if exclude is present in filters, that means paths outside of the exclude by default marked as included.
	- if include and exclude both present, and path is outside of scope of both, marked as excluded
	- if include and exclude both present, and path is inside include and outside of exclude, marked as included.
	- if include and exclude both present, and path is inside exclude and outside of include, marked as excluded.
	- if include and exclude both present, and path is inside both, marked it as included
	*/
	if len(includedPaths) > 0 && len(excludedPaths) == 0 {
		if isIncluded == 1 {
			//SHOW
			excluded = false
		} else {
			//HIDE
			excluded = true
		}
	} else if len(includedPaths) == 0 && len(excludedPaths) > 0 {
		if isExcluded == 1 && outsideOfExcludedScope == 0 {
			//HIDE
			excluded = true
		} else if outsideOfExcludedScope > 0 {
			//SHOW
			excluded = false
		}
	} else if len(includedPaths) > 0 && len(excludedPaths) > 0 {
		if isIncluded == 0 && isExcluded == 0 && (outsideOfIncludedScope > 1 || outsideOfExcludedScope > 1) {
			//TODO - CHECK ORDER AND LAST ONE WILL OVERRIDE
			excluded = false
		} else if isIncluded == 1 && isExcluded == 0 {
			//SHOW
			excluded = false
		} else if isIncluded == 0 && isExcluded == 1 {
			//HIDE
			excluded = true
		} else if isIncluded == 1 && isExcluded == 1 {
			//TODO - CHECK ORDER AND LAST ONE WILL OVERRIDE
			excluded = false
		}
	}

	return excluded
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
