/*
 * Copyright (c) 2024. Devtron Inc.
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
	"github.com/devtron-labs/git-sensor/internals/sql"
	"strings"
)

func BuildExtraEnvironmentVariablesForCi(filterResults []*sql.CiPipelineMaterialWebhookDataMappingFilterResult, extEnvVariable map[string]string) map[string]string {
	extraEnvironmentVariables := make(map[string]string)
	for _, filterResult := range filterResults {
		for k, v := range filterResult.MatchedGroups {
			key := buildEnvVariableKey(k)
			extraEnvironmentVariables[key] = v
		}
	}
	for k, v := range extEnvVariable {
		key := buildEnvVariableKey(k)
		extraEnvironmentVariables[key] = v
	}
	return extraEnvironmentVariables
}

func buildEnvVariableKey(originalKey string) string {
	// upper case
	key := strings.ToUpper(originalKey)
	// replace space with underscore
	key = strings.ReplaceAll(key, " ", "_")
	return key
}
