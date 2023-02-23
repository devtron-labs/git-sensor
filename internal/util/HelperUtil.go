package util

import (
	"github.com/devtron-labs/git-sensor/internal/sql"
	"strings"
)

func BuildExtraEnvironmentVariablesForCi(filterResults []*sql.CiPipelineMaterialWebhookDataMappingFilterResult) map[string]string {
	extraEnvironmentVariables := make(map[string]string)
	for _, filterResult := range filterResults {
		for k, v := range filterResult.MatchedGroups {
			// upper case
			key := strings.ToUpper(k)
			// replace space with underscore
			key = strings.ReplaceAll(key, " ", "_")
			extraEnvironmentVariables[key] = v
		}
	}
	return extraEnvironmentVariables
}
