package server

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/cometbft/cometbft/types"
)

type SnapshotMetadata struct {
	Height     int64     `json:"height"`
	Hash       string    `json:"hash"`
	Time       time.Time `json:"time"`
	ChunkCount int       `json:"chunk_count"`
	ChainID    string    `json:"chain_id"`
}

func (s *Server) startStateSync() error {
	<-s.awaitRpcReady
	logger := s.logger.Child("state_sync")

	if !s.config.StateSync.ServeSnapshots {
		logger.Info("State sync is not enabled, skipping snapshot creation")
		return nil
	}

	node := s.node
	eb := node.EventBus()

	if eb == nil {
		return errors.New("event bus not ready")
	}

	subscriberID := "state-sync-subscriber"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	query := types.EventQueryNewBlock
	subscription, err := eb.Subscribe(ctx, subscriberID, query)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NewBlock events: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping block event subscription")
			return nil
		case msg := <-subscription.Out():
			blockEvent := msg.Data().(types.EventDataNewBlock)
			blockHeight := blockEvent.Block.Height
			if blockHeight%s.config.StateSync.BlockInterval != 0 {
				continue
			}

			if err := s.createSnapshot(logger, blockHeight); err != nil {
				logger.Errorf("error creating snapshot: %v", err)
			}

			if err := s.pruneSnapshots(logger); err != nil {
				logger.Errorf("error pruning snapshots: %v", err)
			}
		case err := <-subscription.Canceled():
			s.logger.Errorf("Subscription cancelled: %v", err)
			return nil
		}
	}
}

func (s *Server) createSnapshot(logger *common.Logger, height int64) error {
	// create snapshot directory if it doesn't exist
	snapshotDir := filepath.Join(s.config.RootDir, fmt.Sprintf("snapshots_%s", s.config.GenesisFile.ChainID))
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("error creating snapshot directory: %v", err)
	}

	if s.rpc == nil {
		return nil
	}

	status, err := s.rpc.Status(context.Background())
	if err != nil {
		return nil
	}

	if status.SyncInfo.CatchingUp {
		return nil
	}

	block, err := s.rpc.Block(context.Background(), &height)
	if err != nil {
		return nil
	}

	logger.Info("Creating snapshot", "height", height)

	blockHeight := height
	blockHash := block.BlockID.Hash.String()
	blockTime := block.Block.Time

	latestSnapshotDirName := fmt.Sprintf("height_%010d", blockHeight)
	latestSnapshotDir := filepath.Join(snapshotDir, latestSnapshotDirName)
	if err := os.MkdirAll(latestSnapshotDir, 0755); err != nil {
		return fmt.Errorf("error creating latest snapshot directory: %v", err)
	}

	logger.Info("Creating pg_dump", "height", blockHeight)

	if err := s.createPgDump(logger, latestSnapshotDir); err != nil {
		return fmt.Errorf("error creating pg_dump: %v", err)
	}

	logger.Info("Chunking pg_dump", "height", blockHeight)

	chunkCount, err := s.chunkPgDump(logger, latestSnapshotDir)
	if err != nil {
		return fmt.Errorf("error chunking pg_dump: %v", err)
	}

	logger.Info("Deleting pg_dump", "height", blockHeight)

	if err := s.deletePgDump(logger, latestSnapshotDir); err != nil {
		return fmt.Errorf("error deleting pg_dump: %v", err)
	}

	logger.Info("Writing snapshot metadata", "height", blockHeight)

	snapshotMetadata := SnapshotMetadata{
		Height:     blockHeight,
		Hash:       blockHash,
		Time:       blockTime,
		ChunkCount: chunkCount,
		ChainID:    s.config.GenesisFile.ChainID,
	}

	snapshotMetadataFile := filepath.Join(latestSnapshotDir, "metadata.json")
	jsonBytes, err := json.Marshal(snapshotMetadata)
	if err != nil {
		return fmt.Errorf("error marshalling snapshot metadata: %v", err)
	}

	if err := os.WriteFile(snapshotMetadataFile, jsonBytes, 0644); err != nil {
		return fmt.Errorf("error writing snapshot metadata: %v", err)
	}

	logger.Info("Snapshot created", "height", blockHeight)

	return nil
}

// createPgDump creates a pg_dump of the database and writes it to the latest snapshot directory
func (s *Server) createPgDump(logger *common.Logger, latestSnapshotDir string) error {
	pgString := s.config.PSQLConn
	dumpPath := filepath.Join(latestSnapshotDir, "data.dump")

	// You can customize this slice with the tables you want to dump
	tables := []string{
		"access_keys",
		"core_app_state",
		"core_blocks",
		"core_db_migrations",
		"core_transactions",
		"core_tx_stats",
		"core_validators",
		"management_keys",
		"sla_node_reports",
		"sla_rollups",
		"sound_recordings",
		"storage_proof_peers",
		"storage_proofs",
		"track_releases",
	}

	// Start building the args
	args := []string{"--dbname=" + pgString, "-Fc"}
	for _, table := range tables {
		args = append(args, "-t", table)
	}
	args = append(args, "-f", dumpPath)

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("pg_dump failed", "error", err, "output", string(output))
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	logger.Info("pg_dump succeeded", "output", string(output))
	return nil
}

// chunkPgDump splits the pg_dump into 16MB gzip-compressed chunks and returns the number of chunks created
func (s *Server) chunkPgDump(logger *common.Logger, latestSnapshotDir string) (int, error) {
	const chunkSize = 16 * 1024 * 1024 // 16MB
	dumpPath := filepath.Join(latestSnapshotDir, "data.dump")

	dumpFile, err := os.Open(dumpPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open pg_dump: %w", err)
	}
	defer dumpFile.Close()

	buffer := make([]byte, chunkSize)
	chunkIndex := 0

	for {
		n, readErr := io.ReadFull(dumpFile, buffer)
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			return chunkIndex, fmt.Errorf("error reading pg_dump: %w", readErr)
		}

		if n == 0 {
			break
		}

		chunkName := fmt.Sprintf("chunk_%04d.gz", chunkIndex)
		chunkPath := filepath.Join(latestSnapshotDir, chunkName)
		chunkFile, err := os.Create(chunkPath)
		if err != nil {
			return chunkIndex, fmt.Errorf("failed to create chunk: %w", err)
		}

		gw := gzip.NewWriter(chunkFile)
		_, err = gw.Write(buffer[:n])
		if err != nil {
			chunkFile.Close()
			return chunkIndex, fmt.Errorf("failed to write gzip chunk: %w", err)
		}
		gw.Close()
		chunkFile.Close()

		logger.Info("Wrote chunk", "path", chunkPath, "size", n)
		chunkIndex++

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	return chunkIndex, nil
}

func (s *Server) deletePgDump(logger *common.Logger, latestSnapshotDir string) error {
	dumpPath := filepath.Join(latestSnapshotDir, "data.dump")
	if err := os.Remove(dumpPath); err != nil {
		return fmt.Errorf("error deleting pg_dump: %w", err)
	}

	return nil
}

// Prunes snapshots by deleting the oldest ones while retaining the most recent ones
// based on the configured retention count
func (s *Server) pruneSnapshots(logger *common.Logger) error {
	snapshotDir := filepath.Join(s.config.RootDir, fmt.Sprintf("snapshots_%s", s.config.GenesisFile.ChainID))
	keep := s.config.StateSync.Keep

	files, err := os.ReadDir(snapshotDir)
	if err != nil {
		return fmt.Errorf("error reading snapshot directory: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for i := range files {
		if i >= len(files)-keep {
			break
		}

		os.RemoveAll(filepath.Join(snapshotDir, files[i].Name()))
		logger.Info("Deleted snapshot", "path", filepath.Join(snapshotDir, files[i].Name()))
	}

	return nil
}

func (s *Server) getStoredSnapshots() ([]SnapshotMetadata, error) {
	snapshotDir := filepath.Join(s.config.RootDir, fmt.Sprintf("snapshots_%s", s.config.GenesisFile.ChainID))

	dirs, err := os.ReadDir(snapshotDir)
	if err != nil {
		return nil, fmt.Errorf("error reading snapshot directory: %w", err)
	}

	snapshots := make([]SnapshotMetadata, 0)
	for _, entry := range dirs {
		if !entry.IsDir() {
			continue
		}

		metadataPath := filepath.Join(snapshotDir, entry.Name(), "metadata.json")
		info, err := os.Stat(metadataPath)
		if err != nil || info.IsDir() {
			continue
		}

		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return nil, fmt.Errorf("error reading metadata file at %s: %w", metadataPath, err)
		}

		var meta SnapshotMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, fmt.Errorf("error unmarshalling metadata at %s: %w", metadataPath, err)
		}

		snapshots = append(snapshots, meta)
	}

	// sort by height, ascending
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Height < snapshots[j].Height
	})

	return snapshots, nil
}
