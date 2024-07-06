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
	"context"
	"github.com/devtron-labs/common-lib/utils"
	"github.com/devtron-labs/git-sensor/internals"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"
	"time"
)

var gitRepoUrl = "https://github.com/devtron-labs/getting-started-nodejs"
var location1 = baseDir + "/git-base/1/github.com/devtron-labs/getting-started-nodejs.git"
var location2 = baseDir + "/git-base/2/github.com/devtron-labs/getting-started-nodejs.git"
var commitHash = "dfde5ecae5cd1ae6a7e3471a63a8277177898a7d"
var tag = "v0.0.2"
var branchName = "do-not-touch-this-branch"
var baseDir = "/Users/subhashish/tmp1"
var privateGitRepoUrl = "https://github.com/devtron-labs/getting-started-nodejs.git"
var privateGitRepoLocation = baseDir + "/git-base/42/github.com/devtron-labs/getting-started-nodejs.git"
var username = ""
var password = ""
var sshPrivateKey = ``

func getRepoManagerAnalyticsImpl(t *testing.T) *RepositoryManagerAnalyticsImpl {
	return &RepositoryManagerAnalyticsImpl{repoManager: getRepoManagerImpl(t)}
}

func getRepoManagerImpl(t *testing.T) *RepositoryManagerImpl {
	logger, err := utils.NewSugardLogger()
	assert.Nil(t, err)
	conf := &internals.Configuration{
		CommitStatsTimeoutInSec: 0,
		EnableFileStats:         true,
		GitHistoryCount:         2,
		UseGitCli:               false,
		GoGitTimeout:            10,
	}
	base := NewGitManagerBaseImpl(logger, conf)
	_ = NewGitCliManagerImpl(base, logger)
	_ = NewGoGitSDKManagerImpl(base, logger)

	gitUtil := NewGitManagerImpl(logger, conf)
	repositoryManagerImpl := NewRepositoryManagerImpl(logger, conf, gitUtil)
	return repositoryManagerImpl
}

func setupSuite(t *testing.T) func(t *testing.T) {

	err := os.MkdirAll(privateGitRepoLocation, 0700)
	assert.Nil(t, err)
	// Return a function to teardown the test
	return func(t *testing.T) {

	}
}

func TestRepositoryManager_Add(t *testing.T) {
	type args struct {
		gitProviderId        int
		location             string
		url                  string
		gitCtx               GitContext
		authMode             sql.AuthMode
		sshPrivateKeyContent string
	}
	tests := []struct {
		name    string
		payload args
		wantErr bool
	}{
		{
			name: "Test1_Add_InvokingWithCorrectArgumentsWithCreds", payload: args{
				gitProviderId: 1,
				location:      privateGitRepoLocation,
				url:           privateGitRepoUrl,
				gitCtx: GitContext{
					Context:  context.Background(),
					Username: username,
					Password: password,
				},
				authMode:             "USERNAME_PASSWORD",
				sshPrivateKeyContent: "",
			}, wantErr: false,
		},
		{
			name: "Test2_Add_InvokingWithCorrectArgumentsWithSSHCreds", payload: args{
				gitProviderId: 1,
				location:      privateGitRepoLocation,
				url:           privateGitRepoUrl,
				gitCtx: GitContext{
					Context:  context.Background(),
					Username: "",
					Password: "",
				},
				authMode:             "SSH",
				sshPrivateKeyContent: sshPrivateKey,
			}, wantErr: false,
		},
		{
			name: "Test3_Add_InvokingWithInvalidGitUrlWithoutCreds", payload: args{
				gitProviderId: 1,
				location:      location1,
				url:           gitRepoUrl + "dhs",
				gitCtx: GitContext{
					Context:  context.Background(),
					Username: "",
					Password: "",
				},
				authMode:             "ANONYMOUS",
				sshPrivateKeyContent: "",
			}, wantErr: true,
		},
		{
			name: "Test4_Add_InvokingWithCorrectArgumentsWithoutCreds", payload: args{
				gitProviderId: 1,
				location:      location2,
				url:           gitRepoUrl,
				gitCtx: GitContext{
					Context:  context.Background(),
					Username: "",
					Password: "",
				},
				authMode:             "ANONYMOUS",
				sshPrivateKeyContent: "",
			}, wantErr: false,
		},
	}
	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		if tt.payload.authMode == "SSH" {
			_, err := repositoryManagerImpl.CreateSshFileIfNotExistsAndConfigureSshCommand(BuildGitContext(context.Background()), tt.payload.location, tt.payload.gitProviderId, tt.payload.sshPrivateKeyContent)
			assert.Nil(t, err)
		}
		t.Run(tt.name, func(t *testing.T) {
			err := repositoryManagerImpl.Add(tt.payload.gitCtx, tt.payload.gitProviderId, tt.payload.location, tt.payload.url, tt.payload.authMode, tt.payload.sshPrivateKeyContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
		})
	}
}

func TestRepositoryManager_Fetch(t *testing.T) {

	type args struct {
		location string
		url      string
		gitCtx   GitContext
	}
	tests := []struct {
		name    string
		payload args
		wantErr bool
	}{
		{
			name: "Test1_Fetch_InvokingWithValidGitUrlWithoutCreds", payload: args{
				location: location2,
				url:      gitRepoUrl,
				gitCtx: GitContext{
					Context:  context.Background(),
					Username: "",
					Password: "",
				},
			}, wantErr: false,
		},
		{
			name: "Test2_Fetch_InvokingWithInvalidGitUrlWithoutCreds", payload: args{
				location: location1,
				url:      gitRepoUrl + "dhs",
				gitCtx: GitContext{
					Context:  context.Background(),
					Username: "",
					Password: "",
				},
			}, wantErr: true,
		},
		{
			name: "Test3_Fetch_InvokingWithCorrectArgumentsWithCreds", payload: args{
				location: privateGitRepoLocation,
				url:      privateGitRepoUrl,
				gitCtx: GitContext{
					Context:  context.Background(),
					Username: username,
					Password: password,
				},
			}, wantErr: false,
		},
		{
			name: "Test4_Fetch_InvokingWithWrongLocationOfLocalDir", payload: args{
				location: baseDir + "/git-base/42/github.com/devtron-labs-private/agetting-started-nodejsgits",
				url:      privateGitRepoUrl,
				gitCtx: GitContext{
					Context:  context.Background(),
					Username: username,
					Password: password,
				},
			}, wantErr: true,
		},
	}
	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := repositoryManagerImpl.Fetch(tt.payload.gitCtx, tt.payload.url, tt.payload.location)

			if (err != nil) != tt.wantErr {
				t.Errorf("Fetch() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
		})
	}
}

func TestRepositoryManager_GetCommitMetadata(t *testing.T) {

	type args struct {
		checkoutPath string
		commitHash   string
	}
	tests := []struct {
		name    string
		payload args
		want    *GitCommitBase
		wantErr bool
	}{
		{
			name: "Test1_GetCommitMetadata_InvokingWithCorrectCheckoutPathAndCorrectCommitHash", payload: args{
				checkoutPath: location2,
				commitHash:   commitHash,
			},
			want: &GitCommitBase{
				Commit:      "dfde5ecae5cd1ae6a7e3471a63a8277177898a7d",
				FileStats:   nil,
				WebhookData: nil,
				Excluded:    false,
			},
			wantErr: false,
		},
		{
			name: "Test2_GetCommitMetadata_InvokingWithInvalidCommitHash", payload: args{
				checkoutPath: location2,
				commitHash:   commitHash[0:6] + "vdsvsdc234rrwffeads",
			}, wantErr: true,
		},
		{
			name: "Test3_GetCommitMetadata_InvokingWithIncorrectLocalGitRepo", payload: args{
				checkoutPath: location2 + "/sgsrsfvdfac",
				commitHash:   commitHash,
			}, wantErr: true,
		},
	}
	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repositoryManagerImpl.GetCommitMetadata(BuildGitContext(context.Background()), tt.payload.checkoutPath, tt.payload.commitHash)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitMetadata() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}

			if tt.want != nil && got != nil {
				got.Date = time.Time{}
				got.Author = ""
				got.Message = ""
				got.Changes = nil

				if !reflect.DeepEqual(*got, *tt.want) {
					t.Errorf("GetCommitMetadata() got = %v, want %v", got, tt.want)
				}
			}

		})
	}
}

func TestRepositoryManager_ChangesSince(t *testing.T) {

	type args struct {
		checkoutPath string
		branch       string
		from         string
		to           string
		count        int
	}
	tests := []struct {
		name    string
		payload args
		want    []*GitCommitBase
		wantErr bool
	}{
		{
			name: "Test1_ChangesSince_InvokingWithCorrectCheckoutPathAndCorrectBranch", payload: args{
				checkoutPath: location2,
				branch:       branchName,
				from:         "",
				to:           "",
				count:        2,
			},
			want: []*GitCommitBase{
				{
					Commit: "2a1683d1c95dd260b311cf59b274792c7b0478ce",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 0,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit: "66054005ca83d6e0f3daff2a93f4f30bc70d9aff",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
			},
			wantErr: false,
		},
		{
			name: "Test2_ChangesSince_InvokingWithInvalidCheckoutPathAndCorrectBranch", payload: args{
				checkoutPath: location2 + "/sffv",
				branch:       branchName,
				from:         "",
				to:           "",
				count:        2,
			}, wantErr: true,
		},
		{
			name: "Test3_ChangesSince_InvokingWithValidCheckoutPathAndInvalidBranch", payload: args{
				checkoutPath: location2,
				branch:       branchName + "-sdvfevfdsc",
				from:         "",
				to:           "",
				count:        2,
			}, wantErr: true,
		},
		{
			name: "Test4_ChangesSince_InvokingWithValidCheckoutPathAndValidBranchWithZeroCount", payload: args{
				checkoutPath: location2,
				branch:       branchName,
				from:         "",
				to:           "",
				count:        0,
			},
			want: []*GitCommitBase{
				{
					Commit: "2a1683d1c95dd260b311cf59b274792c7b0478ce",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 0,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit: "66054005ca83d6e0f3daff2a93f4f30bc70d9aff",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
			},
			wantErr: false,
		},
	}
	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repositoryManagerImpl.ChangesSince(BuildGitContext(context.Background()), tt.payload.checkoutPath, tt.payload.branch, tt.payload.from, tt.payload.to, tt.payload.count)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangesSince() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if len(tt.want) != len(got) {
				t.Errorf("ChangesSince() got = %v, want %v", got, tt.want)
			}
			for index, want := range tt.want {
				if want != nil && got != nil {
					got[index].Date = time.Time{}
					got[index].Author = ""
					got[index].Message = ""
					got[index].Changes = nil

					if !reflect.DeepEqual(*got[index].FileStats, *want.FileStats) {
						t.Errorf("ChangesSince() got = %v, want %v", got, tt.want)
					}

					got[index].FileStats = nil
					want.FileStats = nil

					if !reflect.DeepEqual(*got[index], *want) {
						t.Errorf("ChangesSince() got = %v, want %v", got, tt.want)
					}
				}
			}

		})
	}
}

func TestRepositoryManager_ChangesSinceByRepository(t *testing.T) {

	type args struct {
		checkoutPath string
		branch       string
		from         string
		to           string
		count        int
	}
	tests := []struct {
		name    string
		payload args
		want    []*GitCommitBase
		wantErr bool
	}{
		{
			name: "Test1_ChangesSinceByRepository_InvokingWithCorrectPayloadAndBranchPrefix", payload: args{
				checkoutPath: location2,
				branch:       "refs/heads/" + branchName,
				from:         "",
				to:           "",
				count:        2,
			},
			want: []*GitCommitBase{
				{
					Commit: "2a1683d1c95dd260b311cf59b274792c7b0478ce",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 0,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit: "66054005ca83d6e0f3daff2a93f4f30bc70d9aff",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
			},
			wantErr: false,
		},
		{
			name: "Test2_ChangesSinceByRepository_InvokingWithFromAndTo", payload: args{
				checkoutPath: location2,
				branch:       branchName,
				from:         "8d6f9188cdb7d52b4dc3560caf77477cd67a3fc2",
				to:           "1afbce41f37ce71d0973be2b4a972c6475abc3a7",
				count:        2,
			},
			want: []*GitCommitBase{
				{
					Commit: "1afbce41f37ce71d0973be2b4a972c6475abc3a7",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit: "daa4872b903b0b47c2136d4e6fe50356a6b01d33",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
			},
			wantErr: false,
		},
		{
			name: "Test3_ChangesSinceByRepository_InvokingWithFromHash", payload: args{
				checkoutPath: location2,
				branch:       branchName,
				from:         "8d6f9188cdb7d52b4dc3560caf77477cd67a3fc2",
				to:           "",
				count:        2,
			},
			want: []*GitCommitBase{
				{
					Commit: "2a1683d1c95dd260b311cf59b274792c7b0478ce",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 0,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit: "66054005ca83d6e0f3daff2a93f4f30bc70d9aff",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
			},
			wantErr: false,
		},
		{
			name: "Test4_ChangesSinceByRepository_InvokingWithToHash", payload: args{
				checkoutPath: location2,
				branch:       branchName,
				from:         "",
				to:           "1afbce41f37ce71d0973be2b4a972c6475abc3a7",
				count:        3,
			},
			want: []*GitCommitBase{
				{
					Commit: "1afbce41f37ce71d0973be2b4a972c6475abc3a7",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit: "daa4872b903b0b47c2136d4e6fe50356a6b01d33",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit: "7ffe9bcd668835e9de8d87582c42f8642efb6012",
					FileStats: &FileStats{FileStat{
						Name:     "README.md",
						Addition: 1,
						Deletion: 1,
					}},
					WebhookData: nil,
					Excluded:    false,
				},
			},
			wantErr: false,
		},
		{
			name: "Test5_ChangesSinceByRepository_InvokingWithCountZero", payload: args{
				checkoutPath: location2,
				branch:       branchName,
				from:         "1afbce41f37ce71d0973be2b4a972c6475abc3a7",
				to:           "",
				count:        0,
			},
			want:    nil,
			wantErr: false,
		},
	}

	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		//r, err := git.PlainOpen(tt.payload.checkoutPath)
		r, err := repositoryManagerImpl.gitManager.OpenRepoPlain(tt.payload.checkoutPath)
		assert.Nil(t, err)
		t.Run(tt.name, func(t *testing.T) {
			got, _, _, err := repositoryManagerImpl.ChangesSinceByRepository(BuildGitContext(context.Background()), r, tt.payload.branch, tt.payload.from, tt.payload.to, tt.payload.count, tt.payload.checkoutPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangesSinceByRepository() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}

			if len(tt.want) != len(got) {
				t.Errorf("ChangesSinceByRepository() got = %v, want %v", got, tt.want)
			}

			for index, want := range tt.want {
				if want != nil && got != nil {
					got[index].Date = time.Time{}
					got[index].Author = ""
					got[index].Message = ""
					got[index].Changes = nil

					if !reflect.DeepEqual(*got[index], *want) {
						t.Errorf("ChangesSinceByRepository() got = %v, want %v", got, tt.want)
					}
				}
			}

		})
	}
}

func TestRepositoryManager_GetCommitForTag(t *testing.T) {

	type args struct {
		checkoutPath string
		tag          string
	}
	tests := []struct {
		name    string
		payload args
		want    *GitCommitBase
		wantErr bool
	}{
		{
			name: "Test1_GetCommitForTag_InvokingWithCorrectCheckoutPathAndCorrectTag", payload: args{
				checkoutPath: location2,
				tag:          tag,
			},
			want: &GitCommitBase{
				Commit:      "6e0d605a1c9fbf2717b7fe8a3d4ae23ab006e5c0",
				FileStats:   nil,
				WebhookData: nil,
				Excluded:    false,
			},
			wantErr: false,
		},
		{
			name: "Test2_GetCommitForTag_InvokingWithInvalidTag", payload: args{
				checkoutPath: location2,
				tag:          tag + "vdsvsdc2feads",
			}, wantErr: true,
		},
		{
			name: "Test3_GetCommitForTag_InvokingWithIncorrectLocalGitRepo", payload: args{
				checkoutPath: location2 + "/sgsrsfvdfac",
				tag:          tag,
			}, wantErr: true,
		},
	}
	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repositoryManagerImpl.GetCommitForTag(BuildGitContext(context.Background()), tt.payload.checkoutPath, tt.payload.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitMetadata() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}

			if tt.want != nil && got != nil {
				got.Date = time.Time{}
				got.Author = ""
				got.Message = ""
				got.Changes = nil

				if !reflect.DeepEqual(*got, *tt.want) {
					t.Errorf("GetCommitMetadata() got = %v, want %v", got, tt.want)
				}
			}

		})
	}
}

func TestRepositoryManager_ChangesSinceByRepositoryForAnalytics(t *testing.T) {

	type args struct {
		checkoutPath string
		oldHash      string
		newHash      string
	}
	tests := []struct {
		name    string
		payload args
		want    *GitChanges
		wantErr bool
	}{
		{
			name: "Test1_ChangesSinceByRepositoryForAnalytics_InvokingWithCorrectCheckoutPathAndCorrectOldAndNewHash", payload: args{
				checkoutPath: location2,
				oldHash:      "da3ba3254712965b5944a6271e71bff91fe51f20",
				newHash:      "4da167b5242b79c609e1e92e9e05f00ba325c284",
			},
			want: &GitChanges{
				Commits: []*Commit{
					{
						Hash: &Hash{
							Long:  "17489a358dedf304c267b502be37c21f81cbe5d2",
							Short: "17489a35",
						},
						Tree: &Tree{
							Long:  "a87efbc3ee22e0eb3678401a3d3f6e95da05305a",
							Short: "a87efbc3",
						},
						Author: &Author{
							Name:  "Prashant Ghildiyal",
							Email: "60953820+pghildiyal@users.noreply.github.com",
							Date:  time.Now(),
						},
						Committer: &Committer{
							Name:  "GitHub",
							Email: "noreply@github.com",
							Date:  time.Time{},
						},
						Tag:     nil,
						Subject: "Update app.js",
						Body:    "",
					},
					{
						Hash: &Hash{
							Long:  "4da167b5242b79c609e1e92e9e05f00ba325c284",
							Short: "4da167b5",
						},
						Tree: &Tree{
							Long:  "691f8324102aa3c2d6ca20ec71e9cd1395b419cd",
							Short: "691f8324",
						},
						Author: &Author{
							Name:  "pawan-mehta-dt",
							Email: "117346502+pawan-mehta-dt@users.noreply.github.com",
							Date:  time.Now(),
						},
						Committer: &Committer{
							Name:  "GitHub",
							Email: "noreply@github.com",
							Date:  time.Time{},
						},
						Tag:     nil,
						Subject: "Updated dockerfile for multi-arch support",
						Body:    "",
					},
				},
				FileStats: FileStats{
					FileStat{
						Name:     "Dockerfile",
						Addition: 1,
						Deletion: 1,
					},
					FileStat{
						Name:     "app.js",
						Addition: 2,
						Deletion: 1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Test2_ChangesSinceByRepositoryForAnalytics_InvokingWithCorrectCheckoutPathAndInCorrectOldAndNewHash", payload: args{
				checkoutPath: location2,
				oldHash:      "87234877rfvervrvve34hufda3ba3254712965b5944a6271e71f20",
				newHash:      "4289u34r8924ufhiuwefnoweijfhwe9udwsvda167b5242b325c284",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test3_ChangesSinceByRepositoryForAnalytics_InvokingWithIncorrectCheckoutPathAndCorrectOldAndNewHash", payload: args{
				checkoutPath: location2 + "/dsjnvfuiv",
				oldHash:      "da3ba3254712965b5944a6271e71bff91fe51f20",
				newHash:      "4da167b5242b79c609e1e92e9e05f00ba325c284",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test4_ChangesSinceByRepositoryForAnalytics_InvokingWithIncorrectCheckoutPathAndIncorrectCorrectNewHash", payload: args{
				checkoutPath: location2,
				oldHash:      "da3ba3254712965b5944a6271e71bff91fe51f20",
				newHash:      "4289u34r8924ufhiuwefnoweijfhwe9udwsvda167b5242b325c284",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test5_ChangesSinceByRepositoryForAnalytics_InvokingWithSameOldAndNewHash", payload: args{
				checkoutPath: location2,
				oldHash:      "da3ba3254712965b5944a6271e71bff91fe51f20",
				newHash:      "da3ba3254712965b5944a6271e71bff91fe51f20",
			},
			want: &GitChanges{
				Commits: []*Commit{
					{
						Hash: &Hash{
							Long:  "da3ba3254712965b5944a6271e71bff91fe51f20",
							Short: "da3ba325",
						},
						Tree: &Tree{
							Long:  "d367e0fe1b9f15ccdbf0bb10a53b1e2e554e4c00",
							Short: "d367e0fe",
						},
						Author: &Author{
							Name:  "Prakarsh",
							Email: "71125043+prakarsh-dt@users.noreply.github.com",
							Date:  time.Time{},
						},
						Committer: &Committer{
							Name:  "GitHub",
							Email: "noreply@github.com",
							Date:  time.Time{},
						},
						Tag:     nil,
						Subject: "Update README.md",
						Body:    "",
					},
				},
				FileStats: nil,
			},
			wantErr: false,
		},
		{
			name: "Test6_ChangesSinceByRepositoryForAnalytics_InvokingWithOldAndNewHashReversed", payload: args{
				checkoutPath: location2,
				oldHash:      "4da167b5242b79c609e1e92e9e05f00ba325c284",
				newHash:      "da3ba3254712965b5944a6271e71bff91fe51f20",
			},
			want: &GitChanges{
				Commits: []*Commit{
					{
						Hash: &Hash{
							Long:  "da3ba3254712965b5944a6271e71bff91fe51f20",
							Short: "da3ba325",
						},
						Tree: &Tree{
							Long:  "d367e0fe1b9f15ccdbf0bb10a53b1e2e554e4c00",
							Short: "d367e0fe",
						},
						Author: &Author{
							Name:  "Prakarsh",
							Email: "71125043+prakarsh-dt@users.noreply.github.com",
							Date:  time.Time{},
						},
						Committer: &Committer{
							Name:  "GitHub",
							Email: "noreply@github.com",
							Date:  time.Time{},
						},
						Tag:     nil,
						Subject: "Update README.md",
						Body:    "",
					},
				},
				FileStats: FileStats{
					FileStat{
						Name:     "Dockerfile",
						Addition: 1,
						Deletion: 1,
					},
					FileStat{
						Name:     "app.js",
						Addition: 1,
						Deletion: 2,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Test7_ChangesSinceByRepositoryForAnalytics_InvokingWithEmptyNewHash", payload: args{
				checkoutPath: location2,
				oldHash:      "da3ba3254712965b5944a6271e71bff91fe51f20",
				newHash:      "",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test8_ChangesSinceByRepositoryForAnalytics_InvokingWithEmptyOldHash", payload: args{
				checkoutPath: location2,
				oldHash:      "",
				newHash:      "4da167b5242b79c609e1e92e9e05f00ba325c284",
			},
			want:    nil,
			wantErr: true,
		},
	}
	repositoryManagerImpl := getRepoManagerAnalyticsImpl(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGitChanges, err := repositoryManagerImpl.ChangesSinceByRepositoryForAnalytics(BuildGitContext(context.Background()), tt.payload.checkoutPath, tt.payload.oldHash, tt.payload.newHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangesSinceByRepositoryForAnalytics() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if tt.want != nil && gotGitChanges != nil {

				if !areEqualStruct(*tt.want, *gotGitChanges) {
					t.Errorf("ChangesSinceByRepositoryForAnalytics() got = %v, want %v", gotGitChanges, tt.want)
				}
			}

		})
	}
}

func areEqualStruct(wantt GitChanges, gotGitChanges GitChanges) bool {
	//comparing commits
	//for i, got := range gotGitChanges.Commits {
	got := getOldestCommit(gotGitChanges.Commits)
	want := getOldestCommit(wantt.Commits)
	got.Author.Date = time.Time{}
	got.Committer.Date = time.Time{}
	want.Author.Date = time.Time{}

	if got.Hash.Long != want.Hash.Long {
		return false
	}
	//if !reflect.DeepEqual(*got.Tree, *want.Commits[i].Tree) {
	//	return false
	//}
	if !reflect.DeepEqual(*got.Author, *want.Author) {
		return false
	}
	if !reflect.DeepEqual(*got.Committer, *want.Committer) {
		return false
	}
	if got.Subject != want.Subject || got.Body != want.Body {
		return false
	}
	//}
	if !reflect.DeepEqual(gotGitChanges.FileStats, wantt.FileStats) {
		return false
	}

	return true
}

func TestRepositoryManager_Clean(t *testing.T) {

	type args struct {
		dir string
	}
	tests := []struct {
		name    string
		payload args
		wantErr bool
	}{
		{
			name: "Test1_Clean_InvokingWithCorrectDir", payload: args{
				dir: location2,
			},
			wantErr: false,
		},
		{
			name: "Test2_Clean_InvokingWithInCorrectDir", payload: args{
				dir: "/bh/ij/" + location2 + "/wei/uhfe",
			},
			wantErr: false,
		},
	}
	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repositoryManagerImpl.Clean(tt.payload.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Clean() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			err = os.Chdir(tt.payload.dir)
			if err == nil {
				t.Errorf("error in Test1_Clean_InvokingWithCorrectDir")
				return
			}

		})
	}
}

func TestMain(m *testing.M) {
	var t *testing.T
	tearDownSuite := setupSuite(t)
	code := m.Run()
	tearDownSuite(t)
	os.Exit(code)
}
