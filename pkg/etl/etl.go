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

	dbUrl := os.Getenv("dbUrl")
	if dbUrl == "" {
		return fmt.Errorf("dbUrl environment variable not set")
	}

	pgConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return fmt.Errorf("error parsing database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return fmt.Errorf("error creating database pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("error connecting to database: %v", err)
	}

	logger.Info("Successfully connected to database")

	queries := db.New(pool)

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

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("error acquiring connection: %v", err)
	}
	defer conn.Release()

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

			tx, err := queries.GetTx(ctx, notification.Payload)
			if err != nil {
				logger.Errorf("Error getting transaction: %v", err)
				continue
			}

			if err := processTransaction(ctx, logger, queries, tx); err != nil {
				logger.Errorf("Error processing transaction: %v", err)
			}
		}
	}
}

func processTransaction(ctx context.Context, logger *common.Logger, queries *db.Queries, tx db.CoreTransaction) error {
	var signedTx core_proto.SignedTransaction
	if err := proto.Unmarshal(tx.Transaction, &signedTx); err != nil {
		return fmt.Errorf("error unmarshaling transaction: %v", err)
	}

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

	switch t := signedTx.GetTransaction().(type) {
	case *core_proto.SignedTransaction_Plays:
		for _, play := range t.Plays.Plays {
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

	case *core_proto.SignedTransaction_ValidatorRegistration:
		if err := queries.InsertDecodedValidatorRegistration(ctx, db.InsertDecodedValidatorRegistrationParams{
			TxHash:       tx.TxHash,
			Endpoint:     t.ValidatorRegistration.Endpoint,
			CometAddress: t.ValidatorRegistration.CometAddress,
			EthBlock:     t.ValidatorRegistration.EthBlock,
			NodeType:     t.ValidatorRegistration.NodeType,
			SpID:         t.ValidatorRegistration.SpId,
			PubKey:       t.ValidatorRegistration.PubKey,
			Power:        t.ValidatorRegistration.Power,
			CreatedAt:    pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
		}); err != nil {
			logger.Errorf("failed to insert validator registration: %v", err)
		}

	case *core_proto.SignedTransaction_ValidatorDeregistration:
		if err := queries.InsertDecodedValidatorDeregistration(ctx, db.InsertDecodedValidatorDeregistrationParams{
			TxHash:       tx.TxHash,
			CometAddress: t.ValidatorDeregistration.CometAddress,
			PubKey:       t.ValidatorDeregistration.PubKey,
			CreatedAt:    pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
		}); err != nil {
			logger.Errorf("failed to insert validator deregistration: %v", err)
		}

	case *core_proto.SignedTransaction_SlaRollup:
		if err := queries.InsertDecodedSlaRollup(ctx, db.InsertDecodedSlaRollupParams{
			TxHash:     tx.TxHash,
			BlockStart: t.SlaRollup.BlockStart,
			BlockEnd:   t.SlaRollup.BlockEnd,
			Timestamp:  pgtype.Timestamptz{Time: t.SlaRollup.Timestamp.AsTime(), Valid: true},
			CreatedAt:  pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
		}); err != nil {
			logger.Errorf("failed to insert SLA rollup: %v", err)
		}

	case *core_proto.SignedTransaction_StorageProof:
		if err := queries.InsertDecodedStorageProof(ctx, db.InsertDecodedStorageProofParams{
			TxHash:          tx.TxHash,
			Height:          t.StorageProof.Height,
			Address:         t.StorageProof.Address,
			Cid:             pgtype.Text{String: t.StorageProof.Cid, Valid: t.StorageProof.Cid != ""},
			ProofSignature:  t.StorageProof.ProofSignature,
			ProverAddresses: t.StorageProof.ProverAddresses,
			CreatedAt:       pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
		}); err != nil {
			logger.Errorf("failed to insert storage proof: %v", err)
		}

	case *core_proto.SignedTransaction_StorageProofVerification:
		if err := queries.InsertDecodedStorageProofVerification(ctx, db.InsertDecodedStorageProofVerificationParams{
			TxHash:    tx.TxHash,
			Height:    t.StorageProofVerification.Height,
			Proof:     t.StorageProofVerification.Proof,
			CreatedAt: pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
		}); err != nil {
			logger.Errorf("failed to insert storage proof verification: %v", err)
		}

	case *core_proto.SignedTransaction_ManageEntity:
		if err := queries.InsertDecodedManageEntity(ctx, db.InsertDecodedManageEntityParams{
			TxHash:     tx.TxHash,
			UserID:     t.ManageEntity.UserId,
			EntityType: t.ManageEntity.EntityType,
			EntityID:   t.ManageEntity.EntityId,
			Action:     t.ManageEntity.Action,
			Metadata:   t.ManageEntity.Metadata,
			Signature:  t.ManageEntity.Signature,
			Signer:     t.ManageEntity.Signer,
			Nonce:      t.ManageEntity.Nonce,
			CreatedAt:  pgtype.Timestamptz{Time: tx.CreatedAt.Time, Valid: true},
		}); err != nil {
			logger.Errorf("failed to insert manage entity: %v", err)
		}
	}

	return nil
}
