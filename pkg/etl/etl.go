package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/db"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

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
	// Parse the protobuf message
	var signedTx core_proto.SignedTransaction
	if err := proto.Unmarshal(tx.Transaction, &signedTx); err != nil {
		return fmt.Errorf("error unmarshaling transaction: %v", err)
	}

	// Determine transaction type
	var txType string
	switch signedTx.GetTransaction().(type) {
	case *core_proto.SignedTransaction_Plays:
		txType = "Plays"
	case *core_proto.SignedTransaction_ValidatorRegistration:
		txType = "ValidatorRegistration"
	case *core_proto.SignedTransaction_ValidatorDeregistration:
		txType = "ValidatorDeregistration"
	case *core_proto.SignedTransaction_SlaRollup:
		txType = "SlaRollup"
	case *core_proto.SignedTransaction_StorageProof:
		txType = "StorageProof"
	case *core_proto.SignedTransaction_StorageProofVerification:
		txType = "StorageProofVerification"
	case *core_proto.SignedTransaction_ManageEntity:
		txType = "ManageEntity"
	default:
		txType = "Unknown"
	}

	// Insert into core_tx_decoded
	jsonBytes, err := json.Marshal(signedTx)
	if err != nil {
		logger.Errorf("failed to marshal tx to json: %v", err)
		jsonBytes = []byte("{}")
	}

	if err := queries.InsertDecodedTx(ctx, db.InsertDecodedTxParams{
		BlockHeight: tx.BlockID,
		TxIndex:     tx.Index,
		TxHash:      tx.TxHash,
		TxType:      txType,
		TxData:      jsonBytes,
		CreatedAt:   pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
	}); err != nil {
		return fmt.Errorf("error inserting decoded tx: %v", err)
	}

	// If this is a play transaction, process the plays
	if plays := signedTx.GetPlays(); plays != nil {
		for _, play := range plays.Plays {
			if err := queries.InsertDecodedPlay(ctx, db.InsertDecodedPlayParams{
				TxHash:    tx.TxHash,
				UserID:    play.UserId,
				TrackID:   play.TrackId,
				PlayedAt:  pgtype.Timestamptz{Time: play.Timestamp.AsTime(), Valid: true},
				Signature: play.Signature,
				City:      pgtype.Text{String: play.City, Valid: play.City != ""},
				Region:    pgtype.Text{String: play.Region, Valid: play.Region != ""},
				Country:   pgtype.Text{String: play.Country, Valid: play.Country != ""},
				CreatedAt: pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
			}); err != nil {
				logger.Errorf("failed to insert play record: %v", err)
				continue
			}
		}
	}

	return nil
}
