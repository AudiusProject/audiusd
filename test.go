package main

import (
	"context"
	"log"
	"math/big"
	"strings"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/AudiusProject/audiusd/pkg/eth/contracts"
	"github.com/AudiusProject/audiusd/pkg/eth/contracts/gen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	contentTwoKey       = "1aa14c63d481dcc1185a654eb52c9c0749d07ac8f30ef17d45c3c391d9bf68eb"
	contentThreeKey     = "1166189cdf129cdcb011f2ad0e5be24f967f7b7026d162d7c36073b12020b61c"
	contentThreeAddress = "0x1B569e8f1246907518Ff3386D523dcF373e769B6"
	contentThreeEp      = "https://node4.audiusd.devnet"
)

func main() {
	ctx := context.Background()

	wsRpcUrl := "http://localhost:8545"
	if strings.HasPrefix(wsRpcUrl, "https") {
		wsRpcUrl = "wss" + strings.TrimPrefix(wsRpcUrl, "https")
	} else if strings.HasPrefix(wsRpcUrl, "http:") {
		wsRpcUrl = "ws" + strings.TrimPrefix(wsRpcUrl, "http")
	}

	ethrpc, err := ethclient.Dial(wsRpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer ethrpc.Close()

	// Init contracts
	c, err := contracts.NewAudiusContracts(ethrpc, config.DevRegistryAddress)
	if err != nil {
		log.Fatal(err)
	}

	governanceContract, err := c.GetGovernanceContract()
	if err != nil {
		log.Fatal(err)
	}

	chainID, err := ethrpc.ChainID(ctx)
	if err != nil {
		log.Fatal(err)
	}

	ethKey, err := common.EthToEthKey(contentTwoKey)
	if err != nil {
		log.Fatal(err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(ethKey, chainID)
	if err != nil {
		log.Fatal(err)
	}

	delegateManagerABI, err := gen.DelegateManagerMetaData.GetAbi()
	if err != nil {
		log.Fatal(err)
	}

	contentThreeAddr := ethcommon.HexToAddress(contentThreeAddress)

	callData1, err := delegateManagerABI.Pack("slash", big.NewInt(10), contentThreeAddr)
	if err != nil {
		log.Fatal(err)
	}
	callData2, err := delegateManagerABI.Pack("slash", big.NewInt(100), contentThreeAddr)
	if err != nil {
		log.Fatal(err)
	}
	callData3, err := delegateManagerABI.Pack("slash", big.NewInt(5), contentThreeAddr)
	if err != nil {
		log.Fatal(err)
	}

	_, err = governanceContract.SubmitProposal(
		opts,
		contracts.DelegateManagerKey,
		big.NewInt(0),
		"slash(uint256,address)",
		callData1,
		"Test Slash Proposal 1",
		"Integration test for slash proposal 1",
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = governanceContract.SubmitProposal(
		opts,
		contracts.DelegateManagerKey,
		big.NewInt(0),
		"slash(uint256,address)",
		callData2,
		"Test Slash Proposal 2",
		"Integration test for slash proposal 2",
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = governanceContract.SubmitProposal(
		opts,
		contracts.DelegateManagerKey,
		big.NewInt(0),
		"slash(uint256,address)",
		callData3,
		"Test Slash Proposal 3",
		"Integration test for slash proposal 3",
	)
	if err != nil {
		log.Fatal(err)
	}

}
