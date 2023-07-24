package git

import (
	"github.com/devtron-labs/common-lib/utils"
	"github.com/devtron-labs/git-sensor/internal"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"
	"time"
)

var gitRepoUrl = "https://github.com/devtron-labs/getting-started-nodejs.git"
var location1 = "/tmp/git-base/1/github.com/devtron-labs/getting-started-nodejs.git"
var location2 = "/tmp/git-base/2/github.com/devtron-labs/getting-started-nodejs.git"
var commitHash = "dfde5ecae5cd1ae6a7e3471a63a8277177898a7d"
var tag = "v0.0.2"
var branchName = "master-1"
var baseDir = "tmp/"
var privateGitRepoUrl = "https://github.com/prakash100198/HelloWorldProject.git"
var privateGitRepoLocation = "/tmp/git-base/42/github.com/prakash100198/HelloWorldProject.git"
var username = "prakash100198"
var password = ""
var sshPrivateKey = ``

func getRepoManagerImpl(t *testing.T) *RepositoryManagerImpl {
	logger, err := utils.NewSugardLogger()
	assert.Nil(t, err)
	gitCliImpl := NewGitUtil(logger)
	repositoryManagerImpl := NewRepositoryManagerImpl(logger, gitCliImpl, &internal.Configuration{
		CommitStatsTimeoutInSec: 0,
		EnableFileStats:         false,
		GitHistoryCount:         2,
	})
	return repositoryManagerImpl
}
func setupSuite(t *testing.T) func(t *testing.T) {
	//Add(1, "/tmp/git-base/31/github.com/prakash100198/SampleGoLangProject.git", "https://github.com/prakash100198/SampleGoLangProject.git", "", "", "ANONYMOUS", "")
	//Fetch("", "", "https://github.com/prakash100198/SampleGoLangProject.git", "/tmp/git-base/31/github.com/prakash100198/SampleGoLangProject.git")

	//err := os.MkdirAll(location, 0700)
	//assert.Nil(t, err)
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
		username             string
		password             string
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
				gitProviderId:        1,
				location:             privateGitRepoLocation,
				url:                  privateGitRepoUrl,
				username:             username,
				password:             password,
				authMode:             "USERNAME_PASSWORD",
				sshPrivateKeyContent: "",
			}, wantErr: false,
		},
		{
			name: "Test2_Add_InvokingWithCorrectArgumentsWithSSHCreds", payload: args{
				gitProviderId:        1,
				location:             privateGitRepoLocation,
				url:                  privateGitRepoUrl,
				username:             "",
				password:             "",
				authMode:             "SSH",
				sshPrivateKeyContent: sshPrivateKey,
			}, wantErr: false,
		},
		{
			name: "Test3_Add_InvokingWithInvalidGitUrlWithoutCreds", payload: args{
				gitProviderId:        1,
				location:             location1,
				url:                  gitRepoUrl + "dhs",
				username:             "",
				password:             "",
				authMode:             "ANONYMOUS",
				sshPrivateKeyContent: "",
			}, wantErr: true,
		},
		{
			name: "Test4_Add_InvokingWithCorrectArgumentsWithoutCreds", payload: args{
				gitProviderId:        1,
				location:             location2,
				url:                  gitRepoUrl,
				username:             "",
				password:             "",
				authMode:             "ANONYMOUS",
				sshPrivateKeyContent: "",
			}, wantErr: false,
		},
	}
	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		if tt.payload.authMode == "SSH" {
			err := repositoryManagerImpl.CreateSshFileIfNotExistsAndConfigureSshCommand(tt.payload.location, tt.payload.gitProviderId, tt.payload.sshPrivateKeyContent)
			assert.Nil(t, err)
		}
		t.Run(tt.name, func(t *testing.T) {
			err := repositoryManagerImpl.Add(tt.payload.gitProviderId, tt.payload.location, tt.payload.url, tt.payload.username, tt.payload.password, tt.payload.authMode, tt.payload.sshPrivateKeyContent)
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
		username string
		password string
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
				username: "",
				password: "",
			}, wantErr: false,
		},
		{
			name: "Test2_Fetch_InvokingWithInvalidGitUrlWithoutCreds", payload: args{
				location: location1,
				url:      gitRepoUrl + "dhs",
				username: "",
				password: "",
			}, wantErr: true,
		},
		{
			name: "Test3_Fetch_InvokingWithCorrectArgumentsWithCreds", payload: args{
				location: privateGitRepoLocation,
				url:      privateGitRepoUrl,
				username: username,
				password: password,
			}, wantErr: false,
		},
		{
			name: "Test4_Fetch_InvokingWithWrongLocationOfLocalDir", payload: args{
				location: privateGitRepoLocation + "/hjwbwfdj",
				url:      privateGitRepoUrl,
				username: "",
				password: "",
			}, wantErr: true,
		},
	}
	repositoryManagerImpl := getRepoManagerImpl(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := repositoryManagerImpl.Fetch(tt.payload.username, tt.payload.password, tt.payload.url, tt.payload.location)
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
		want    *GitCommit
		wantErr bool
	}{
		{
			name: "Test1_GetCommitMetadata_InvokingWithCorrectCheckoutPathAndCorrectCommitHash", payload: args{
				checkoutPath: location2,
				commitHash:   commitHash,
			},
			want: &GitCommit{
				Commit:      "dfde5ecae5cd1ae6a7e3471a63a8277177898a7d",
				Author:      "Vivekanand Pandey <95338474+vivek-devtron@users.noreply.github.com>",
				Message:     "Update README.md",
				Date:        time.Time{},
				Changes:     nil,
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
			got, err := repositoryManagerImpl.GetCommitMetadata(tt.payload.checkoutPath, tt.payload.commitHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitMetadata() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if tt.want != nil && got != nil {
				got.Date = time.Time{}
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
		want    []*GitCommit
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
			want: []*GitCommit{
				{
					Commit:      "2a1683d1c95dd260b311cf59b274792c7b0478ce",
					Author:      "nishant kumar <4nishantkumar@gmail.com>",
					Message:     "Update README.md",
					Date:        time.Time{},
					Changes:     nil,
					FileStats:   nil,
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit:      "66054005ca83d6e0f3daff2a93f4f30bc70d9aff",
					Author:      "nishant kumar <4nishantkumar@gmail.com>",
					Message:     "Update README.md",
					Date:        time.Time{},
					Changes:     nil,
					FileStats:   nil,
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
			want: []*GitCommit{
				{
					Commit:      "2a1683d1c95dd260b311cf59b274792c7b0478ce",
					Author:      "nishant kumar <4nishantkumar@gmail.com>",
					Message:     "Update README.md",
					Date:        time.Time{},
					Changes:     nil,
					FileStats:   nil,
					WebhookData: nil,
					Excluded:    false,
				},
				{
					Commit:      "66054005ca83d6e0f3daff2a93f4f30bc70d9aff",
					Author:      "nishant kumar <4nishantkumar@gmail.com>",
					Message:     "Update README.md",
					Date:        time.Time{},
					Changes:     nil,
					FileStats:   nil,
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
			got, err := repositoryManagerImpl.ChangesSince(tt.payload.checkoutPath, tt.payload.branch, tt.payload.from, tt.payload.to, tt.payload.count)
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
					if !reflect.DeepEqual(*got[index], *want) {
						t.Errorf("ChangesSince() got = %v, want %v", got, tt.want)
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
		want    *GitCommit
		wantErr bool
	}{
		{
			name: "Test1_GetCommitForTag_InvokingWithCorrectCheckoutPathAndCorrectTag", payload: args{
				checkoutPath: location2,
				tag:          tag,
			},
			want: &GitCommit{
				Commit:      "6e0d605a1c9fbf2717b7fe8a3d4ae23ab006e5c0",
				Author:      "pghildiy <pghildiyal82@gmail.com>",
				Message:     "Update app.js",
				Date:        time.Time{},
				Changes:     nil,
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
			got, err := repositoryManagerImpl.GetCommitForTag(tt.payload.checkoutPath, tt.payload.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitMetadata() error in %s, error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if tt.want != nil && got != nil {
				got.Date = time.Time{}
				if !reflect.DeepEqual(*got, *tt.want) {
					t.Errorf("GetCommitMetadata() got = %v, want %v", got, tt.want)
				}
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
