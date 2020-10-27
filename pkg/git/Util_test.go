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

package git

import "testing"

func TestGetLocationForUrl(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name         string
		args         args
		wantLocation string
		wantErr      bool
	}{
		{
			name:         "gitlab-ssh",
			args:         args{url: "git@gitlab.com:devtron-client-gitops/wms-user-management.git"},
			wantErr:      false,
			wantLocation: "gitlab.com/devtron-client-gitops/wms-user-management.git",
		},
		{
			name:         "gitlab-ssh",
			args:         args{url: "https://gitlab.com/devtron-client-gitops/wms-user-management.git"},
			wantErr:      false,
			wantLocation: "gitlab.com/devtron-client-gitops/wms-user-management.git",
		},
		{
			name:         "gitlab-ssh",
			args:         args{url: "git@bitbucket.org:DelhiveryTech/kafka-consumer-config.git"},
			wantErr:      false,
			wantLocation: "bitbucket.org/DelhiveryTech/kafka-consumer-config.git",
		},
		{
			name:         "gitlab-ssh",
			args:         args{url: "https://prashant-delhivery@bitbucket.org/DelhiveryTech/kafka-consumer-config.git"},
			wantErr:      false,
			wantLocation: "bitbucket.org/DelhiveryTech/kafka-consumer-config.git",
		},
		{
			name:         "gitlab-ssh",
			args:         args{url: "efcererer"},
			wantErr:      true,
			wantLocation: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLocation, err := GetLocationForUrl(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLocationForUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotLocation != tt.wantLocation {
				t.Errorf("GetLocationForUrl() = %v, want %v", gotLocation, tt.wantLocation)
			}
		})
	}
}
