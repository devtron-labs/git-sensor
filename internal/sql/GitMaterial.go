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
	RefGitMaterialId    int       `sql:"ref_git_material_id"`
	GitProvider         *GitProvider
	CiPipelineMaterials []*CiPipelineMaterial
}

type MaterialRepository interface {
	GetConnection() *pg.DB
	FindById(id int) (*GitMaterial, error)
	Update(material *GitMaterial) error
	Save(material *GitMaterial) error
	SaveWithTransaction(material *GitMaterial, tx *pg.Tx) error
	FindActive() ([]*GitMaterial, error)
	FindAll() ([]*GitMaterial, error)
	FindAllActiveByUrls(urls []string) ([]*GitMaterial, error)
	FindAllReferencedGitMaterials() ([]*GitMaterial, error)
	FindReferencedGitMaterial(materialId int) (*GitMaterial, error)
	GetMaterialWithSameRepoUrl(repoUrl string, limit int) (*GitMaterial, error)
	UpdateRefMaterialId(material *GitMaterial, tx *pg.Tx) error
	FindByReferenceId(refGitMaterialId int, limit int, excludingIds []int) (*GitMaterial, error)
	UpdateWithTransaction(material *GitMaterial, tx *pg.Tx) error
	UpdateRefMaterialIdForAllWithSameUrl(url string, refGitMaterialId int, tx *pg.Tx) error
}
type MaterialRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewMaterialRepositoryImpl(dbConnection *pg.DB) *MaterialRepositoryImpl {
	return &MaterialRepositoryImpl{dbConnection: dbConnection}
}

func (repo MaterialRepositoryImpl) GetConnection() *pg.DB {
	return repo.dbConnection
}

func (repo MaterialRepositoryImpl) Save(material *GitMaterial) error {
	err := repo.dbConnection.Insert(material)
	return err
}

func (repo MaterialRepositoryImpl) SaveWithTransaction(material *GitMaterial, tx *pg.Tx) error {
	err := tx.Insert(material)
	return err
}

func (repo MaterialRepositoryImpl) Update(material *GitMaterial) error {
	_, err := repo.dbConnection.Model(material).WherePK().Update()
	return err
}

func (repo MaterialRepositoryImpl) UpdateWithTransaction(material *GitMaterial, tx *pg.Tx) error {
	_, err := tx.Model(material).
		WherePK().
		Update()

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

func (repo MaterialRepositoryImpl) FindAllReferencedGitMaterials() ([]*GitMaterial, error) {

	var materials []*GitMaterial

	err := repo.dbConnection.Model(&materials).
		ColumnExpr("DISTINCT(git_material.id)").
		Column("git_material.*", "GitProvider").
		Join("RIGHT JOIN git_material gm ON git_material.id = gm.ref_git_material_id").
		Where("gm.deleted = ? ", false).
		Where("gm.checkout_status = ? ", true).
		Order("git_material.id ASC").
		Select()

	return materials, err
}

func (repo MaterialRepositoryImpl) FindReferencedGitMaterial(materialId int) (*GitMaterial, error) {

	material := &GitMaterial{}
	err := repo.dbConnection.Model(material).
		Column("git_material.*", "GitProvider").
		Join("RIGHT JOIN git_material gm ON git_material.id = gm.ref_git_material_id").
		Where("gm.id = ? ", materialId).
		Select()

	return material, err
}

func (repo MaterialRepositoryImpl) GetMaterialWithSameRepoUrl(repoUrl string, limit int) (*GitMaterial, error) {

	material := &GitMaterial{}
	err := repo.dbConnection.Model(material).
		Where("url = ?", repoUrl).
		Where("deleted = false").
		Limit(limit).
		Select()

	return material, err
}

func (repo MaterialRepositoryImpl) UpdateRefMaterialId(material *GitMaterial, tx *pg.Tx) error {

	_, err := tx.Model(material).
		Column("ref_git_material_id").
		WherePK().
		Update()

	return err
}

func (repo MaterialRepositoryImpl) UpdateRefMaterialIdForAllWithSameUrl(url string, refGitMaterialId int, tx *pg.Tx) error {

	material := &GitMaterial{}
	_, err := tx.Model(material).
		Set("ref_git_material_id", refGitMaterialId).
		Where("url = ?", url).
		Where("deleted = ?", false).
		Update()

	return err
}

func (repo MaterialRepositoryImpl) FindByReferenceId(refGitMaterialId int, limit int, excludingIds []int) (*GitMaterial, error) {
	var material GitMaterial
	qry := repo.dbConnection.Model(&material).
		Column("git_material.*", "GitProvider").
		Where("git_material.ref_git_material_id = ? ", refGitMaterialId).
		Where("git_material.deleted =? ", false).
		Limit(limit)

	if len(excludingIds) > 0 {
		qry = qry.Where("git_material.id not in (?)", pg.In(excludingIds))
	}

	err := qry.Select()
	return &material, err
}
