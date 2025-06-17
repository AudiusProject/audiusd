-- Normalized write queries for ETL database
-- Uses the new schema with proper foreign key relationships

-- Helper function to get or create address
-- name: GetOrCreateAddress :one
INSERT INTO etl_addresses (address, first_seen_block_id)
VALUES ($1, $2)
ON CONFLICT (address) DO UPDATE SET address = EXCLUDED.address
RETURNING id;

-- Update transaction type
-- name: UpdateTransactionType :exec
UPDATE etl_transactions_v2
SET tx_type = $2
WHERE id = $1;

-- insert a new block record (unchanged)
-- name: InsertBlock :one
INSERT INTO etl_blocks (
    proposer_address,
    block_height,
    block_time
) VALUES ($1, $2, $3)
RETURNING *;

-- insert a new transaction record with normalized schema
-- name: InsertTransaction :one
INSERT INTO etl_transactions_v2 (
    tx_hash,
    block_id,
    tx_index,
    tx_type
) VALUES ($1, $2, $3, $4)
ON CONFLICT (tx_hash, block_id) DO NOTHING
RETURNING *;

-- insert a new play record with normalized schema
-- name: InsertPlay :one
INSERT INTO etl_plays_v2 (
    transaction_id,
    address_id,
    track_id,
    city,
    region,
    country,
    played_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- insert multiple play records with normalized schema
-- name: InsertPlays :many
INSERT INTO etl_plays_v2 (
    transaction_id,
    address_id,
    track_id,
    city,
    region,
    country,
    played_at
) VALUES (
    unnest($1::int[]),
    unnest($2::int[]),
    unnest($3::text[]),
    unnest($4::text[]),
    unnest($5::text[]),
    unnest($6::text[]),
    unnest($7::timestamp[])
) ON CONFLICT DO NOTHING
RETURNING *;

-- insert a new manage entity record with normalized schema
-- name: InsertManageEntity :one
INSERT INTO etl_manage_entities_v2 (
    transaction_id,
    address_id,
    entity_type,
    entity_id,
    action,
    metadata,
    signature,
    signer_address_id,
    nonce
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- insert multiple manage entity records with normalized schema
-- name: InsertManageEntities :many
INSERT INTO etl_manage_entities_v2 (
    transaction_id,
    address_id,
    entity_type,
    entity_id,
    action,
    metadata,
    signature,
    signer_address_id,
    nonce
) VALUES (
    unnest($1::int[]),
    unnest($2::int[]),
    unnest($3::text[]),
    unnest($4::bigint[]),
    unnest($5::text[]),
    unnest($6::text[]),
    unnest($7::text[]),
    unnest($8::int[]),
    unnest($9::text[])
) ON CONFLICT DO NOTHING
RETURNING *;

-- insert a new validator registration record with normalized schema
-- name: InsertValidatorRegistration :one
INSERT INTO etl_validator_registrations_v2 (
    transaction_id,
    address_id,
    endpoint,
    comet_address,
    eth_block,
    node_type,
    spid,
    comet_pubkey,
    voting_power
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- insert a new validator deregistration record with normalized schema
-- name: InsertValidatorDeregistration :one
INSERT INTO etl_validator_deregistrations_v2 (
    transaction_id,
    comet_address,
    comet_pubkey
) VALUES ($1, $2, $3)
RETURNING *;

-- insert a new legacy validator registration record with normalized schema
-- name: InsertValidatorRegistrationLegacy :one
INSERT INTO etl_validator_registrations_legacy_v2 (
    transaction_id,
    endpoint,
    comet_address,
    eth_block,
    node_type,
    sp_id,
    pub_key,
    power
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- insert a new SLA rollup record with normalized schema
-- name: InsertSlaRollup :one
INSERT INTO etl_sla_rollups_v2 (
    transaction_id,
    timestamp,
    block_start,
    block_end
) VALUES ($1, $2, $3, $4)
RETURNING *;

-- insert a new SLA node report record with normalized schema
-- name: InsertSlaNodeReport :one
INSERT INTO etl_sla_node_reports_v2 (
    sla_rollup_id,
    address_id,
    num_blocks_proposed
) VALUES ($1, $2, $3)
RETURNING *;

-- insert a new validator misbehavior deregistration record with normalized schema
-- name: InsertValidatorMisbehaviorDeregistration :one
INSERT INTO etl_validator_misbehavior_deregistrations_v2 (
    transaction_id,
    comet_address,
    pub_key
) VALUES ($1, $2, $3)
RETURNING *;

-- insert a new storage proof record with normalized schema
-- name: InsertStorageProof :one
INSERT INTO etl_storage_proofs_v2 (
    transaction_id,
    height,
    address_id,
    prover_addresses,
    cid,
    proof_signature
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- insert a new storage proof verification record with normalized schema
-- name: InsertStorageProofVerification :one
INSERT INTO etl_storage_proof_verifications_v2 (
    transaction_id,
    height,
    proof
) VALUES ($1, $2, $3)
RETURNING *;

-- insert a new release record with normalized schema
-- name: InsertRelease :one
INSERT INTO etl_releases_v2 (
    transaction_id,
    release_data
) VALUES ($1, $2)
RETURNING *;

-- delete plays by transaction IDs (useful for reindexing)
-- name: DeletePlaysByTransactionIds :exec
DELETE FROM etl_plays_v2
WHERE transaction_id = ANY($1::int[]);

-- delete manage entities by transaction IDs (useful for reindexing)
-- name: DeleteManageEntitiesByTransactionIds :exec
DELETE FROM etl_manage_entities_v2
WHERE transaction_id = ANY($1::int[]);

-- delete transactions by block height range (useful for reindexing)
-- name: DeleteTransactionsByBlockRange :exec
DELETE FROM etl_transactions_v2
WHERE block_id IN (
    SELECT id FROM etl_blocks
    WHERE block_height BETWEEN $1 AND $2
);
