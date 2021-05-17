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
	"fmt"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	GIT_BASE_DIR      = "/git-base/"
	CLONE_TIMEOUT_SEC = 600
	FETCH_TIMEOUT_SEC = 30
)

//git@gitlab.com:devtron-client-gitops/wms-user-management.git
//https://gitlab.com/devtron-client-gitops/wms-user-management.git

//git@bitbucket.org:DelhiveryTech/kafka-consumer-config.git
//https://prashant-delhivery@bitbucket.org/DelhiveryTech/kafka-consumer-config.git

func GetLocationForMaterial(material *sql.GitMaterial) (location string, err error) {
	//gitRegex := `/(?:git|ssh|https?|git@[-\w.]+):(\/\/)?(.*?)(\.git)(\/?|\#[-\d\w._]+?)$/`
	gitRegex := `^https.*`
	matched, err := regexp.MatchString(gitRegex, material.Url)
	if matched {
		location := strings.ReplaceAll(material.Url, "https://", "")
		checkoutPath := path.Join(GIT_BASE_DIR, strconv.Itoa(material.Id), location)
		return checkoutPath, nil
	}
	return "", fmt.Errorf("unsupported format url %s", material.Url)
}

func GetAuthMethod(gitProvider *sql.GitProvider) (transport.AuthMethod, error) {

	var auth transport.AuthMethod
	switch gitProvider.AuthMode {
	case sql.AUTH_MODE_USERNAME_PASSWORD:
		auth = &http.BasicAuth{Password: gitProvider.Password, Username: gitProvider.UserName}
	case sql.AUTH_MODE_ACCESS_TOKEN:
		auth = &http.BasicAuth{Password: gitProvider.AccessToken, Username: gitProvider.UserName}
	case sql.AUTH_MODE_ANONYMOUS:
		auth = nil
	case sql.AUTH_MODE_SSH:
		//signer, err := ssh.ParsePrivateKey([]byte(""))
		//auth = &ssh.PublicKeys{User: "git", Signer: signer}
		return nil, fmt.Errorf("ssh not supported")
	default:
		return nil, fmt.Errorf("unsupported %s", gitProvider.AuthMode)
	}
	return auth, nil
}

func GetUserNamePassword(gitProvider *sql.GitProvider) (userName, password string, err error) {
	switch gitProvider.AuthMode {
	case sql.AUTH_MODE_USERNAME_PASSWORD:
		return gitProvider.UserName, gitProvider.Password, nil
	case sql.AUTH_MODE_ACCESS_TOKEN:

		return gitProvider.UserName, gitProvider.AccessToken, nil
	case sql.AUTH_MODE_ANONYMOUS:
		return "", "", nil
	case sql.AUTH_MODE_SSH:
		return "", "", fmt.Errorf("ssh not supported")
	default:
		return "", "", fmt.Errorf("unsupported %s", gitProvider.AuthMode)
	}
}
