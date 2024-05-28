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

package api

import (
	"context"
	"encoding/json"
	"github.com/devtron-labs/git-sensor/internals/sql"
	"github.com/devtron-labs/git-sensor/pkg"
	"github.com/devtron-labs/git-sensor/pkg/git"
	"github.com/devtron-labs/git-sensor/pkg/mocks"
	pb "github.com/devtron-labs/protos/gitSensor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"testing"
	"time"
)

// Initialize gRPC server, to be called by each method before executing any tests
func initServer(t *testing.T, repositoryManager pkg.RepoManager) (*grpc.ClientConn, error) {

	// Stop existing listener and initialize new listener
	lis := bufconn.Listen(1024 * 1024)
	t.Cleanup(func() {
		_ = lis.Close()
	})

	// Stop existing gRPC server and initialize new gRPC server
	srv := grpc.NewServer()
	t.Cleanup(func() {
		srv.Stop()
	})

	// Initialize service implementation
	gitService := GrpcHandlerImpl{
		repositoryManager: repositoryManager,
	}

	// Register service
	pb.RegisterGitSensorServiceServer(srv, &gitService)

	// Start serving requests
	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Error("failed to serve requests on " + lis.Addr().String())
			return
		}
	}()

	// Initialize client connection
	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(context.Background(),
		"",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	t.Cleanup(func() {
		_ = conn.Close()
	})

	if err != nil {
		t.Error("failed to initialize client connection")
		return nil, err
	}

	return conn, nil
}

// Test SaveGitProvider handler - happy path
func TestSaveGitProvider(t *testing.T) {

	providedGitProvider := &pb.GitProvider{
		Id:            1,
		Name:          "test",
		UserName:      "username",
		Password:      "password",
		SshPrivateKey: "ssh-pvt-key",
		AccessToken:   "access-token",
		Active:        true,
		AuthMode:      "SSH",
	}

	// Mock repository manager
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("SaveGitProvider", mock.IsType(&sql.GitProvider{})).
		Return(&sql.GitProvider{}, nil)

	// Initialize gRPC server
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}

	// Get the gRPC client
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	_, err = client.SaveGitProvider(context.Background(), providedGitProvider)

	// Assert
	assert.True(t, err == nil)
}

// Test AddRepo
func TestAddRepo(t *testing.T) {

	// Arrange
	gitMaterial := &pb.GitMaterial{
		Id: 1,
	}
	gitMaterialList := []*pb.GitMaterial{gitMaterial}
	addRepoRequest := &pb.AddRepoRequest{
		GitMaterialList: gitMaterialList,
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("AddRepo", mock.IsType([]*sql.GitMaterial{})).
		Return([]*sql.GitMaterial{}, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	_, err = client.AddRepo(context.Background(), addRepoRequest)

	// Assert
	assert.True(t, err == nil)
}

// Test UpdateRepo
func TestUpdateRepo(t *testing.T) {

	// Arrange
	gitMaterial := &pb.GitMaterial{
		Id: 1,
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("UpdateRepo", mock.IsType(&sql.GitMaterial{})).
		Return(&sql.GitMaterial{}, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	_, err = client.UpdateRepo(context.Background(), gitMaterial)

	// Assert
	assert.True(t, err == nil)
}

// Test FetchChanges
func TestFetchChanges(t *testing.T) {

	// Arrange
	req := &pb.FetchScmChangesRequest{
		Count:              1,
		From:               "from",
		To:                 "to",
		PipelineMaterialId: 1,
	}

	lastFetchTime := time.Now()
	lastFetchTimePb := timestamppb.New(lastFetchTime)

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

	commits := make([]*git.GitCommit, 0)
	err := json.Unmarshal([]byte(gitCommits), &commits)

	if err != nil {
		t.Error("failed to deserialize git commits data")
		t.FailNow()
	}

	materialChangeRes := git.MaterialChangeResp{
		LastFetchTime: lastFetchTime,
		Commits:       commits,
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("FetchChanges", int(req.PipelineMaterialId), req.From, req.To, int(req.Count)).
		Return(&materialChangeRes, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.FetchChanges(context.Background(), req)

	// Assert
	assert.Equal(t, res.LastFetchTime, lastFetchTimePb)

	// Assert if FileStats of go-git and pb package are same
	serializedFileStats, _ := json.Marshal(commits[0].FileStats)
	serializedFileStatsPb, _ := json.Marshal(res.Commits[0].FileStats)

	assert.Equal(t, string(serializedFileStats), string(serializedFileStatsPb))
}

// Test GetHeadForPipelineMaterials
func TestGetHeadForPipelineMaterials(t *testing.T) {

	// Arrange
	req := &pb.HeadRequest{
		MaterialIds: []int64{1, 2, 3},
	}

	materials := []*git.CiPipelineMaterialBean{{
		Id: 1,
	}}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetHeadForPipelineMaterials", mock.IsType([]int{})).
		Return(materials, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetHeadForPipelineMaterials(context.Background(), req)

	// Assert
	assert.Equal(t, len(res.Materials), 1)
}

// Test GetCommitMetadata
func TestGetCommitMetadata(t *testing.T) {

	// Arrange
	req := &pb.CommitMetadataRequest{
		PipelineMaterialId: 1,
		GitHash:            "hash",
	}

	gitCommitString := "{\"commit\":\"xyz\",\"author\":\"devtron\"," +
		"\"date\":\"2021-02-18T21:54:42.123Z\",\"message\":\"message\"," +
		"\"changes\":[\"1\",\"2\",\"3\"],\"fileStats\":[{\"name\":\"z\",\"addition\":50," +
		"\"deletion\":10},{\"name\":\"z\",\"addition\":10,\"deletion\":40}]," +
		"\"webhookData\":{\"id\":1,\"eventActionType\":\"action\"," +
		"\"data\":{\"a\":\"b\",\"c\":\"d\"}}}"

	var gitCommit git.GitCommit
	err := json.Unmarshal([]byte(gitCommitString), &gitCommit)

	if err != nil {
		t.Error("failed to deserialize git commits data")
		t.FailNow()
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetCommitMetadata", int(req.PipelineMaterialId), req.GitHash).
		Return(&gitCommit, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetCommitMetadata(context.Background(), req)

	// Assert if FileStats of go-git and pb package are same
	serializedFileStats, _ := json.Marshal(gitCommit.FileStats)
	serializedFileStatsPb, _ := json.Marshal(res.FileStats)

	assert.Equal(t, string(serializedFileStats), string(serializedFileStatsPb))
}

// Test GetCommitMetadataForPipelineMaterial
func TestGetCommitMetadataForPipelineMaterial(t *testing.T) {

	// Arrange
	req := &pb.CommitMetadataRequest{
		PipelineMaterialId: 1,
		GitHash:            "hash",
	}

	gitCommitString := "{\"commit\":\"xyz\",\"author\":\"devtron\"," +
		"\"date\":\"2021-02-18T21:54:42.123Z\",\"message\":\"message\"," +
		"\"changes\":[\"1\",\"2\",\"3\"],\"fileStats\":[{\"name\":\"z\",\"addition\":50," +
		"\"deletion\":10},{\"name\":\"z\",\"addition\":10,\"deletion\":40}]," +
		"\"webhookData\":{\"id\":1,\"eventActionType\":\"action\"," +
		"\"data\":{\"a\":\"b\",\"c\":\"d\"}}}"

	var gitCommit git.GitCommit
	err := json.Unmarshal([]byte(gitCommitString), &gitCommit)

	if err != nil {
		t.Error("failed to deserialize git commits data")
		t.FailNow()
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetCommitMetadataForPipelineMaterial", int(req.PipelineMaterialId), req.GitHash).
		Return(&gitCommit, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetCommitMetadataForPipelineMaterial(context.Background(), req)

	// Assert if FileStats of go-git and pb package are same
	serializedFileStats, _ := json.Marshal(gitCommit.FileStats)
	serializedFileStatsPb, _ := json.Marshal(res.FileStats)

	assert.Equal(t, string(serializedFileStats), string(serializedFileStatsPb))
}

// Test GetCommitInfoForTag
func TestGetCommitInfoForTag(t *testing.T) {

	// Arrange
	req := &pb.CommitMetadataRequest{
		PipelineMaterialId: 1,
		GitTag:             "v0.1",
	}

	gitCommitString := "{\"commit\":\"xyz\",\"author\":\"devtron\"," +
		"\"date\":\"2021-02-18T21:54:42.123Z\",\"message\":\"message\"," +
		"\"changes\":[\"1\",\"2\",\"3\"],\"fileStats\":[{\"name\":\"z\",\"addition\":50," +
		"\"deletion\":10},{\"name\":\"z\",\"addition\":10,\"deletion\":40}]," +
		"\"webhookData\":{\"id\":1,\"eventActionType\":\"action\"," +
		"\"data\":{\"a\":\"b\",\"c\":\"d\"}}}"

	var gitCommit git.GitCommit
	err := json.Unmarshal([]byte(gitCommitString), &gitCommit)

	if err != nil {
		t.Error("failed to deserialize git commits data")
		t.FailNow()
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetCommitInfoForTag", mock.IsType(&git.CommitMetadataRequest{})).
		Return(&gitCommit, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetCommitInfoForTag(context.Background(), req)

	// Assert if FileStats of go-git and pb package are same
	serializedFileStats, _ := json.Marshal(gitCommit.FileStats)
	serializedFileStatsPb, _ := json.Marshal(res.FileStats)

	assert.Equal(t, string(serializedFileStats), string(serializedFileStatsPb))
}

// Test RefreshGitMaterial
func TestRefreshGitMaterial(t *testing.T) {

	// Arrange
	req := &pb.RefreshGitMaterialRequest{
		GitMaterialId: 1,
	}

	lastFetchedTime := time.Now()
	lastFetchedTimePb := timestamppb.New(lastFetchedTime)

	refreshGitMaterialResp := &git.RefreshGitMaterialResponse{
		Message:       "msg",
		LastFetchTime: lastFetchedTime,
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("RefreshGitMaterial", mock.IsType(&git.RefreshGitMaterialRequest{})).
		Return(refreshGitMaterialResp, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.RefreshGitMaterial(context.Background(), req)

	// Assert
	assert.Equal(t, lastFetchedTimePb, res.LastFetchTime)
}

// Test ReloadAllMaterial
func TestReloadAllMaterial(t *testing.T) {

	// Arrange
	req := &pb.Empty{}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("ReloadAllRepo").
		Return()

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	_, err = client.ReloadAllMaterial(context.Background(), req)

	// Assert
	assert.Equal(t, nil, err)
}

// Test ReloadMaterial
func TestReloadMaterial(t *testing.T) {

	// Arrange
	req := &pb.ReloadMaterialRequest{
		MaterialId: 1,
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("ResetRepo", 1).
		Return(nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.ReloadMaterial(context.Background(), req)

	// Assert
	assert.Equal(t, res.Message, "reloaded")
}

// Test GetChangesInRelease
func TestGetChangesInRelease(t *testing.T) {

	// Arrange
	req := &pb.ReleaseChangeRequest{
		PipelineMaterialId: 1,
		OldCommit:          "old",
		NewCommit:          "new",
	}

	gitChanges := &git.GitChanges{
		Commits: []*git.Commit{{
			Subject: "test",
		}},
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetReleaseChanges", mock.IsType(&pkg.ReleaseChangesRequest{})).
		Return(gitChanges, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetChangesInRelease(context.Background(), req)

	// Assert
	assert.Equal(t, res.Commits[0].Subject, "test")
}

// Test GetWebhookData
func TestGetWebhookData(t *testing.T) {

	// Arrange
	req := &pb.WebhookDataRequest{
		Id:                   1,
		CiPipelineMaterialId: 2,
	}

	webhookAndCiData := &git.WebhookAndCiData{
		WebhookData: &git.WebhookData{
			Id: 5,
		},
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetWebhookAndCiDataById", int(req.Id), int(req.CiPipelineMaterialId)).
		Return(webhookAndCiData, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetWebhookData(context.Background(), req)

	// Assert
	assert.Equal(t, int(res.WebhookData.Id), 5)
}

// Test GetAllWebhookEventConfigForHost
func TestGetAllWebhookEventConfigForHosta(t *testing.T) {

	// Arrange
	req := &pb.WebhookEventConfigRequest{
		GitHostId: 1,
	}

	webhookEventConfig := []*git.WebhookEventConfig{{
		GitHostId: 1,
	}}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetAllWebhookEventConfigForHost", int(req.GitHostId)).
		Return(webhookEventConfig, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetAllWebhookEventConfigForHost(context.Background(), req)

	// Assert
	assert.Equal(t, int(res.WebhookEventConfig[0].GitHostId), 1)
}

// Test GetWebhookEventConfig
func TestGetWebhookEventConfig(t *testing.T) {

	// Arrange
	req := &pb.WebhookEventConfigRequest{
		EventId: 1,
	}

	webhookEventConfig := &git.WebhookEventConfig{
		GitHostId: 1,
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetWebhookEventConfig", int(req.EventId)).
		Return(webhookEventConfig, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetWebhookEventConfig(context.Background(), req)

	// Assert
	assert.Equal(t, int(res.GitHostId), 1)
}

// Test GetWebhookPayloadDataForPipelineMaterialId
func TestGetWebhookPayloadDataForPipelineMaterialId(t *testing.T) {

	// Arrange
	req := &pb.WebhookPayloadDataRequest{
		CiPipelineMaterialId: 1,
	}

	payloadData := &git.WebhookPayloadDataResponse{
		RepositoryUrl: "localhost:1234/repo",
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetWebhookPayloadDataForPipelineMaterialId", mock.IsType(&git.WebhookPayloadDataRequest{})).
		Return(payloadData, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetWebhookPayloadDataForPipelineMaterialId(context.Background(), req)

	// Assert
	assert.Equal(t, res.RepositoryUrl, "localhost:1234/repo")
}

// Test GetWebhookPayloadFilterDataForPipelineMaterialId
func TestGetWebhookPayloadFilterDataForPipelineMaterialId(t *testing.T) {

	// Arrange
	req := &pb.WebhookPayloadFilterDataRequest{}

	payloadRes := &git.WebhookPayloadFilterDataResponse{
		PayloadId: 1,
	}

	// Mocking
	repositoryManager := mocks.NewRepoManager(t)
	repositoryManager.On("GetWebhookPayloadFilterDataForPipelineMaterialId", mock.IsType(&git.WebhookPayloadFilterDataRequest{})).
		Return(payloadRes, nil)

	// Initializing gRPC server and client connection
	conn, err := initServer(t, repositoryManager)
	if err != nil {
		t.FailNow()
	}
	client := pb.NewGitSensorServiceClient(conn)

	// Action
	res, err := client.GetWebhookPayloadFilterDataForPipelineMaterialId(context.Background(), req)

	// Assert
	assert.Equal(t, 1, int(res.PayloadId))
}
