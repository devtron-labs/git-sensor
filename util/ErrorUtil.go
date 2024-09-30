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

package util

import (
	"github.com/go-pg/pg"
	"strings"
)

func IsErrNoRows(err error) bool {
	return pg.ErrNoRows == err
}

func BuildDisplayErrorMessage(cliMessage string, err error) string {
	customErrorMessage := GetErrMsgFromCliMessage(cliMessage, err)
	if customErrorMessage != "" {
		return customErrorMessage
	} else {
		if cliMessage != "" {
			return cliMessage
		} else {
			return err.Error()
		}
	}
}

// This function returns custom error message. If cliMessage is empty then it checks same handling in err.Error()
func GetErrMsgFromCliMessage(cliMessage string, err error) string {
	errMsg := strings.TrimSpace(cliMessage)
	if errMsg == "" {
		if err == nil {
			return ""
		}
		errMsg = err.Error()
	}
	if strings.Contains(errMsg, AUTHENTICATION_FAILED_ERROR) || strings.Contains(errMsg, DIRECTORY_NOT_EXISTS_ERROR) {
		return CHECK_REPO_MESSAGE_RESPONSE
	}
	return ""
}
