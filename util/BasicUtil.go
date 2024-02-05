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

package util

import (
	"github.com/devtron-labs/git-sensor/internals/middleware"
	"math/rand"
	"strings"
	"time"
)

var chars = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

// Generates random string
func Generate(size int) string {
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < size; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	str := b.String()
	return str
}

func TriggerGitOperationMetrics(method string, startTime time.Time, err error) {
	status := "Success"
	if err != nil {
		status = "Failed"
	}
	middleware.GitOperationDuration.WithLabelValues(method, status).Observe(time.Since(startTime).Seconds())
}

func GetPathRegex(path string) string {
	const MultiDirectoryMatch = "(/[0-9A-Za-z_./-]+)+"
	const singleDirectoryMatch = "(/[0-9A-Za-z_./-]+)"
	const fileMatch = "([0-9A-Za-z_./-]+)"
	path = strings.ReplaceAll(path, "/**", MultiDirectoryMatch)
	path = strings.ReplaceAll(path, "/*", singleDirectoryMatch)
	path = strings.ReplaceAll(path, "*", fileMatch)
	return path
}

func ReverseSlice(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}
