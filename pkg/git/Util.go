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
	"github.com/devtron-labs/git-sensor/internals/sql"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	GIT_BASE_DIR              = "/git-base/"
	SSH_PRIVATE_KEY_DIR       = GIT_BASE_DIR + "ssh-keys/"
	SSH_PRIVATE_KEY_FILE_NAME = "ssh_pvt_key"
	CLONE_TIMEOUT_SEC         = 600
	FETCH_TIMEOUT_SEC         = 30
)

//git@gitlab.com:devtron-client-gitops/wms-user-management.git
//https://gitlab.com/devtron-client-gitops/wms-user-management.git

//git@bitbucket.org:DelhiveryTech/kafka-consumer-config.git
//https://prashant-delhivery@bitbucket.org/DelhiveryTech/kafka-consumer-config.git

func GetLocationForMaterial(material *sql.GitMaterial) (location string, err error) {
	//gitRegex := `/(?:git|ssh|https?|git@[-\w.]+):(\/\/)?(.*?)(\.git)(\/?|\#[-\d\w._]+?)$/`
	httpsRegex := `^https.*`
	httpsMatched, err := regexp.MatchString(httpsRegex, material.Url)
	if httpsMatched {
		locationWithoutProtocol := strings.ReplaceAll(material.Url, "https://", "")
		checkoutPath := path.Join(GIT_BASE_DIR, strconv.Itoa(material.Id), locationWithoutProtocol)
		return checkoutPath, nil
	}

	sshRegex := `^git@.*`
	sshMatched, err := regexp.MatchString(sshRegex, material.Url)
	if sshMatched {
		checkoutPath := path.Join(GIT_BASE_DIR, strconv.Itoa(material.Id), material.Url)
		return checkoutPath, nil
	}

	return "", fmt.Errorf("unsupported format url %s", material.Url)
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
		return "", "", nil
	default:
		return "", "", fmt.Errorf("unsupported %s", gitProvider.AuthMode)
	}
}

func GetOrCreateSshPrivateKeyOnDisk(gitProviderId int, sshPrivateKeyContent string) (privateKeyPath string, err error) {
	sshPrivateKeyFolderPath := path.Join(SSH_PRIVATE_KEY_DIR, strconv.Itoa(gitProviderId))
	sshPrivateKeyFilePath := path.Join(sshPrivateKeyFolderPath, SSH_PRIVATE_KEY_FILE_NAME)

	// if file exists then return
	if _, err := os.Stat(sshPrivateKeyFilePath); os.IsExist(err) {
		return sshPrivateKeyFilePath, nil
	}

	// create dirs
	err = os.MkdirAll(sshPrivateKeyFolderPath, 0755)
	if err != nil {
		return "", err
	}

	// create file with content
	err = ioutil.WriteFile(sshPrivateKeyFilePath, []byte(sshPrivateKeyContent), 0600)
	if err != nil {
		return "", err
	}

	return sshPrivateKeyFilePath, nil
}

func CreateOrUpdateSshPrivateKeyOnDisk(gitProviderId int, sshPrivateKeyContent string) error {
	sshPrivateKeyFolderPath := path.Join(SSH_PRIVATE_KEY_DIR, strconv.Itoa(gitProviderId))
	sshPrivateKeyFilePath := path.Join(sshPrivateKeyFolderPath, SSH_PRIVATE_KEY_FILE_NAME)

	// if file exists then delete file
	if _, err := os.Stat(sshPrivateKeyFilePath); os.IsExist(err) {
		os.Remove(sshPrivateKeyFilePath)
	}

	// create dirs
	err := os.MkdirAll(sshPrivateKeyFolderPath, 0755)
	if err != nil {
		return err
	}

	// create file with content
	err = ioutil.WriteFile(sshPrivateKeyFilePath, []byte(sshPrivateKeyContent), 0600)
	if err != nil {
		return err
	}

	return nil
}

// sample commitDiff :=4\t3\tModels/models.go\n2\t2\tRepository/Repository.go\n0\t2\main.go
func getFileStat(commitDiff string) (FileStats, error) {
	filestat := FileStats{}
	lines := strings.Split(strings.TrimSpace(commitDiff), "\n")

	for _, line := range lines {
		parts := strings.Fields(line)

		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid git diff --numstat output")
		}

		added, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse number of lines added: %w", err)
		}

		deleted, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse number of lines deleted: %w", err)
		}
		filestat = append(filestat, FileStat{
			Name:     parts[2],
			Addition: added,
			Deletion: deleted,
		})
	}

	return filestat, nil
}
