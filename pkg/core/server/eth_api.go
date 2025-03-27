package server

import (
	"context"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
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
		w.Header().Set("Content-Type", "application/json")

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
	return "1056801", nil
}

// Web3API provides stubs for the "web3" namespace
type Web3API struct{}

// Stub: web3_clientVersion
func (api *Web3API) ClientVersion(ctx context.Context) (string, error) {
	return "MyCustomNode/v1.0.0", nil // Return a custom client version string
}

func (api *EthAPI) ChainId(ctx context.Context) (*hexutil.Big, error) {
	return (*hexutil.Big)(big.NewInt(1056801)), nil
}

// Stub: eth_blockNumber
func (api *EthAPI) BlockNumber(ctx context.Context) (*hexutil.Uint64, error) {
	blockHeight := uint64(api.server.cache.currentHeight.Load())
	return (*hexutil.Uint64)(&blockHeight), nil
}

func (api *EthAPI) GetBlockByNumber(ctx context.Context, blockNumber string, fullTx bool) (map[string]interface{}, error) {
	var height uint64

	switch blockNumber {
	case "latest":
		height = uint64(api.server.cache.currentHeight.Load())
	default:
		n := new(big.Int)
		if err := n.UnmarshalText([]byte(blockNumber)); err != nil {
			return nil, fmt.Errorf("invalid block number: %s", blockNumber)
		}
		height = n.Uint64()
	}

	// TODO: Replace this with real block fetching logic from your chain
	block := map[string]interface{}{
		"number":           hexutil.EncodeUint64(height),
		"hash":             "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"parentHash":       "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdef",
		"nonce":            "0x0000000000000000",
		"sha3Uncles":       "0x1dcc4de8dec75d7aab85b567b6ccfdf55ed6e8195b14701d08cb5b2f89be1e11",
		"logsBloom":        "0x" + string(make([]byte, 256)),
		"transactionsRoot": "0x" + string(make([]byte, 32)),
		"stateRoot":        "0x" + string(make([]byte, 32)),
		"receiptsRoot":     "0x" + string(make([]byte, 32)),
		"miner":            "0x0000000000000000000000000000000000000000",
		"difficulty":       hexutil.EncodeBig(big.NewInt(1)),
		"totalDifficulty":  hexutil.EncodeBig(big.NewInt(1)),
		"extraData":        "0x",
		"size":             hexutil.Uint64(1000),
		"gasLimit":         hexutil.Uint64(10000000),
		"gasUsed":          hexutil.Uint64(0),
		"timestamp":        hexutil.Uint64(1711830000),
		"transactions":     []any{}, // empty list unless fullTx == true
		"uncles":           []string{},
	}

	return block, nil
}

// Stub: eth_getBalance
func (api *EthAPI) GetBalance(ctx context.Context, address string, block string) (*hexutil.Big, error) {
	// Return dummy balance for now
	return (*hexutil.Big)(big.NewInt(1000000000000000000)), nil // 1 ETH
}
