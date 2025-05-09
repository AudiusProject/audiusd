package etl

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (etl *ETLService) Run() error {
	dbUrl := os.Getenv("dbUrl")
	if dbUrl == "" {
		return fmt.Errorf("dbUrl environment variable not set")
	}

	err := db.RunMigrations(etl.logger, dbUrl, true)
	if err != nil {
		return fmt.Errorf("error running migrations: %v", err)
	}

	pgConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return fmt.Errorf("error parsing database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), pgConfig)
	if err != nil {
		return fmt.Errorf("error creating database pool: %v", err)
	}

	etl.db = db.New(pool)

	return etl.indexBlocks()
}

func (etl *ETLService) indexBlocks() error {
	for {
		// Get the latest indexed block height
		latestHeight, err := etl.db.GetLatestIndexedBlock(context.Background())
		if err != nil {
			// If no records exist, start from block 1
			if err == sql.ErrNoRows {
				latestHeight = 0 // Start from block 1 (nextHeight will be 1)
			} else {
				continue
			}
		}

		// Get the next block
		nextHeight := latestHeight + 1
		block, err := etl.core.GetBlock(context.Background(), connect.NewRequest(&v1.GetBlockRequest{
			Height: nextHeight,
		}))
		if err != nil {
			etl.logger.Errorf("error getting block %d: %v", nextHeight, err)
			continue
		}

		// If block doesn't exist yet (height = -1), wait and try again
		if block.Msg.Block.Height == -1 {
			time.Sleep(time.Second)
			continue
		}

		// Store the block in etl_blocks
		_, err = etl.db.UpdateLatestIndexedBlock(context.Background(), nextHeight)
		if err != nil {
			etl.logger.Errorf("error updating latest indexed block: %v", err)
			continue
		}

		etl.logger.Infof("indexed block %d", nextHeight)
	}
}
