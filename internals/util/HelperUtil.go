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
