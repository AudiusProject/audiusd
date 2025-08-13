package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/AudiusProject/audiusd/pkg/core/db"
	cmtcfg "github.com/cometbft/cometbft/config"
)

// nodes that are going to reset and sync via state sync

var (
	devTargetBlock = 50
	devTargetNodes = []string{
		"https://node2.audiusd.devnet",
	}

	stageTargetBlock = 10000
	stageTargetNodes = []string{
		"creatornode9.staging.audius.co",
	}

	prodTargetBlock = 7868510
	prodTargetNodes = []string{}
)

func (s *Server) resetChain() error {
	// if node in targetNodes, delete chain dir and restart using default state sync params

	chainDir := s.cometbftConfig.RootDir
	s.logger.Infof("chain dir: %s", chainDir)

	targetBlock := 0
	targetNodes := []string{}

	isTest := true
	isProd := false

	if s.config.Environment == "dev" {
		targetBlock = devTargetBlock
		targetNodes = devTargetNodes
	}

	if s.config.Environment == "staging" {
		targetBlock = stageTargetBlock
		targetNodes = stageTargetNodes
	}

	if s.config.Environment == "prod" {
		isProd = true
		targetBlock = prodTargetBlock
		targetNodes = prodTargetNodes
	}

	if targetBlock == 0 || len(targetNodes) == 0 {
		return nil
	}

	if !slices.Contains(targetNodes, s.config.NodeEndpoint) {
		// if endpoint not in this we aren't supposed to clear chain dir
		return nil
	}

	latestBlock, err := s.db.GetLatestBlock(context.Background())
	if err != nil {
		return err
	}

	latestBlockHeight := int(latestBlock.Height)

	if isProd && latestBlockHeight != targetBlock {
		// we are either above (operating) or below (syncing) target block
		// do not reset chain dir
		return nil
	}

	highestTargetBlock := targetBlock + 100
	latestBlockInWindow := latestBlockHeight >= targetBlock && latestBlockHeight <= highestTargetBlock
	if isTest && !latestBlockInWindow {
		// in testing envs only test if we have passed the block, not stuck
		return nil
	}

	// we are stuck at target block, remove chaindir and os exit to restart
	if err := ResetForResync(s.cometbftConfig); err != nil {
		return fmt.Errorf("reset for resync: %w", err)
	}

	// run pg down migrations before restart
	db.RunDownMigrations(s.logger, s.config.PSQLConn)

	// clowny
	os.Exit(1)

	// check if target block
	return nil
}

// ResetForResync deletes all contents of the CometBFT data dir
// except priv_validator_state.json, which is preserved to avoid double-signing.
func ResetForResync(cfg *cmtcfg.Config) error {
	dataDir := filepath.Join(cfg.RootDir, "data")
	stateFile := filepath.Join(dataDir, "priv_validator_state.json")

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("read data dir: %w", err)
	}

	for _, e := range entries {
		// Skip priv_validator_state.json
		if e.Name() == "priv_validator_state.json" {
			continue
		}
		path := filepath.Join(dataDir, e.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}

	// Ensure priv_validator_state.json still exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return fmt.Errorf("missing priv_validator_state.json — restore from backup before starting")
	}

	return nil
}
