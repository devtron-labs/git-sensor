package util

import (
	"fmt"
	"github.com/devtron-labs/git-sensor/internal/sql"
)

func BuildExtraEnvironmentVariablesForCi(filterResults []*sql.CiPipelineMaterialWebhookDataMappingFilterResult) map[string]string {
	extraEnvironmentVariables := make(map[string]string)
	for _, filterResult := range filterResults {
		for k, v := range filterResult.MatchedGroups {
			extraEnvironmentVariables[fmt.Sprintf("%s_%s", filterResult.SelectorName, k)] = v
		}
	}
	return extraEnvironmentVariables
}
