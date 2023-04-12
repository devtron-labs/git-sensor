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

type CiPipelineMaterial struct {
	tableName     struct{}   `sql:"ci_pipeline_material" pg:",discard_unknown_columns"`
	Id            int        `sql:"id"`
	GitMaterialId int        `sql:"git_material_id"` //id stored in db GitMaterial( foreign key)
	Type          SourceType `sql:"type"`
	Value         string     `sql:"value"`
	Active        bool       `sql:"active,notnull"`
	LastSeenHash  string     `sql:"last_seen_hash,notnull"`
	CommitAuthor  string     `sql:"commit_author"`
	CommitDate    time.Time  `sql:"commit_date"`

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
	FindSimilarCiPipelineMaterials(repoUrl string, branch string) ([]*CiPipelineMaterial, error)
	FindAllCiPipelineMaterialsReferencingGivenMaterial(gitMaterialId int) ([]*CiPipelineMaterial, error)
	UpdateErroredCiPipelineMaterialsReferencingGivenGitMaterial(gitMaterialId int, branch string, material *CiPipelineMaterial) error
	UpdateCiPipelineMaterialsReferencingGivenGitMaterial(gitMaterialId int, branch string, material *CiPipelineMaterial) error
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

func (impl CiPipelineMaterialRepositoryImpl) FindSimilarCiPipelineMaterials(repoUrl string, branch string) ([]*CiPipelineMaterial, error) {
	materials := make([]*CiPipelineMaterial, 0)
	err := impl.dbConnection.Model(&materials).
		Join("INNER JOIN git_material gm ON gm.id = ci_pipeline_material.git_material_id").
		Where("gm.url = ?", repoUrl).
		Where("gm.deleted = ?", false).
		Where("ci_pipeline_material.value = ?", branch).
		Where("ci_pipeline_material.active = ?", true).
		Select()

	return materials, err
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

func (impl CiPipelineMaterialRepositoryImpl) FindAllCiPipelineMaterialsReferencingGivenMaterial(gitMaterialId int) ([]*CiPipelineMaterial, error) {
	ciPipelineMaterials := make([]*CiPipelineMaterial, 0)
	//err := impl.dbConnection.Model(&ciPipelineMaterials).
	//	ColumnExpr("DISTINCT ci_pipeline_material.value").
	//	Column("ci_pipeline_material.*").
	//	Join("INNER JOIN git_material gm ON gm.id = ci_pipeline_material.git_material_id").
	//	Where("gm.ref_git_material_id = ?", gitMaterialId).
	//	Where("gm.deleted = ?", false).
	//	Where("ci_pipeline_material.active = ?", true).
	//	Where("ci_pipeline_material.type = ?", "SOURCE_TYPE_BRANCH_FIXED").
	//	Select()

	query := "select * from (" +
		"select *, row_number() over (partition by value order by commit_date desc) as row_num from ci_pipeline_material cpm " +
		"inner join git_material gm on gm.id = cpm.git_material_id " +
		"where (gm.ref_git_material_id  = ?) and (gm.deleted = false) and (cpm.active = true) and (cpm.\"type\" = 'SOURCE_TYPE_BRANCH_FIXED')" +
		") materials where row_num <= 1"

	_, err := impl.dbConnection.Query(&ciPipelineMaterials, query, gitMaterialId)
	return ciPipelineMaterials, err
}

func (impl CiPipelineMaterialRepositoryImpl) UpdateErroredCiPipelineMaterialsReferencingGivenGitMaterial(gitMaterialId int, branch string, material *CiPipelineMaterial) error {

	//_, err := impl.dbConnection.Model(material).
	//	Column("ci_pipeline_material.errored", "ci_pipeline_material.error_msg").
	//	Join("INNER JOIN git_material gm ON gm.id = ci_pipeline_material.git_material_id").
	//	Where("gm.ref_git_material_id = ?", gitMaterialId).
	//	Where("ci_pipeline_material.value = ?", branch).
	//	Update()

	query := "UPDATE ci_pipeline_material cpm " +
		"SET errored = ?, " +
		"error_msg = ? " +
		"FROM git_material gm WHERE (cpm.git_material_id = gm.id) " +
		"AND (gm.ref_git_material_id = ?) " +
		"AND (cpm.value = ?)"

	_, err := impl.dbConnection.Query(material, query, material.Errored, material.ErrorMsg, gitMaterialId, branch)
	return err
}

func (impl CiPipelineMaterialRepositoryImpl) UpdateCiPipelineMaterialsReferencingGivenGitMaterial(gitMaterialId int, branch string, material *CiPipelineMaterial) error {

	//_, err := impl.dbConnection.Model(material).
	//	Column("ci_pipeline_material.last_seen_hash", "ci_pipeline_material.commit_author", "ci_pipeline_material.commit_date",
	//		"ci_pipeline_material.commit_history", "ci_pipeline_material.errored", "ci_pipeline_material.error_msg").
	//	Join("INNER JOIN git_material gm ON gm.id = ci_pipeline_material.git_material_id").
	//	Where("gm.ref_git_material_id = ?", gitMaterialId).
	//	Where("ci_pipeline_material.value = ?", branch).
	//	Update()

	query := "UPDATE ci_pipeline_material cpm " +
		"SET errored = ?, " +
		"error_msg = ?, " +
		"last_seen_hash = ?, " +
		"commit_author = ?, " +
		"commit_date = ?, " +
		"commit_history = ? " +
		"FROM git_material gm WHERE (cpm.git_material_id = gm.id) " +
		"AND (gm.ref_git_material_id = ?) " +
		"AND (cpm.value = ?)"

	_, err := impl.dbConnection.Query(material, query, material.Errored, material.ErrorMsg, material.LastSeenHash,
		material.CommitAuthor, material.CommitDate, material.CommitHistory, gitMaterialId, branch)

	return err
}
