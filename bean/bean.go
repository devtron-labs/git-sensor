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

package bean

import "fmt"

type StartupConfig struct {
	RestPort int `env:"SERVER_REST_PORT" envDefault:"8080"`
	GrpcPort int `env:"SERVER_GRPC_PORT" envDefault:"8081"`
}

const ReloadAllLogPrefix = "RELOAD_ALL_LOG"

func GetReloadAllLog(message string) string {
	return fmt.Sprintf("%s: %s", ReloadAllLogPrefix, message)
}

type ReloadAllMaterialQuery struct {
	Start int `json:"start"`
	End   int `json:"end"`
}
