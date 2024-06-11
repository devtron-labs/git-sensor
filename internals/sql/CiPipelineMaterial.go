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

package sql

import (
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type CiPipelineMaterial struct {
	tableName     struct{}   `sql:"ci_pipeline_material"`
	Id            int        `sql:"id"`
	GitMaterialId int        `sql:"git_material_id"` //id stored in db GitMaterial( foreign key)
	Type          SourceType `sql:"type"`
	Value         string     `sql:"value"`
	Active        bool       `sql:"active,notnull"`
	LastSeenHash  string     `sql:"last_seen_hash,notnull"`
	CommitAuthor  string     `sql:"commit_author"`
	CommitDate    time.Time  `sql:"commit_date"`
	CommitMessage string     `sql:"commit_message"`

	CommitHistory string `sql:"commit_history"` //last five commit for caching purpose1
	Errored       bool   `sql:"errored,notnull"`
	ErrorMsg      string `sql:"error_msg,notnull"`
}

type CiPipelineMaterialRepository interface {
	FindByGitMaterialId(gitMaterialId int) ([]*CiPipelineMaterial, error)
	Update(material []*CiPipelineMaterial) error
	FindByIds(ids []int) ([]*CiPipelineMaterial, error)
	FindById(id int) (*CiPipelineMaterial, error)
	Exists(id int) (bool, error)
	Save(material []*CiPipelineMaterial) ([]*CiPipelineMaterial, error)
}

type CiPipelineMaterialRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
}

func NewCiPipelineMaterialRepositoryImpl(dbConnection *pg.DB,
	logger *zap.SugaredLogger) *CiPipelineMaterialRepositoryImpl {
	return &CiPipelineMaterialRepositoryImpl{dbConnection: dbConnection, logger: logger}

}
func (impl CiPipelineMaterialRepositoryImpl) FindByGitMaterialId(gitMaterialId int) ([]*CiPipelineMaterial, error) {
	var materials []*CiPipelineMaterial
	err := impl.dbConnection.Model(&materials).
		Where("git_material_id =?", gitMaterialId).
		Where("active = ?", true).Select()
	return materials, err
}

func (impl CiPipelineMaterialRepositoryImpl) Exists(id int) (bool, error) {
	var materials *CiPipelineMaterial
	exists, err := impl.dbConnection.Model(materials).Where("id =? ", id).Exists()
	return exists, err
}

func (impl CiPipelineMaterialRepositoryImpl) Save(material []*CiPipelineMaterial) ([]*CiPipelineMaterial, error) {
	_, err := impl.dbConnection.Model(&material).Insert()
	return material, err
}

func (impl CiPipelineMaterialRepositoryImpl) Update(materials []*CiPipelineMaterial) error {
	err := impl.dbConnection.RunInTransaction(func(tx *pg.Tx) error {
		for _, material := range materials {
			_, err := tx.Model(material).WherePK().UpdateNotNull()
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (impl CiPipelineMaterialRepositoryImpl) FindByIds(ids []int) (materials []*CiPipelineMaterial, err error) {
	err = impl.dbConnection.Model(&materials).
		Where("id in (?)", pg.In(ids)).
		Where("active = ?", true).Select()
	return materials, err
}

func (impl CiPipelineMaterialRepositoryImpl) FindById(id int) (*CiPipelineMaterial, error) {
	materials := &CiPipelineMaterial{}
	err := impl.dbConnection.Model(materials).
		Where("id =?", id).
		Where("active = ?", true).Select()
	return materials, err
}
