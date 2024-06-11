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
