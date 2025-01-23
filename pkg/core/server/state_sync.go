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

	if err := s.createPgDump(height); err != nil {
		return err
	}

	s.logger.Info("backup completed")

	return nil
}

func (s *Server) createPgDump(height int64) error {
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
	return cmd.Run()
}
