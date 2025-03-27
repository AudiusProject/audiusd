package server

import (
	"context"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/labstack/echo/v4"
)

type EthAPI struct {
	server *Server
}

func (s *Server) createEthRPC() error {
	ethRpc := rpc.NewServer()

	// Register the "eth" namespace
	if err := ethRpc.RegisterName("eth", &EthAPI{server: s}); err != nil {
		return fmt.Errorf("failed to register eth rpc: %v", err)
	}

	// Register the "net" namespace
	if err := ethRpc.RegisterName("net", &NetAPI{server: s}); err != nil {
		return fmt.Errorf("failed to register net rpc: %v", err)
	}

	// Register the "web3" namespace
	if err := ethRpc.RegisterName("web3", &Web3API{}); err != nil {
		return fmt.Errorf("failed to register web3 rpc: %v", err)
	}

	e := s.GetEcho()

	e.POST("/core/erpc", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		ethRpc.ServeHTTP(w, r)
	})))

	e.OPTIONS("/core/erpc", func(c echo.Context) error {
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type")
		return c.NoContent(http.StatusOK)
	})

	return nil
}

// NetAPI provides stubs for the "net" namespace
type NetAPI struct {
	server *Server
}

// Stub: net_version
func (api *NetAPI) Version(ctx context.Context) (string, error) {
	return "1", nil // Return the network ID (e.g., "1" for mainnet)
}

// Stub: net_listening
func (api *NetAPI) Listening(ctx context.Context) (bool, error) {
	return true, nil // Indicate that the node is listening
}

// Stub: net_peerCount
func (api *NetAPI) PeerCount(ctx context.Context) (hexutil.Uint, error) {
	return hexutil.Uint(0), nil // Return the number of connected peers
}

// Web3API provides stubs for the "web3" namespace
type Web3API struct{}

// Stub: web3_clientVersion
func (api *Web3API) ClientVersion(ctx context.Context) (string, error) {
	return "MyCustomNode/v1.0.0", nil // Return a custom client version string
}

// Stub: web3_sha3
func (api *Web3API) Sha3(ctx context.Context, input hexutil.Bytes) (hexutil.Bytes, error) {
	hash := common.BytesToHash(crypto.Keccak256(input))
	return hash[:], nil // Return the Keccak-256 hash of the input
}

func (api *EthAPI) ChainId(ctx context.Context) (*hexutil.Big, error) {
	return (*hexutil.Big)(big.NewInt(1056801)), nil
}

// Stub: eth_blockNumber
func (api *EthAPI) BlockNumber(ctx context.Context) (*hexutil.Uint64, error) {
	blockHeight := uint64(api.server.cache.currentHeight.Load())
	return (*hexutil.Uint64)(&blockHeight), nil
}

// Stub: eth_getBalance
func (api *EthAPI) GetBalance(ctx context.Context, address string, block string) (*hexutil.Big, error) {
	// Return dummy balance for now
	return (*hexutil.Big)(big.NewInt(1000000000000000000)), nil // 1 ETH
}

// Stub: eth_sendRawTransaction
func (api *EthAPI) SendRawTransaction(ctx context.Context, rawTx hexutil.Bytes) (string, error) {
	// Decode, verify, and submit transaction to your backend here
	return "0xdeadbeef", nil
}

func (api *EthAPI) GetTransactionCount(ctx context.Context, address string, blockParam string) (*hexutil.Uint64, error) {
	// Validate Ethereum address
	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid address: %s", address)
	}

	// Replace this with your logic — lookup nonce from your chain/backend
	// You might call a gRPC method or query your DB for the account's sequence/nonce
	var nonce uint64 = 0

	// Example: if you're storing nonce in your server cache/db
	// nonce, err := api.server.cache.GetAccountNonce(addr)
	// if err != nil {
	//     return nil, err
	// }

	return (*hexutil.Uint64)(&nonce), nil
}
