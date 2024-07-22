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
	"github.com/go-pg/pg/orm"
	"time"
)

type SourceType string

const (
	SOURCE_TYPE_BRANCH_FIXED SourceType = "SOURCE_TYPE_BRANCH_FIXED"
	SOURCE_TYPE_BRANCH_REGEX SourceType = "SOURCE_TYPE_BRANCH_REGEX"
	SOURCE_TYPE_TAG_ANY      SourceType = "SOURCE_TYPE_TAG_ANY"
	SOURCE_TYPE_WEBHOOK      SourceType = "WEBHOOK"
)

// TODO: add support for submodule
type GitMaterial struct {
	tableName        struct{} `sql:"git_material"`
	Id               int      `sql:"id,pk"`
	GitProviderId    int      `sql:"git_provider_id,notnull"`
	Url              string   `sql:"url,omitempty"`
	FetchSubmodules  bool     `sql:"fetch_submodules,notnull"`
	Name             string   `sql:"name, omitempty"`
	CheckoutLocation string   `sql:"checkout_location"`
	CheckoutStatus   bool     `sql:"checkout_status,notnull"`
	CheckoutMsgAny   string   `sql:"checkout_msg_any"`
	Deleted          bool     `sql:"deleted,notnull"`
	//------
	LastFetchTime       time.Time `json:"last_fetch_time"`
	FetchStatus         bool      `json:"fetch_status"`
	LastFetchErrorCount int       `json:"last_fetch_error_count"` //continues fetch error
	FetchErrorMessage   string    `json:"fetch_error_message"`
	CloningMode         string    `json:"cloning_mode" sql:"-"`
	FilterPattern       []string  `sql:"filter_pattern"`
	GitProvider         *GitProvider
	CiPipelineMaterials []*CiPipelineMaterial
}

type MaterialRepository interface {
	FindById(id int) (*GitMaterial, error)
	Update(material *GitMaterial) error
	Save(material *GitMaterial) error
	FindActive() ([]*GitMaterial, error)
	FindAll() ([]*GitMaterial, error)
	FindInRage(startFrom int, endAt int) ([]*GitMaterial, error)
	FindAllActiveByUrls(urls []string) ([]*GitMaterial, error)
}
type MaterialRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewMaterialRepositoryImpl(dbConnection *pg.DB) *MaterialRepositoryImpl {
	return &MaterialRepositoryImpl{dbConnection: dbConnection}
}

func (repo MaterialRepositoryImpl) Save(material *GitMaterial) error {
	err := repo.dbConnection.Insert(material)
	return err
}

func (repo MaterialRepositoryImpl) Update(material *GitMaterial) error {
	_, err := repo.dbConnection.Model(material).WherePK().Update()
	return err
}

func (repo MaterialRepositoryImpl) FindActive() ([]*GitMaterial, error) {
	var materials []*GitMaterial
	err := repo.dbConnection.Model(&materials).
		Column("git_material.*", "GitProvider").
		Relation("CiPipelineMaterials", func(q *orm.Query) (*orm.Query, error) {
			return q.Where("active IS TRUE"), nil
		}).
		Where("deleted =? ", false).
		Where("checkout_status=? ", true).
		Order("id ASC").
		Select()
	return materials, err
}

func (repo MaterialRepositoryImpl) FindAll() ([]*GitMaterial, error) {
	var materials []*GitMaterial
	err := repo.dbConnection.Model(&materials).
		Column("git_material.*", "GitProvider").
		Where("deleted =? ", false).
		Select()
	return materials, err
}

func (repo MaterialRepositoryImpl) FindInRage(startFrom int, endAt int) ([]*GitMaterial, error) {
	var materials []*GitMaterial
	query := repo.dbConnection.Model(&materials).
		Column("git_material.*", "GitProvider").
		Where("deleted =? ", false)
	if startFrom != 0 {
		query.Where("git_material.id >= ?", startFrom)
	}
	if endAt != 0 {
		query.Where("git_material.id <= ?", endAt)
	}
	err := query.Select()
	return materials, err
}

func (repo MaterialRepositoryImpl) FindById(id int) (*GitMaterial, error) {
	var material GitMaterial
	err := repo.dbConnection.Model(&material).
		Column("git_material.*", "GitProvider").
		Where("git_material.id =? ", id).
		Where("git_material.deleted =? ", false).
		Select()
	return &material, err
}

func (repo MaterialRepositoryImpl) FindAllActiveByUrls(urls []string) ([]*GitMaterial, error) {
	var materials []*GitMaterial
	err := repo.dbConnection.Model(&materials).
		Relation("CiPipelineMaterials", func(q *orm.Query) (*orm.Query, error) {
			return q.Where("active IS TRUE"), nil
		}).
		Where("deleted =? ", false).
		Where("url in (?) ", pg.In(urls)).
		Select()
	return materials, err
}
