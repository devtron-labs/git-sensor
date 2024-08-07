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

package git

import (
	"fmt"
	"github.com/devtron-labs/git-sensor/internals/middleware"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	GIT_BASE_DIR              = "/git-base/"
	SSH_PRIVATE_KEY_DIR       = GIT_BASE_DIR + "ssh-keys/"
	TLS_FILES_DIR             = GIT_BASE_DIR + "tls-files/"
	SSH_PRIVATE_KEY_FILE_NAME = "ssh_pvt_key"
	TLS_KEY_FILE_NAME         = "tls_key.key"
	TLS_CERT_FILE_NAME        = "tls_cert.pem"
	CA_CERT_FILE_NAME         = "ca_cert.pem"
	CLONE_TIMEOUT_SEC         = 600
	FETCH_TIMEOUT_SEC         = 30
	GITHUB_PROVIDER           = "github.com"
	GITLAB_PROVIDER           = "gitlab.com"
	CloningModeShallow        = "SHALLOW"
	CloningModeFull           = "FULL"
	NO_COMMIT_GIT_ERROR_MESSAGE    = "unknown revision or path not in the working tree."
	NO_COMMIT_CUSTOM_ERROR_MESSAGE = "No Commit Found"
)

//git@gitlab.com:devtron-client-gitops/wms-user-management.git
//https://gitlab.com/devtron-client-gitops/wms-user-management.git

//git@bitbucket.org:DelhiveryTech/kafka-consumer-config.git
//https://prashant-delhivery@bitbucket.org/DelhiveryTech/kafka-consumer-config.git

func GetProjectName(url string) string {
	//if url = https://github.com/devtron-labs/git-sensor.git then it will return git-sensor
	url = url[strings.LastIndex(url, "/")+1:]
	return strings.TrimSuffix(url, ".git")
}
func GetCheckoutPath(url string, cloneLocation string) string {
	//url= https://github.com/devtron-labs/git-sensor.git cloneLocation= git-base/1/github.com/prakash100198
	//then this function returns git-base/1/github.com/prakash100198/SampleGoLangProject/.git
	projectName := GetProjectName(url)
	projRootDir := cloneLocation + "/" + projectName + "/.git"
	return projRootDir
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

func CreateTlsPathFilesWithData(gitProviderId int, content string, fileName string) (string, error) {
	tlsFolderPath := path.Join(TLS_FILES_DIR, strconv.Itoa(gitProviderId))
	tlsFilePath := path.Join(tlsFolderPath, fileName)

	// if file exists then delete file
	if _, err := os.Stat(tlsFilePath); os.IsExist(err) {
		os.Remove(tlsFilePath)
	}
	// create dirs
	err := os.MkdirAll(tlsFolderPath, 0755)
	if err != nil {
		return "", err
	}

	// create file with content
	err = ioutil.WriteFile(tlsFilePath, []byte(content), 0600)
	if err != nil {
		return "", err
	}
	return tlsFilePath, nil
}

func DeleteAFileIfExists(path string) error {
	if _, err := os.Stat(path); os.IsExist(err) {
		err = os.Remove(path)
		return err
	}
	return nil
}

// sample commitDiff :=4\t3\tModels/models.go\n2\t2\tRepository/Repository.go\n0\t2\main.go
func processFileStatOutputWithNumstat(commitDiff string) (FileStats, error) {
	filestat := FileStats{}
	lines := strings.Split(strings.TrimSpace(commitDiff), "\n")

	for _, line := range lines {
		parts := strings.Split(line, "\t")

		if len(parts) != 3 {
			fmt.Printf("invalid git diff --numstat output, parts: %v\n", parts)
			middleware.CommitStatParsingErrorCounter.WithLabelValues().Inc()
			continue
		}

		if parts[0] == "-" && parts[1] == "-" {
			// ignoring binary file
			continue
		}
		var isParsingError bool
		//TODO not ignoring in case of error in below cases because of include/exclude feature where file name is important
		added, err := strconv.Atoi(parts[0])
		if err != nil {
			fmt.Printf("failed to parse number of lines added: %v\n", err)
			isParsingError = true
		}

		deleted, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Printf("failed to parse number of lines deleted: %v\n", err)
			isParsingError = true
		}
		if isParsingError {
			middleware.CommitStatParsingErrorCounter.WithLabelValues().Inc()
		}

		filestat = append(filestat, FileStat{
			Name:     parts[2],
			Addition: added,
			Deletion: deleted,
		})
	}

	return filestat, nil
}

// sample commitDiff :=pkg/bulkAction/BulkUpdateService.go\nscripts/sql/244_alter_resource_release_feature.down.sql\nscripts/sql/244_alter_resource_release_feature.up.sql
func processFileStatOutputNameOnly(commitDiff string) (FileStats, error) {
	filestat := FileStats{}
	lines := strings.Split(strings.TrimSpace(commitDiff), "\n")

	for _, line := range lines {

		filestat = append(filestat, FileStat{
			Name: line,
		})
	}

	return filestat, nil
}
func IsRepoShallowCloned(checkoutPath string) bool {
	return strings.Contains(checkoutPath, "/.git")
}
