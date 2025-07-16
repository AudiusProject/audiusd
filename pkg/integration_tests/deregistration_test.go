package integration_tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	ethv1 "github.com/AudiusProject/audiusd/pkg/api/eth/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/AudiusProject/audiusd/pkg/eth/contracts"
	"github.com/AudiusProject/audiusd/pkg/integration_tests/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const (
	contentThreeKey = "1166189cdf129cdcb011f2ad0e5be24f967f7b7026d162d7c36073b12020b61c"
	contentThreeEp  = "https://node4.audiusd.devnet"
	contentTwoEp    = "https://node3.audiusd.devnet"
)

type CometRPCResponse struct {
	Result struct {
		Validators []struct {
			Address string `json:"address"`
		} `json:"validators"`
	} `json:"result"`
}

func TestDeregisterNode(t *testing.T) {
	ctx := context.Background()

	wsRpcUrl := config.DevEthRpc
	if strings.HasPrefix(wsRpcUrl, "https") {
		wsRpcUrl = "wss" + strings.TrimPrefix(wsRpcUrl, "https")
	} else if strings.HasPrefix(wsRpcUrl, "http:") {
		wsRpcUrl = "ws" + strings.TrimPrefix(wsRpcUrl, "http")
	}

	err := utils.WaitForDevnetHealthy(30 * time.Second)
	require.NoError(t, err)

	ethrpc, err := ethclient.Dial(wsRpcUrl)
	require.NoError(t, err, "eth client dial err")
	defer ethrpc.Close()

	// Init contracts
	c, err := contracts.NewAudiusContracts(ethrpc, config.DevRegistryAddress)
	require.NoError(t, err, "failed to initialize eth contracts")

	serviceProviderFactoryContract, err := c.GetServiceProviderFactoryContract()
	require.NoError(t, err, "failed to get service provider factory contract")

	chainID, err := ethrpc.ChainID(ctx)
	require.NoError(t, err, "failed to get chainID")

	ethKey, err := common.EthToEthKey(contentThreeKey)
	require.NoError(t, err, "failed to create ethereum key")

	opts, err := bind.NewKeyedTransactorWithChainID(ethKey, chainID)
	require.NoError(t, err, "failed to create keyed transactor")

	_, err = serviceProviderFactoryContract.Deregister(opts, contracts.ContentNode, contentThreeEp)
	require.NoError(t, err, "failed to deregister node4")

	time.Sleep(1 * time.Second)

	epResp, err := utils.ContentTwo.Eth.GetRegisteredEndpoints(ctx, connect.NewRequest(&ethv1.GetRegisteredEndpointsRequest{}))
	require.NoError(t, err, "failed to get registered endpoints from node3 eth service")
	require.Equal(t, 3, len(epResp.Msg.Endpoints), "unexpected number of endpoints returned by node3 eth service", epResp.Msg.Endpoints)

	for _, ep := range epResp.Msg.Endpoints {
		require.NotEqual(t, contentThreeEp, ep.Endpoint, "node4 should not be in returned endpoints")
	}

	time.Sleep(10 * time.Second)
	statusResp, err := utils.ContentTwo.Core.GetStatus(ctx, connect.NewRequest(&corev1.GetStatusRequest{}))
	require.NoError(t, err, "failed to get registered endpoints from node3 eth service")
	require.True(t, statusResp.Msg.Ready, "node3 is unhealthy")
	for _, peer := range statusResp.Msg.Peers.P2P {
		require.NotEqual(t, contentThreeEp, peer.Endpoint, "node4 should not be in returned peers")
	}

	cometRpcResp, err := http.Get(contentTwoEp + "/core/crpc/validators")
	require.NoError(t, err, "failed to get validators from node3 cometRPC")
	defer cometRpcResp.Body.Close()

	body, err := io.ReadAll(cometRpcResp.Body)
	require.NoError(t, err, "failed to read comet rpc response body")

	var r CometRPCResponse
	err = json.Unmarshal(body, &r)
	require.NoError(t, err, "failed to marshall comet rpc response body")
	require.Equal(t, 3, len(r.Result.Validators))
}
