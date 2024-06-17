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

import "github.com/go-pg/pg"

type AuthMode string

const (
	AUTH_MODE_USERNAME_PASSWORD AuthMode = "USERNAME_PASSWORD"
	AUTH_MODE_SSH               AuthMode = "SSH"
	AUTH_MODE_ACCESS_TOKEN      AuthMode = "ACCESS_TOKEN"
	AUTH_MODE_ANONYMOUS         AuthMode = "ANONYMOUS"
)

type GitProvider struct {
	tableName             struct{} `sql:"git_provider"`
	Id                    int      `sql:"id,pk"`
	Name                  string   `sql:"name,notnull"`
	Url                   string   `sql:"url,notnull"`
	UserName              string   `sql:"user_name"`
	Password              string   `sql:"password"`
	SshPrivateKey         string   `sql:"ssh_private_key"`
	AccessToken           string   `sql:"access_token"`
	AuthMode              AuthMode `sql:"auth_mode,notnull"`
	Active                bool     `sql:"active,notnull"`
	TlsCert               string   `sql:"tls_cert"`
	TlsKey                string   `sql:"tls_key"`
	CaCert                string   `sql:"ca_cert"`
	EnableTLSVerification bool     `sql:"enable_tls_verification"`
	//models.AuditLog
}

type GitProviderRepository interface {
	GetById(id int) (*GitProvider, error)
	Save(provider *GitProvider) error
	Update(provider *GitProvider) error
	Exists(id int) (bool, error)
}

type GitProviderRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewGitProviderRepositoryImpl(dbConnection *pg.DB) *GitProviderRepositoryImpl {
	return &GitProviderRepositoryImpl{dbConnection: dbConnection}
}

func (impl GitProviderRepositoryImpl) Save(provider *GitProvider) error {
	_, err := impl.dbConnection.Model(provider).Insert()
	return err
}

func (impl GitProviderRepositoryImpl) Update(provider *GitProvider) error {
	_, err := impl.dbConnection.Model(provider).WherePK().Update()
	return err
}
func (impl GitProviderRepositoryImpl) GetById(id int) (*GitProvider, error) {
	var provider GitProvider
	err := impl.dbConnection.Model(&provider).Where("id =? ", id).
		Where("active = ?", true).Select()
	return &provider, err
}

func (impl GitProviderRepositoryImpl) Exists(id int) (bool, error) {
	var provider GitProvider
	exists, err := impl.dbConnection.Model(&provider).Where("id =? ", id).Exists()
	return exists, err
}
