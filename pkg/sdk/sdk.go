package sdk

import (
	"context"
	"crypto/ecdsa"
	"net/http"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	corev1connect "github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	ethv1connect "github.com/AudiusProject/audiusd/pkg/api/eth/v1/v1connect"
	etlv1connect "github.com/AudiusProject/audiusd/pkg/api/etl/v1/v1connect"
	storagev1connect "github.com/AudiusProject/audiusd/pkg/api/storage/v1/v1connect"
	systemv1connect "github.com/AudiusProject/audiusd/pkg/api/system/v1/v1connect"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/sdk/mediorum"
	"github.com/AudiusProject/audiusd/pkg/sdk/rewards"
)

type AudiusdSDK struct {
	privKey *ecdsa.PrivateKey
	chainID string

	Core    corev1connect.CoreServiceClient
	Storage storagev1connect.StorageServiceClient
	ETL     etlv1connect.ETLServiceClient
	System  systemv1connect.SystemServiceClient
	Eth     ethv1connect.EthServiceClient

	// helper instances
	Rewards  *rewards.Rewards
	Mediorum *mediorum.Mediorum
}

func NewAudiusdSDK(nodeURL string) *AudiusdSDK {
	httpClient := http.DefaultClient
	url := common.EnsureURLProtocol(nodeURL)

	coreClient := corev1connect.NewCoreServiceClient(httpClient, url)
	storageClient := storagev1connect.NewStorageServiceClient(httpClient, url)
	etlClient := etlv1connect.NewETLServiceClient(httpClient, url)
	systemClient := systemv1connect.NewSystemServiceClient(httpClient, url)
	ethClient := ethv1connect.NewEthServiceClient(httpClient, url)
	mediorumClient := mediorum.NewWithCore(url, coreClient)
	rewardsClient := rewards.NewRewards(coreClient)

	sdk := &AudiusdSDK{
		Core:     coreClient,
		Storage:  storageClient,
		ETL:      etlClient,
		System:   systemClient,
		Eth:      ethClient,
		Mediorum: mediorumClient,
		Rewards:  rewardsClient,
	}

	return sdk
}

func (s *AudiusdSDK) Init(ctx context.Context) error {
	nodeInfoResp, err := s.Core.GetNodeInfo(ctx, connect.NewRequest(&corev1.GetNodeInfoRequest{}))
	if err != nil {
		return err
	}

	s.chainID = nodeInfoResp.Msg.Chainid
	return nil
}

func (s *AudiusdSDK) ChainID() string {
	return s.chainID
}
