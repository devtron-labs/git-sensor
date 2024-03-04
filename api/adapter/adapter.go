package adapter

import (
	"github.com/devtron-labs/git-sensor/internals/sql"
	pb "github.com/devtron-labs/protos/gitSensor"
)

func GenerateMaterialRequest(updatedGitMaterial *sql.GitMaterial) *pb.GitMaterial {
	return &pb.GitMaterial{
		Id:               int64(updatedGitMaterial.Id),
		GitProviderId:    int64(updatedGitMaterial.GitProviderId),
		Url:              updatedGitMaterial.Url,
		FetchSubmodules:  updatedGitMaterial.FetchSubmodules,
		Name:             updatedGitMaterial.Name,
		CheckoutLocation: updatedGitMaterial.CheckoutLocation,
		CheckoutStatus:   updatedGitMaterial.CheckoutStatus,
		CheckoutMsgAny:   updatedGitMaterial.CheckoutMsgAny,
		Deleted:          updatedGitMaterial.Deleted,
		FilterPattern:    updatedGitMaterial.FilterPattern,
	}
}

func ConvertToMaterialModel(req *pb.GitMaterial) *sql.GitMaterial {
	return &sql.GitMaterial{
		Id:               int(req.Id),
		GitProviderId:    int(req.GitProviderId),
		Url:              req.Url,
		FetchSubmodules:  req.FetchSubmodules,
		Name:             req.Name,
		CheckoutLocation: req.CheckoutLocation,
		CheckoutStatus:   req.CheckoutStatus,
		CheckoutMsgAny:   req.CheckoutMsgAny,
		Deleted:          req.Deleted,
		FilterPattern:    req.FilterPattern,
	}
}
