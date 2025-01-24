package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	stateSyncTables = []string{
		"core_blocks", "core_tx_results", "core_events",
		"core_attributes", "core_app_state", "core_validators",
		"sla_rollups", "sla_node_reports", "core_tx_stats", "",
	}
)

type StateSyncStatus struct {
	backingUpMU         sync.Mutex
	lastBackupStarted   time.Time
	lastBackupCompleted time.Time
}

func NewStateSyncStatus() *StateSyncStatus {
	return &StateSyncStatus{}
}

func (s *Server) createStateBackup(height int64) error {
	<-s.awaitRpcReady
	status := s.stateSyncStatus

	if !status.backingUpMU.TryLock() {
		return nil
	}

	rpcStatus, err := s.rpc.Status(context.Background())
	if err != nil {
		return err
	}

	if rpcStatus.SyncInfo.CatchingUp {
		return nil
	}

	status.lastBackupStarted = time.Now()
	defer func() {
		status.lastBackupCompleted = time.Now()
		status.backingUpMU.Unlock()
	}()

	pgdumpFilename, err := s.createPgDump(height)
	if err != nil {
		return fmt.Errorf("could not create state pgdump: %v", err)
	}

	s.logger.Infof("created pgdump: %s", pgdumpFilename)

	compressedPgdumpFilename, err := s.compressPgDump(pgdumpFilename)
	if err != nil {
		return fmt.Errorf("could not compress pgdump: %v", err)
	}

	s.logger.Infof("compressed pgdump: %s", compressedPgdumpFilename)

	if err := s.splitPgDump(compressedPgdumpFilename); err != nil {
		return fmt.Errorf("could not split pgdump: %v", err)
	}

	s.logger.Info("split pgdump")

	if err := s.sweepOldBackups(); err != nil {
		return fmt.Errorf("could not sweep old backups: %v", err)
	}

	s.logger.Info("backup completed")

	return nil
}

func (s *Server) createPgDump(height int64) (string, error) {
	chainID := s.config.GenesisFile.ChainID
	stateBackupFilename := fmt.Sprintf("state_backup_%s_block_%d.sql", chainID, height)

	cmdArgs := []string{
		"--dbname=" + s.config.PSQLConn,
		"--file=" + stateBackupFilename,
	}

	cmd := exec.Command("pg_dump", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for _, table := range stateSyncTables {
		cmdArgs = append(cmdArgs, "--table="+table)
	}

	s.logger.Info("starting state backup")

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return stateBackupFilename, nil
}

func (s *Server) compressPgDump(pgdumpFilename string) (string, error) {
	compressedFilename := pgdumpFilename + ".gz"

	cmd := exec.Command("gzip", "-c", pgdumpFilename)
	out, err := os.Create(compressedFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer out.Close()

	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to compress file: %w", err)
	}

	return compressedFilename, nil
}

func (s *Server) splitPgDump(compressedFilename string) error {
	chunkPrefix := compressedFilename + ".part_"

	// cometbft requires chunks be smaller than 16mbs, use 15 for some headroom
	cmd := exec.Command("split", "-b", "15m", compressedFilename, chunkPrefix)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to split compressed file: %w", err)
	}

	return nil
}

func (s *Server) sweepOldBackups() error {
	return nil
}
