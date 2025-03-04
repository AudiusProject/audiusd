package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/gen/core_proto"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ETL handles writing decoded transaction data to PostgreSQL during FinalizeBlock
type ETL struct {
	pool   *pgxpool.Pool
	logger *common.Logger
}

// NewETL creates a new ETL instance
func NewETL(pool *pgxpool.Pool, logger *common.Logger) *ETL {
	return &ETL{
		pool:   pool,
		logger: logger.Child("etl"),
	}
}

// DecodedTransaction represents a decoded transaction with its metadata
type DecodedTransaction struct {
	BlockHeight int64
	TxIndex     int32
	TxHash      string
	TxType      string
	TxData      *core_proto.SignedTransaction
	CreatedAt   time.Time
}

// WriteTx writes a decoded transaction to PostgreSQL
func (e *ETL) WriteTx(ctx context.Context, tx *DecodedTransaction) error {
	// Convert the protobuf message to JSON
	txDataJSON, err := json.Marshal(tx.TxData)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction data: %w", err)
	}

	// Write to core_tx_decoded
	query := `
		INSERT INTO core_tx_decoded (
			block_height,
			tx_index,
			tx_hash,
			tx_type,
			tx_data,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tx_hash) DO NOTHING
	`

	_, err = e.pool.Exec(ctx, query,
		tx.BlockHeight,
		tx.TxIndex,
		tx.TxHash,
		tx.TxType,
		txDataJSON,
		tx.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert decoded transaction: %w", err)
	}

	// If this is a play transaction, also write to core_tx_decoded_plays
	if plays := tx.TxData.GetPlays(); plays != nil {
		// Insert each play into core_tx_decoded_plays
		playsQuery := `
			INSERT INTO core_tx_decoded_plays (
				tx_hash,
				user_id,
				track_id,
				played_at,
				signature,
				city,
				region,
				country,
				created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (tx_hash, user_id, track_id) DO NOTHING
		`

		for _, play := range plays.Plays {
			_, err = e.pool.Exec(ctx, playsQuery,
				tx.TxHash,
				play.UserId,
				play.TrackId,
				play.Timestamp.AsTime(),
				play.Signature,
				play.City,
				play.Region,
				play.Country,
				tx.CreatedAt,
			)
			if err != nil {
				e.logger.Error("failed to insert play record", "error", err)
				// Continue inserting other plays even if one fails
				continue
			}
		}
	}

	return nil
}

// GetProtoTypeName returns the protobuf type name for a transaction
func GetProtoTypeName(tx *core_proto.SignedTransaction) string {
	switch tx.Transaction.(type) {
	case *core_proto.SignedTransaction_Plays:
		return "Plays"
	case *core_proto.SignedTransaction_ValidatorRegistration:
		return "ValidatorRegistration"
	case *core_proto.SignedTransaction_ValidatorDeregistration:
		return "ValidatorDeregistration"
	case *core_proto.SignedTransaction_SlaRollup:
		return "SlaRollup"
	case *core_proto.SignedTransaction_StorageProof:
		return "StorageProof"
	case *core_proto.SignedTransaction_StorageProofVerification:
		return "StorageProofVerification"
	case *core_proto.SignedTransaction_ManageEntity:
		return "ManageEntity"
	default:
		return "Unknown"
	}
}
