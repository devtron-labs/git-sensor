package api

import (
	"context"
	"github.com/devtron-labs/git-sensor/internal/sql"
	"github.com/devtron-labs/git-sensor/pkg"
	"github.com/devtron-labs/git-sensor/pkg/git"
	pb "github.com/devtron-labs/protos/gitSensor"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcController interface {
	SaveGitProvider(ctx context.Context, req *pb.GitProvider) (*pb.Empty, error)
	AddRepo(ctx context.Context, req *pb.AddRepoRequest) (*pb.Empty, error)
	UpdateRepo(ctx context.Context, req *pb.GitMaterial) (*pb.Empty, error)
	SavePipelineMaterial(ctx context.Context, req *pb.SavePipelineMaterialRequest) (*pb.Empty, error)

	// ----
	FetchChanges(ctx context.Context, req *pb.FetchScmChangesRequest) (*pb.MaterialChangeResponse, error)
	GetHeadForPipelineMaterials(ctx context.Context, req *pb.HeadRequest) (*pb.GetHeadForPipelineMaterialsResponse, error)
	GetCommitMetadata(ctx context.Context, req *pb.CommitMetadataRequest) (*pb.GitCommit, error)
	GetCommitMetadataForPipelineMaterial(ctx context.Context, req *pb.CommitMetadataRequest) (*pb.GitCommit, error)
	GetCommitInfoForTag(ctx context.Context, req *pb.CommitMetadataRequest) (*pb.GitCommit, error)
	RefreshGitMaterial(ctx context.Context, req *pb.RefreshGitMaterialRequest) (*pb.RefreshGitMaterialResponse, error)
	ReloadAllMaterial(ctx context.Context, req *pb.Empty) (*pb.Empty, error)
	ReloadMaterial(ctx context.Context, req *pb.ReloadMaterialRequest) (*pb.GenericResponse, error)
	GetChangesInRelease(ctx context.Context, req *pb.ReleaseChangeRequest) (*pb.GitChanges, error)
	GetWebhookData(ctx context.Context, req *pb.WebhookDataRequest) (*pb.WebhookAndCiData, error)
	GetAllWebhookEventConfigForHost(ctx context.Context, req *pb.WebhookEventConfigRequest) (*pb.WebhookEventConfigResponse, error)
	GetWebhookEventConfig(ctx context.Context, req *pb.WebhookEventConfigRequest) (*pb.WebhookEventConfig, error)
	GetWebhookPayloadDataForPipelineMaterialId(ctx context.Context, req *pb.WebhookPayloadDataRequest) (*pb.WebhookPayloadDataResponse, error)
	GetWebhookPayloadFilterDataForPipelineMaterialId(ctx context.Context, req *pb.WebhookPayloadFilterDataRequest) (*pb.WebhookPayloadFilterDataResponse, error)
}

type GrpcControllerImpl struct {
	pb.GitSensorServiceServer
	logger            *zap.SugaredLogger
	repositoryManager pkg.RepoManager
}

func NewGrpcControllerImpl(
	repositoryManager pkg.RepoManager, logger *zap.SugaredLogger) *GrpcControllerImpl {

	return &GrpcControllerImpl{
		repositoryManager: repositoryManager,
		logger:            logger,
	}
}

// SaveGitProvider saves the GitProvider details
func (controller *GrpcControllerImpl) SaveGitProvider(ctx context.Context, req *pb.GitProvider) (
	*pb.Empty, error) {

	// mapping req
	gitProvider := &sql.GitProvider{
		Id:            int(req.Id),
		Name:          req.Name,
		Url:           req.Url,
		UserName:      req.UserName,
		Password:      req.Password,
		AccessToken:   req.AccessToken,
		SshPrivateKey: req.SshPrivateKey,
		AuthMode:      sql.AuthMode(req.AuthMode),
		Active:        req.Active,
	}

	_, err := controller.repositoryManager.SaveGitProvider(gitProvider)
	if err != nil {
		controller.logger.Errorw("error while saving git provider",
			"authMode", gitProvider.AuthMode,
			"err", err)

		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Empty{}, nil
}

// AddRepo save git materials
func (controller *GrpcControllerImpl) AddRepo(ctx context.Context, req *pb.AddRepoRequest) (
	*pb.Empty, error) {

	// Mapping to sql package specified struct type
	gitMaterials := make([]*sql.GitMaterial, 0)
	for _, item := range req.GitMaterialList {

		gitMaterials = append(gitMaterials, &sql.GitMaterial{
			Id:               int(item.Id),
			GitProviderId:    int(item.GitProviderId),
			Url:              item.Url,
			FetchSubmodules:  item.FetchSubmodules,
			Name:             item.Name,
			CheckoutLocation: item.CheckoutLocation,
			CheckoutStatus:   item.CheckoutStatus,
			CheckoutMsgAny:   item.CheckoutMsgAny,
			Deleted:          item.Deleted,
		})
	}

	_, err := controller.repositoryManager.AddRepo(gitMaterials)
	if err != nil {
		controller.logger.Errorw("error while adding repo",
			"err", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Empty{}, nil
}

// UpdateRepo updates GitMaterial
func (controller *GrpcControllerImpl) UpdateRepo(ctx context.Context, req *pb.GitMaterial) (
	*pb.Empty, error) {

	// Mapping
	mappedGitMaterial := &sql.GitMaterial{
		Id:               int(req.Id),
		GitProviderId:    int(req.GitProviderId),
		Url:              req.Url,
		FetchSubmodules:  req.FetchSubmodules,
		Name:             req.Name,
		CheckoutLocation: req.CheckoutLocation,
		CheckoutStatus:   req.CheckoutStatus,
		CheckoutMsgAny:   req.CheckoutMsgAny,
		Deleted:          req.Deleted,
	}

	// Update repo
	_, err := controller.repositoryManager.UpdateRepo(mappedGitMaterial)
	if err != nil {
		controller.logger.Errorw("error while updating repo",
			"name", mappedGitMaterial.Name,
			"err", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Empty{}, nil
}

// SavePipelineMaterial saves pipeline material
func (controller *GrpcControllerImpl) SavePipelineMaterial(ctx context.Context, req *pb.SavePipelineMaterialRequest) (
	*pb.Empty, error) {

	// Mapping to sql package specified struct type
	ciPipelineMaterials := make([]*sql.CiPipelineMaterial, 0)
	for _, item := range req.CiPipelineMaterials {

		ciPipelineMaterials = append(ciPipelineMaterials, &sql.CiPipelineMaterial{
			Id:            int(item.Id),
			GitMaterialId: int(item.GitMaterialId),
			Type:          sql.SourceType(item.Type),
			Value:         item.Value,
			Active:        item.Active,
			LastSeenHash:  item.LastSeenHash,
			CommitAuthor:  item.CommitAuthor,
			CommitDate:    item.CommitDate.AsTime(),
			CommitHistory: item.CommitHistory,
			Errored:       item.Errored,
			ErrorMsg:      item.ErrorMsg,
		})
	}

	// TODO: Check if we can change the argument type for the below method to avoid mapping
	_, err := controller.repositoryManager.SavePipelineMaterial(ciPipelineMaterials)
	if err != nil {
		controller.logger.Errorw("error while adding repo",
			"err", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Empty{}, nil
}

// FetchChanges
func (controller *GrpcControllerImpl) FetchChanges(ctx context.Context, req *pb.FetchScmChangesRequest) (
	*pb.MaterialChangeResponse, error) {

	res, err := controller.repositoryManager.FetchChanges(int(req.PipelineMaterialId), req.From, req.To, int(req.Count))
	if err != nil {
		controller.logger.Errorw("error while fetching scm changes",
			"pipelineMaterialId", req.PipelineMaterialId,
			"err", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	if res == nil {
		return nil, nil
	}

	// Mapping GitCommit
	pbGitCommits := make([]*pb.GitCommit, 0)
	for _, item := range res.Commits {

		mappedCommit, err := controller.mapGitCommit(item)
		if err != nil {
			continue
		}
		pbGitCommits = append(pbGitCommits, mappedCommit)
	}

	return &pb.MaterialChangeResponse{
		Commits:        pbGitCommits,
		LastFetchTime:  timestamppb.New(res.LastFetchTime),
		IsRepoError:    res.IsRepoError,
		RepoErrorMsg:   res.RepoErrorMsg,
		IsBranchError:  res.IsBranchError,
		BranchErrorMsg: res.BranchErrorMsg,
	}, nil
}

func (controller *GrpcControllerImpl) GetHeadForPipelineMaterials(ctx context.Context, req *pb.HeadRequest) (
	*pb.GetHeadForPipelineMaterialsResponse, error) {

	// Map int64 to int
	materialIds := make([]int, 0)
	for _, id := range req.MaterialIds {
		materialIds = append(materialIds, int(id))
	}

	// Fetch
	res, err := controller.repositoryManager.GetHeadForPipelineMaterials(materialIds)
	if err != nil {
		controller.logger.Errorw("error while fetching head for pipeline materials",
			"err", err)
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// Mapping to pb type
	ciPipelineMaterialBeans := make([]*pb.CiPipelineMaterialBean, 0)
	for _, item := range res {

		var mappedGitCommit *pb.GitCommit
		if item.GitCommit != nil {
			mappedGitCommit, _ = controller.mapGitCommit(item.GitCommit)
			if err != nil {
				continue
			}
		}

		ciPipelineMaterialBeans = append(ciPipelineMaterialBeans, &pb.CiPipelineMaterialBean{
			Id:                        int64(item.Id),
			GitMaterialId:             int64(item.GitMaterialId),
			Type:                      string(item.Type),
			Value:                     item.Value,
			Active:                    item.Active,
			GitCommit:                 mappedGitCommit,
			ExtraEnvironmentVariables: item.ExtraEnvironmentVariables,
		})
	}
	return &pb.GetHeadForPipelineMaterialsResponse{
		Materials: ciPipelineMaterialBeans,
	}, nil
}

func (controller *GrpcControllerImpl) GetCommitMetadata(ctx context.Context, req *pb.CommitMetadataRequest) (
	*pb.GitCommit, error) {

	// Mapping req body
	mappedReq := &git.CommitMetadataRequest{
		PipelineMaterialId: int(req.PipelineMaterialId),
		GitHash:            req.GitHash,
		GitTag:             req.GitTag,
		BranchName:         req.BranchName,
	}

	var gitCommit *git.GitCommit
	var err error

	if len(req.GitTag) > 0 {
		gitCommit, err = controller.repositoryManager.GetCommitInfoForTag(mappedReq)

	} else if len(req.BranchName) > 0 {
		gitCommit, err = controller.repositoryManager.GetLatestCommitForBranch(mappedReq.PipelineMaterialId,
			mappedReq.BranchName)

	} else {
		gitCommit, err = controller.repositoryManager.GetCommitMetadata(mappedReq.PipelineMaterialId,
			mappedReq.GitHash)
	}

	if err != nil {
		controller.logger.Errorw("error while fetching commit metadata",
			"pipelineMaterialId", req.PipelineMaterialId,
			"err", err)

		return nil, err
	}
	if gitCommit == nil {
		return nil, nil
	}

	// Mapping GitCommit
	mappedGitCommit, err := controller.mapGitCommit(gitCommit)
	if err != nil {
		controller.logger.Errorw("error mapping git commit",
			"pipelineMaterialId", req.PipelineMaterialId,
			"err", err)

		return nil, err
	}
	return mappedGitCommit, nil
}

func (controller *GrpcControllerImpl) GetCommitMetadataForPipelineMaterial(ctx context.Context, req *pb.CommitMetadataRequest) (
	*pb.GitCommit, error) {

	// Mapping req body
	mappedReq := &git.CommitMetadataRequest{
		PipelineMaterialId: int(req.PipelineMaterialId),
		GitHash:            req.GitHash,
		GitTag:             req.GitTag,
		BranchName:         req.BranchName,
	}

	res, err := controller.repositoryManager.GetCommitMetadataForPipelineMaterial(mappedReq.PipelineMaterialId,
		mappedReq.GitHash)

	if err != nil {
		controller.logger.Errorw("error while fetching commit metadata for pipeline material",
			"pipelineMaterialId", req.PipelineMaterialId,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// Mapping GitCommit
	mappedGitCommit, err := controller.mapGitCommit(res)
	if err != nil {
		controller.logger.Errorw("error mapping git commit",
			"pipelineMaterialId", req.PipelineMaterialId,
			"err", err)

		return nil, err
	}
	return mappedGitCommit, nil
}

func (controller *GrpcControllerImpl) GetCommitInfoForTag(ctx context.Context, req *pb.CommitMetadataRequest) (
	*pb.GitCommit, error) {

	// Mapping req body
	mappedReq := &git.CommitMetadataRequest{
		PipelineMaterialId: int(req.PipelineMaterialId),
		GitHash:            req.GitHash,
		GitTag:             req.GitTag,
		BranchName:         req.BranchName,
	}

	res, err := controller.repositoryManager.GetCommitInfoForTag(mappedReq)

	if err != nil {
		controller.logger.Errorw("error while fetching commit info for tag",
			"pipelineMaterialId", req.PipelineMaterialId,
			"gitTag", mappedReq.GitTag,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// Mapping GitCommit
	mappedGitCommit, err := controller.mapGitCommit(res)
	if err != nil {
		controller.logger.Errorw("error mapping git commit",
			"pipelineMaterialId", req.PipelineMaterialId,
			"err", err)

		return nil, err
	}
	return mappedGitCommit, nil
}

func (controller *GrpcControllerImpl) RefreshGitMaterial(ctx context.Context, req *pb.RefreshGitMaterialRequest) (
	*pb.RefreshGitMaterialResponse, error) {

	// Mapping req body
	mappedRequest := &git.RefreshGitMaterialRequest{
		GitMaterialId: int(req.GitMaterialId),
	}

	res, err := controller.repositoryManager.RefreshGitMaterial(mappedRequest)
	if err != nil {
		controller.logger.Errorw("error while refreshing git material",
			"gitMaterialId", req.GitMaterialId,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// mapping res
	mappedRes := &pb.RefreshGitMaterialResponse{
		Message:  res.Message,
		ErrorMsg: res.ErrorMsg,
	}
	if !res.LastFetchTime.IsZero() {
		mappedRes.LastFetchTime = timestamppb.New(res.LastFetchTime)
	}

	return mappedRes, nil
}

func (controller *GrpcControllerImpl) ReloadAllMaterial(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	controller.repositoryManager.ReloadAllRepo()
	return &pb.Empty{}, nil
}

func (controller *GrpcControllerImpl) ReloadMaterial(ctx context.Context, req *pb.ReloadMaterialRequest) (
	*pb.GenericResponse, error) {

	err := controller.repositoryManager.ResetRepo(int(req.MaterialId))
	if err != nil {
		controller.logger.Errorw("error while reloading material",
			"materialId", req.MaterialId,
			"err", err)

		return nil, err
	}
	return &pb.GenericResponse{
		Message: "reloaded",
	}, nil
}

func (controller *GrpcControllerImpl) GetChangesInRelease(ctx context.Context, req *pb.ReleaseChangeRequest) (
	*pb.GitChanges, error) {

	// Mapping req body
	mappedReq := &pkg.ReleaseChangesRequest{
		PipelineMaterialId: int(req.PipelineMaterialId),
		OldCommit:          req.OldCommit,
		NewCommit:          req.NewCommit,
	}

	res, err := controller.repositoryManager.GetReleaseChanges(mappedReq)
	if err != nil {
		controller.logger.Errorw("error while fetching release changes",
			"pipelineMaterialId", mappedReq.PipelineMaterialId,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	return controller.mapGitChanges(res), nil
}

func (controller *GrpcControllerImpl) GetWebhookData(ctx context.Context, req *pb.WebhookDataRequest) (
	*pb.WebhookAndCiData, error) {

	res, err := controller.repositoryManager.GetWebhookAndCiDataById(int(req.Id), int(req.CiPipelineMaterialId))
	if err != nil {
		controller.logger.Errorw("error while fetching webhook and ci data",
			"ciPipelineMaterialId", req.CiPipelineMaterialId,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// Mapping response
	mappedResponse := &pb.WebhookAndCiData{}
	mappedResponse.ExtraEnvironmentVariables = res.ExtraEnvironmentVariables
	if res.WebhookData != nil {
		mappedResponse.WebhookData = &pb.WebhookData{
			Id:              int64(res.WebhookData.Id),
			EventActionType: res.WebhookData.EventActionType,
			Data:            res.WebhookData.Data,
		}
	}
	return mappedResponse, nil
}

func (controller *GrpcControllerImpl) GetAllWebhookEventConfigForHost(ctx context.Context, req *pb.WebhookEventConfigRequest) (
	*pb.WebhookEventConfigResponse, error) {

	res, err := controller.repositoryManager.GetAllWebhookEventConfigForHost(int(req.GitHostId))
	if err != nil {
		controller.logger.Errorw("error while fetching webhook event config",
			"gitHostId", req.GitHostId,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// Mapping response
	mappedEventConfig := make([]*pb.WebhookEventConfig, 0)
	for _, item := range res {
		mappedEventConfig = append(mappedEventConfig, controller.mapWebhookEventConfig(item))
	}

	return &pb.WebhookEventConfigResponse{
		WebhookEventConfig: mappedEventConfig,
	}, nil
}

func (controller *GrpcControllerImpl) GetWebhookEventConfig(ctx context.Context, req *pb.WebhookEventConfigRequest) (
	*pb.WebhookEventConfig, error) {

	res, err := controller.repositoryManager.GetWebhookEventConfig(int(req.EventId))
	if err != nil {
		controller.logger.Errorw("error while fetching webhook event config",
			"eventId", req.EventId,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return controller.mapWebhookEventConfig(res), nil
}

func (controller *GrpcControllerImpl) GetWebhookPayloadDataForPipelineMaterialId(ctx context.Context,
	req *pb.WebhookPayloadDataRequest) (*pb.WebhookPayloadDataResponse, error) {

	mappedReq := &git.WebhookPayloadDataRequest{
		CiPipelineMaterialId: int(req.CiPipelineMaterialId),
		Limit:                int(req.Limit),
		Offset:               int(req.Offset),
		EventTimeSortOrder:   req.EventTimeSortOrder,
	}

	res, err := controller.repositoryManager.GetWebhookPayloadDataForPipelineMaterialId(mappedReq)
	if err != nil {
		controller.logger.Errorw("error while fetching webhook payload data for pipeline material id",
			"ciPipelineMaterialId", mappedReq.CiPipelineMaterialId,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// Mapping payloads
	payloads := make([]*pb.WebhookPayload, 0)
	for _, item := range res.Payloads {

		payload := &pb.WebhookPayload{
			ParsedDataId:        int64(item.ParsedDataId),
			MatchedFiltersCount: int64(item.MatchedFiltersCount),
			FailedFiltersCount:  int64(item.FailedFiltersCount),
			MatchedFilters:      item.MatchedFilters,
		}
		if !item.EventTime.IsZero() {
			payload.EventTime = timestamppb.New(item.EventTime)
		}
		payloads = append(payloads, payload)
	}

	return &pb.WebhookPayloadDataResponse{
		Filters:       res.Filters,
		RepositoryUrl: res.RepositoryUrl,
		Payloads:      payloads,
	}, nil
}

func (controller *GrpcControllerImpl) GetWebhookPayloadFilterDataForPipelineMaterialId(ctx context.Context,
	req *pb.WebhookPayloadFilterDataRequest) (*pb.WebhookPayloadFilterDataResponse, error) {

	// Mapping request
	mappedReq := &git.WebhookPayloadFilterDataRequest{
		ParsedDataId:         int(req.ParsedDataId),
		CiPipelineMaterialId: int(req.CiPipelineMaterialId),
	}

	res, err := controller.repositoryManager.GetWebhookPayloadFilterDataForPipelineMaterialId(mappedReq)
	if err != nil {
		controller.logger.Errorw("error while fetching webhook payload data for pipeline material id with filter",
			"ciPipelineMaterialId", mappedReq.CiPipelineMaterialId,
			"err", err)

		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	// Mapping response
	selectorsData := make([]*pb.WebhookPayloadFilterDataSelectorResponse, 0)
	for _, item := range res.SelectorsData {

		selectorsData = append(selectorsData, &pb.WebhookPayloadFilterDataSelectorResponse{
			SelectorName:      item.SelectorName,
			SelectorCondition: item.SelectorCondition,
			SelectorValue:     item.SelectorValue,
			Match:             item.Match,
		})
	}

	return &pb.WebhookPayloadFilterDataResponse{
		PayloadId:     int64(res.PayloadId),
		SelectorsData: selectorsData,
	}, nil
}

func (controller *GrpcControllerImpl) mapWebhookEventConfig(config *git.WebhookEventConfig) *pb.WebhookEventConfig {

	selectors := make([]*pb.WebhookEventSelectors, 0)
	for _, item := range config.Selectors {

		selector := &pb.WebhookEventSelectors{
			Id:               int64(item.Id),
			EventId:          int64(item.EventId),
			Name:             item.Name,
			ToShow:           item.ToShow,
			ToShowInCiFilter: item.ToShowInCiFilter,
			FixValue:         item.FixValue,
			PossibleValues:   item.PossibleValues,
			IsActive:         item.IsActive,
		}
		if !item.CreatedOn.IsZero() {
			selector.CreatedOn = timestamppb.New(item.CreatedOn)
		}
		if !item.UpdatedOn.IsZero() {
			selector.UpdatedOn = timestamppb.New(item.UpdatedOn)
		}
		selectors = append(selectors, selector)
	}

	mappedConfig := &pb.WebhookEventConfig{
		Id:            int64(config.Id),
		GitHostId:     int64(config.GitHostId),
		Name:          config.Name,
		EventTypesCsv: config.EventTypesCsv,
		ActionType:    config.ActionType,
		IsActive:      config.IsActive,
		Selectors:     selectors,
	}
	if !config.CreatedOn.IsZero() {
		mappedConfig.CreatedOn = timestamppb.New(config.CreatedOn)
	}
	if !config.UpdatedOn.IsZero() {
		mappedConfig.UpdatedOn = timestamppb.New(config.UpdatedOn)
	}
	return mappedConfig
}

func (controller *GrpcControllerImpl) mapGitChanges(gitChanges *git.GitChanges) *pb.GitChanges {

	// Mapping Commits
	commitsPb := make([]*pb.Commit, 0)
	for _, item := range gitChanges.Commits {

		commitPb := &pb.Commit{}

		// Map Hash
		if item.Hash != nil {
			commitPb.Hash = &pb.Hash{
				Long:  item.Hash.Long,
				Short: item.Hash.Short,
			}
		}

		// Map Tree
		if item.Tree != nil {
			commitPb.Tree = &pb.Tree{
				Long:  item.Tree.Long,
				Short: item.Tree.Short,
			}
		}

		// Map Author
		if item.Author != nil {
			commitPb.Author = &pb.Author{
				Name:  item.Author.Name,
				Email: item.Author.Email,
			}
			if !item.Author.Date.IsZero() {
				commitPb.Author.Date = timestamppb.New(item.Author.Date)
			}
		}

		// Map Committer
		if item.Committer != nil {
			commitPb.Committer = &pb.Committer{
				Name:  item.Committer.Name,
				Email: item.Committer.Email,
			}
			if !item.Committer.Date.IsZero() {
				commitPb.Committer.Date = timestamppb.New(item.Committer.Date)
			}
		}

		// Map Tag
		if item.Tag != nil {
			commitPb.Tag = &pb.Tag{
				Name: item.Tag.Name,
			}
			if !item.Tag.Date.IsZero() {
				commitPb.Tag.Date = timestamppb.New(item.Tag.Date)
			}
		}

		commitPb.Subject = item.Subject
		commitPb.Body = item.Body

		commitsPb = append(commitsPb, commitPb)
	}

	// Mapping FileStats
	mappedFileStats := make([]*pb.FileStat, 0)
	for _, item := range gitChanges.FileStats {

		mappedFileStats = append(mappedFileStats, &pb.FileStat{
			Name:     item.Name,
			Addition: int64(item.Addition),
			Deletion: int64(item.Deletion),
		})
	}

	return &pb.GitChanges{
		Commits:   commitsPb,
		FileStats: mappedFileStats,
	}
}

func (controller *GrpcControllerImpl) mapGitCommit(commit *git.GitCommit) (*pb.GitCommit, error) {

	// mapping FileStats
	mappedFileStats := make([]*pb.FileStat, 0)
	if commit.FileStats != nil {
		for _, item := range *commit.FileStats {

			mappedFileStats = append(mappedFileStats, &pb.FileStat{
				Name:     item.Name,
				Addition: int64(item.Addition),
				Deletion: int64(item.Deletion),
			})
		}
	}

	// Mapping GitCommit
	mappedRes := &pb.GitCommit{
		Commit:    commit.Commit,
		Author:    commit.Author,
		Message:   commit.Message,
		Changes:   commit.Changes,
		FileStats: mappedFileStats,
	}

	if commit.WebhookData != nil {
		mappedRes.WebhookData = &pb.WebhookData{
			Id:              int64(commit.WebhookData.Id),
			EventActionType: commit.WebhookData.EventActionType,
			Data:            commit.WebhookData.Data,
		}
	}
	if !commit.Date.IsZero() {
		mappedRes.Date = timestamppb.New(commit.Date)
	}
	return mappedRes, nil
}
