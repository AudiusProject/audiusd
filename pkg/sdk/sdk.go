package sdk

import (
	"net/http"

	corev1connect "github.com/AudiusProject/audiusd/pkg/api/core/v1/v1connect"
	etlv1connect "github.com/AudiusProject/audiusd/pkg/api/etl/v1/v1connect"
	storagev1connect "github.com/AudiusProject/audiusd/pkg/api/storage/v1/v1connect"
	systemv1connect "github.com/AudiusProject/audiusd/pkg/api/system/v1/v1connect"
)

type AudiusdSDK struct {
	Core    corev1connect.CoreServiceClient
	Storage storagev1connect.StorageServiceClient
	ETL     etlv1connect.ETLServiceClient
	System  systemv1connect.SystemServiceClient
}

func NewAudiusdSDK(nodeURL string) *AudiusdSDK {
	httpClient := http.DefaultClient
	sdk := &AudiusdSDK{
		Core:    corev1connect.NewCoreServiceClient(httpClient, nodeURL),
		Storage: storagev1connect.NewStorageServiceClient(httpClient, nodeURL),
		ETL:     etlv1connect.NewETLServiceClient(httpClient, nodeURL),
		System:  systemv1connect.NewSystemServiceClient(httpClient, nodeURL),
	}

	return sdk
}
