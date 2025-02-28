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

// WriteTx writes a decoded transaction to PostgreSQL with type-specific columns
func (e *ETL) WriteTx(ctx context.Context, tx *DecodedTransaction) error {
	// Start a database transaction
	dbTx, err := e.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback(ctx)

	query := `
		INSERT INTO core_decoded_tx (
			block_height,
			tx_index,
			tx_hash,
			tx_type,
			created_at,
			signature,
			request_id,
			validator_endpoint,
			validator_comet_address,
			validator_eth_block,
			validator_node_type,
			validator_sp_id,
			validator_pub_key,
			validator_power,
			deregistration_comet_address,
			deregistration_pub_key,
			sla_timestamp,
			sla_block_start,
			sla_block_end,
			sla_reports,
			storage_proof_height,
			storage_proof_address,
			storage_proof_prover_addresses,
			storage_proof_cid,
			storage_proof_signature,
			storage_verification_height,
			storage_verification_proof,
			manage_entity_user_id,
			manage_entity_type,
			manage_entity_id,
			manage_entity_action,
			manage_entity_metadata,
			manage_entity_signature,
			manage_entity_signer,
			manage_entity_nonce
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35)
		ON CONFLICT (tx_hash) DO NOTHING
	`

	// Common fields
	args := []interface{}{
		tx.BlockHeight,
		tx.TxIndex,
		tx.TxHash,
		tx.TxType,
		tx.CreatedAt,
		tx.TxData.Signature,
		tx.TxData.RequestId,
	}

	// Initialize all type-specific fields as nil
	var (
		validatorEndpoint       interface{}
		validatorCometAddress   interface{}
		validatorEthBlock       interface{}
		validatorNodeType       interface{}
		validatorSpId           interface{}
		validatorPubKey         interface{}
		validatorPower          interface{}
		deregCometAddress       interface{}
		deregPubKey             interface{}
		slaTimestamp            interface{}
		slaBlockStart           interface{}
		slaBlockEnd             interface{}
		slaReports              interface{}
		storageProofHeight      interface{}
		storageProofAddress     interface{}
		storageProofProverAddrs interface{}
		storageProofCid         interface{}
		storageProofSignature   interface{}
		verificationHeight      interface{}
		verificationProof       interface{}
		manageEntityUserId      interface{}
		manageEntityType        interface{}
		manageEntityId          interface{}
		manageEntityAction      interface{}
		manageEntityMetadata    interface{}
		manageEntitySignature   interface{}
		manageEntitySigner      interface{}
		manageEntityNonce       interface{}
	)

	// Set fields based on transaction type
	var playsToInsert []*core_proto.TrackPlay
	switch t := tx.TxData.Transaction.(type) {
	case *core_proto.SignedTransaction_Plays:
		// Store plays for later insertion
		playsToInsert = t.Plays.Plays

	case *core_proto.SignedTransaction_ValidatorRegistration:
		validatorEndpoint = t.ValidatorRegistration.Endpoint
		validatorCometAddress = t.ValidatorRegistration.CometAddress
		validatorEthBlock = t.ValidatorRegistration.EthBlock
		validatorNodeType = t.ValidatorRegistration.NodeType
		validatorSpId = t.ValidatorRegistration.SpId
		validatorPubKey = t.ValidatorRegistration.PubKey
		validatorPower = t.ValidatorRegistration.Power

	case *core_proto.SignedTransaction_ValidatorDeregistration:
		deregCometAddress = t.ValidatorDeregistration.CometAddress
		deregPubKey = t.ValidatorDeregistration.PubKey

	case *core_proto.SignedTransaction_SlaRollup:
		slaTimestamp = t.SlaRollup.Timestamp.AsTime()
		slaBlockStart = t.SlaRollup.BlockStart
		slaBlockEnd = t.SlaRollup.BlockEnd
		reportsJSON, err := json.Marshal(t.SlaRollup.Reports)
		if err != nil {
			return fmt.Errorf("failed to marshal SLA reports: %w", err)
		}
		slaReports = reportsJSON

	case *core_proto.SignedTransaction_StorageProof:
		storageProofHeight = t.StorageProof.Height
		storageProofAddress = t.StorageProof.Address
		storageProofProverAddrs = t.StorageProof.ProverAddresses
		storageProofCid = t.StorageProof.Cid
		storageProofSignature = t.StorageProof.ProofSignature

	case *core_proto.SignedTransaction_StorageProofVerification:
		verificationHeight = t.StorageProofVerification.Height
		verificationProof = t.StorageProofVerification.Proof

	case *core_proto.SignedTransaction_ManageEntity:
		manageEntityUserId = t.ManageEntity.UserId
		manageEntityType = t.ManageEntity.EntityType
		manageEntityId = t.ManageEntity.EntityId
		manageEntityAction = t.ManageEntity.Action
		manageEntityMetadata = t.ManageEntity.Metadata
		manageEntitySignature = t.ManageEntity.Signature
		manageEntitySigner = t.ManageEntity.Signer
		manageEntityNonce = t.ManageEntity.Nonce
	}

	// Append all type-specific fields
	args = append(args,
		validatorEndpoint,
		validatorCometAddress,
		validatorEthBlock,
		validatorNodeType,
		validatorSpId,
		validatorPubKey,
		validatorPower,
		deregCometAddress,
		deregPubKey,
		slaTimestamp,
		slaBlockStart,
		slaBlockEnd,
		slaReports,
		storageProofHeight,
		storageProofAddress,
		storageProofProverAddrs,
		storageProofCid,
		storageProofSignature,
		verificationHeight,
		verificationProof,
		manageEntityUserId,
		manageEntityType,
		manageEntityId,
		manageEntityAction,
		manageEntityMetadata,
		manageEntitySignature,
		manageEntitySigner,
		manageEntityNonce,
	)

	// Insert the main transaction
	_, err = dbTx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert decoded transaction: %w", err)
	}

	// If this is a plays transaction, insert the plays
	if len(playsToInsert) > 0 {
		playsQuery := `
			INSERT INTO core_decoded_tx_plays (
				tx_hash,
				user_id,
				track_id,
				timestamp,
				signature,
				city,
				region,
				country
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		// Insert each play
		for _, play := range playsToInsert {
			_, err = dbTx.Exec(ctx, playsQuery,
				tx.TxHash,
				play.UserId,
				play.TrackId,
				play.Timestamp.AsTime(),
				play.Signature,
				play.City,
				play.Region,
				play.Country,
			)
			if err != nil {
				return fmt.Errorf("failed to insert play: %w", err)
			}
		}
	}

	// Commit the transaction
	if err := dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
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
