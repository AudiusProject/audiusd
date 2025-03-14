package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlayData struct {
	UserID    string    `json:"userId"`
	TrackID   string    `json:"trackId"`
	PlayedAt  time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
	Location  struct {
		City    string `json:"city"`
		Region  string `json:"region"`
		Country string `json:"country"`
	} `json:"location"`
}

func Run(ctx context.Context, logger *common.Logger) error {
	logger.Info("Starting ETL service...")

	// Get database connection string from environment
	dbUrl := os.Getenv("dbUrl")
	if dbUrl == "" {
		return fmt.Errorf("dbUrl environment variable not set")
	}

	// Create connection pool
	pgConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return fmt.Errorf("error parsing database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return fmt.Errorf("error creating database pool: %v", err)
	}
	defer pool.Close()

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("error connecting to database: %v", err)
	}

	logger.Info("Successfully connected to database")

	// Create a db.Queries instance
	queries := db.New(pool)

	// Set up notification trigger
	if _, err := pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION notify_new_transaction() RETURNS TRIGGER AS $$
		BEGIN
			PERFORM pg_notify('new_transaction', NEW.tx_hash);
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS new_transaction_trigger ON core_transactions;
		
		CREATE TRIGGER new_transaction_trigger
			AFTER INSERT ON core_transactions
			FOR EACH ROW
			EXECUTE FUNCTION notify_new_transaction();
	`); err != nil {
		return fmt.Errorf("error setting up notification trigger: %v", err)
	}

	// Start listening for notifications
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("error acquiring connection: %v", err)
	}
	defer conn.Release()

	// Listen for new transactions
	if _, err := conn.Exec(ctx, "LISTEN new_transaction"); err != nil {
		return fmt.Errorf("error setting up LISTEN: %v", err)
	}

	logger.Info("Listening for new transactions...")

	for {
		select {
		case <-ctx.Done():
			logger.Info("ETL service shutting down...")
			return nil
		default:
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				logger.Errorf("Error waiting for notification: %v", err)
				continue
			}

			// Get the transaction by hash
			tx, err := queries.GetTx(ctx, notification.Payload)
			if err != nil {
				logger.Errorf("Error getting transaction: %v", err)
				continue
			}

			// Process the transaction
			if err := processTransaction(ctx, logger, queries, tx); err != nil {
				logger.Errorf("Error processing transaction: %v", err)
			}
		}
	}
}

func processTransaction(ctx context.Context, logger *common.Logger, queries *db.Queries, tx db.CoreTransaction) error {
	// Decode the transaction data
	var txData struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(tx.Transaction, &txData); err != nil {
		return fmt.Errorf("error unmarshaling transaction: %v", err)
	}

	// Insert into core_tx_decoded
	if err := queries.InsertDecodedTx(ctx, db.InsertDecodedTxParams{
		BlockHeight: tx.BlockID,
		TxIndex:     tx.Index,
		TxHash:      tx.TxHash,
		TxType:      txData.Type,
		TxData:      tx.Transaction,
		CreatedAt:   pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
	}); err != nil {
		return fmt.Errorf("error inserting decoded tx: %v", err)
	}

	// If this is a play transaction, process it further
	if txData.Type == "play" {
		var playData PlayData
		if err := json.Unmarshal(txData.Data, &playData); err != nil {
			return fmt.Errorf("error unmarshaling play data: %v", err)
		}

		// Insert into core_tx_decoded_plays
		if err := queries.InsertDecodedPlay(ctx, db.InsertDecodedPlayParams{
			TxHash:    tx.TxHash,
			UserID:    playData.UserID,
			TrackID:   playData.TrackID,
			PlayedAt:  pgtype.Timestamptz{Time: playData.PlayedAt, Valid: true},
			Signature: playData.Signature,
			City:      pgtype.Text{String: playData.Location.City, Valid: playData.Location.City != ""},
			Region:    pgtype.Text{String: playData.Location.Region, Valid: playData.Location.Region != ""},
			Country:   pgtype.Text{String: playData.Location.Country, Valid: playData.Location.Country != ""},
			CreatedAt: pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
		}); err != nil {
			return fmt.Errorf("error inserting decoded play: %v", err)
		}
	}

	return nil
}
