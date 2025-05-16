package server

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

func (s *Server) startFundingRoundManager() error {
	ticker := time.NewTicker(6 * time.Hour)
	if s.isDevEnvironment() {
		// query eth chain more aggressively on dev
		ticker = time.NewTicker(5 * time.Second)
	}

	defer ticker.Stop()

	for range ticker.C {
		r, err := s.getLatestFundingRound()
		if err != nil {
			s.logger.Errorf("error getting latest funding round: %v", err)
		}
		if s.fundingRoundPending {
			go s.initFundingRound(r)
		}
	}
	return nil
}

func (s *Server) getLatestFundingRound() (int64, error) {
	ctx := context.Background()
	// TODO why isn't there just a getter on ClaimsManager.currentRound ?
	blockNum, err := s.contracts.ClaimsManager.GetLastFundedBlock(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, fmt.Errorf("could not get latest funded block number: %w", err)
	}

	query := ethereum.FilterQuery{
		FromBlock: blockNum,
		ToBlock:   blockNum,
	}
	logs, err := s.eth.FilterLogs(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("could not get logs from latest funded block: %w", err)
	}

	if len(logs) < 2 {
		return 0, fmt.Errorf("not enough logs from latest funded block. Got: %v", logs)
	}
	if len(logs[1].Topics) < 2 {
		return 0, fmt.Errorf("not enough logs from latest funded block. Got: %v", logs)
	}

	b := logs[1].Topics[2].Big()
	if b.Cmp(big.NewInt(math.MinInt64)) < 0 || b.Cmp(big.NewInt(math.MaxInt64)) > 0 {
		return 0, fmt.Errorf("value out of int range")
	}

	return b.Int64(), nil
}

func (s *Server) initFundingRound(newRound int64) {
	ctx := context.Background()
	validators, err := s.db.GetAllRegisteredNodesSorted(ctx)
	if err != nil {
		s.logger.Errorf("could not init funding round: error getting validators: %w", err)
		return
	}
	roundResults := make([]v1.FundingRoundSLAResult, 0, len(s.peers))
	tx := v1.FundingRoundUpdate{
		NewRoundNum: newRound,
		Results:     nil,
	}
}
