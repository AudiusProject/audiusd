package utils

import (
	"os"

	"github.com/AudiusProject/audiusd/pkg/sdk"
)

var (
	DiscoveryOneRPC = getEnvWithDefault("discoveryOneRPC", "https://node1.audiusd.devnet")
	ContentOneRPC   = getEnvWithDefault("contentOneRPC", "https://node2.audiusd.devnet")
	ContentTwoRPC   = getEnvWithDefault("contentTwoRPC", "https://node3.audiusd.devnet")
	ContentThreeRPC = getEnvWithDefault("contentThreeRPC", "https://node4.audiusd.devnet")

	DiscoveryOne *sdk.AudiusdSDK
	ContentOne   *sdk.AudiusdSDK
	ContentTwo   *sdk.AudiusdSDK
	ContentThree *sdk.AudiusdSDK
)

func init() {
	DiscoveryOne = sdk.NewAudiusdSDK(DiscoveryOneRPC)
	ContentOne = sdk.NewAudiusdSDK(ContentOneRPC)
	ContentTwo = sdk.NewAudiusdSDK(ContentTwoRPC)
	ContentThree = sdk.NewAudiusdSDK(ContentThreeRPC)
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
