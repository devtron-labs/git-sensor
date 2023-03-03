package git

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGitCommitDeserialization(t *testing.T) {

	// Arrange
	gitCommits := "[{\"commit\":\"xyz\",\"author\":\"devtron\"," +
		"\"date\":\"2021-02-18T21:54:42.123Z\",\"message\":\"message\"," +
		"\"changes\":[\"1\",\"2\",\"3\"],\"fileStats\":[{\"name\":\"z\",\"addition\":50," +
		"\"deletion\":10},{\"name\":\"z\",\"addition\":10,\"deletion\":40}]," +
		"\"webhookData\":{\"id\":1,\"eventActionType\":\"action\"," +
		"\"data\":{\"a\":\"b\",\"c\":\"d\"}}},{\"commit\":\"vtu\",\"author\":\"devtron\"," +
		"\"date\":\"2021-01-18T21:54:42.123Z\",\"message\":\"message\"," +
		"\"changes\":[\"4\",\"5\",\"6\"],\"fileStats\":[{\"name\":\"r\",\"addition\":50," +
		"\"deletion\":10},{\"name\":\"q\",\"addition\":10,\"deletion\":40}]," +
		"\"webhookData\":{\"id\":1,\"eventActionType\":\"action\",\"data\":{\"a\":\"b\",\"c\":\"d\"}}}]"

	commits := make([]*GitCommit, 0)
	_ = json.Unmarshal([]byte(gitCommits), &commits)

	assert.Equal(t, len(commits), 2)
}
