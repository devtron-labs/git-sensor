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

package internals

import (
	"go.uber.org/zap"
	"sync"
)

type RepositoryLocker struct {
	logger *zap.SugaredLogger
	Mutex  sync.Mutex
	Bank   map[int]*RepositoryLock
}

func NewRepositoryLocker(logger *zap.SugaredLogger) *RepositoryLocker {
	return &RepositoryLocker{
		logger: logger,
		Bank:   map[int]*RepositoryLock{},
	}
}

func (locker *RepositoryLocker) LeaseLocker(RepositoryId int) *RepositoryLock {
	locker.logger.Debugw("lease req get for ", "repo", RepositoryId)
	locker.Mutex.Lock()
	defer locker.Mutex.Unlock()
	repositoryLock := locker.Bank[RepositoryId]
	if repositoryLock == nil {
		repositoryLock = &RepositoryLock{} //check for initialization
		locker.Bank[RepositoryId] = repositoryLock
	}
	repositoryLock.counter = repositoryLock.counter + 1
	return repositoryLock
}

func (locker *RepositoryLocker) ReturnLocker(appId int) {
	locker.logger.Debugw("lease req release for ", "repo", appId)
	locker.Mutex.Lock()
	defer locker.Mutex.Unlock()
	repositoryLock := locker.Bank[appId]
	repositoryLock.counter = repositoryLock.counter - 1
	if repositoryLock.counter == 0 {
		delete(locker.Bank, appId)
	}
}

type RepositoryLock struct {
	Mutex   sync.Mutex
	counter int
}

/*func SyncHandler(locker *RepositoryLocker, appId int) {
	lock := locker.LeaseLocker(appId)
	defer locker.ReturnLocker(appId)
	lock.Mutex.Lock()
	defer lock.Mutex.Unlock()
	fmt.Println("running : " + strconv.Itoa(appId))

}*/
