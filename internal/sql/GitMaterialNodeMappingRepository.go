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

package sql

import (
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type GitMaterialNodeMapping struct {
	tableName     struct{}  `sql:"git_material_node_mapping" pg:",discard_unknown_columns"`
	Id            int       `sql:"id,pk"`
	GitMaterialId int       `sql:"git_material_id"`
	OrdinalIndex  int       `sql:"ordinal_index"`
	CreatedBy     int       `sql:"created_by"`
	CreatedOn     time.Time `sql:"created_on"`
	UpdatedBy     int       `sql:"updated_by"`
	UpdatedOn     time.Time `sql:"updated_on"`
}

type GitMaterialNodeMappingRepository interface {
	GetConnection() *pg.DB
	InsertWithTransaction(mapping *GitMaterialNodeMapping, tx *pg.Tx) error
}

type GitMaterialNodeMappingRepositoryImpl struct {
	logger       *zap.SugaredLogger
	dbConnection *pg.DB
}

func NewGitMaterialNodeMappingRepositoryImpl(logger *zap.SugaredLogger,
	dbConnection *pg.DB) *GitMaterialNodeMappingRepositoryImpl {
	return &GitMaterialNodeMappingRepositoryImpl{
		logger:       logger,
		dbConnection: dbConnection,
	}
}

func (impl *GitMaterialNodeMappingRepositoryImpl) GetConnection() *pg.DB {
	return impl.dbConnection
}

func (impl *GitMaterialNodeMappingRepositoryImpl) InsertWithTransaction(mapping *GitMaterialNodeMapping, tx *pg.Tx) error {

	err := tx.Insert(mapping)
	return err
}
