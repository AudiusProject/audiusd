package server

import (
	"context"
	"crypto/sha256"
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
	if err := ethRpc.RegisterName("eth", &EthAPI{server: s}); err != nil {
		return fmt.Errorf("failed to register eth rpc: %v", err)
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

func (api *EthAPI) ChainId(ctx context.Context) (*hexutil.Big, error) {
	chainID := api.server.config.GenesisFile.ChainID
	hash := sha256.Sum256([]byte(chainID))
	numericChainID := hash[:8]
	return (*hexutil.Big)(new(big.Int).SetBytes(numericChainID)), nil
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
