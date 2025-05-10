package etl

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/etl/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (etl *ETLService) Run() error {
	dbUrl := etl.dbURL
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

	etl.pool = pool
	etl.db = db.New(pool)

	etl.logger.Infof("starting etl service")

	err = etl.indexBlocks()
	if err != nil {
		return fmt.Errorf("indexer crashed: %v", err)
	}

	return nil
}

func (etl *ETLService) indexBlocks() error {
	for {
		// Get the latest indexed block height
		latestHeight, err := etl.db.GetLatestIndexedBlock(context.Background())
		if err != nil {
			// If no records exist, start from block 1
			if errors.Is(err, pgx.ErrNoRows) {
				latestHeight = 0 // Start from block 1 (nextHeight will be 1)
			} else {
				etl.logger.Errorf("error getting latest indexed block: %v", err)
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

		if block.Msg.Block.Height < 0 {
			continue
		}

		_, err = etl.db.InsertBlock(context.Background(), db.InsertBlockParams{
			BlockHeight: block.Msg.Block.Height,
			BlockTime:   pgtype.Timestamp{Time: block.Msg.Block.Timestamp.AsTime(), Valid: true},
		})
		if err != nil {
			etl.logger.Errorf("error inserting block %d: %v", nextHeight, err)
			continue
		}

		txs := block.Msg.Block.Transactions
		for _, tx := range txs {
			switch signedTx := tx.Transaction.Transaction.(type) {
			case *v1.SignedTransaction_Plays:
				for _, play := range signedTx.Plays.GetPlays() {
					etl.db.InsertPlay(context.Background(), db.InsertPlayParams{
						Address:     play.UserId,
						TrackID:     play.TrackId,
						City:        play.City,
						Region:      play.Region,
						Country:     play.Country,
						PlayedAt:    pgtype.Timestamp{Time: play.Timestamp.AsTime(), Valid: true},
						BlockHeight: block.Msg.Block.Height,
						TxHash:      tx.Hash,
					})
				}
			case *v1.SignedTransaction_ManageEntity:
				me := signedTx.ManageEntity
				etl.db.InsertManageEntity(context.Background(), db.InsertManageEntityParams{
					Address:     me.GetSigner(),
					EntityType:  me.GetEntityType(),
					EntityID:    me.GetEntityId(),
					Action:      me.GetAction(),
					Metadata:    pgtype.Text{String: me.GetMetadata(), Valid: true},
					Signature:   me.GetSignature(),
					Signer:      me.GetSigner(),
					Nonce:       me.GetNonce(),
					BlockHeight: block.Msg.Block.Height,
					TxHash:      tx.Hash,
				})
			}
		}
	}
}
