-- Normalized read queries for ETL database
-- Uses the new schema with proper JOIN operations for efficiency

-- get latest indexed block height
-- name: GetLatestIndexedBlock :one
SELECT block_height
FROM etl_blocks
ORDER BY id DESC
LIMIT 1;

-- name: GetIndexedBlock :one
SELECT *
FROM etl_blocks
WHERE block_height = $1;

-- name: GetLatestBlocks :many
SELECT *
FROM etl_blocks
ORDER BY block_height DESC
LIMIT $1 OFFSET $2;

-- name: GetTotalBlocksCount :one
SELECT count(*) as total
FROM etl_blocks;

-- Get latest transactions with block info using normalized schema
-- name: GetLatestTransactions :many
SELECT 
    t.id,
    t.tx_hash,
    b.block_height,
    t.tx_index as index,
    t.tx_type,
    b.block_time,
    b.proposer_address,
    t.created_at
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
ORDER BY t.id DESC
LIMIT $1 OFFSET $2;

-- name: GetTotalTransactionsCount :one
SELECT count(*) as total
FROM etl_transactions_v2;

-- name: GetBlockRangeByTime :one
SELECT min(block_height) as start_block,
    max(block_height) as end_block
FROM etl_blocks
WHERE block_time BETWEEN $1 AND $2;

-- Get plays by address using normalized schema
-- name: GetPlaysByAddress :many
SELECT 
    a.address,
    p.track_id,
    EXTRACT(epoch FROM p.played_at)::bigint as timestamp,
    p.city,
    p.country,
    p.region,
    b.block_height,
    t.tx_hash
FROM etl_plays_v2 p
JOIN etl_addresses a ON p.address_id = a.id
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
WHERE a.address = $1
    AND b.block_height BETWEEN $2 AND $3
ORDER BY p.played_at DESC
LIMIT $4 OFFSET $5;

-- Get plays by track using normalized schema
-- name: GetPlaysByTrack :many
SELECT 
    a.address,
    p.track_id,
    EXTRACT(epoch FROM p.played_at)::bigint as timestamp,
    p.city,
    p.country,
    p.region,
    b.block_height,
    t.tx_hash
FROM etl_plays_v2 p
JOIN etl_addresses a ON p.address_id = a.id
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
WHERE p.track_id = $1
    AND b.block_height BETWEEN $2 AND $3
ORDER BY p.played_at DESC
LIMIT $4 OFFSET $5;

-- Get all plays using normalized schema
-- name: GetPlays :many
SELECT 
    a.address,
    p.track_id,
    EXTRACT(epoch FROM p.played_at)::bigint as timestamp,
    p.city,
    p.country,
    p.region,
    b.block_height,
    t.tx_hash
FROM etl_plays_v2 p
JOIN etl_addresses a ON p.address_id = a.id
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
WHERE b.block_height BETWEEN $1 AND $2
ORDER BY p.played_at DESC
LIMIT $3 OFFSET $4;

-- Get total count of plays with filtering (normalized schema)
-- name: GetPlaysCount :one
SELECT count(*) as total
FROM etl_plays_v2 p
JOIN etl_addresses a ON p.address_id = a.id
WHERE ($1::text IS NULL OR a.address = $1)
    AND ($2::text IS NULL OR p.track_id = $2)
    AND ($3::timestamp IS NULL OR $4::timestamp IS NULL OR p.played_at BETWEEN $3 AND $4);

-- Get play count by track
-- name: GetPlayCountByTrack :one
SELECT count(*) as play_count
FROM etl_plays_v2
WHERE track_id = $1;

-- Get play count by address
-- name: GetPlayCountByAddress :one
SELECT count(*) as play_count
FROM etl_plays_v2 p
JOIN etl_addresses a ON p.address_id = a.id
WHERE a.address = $1;

-- Get validator registrations using normalized schema
-- name: GetValidatorRegistrations :many
SELECT DISTINCT ON (a.address) 
    a.address,
    vr.endpoint,
    vr.comet_address,
    vr.comet_pubkey,
    vr.eth_block,
    vr.node_type,
    vr.spid,
    vr.voting_power,
    b.block_height,
    t.tx_hash
FROM etl_validator_registrations_v2 vr
JOIN etl_addresses a ON vr.address_id = a.id
JOIN etl_transactions_v2 t ON vr.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
WHERE ($1::text IS NULL OR LOWER(vr.endpoint) LIKE '%' || LOWER($1) || '%')
ORDER BY a.address, b.block_height DESC;

-- Get validator deregistrations using normalized schema
-- name: GetValidatorDeregistrations :many
SELECT 
    vd.comet_address,
    vd.comet_pubkey,
    b.block_height,
    t.tx_hash
FROM etl_validator_deregistrations_v2 vd
JOIN etl_transactions_v2 t ON vd.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id;

-- Get plays by location using normalized schema
-- name: GetPlaysByLocation :many
SELECT 
    t.tx_hash,
    a.address,
    p.track_id,
    p.played_at,
    p.city,
    p.region,
    p.country,
    p.played_at as created_at
FROM etl_plays_v2 p
JOIN etl_addresses a ON p.address_id = a.id
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
WHERE (NULLIF($1, '')::text IS NULL OR LOWER(p.city) = LOWER($1))
    AND (NULLIF($2, '')::text IS NULL OR LOWER(p.region) = LOWER($2))
    AND (NULLIF($3, '')::text IS NULL OR LOWER(p.country) = LOWER($3))
ORDER BY p.played_at DESC
LIMIT $4;

-- Get available cities using normalized schema
-- name: GetAvailableCities :many
SELECT 
    p.city,
    p.region,
    p.country,
    count(*) as play_count
FROM etl_plays_v2 p
WHERE p.city IS NOT NULL
    AND (NULLIF($1, '')::text IS NULL OR LOWER(p.country) = LOWER($1))
    AND (NULLIF($2, '')::text IS NULL OR LOWER(p.region) = LOWER($2))
GROUP BY p.city, p.region, p.country
ORDER BY count(*) DESC
LIMIT $3;

-- Get available regions using normalized schema
-- name: GetAvailableRegions :many
SELECT 
    p.region,
    p.country,
    count(*) as play_count
FROM etl_plays_v2 p
WHERE p.region IS NOT NULL
    AND (NULLIF($1, '')::text IS NULL OR LOWER(p.country) = LOWER($1))
GROUP BY p.region, p.country
ORDER BY count(*) DESC
LIMIT $2;

-- Get available countries using normalized schema
-- name: GetAvailableCountries :many
SELECT 
    p.country,
    count(*) as play_count
FROM etl_plays_v2 p
WHERE p.country IS NOT NULL
GROUP BY p.country
ORDER BY count(*) DESC
LIMIT $1;

-- Get block transactions using normalized schema
-- name: GetBlockTransactions :many
SELECT 
    t.tx_hash,
    t.tx_type,
    t.tx_index as index
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
WHERE b.block_height = $1
ORDER BY t.tx_index;

-- Get transaction using normalized schema
-- name: GetTransaction :one
SELECT 
    t.tx_hash,
    t.tx_type,
    b.block_height,
    t.tx_index as index,
    b.block_time,
    b.proposer_address
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
WHERE t.tx_hash = $1
ORDER BY b.block_height DESC, t.tx_index DESC
LIMIT 1;

-- Get blocks using normalized schema
-- name: GetBlocks :many
SELECT *
FROM etl_blocks
WHERE block_height BETWEEN $1 AND $2
ORDER BY block_height DESC;

-- Get transactions count in block range
-- name: GetTransactionsCount :one
SELECT count(*) as total
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
WHERE b.block_height BETWEEN $1 AND $2;

-- Get transactions count in time range
-- name: GetTransactionsCountTimeRange :one
SELECT count(*) as total
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
WHERE b.block_time BETWEEN $1 AND $2;

-- Get latest SLA rollup using normalized schema
-- name: GetLatestSLARollup :one
SELECT 
    sr.timestamp,
    sr.block_start,
    sr.block_end,
    b.block_height,
    t.tx_hash
FROM etl_sla_rollups_v2 sr
JOIN etl_transactions_v2 t ON sr.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
ORDER BY sr.timestamp DESC
LIMIT 1;

-- Get active validators count
-- name: GetActiveValidatorsCount :one
SELECT count(DISTINCT vr.comet_address) as total
FROM etl_validator_registrations_v2 vr
WHERE vr.comet_address NOT IN (
    SELECT DISTINCT vd.comet_address 
    FROM etl_validator_deregistrations_v2 vd
);

-- Get transaction type breakdown
-- name: GetTransactionTypeBreakdown :many
SELECT 
    t.tx_type as type,
    count(*) as count
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
WHERE b.block_time BETWEEN $1 AND $2
GROUP BY t.tx_type
ORDER BY count(*) DESC;

-- Get plays by tx hash using normalized schema
-- name: GetPlaysByTxHash :many
SELECT 
    a.address,
    p.track_id,
    EXTRACT(epoch FROM p.played_at)::bigint as timestamp,
    p.city,
    p.region,
    p.country
FROM etl_plays_v2 p
JOIN etl_addresses a ON p.address_id = a.id
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Get manage entities by tx hash using normalized schema
-- name: GetManageEntitiesByTxHash :many
SELECT 
    a.address,
    me.entity_type,
    me.entity_id,
    me.action,
    me.metadata,
    me.signature,
    sa.address as signer,
    me.nonce
FROM etl_manage_entities_v2 me
JOIN etl_addresses a ON me.address_id = a.id
JOIN etl_addresses sa ON me.signer_address_id = sa.id
JOIN etl_transactions_v2 t ON me.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Get validator registrations by tx hash using normalized schema
-- name: GetValidatorRegistrationsByTxHash :many
SELECT 
    a.address,
    vr.comet_address,
    vr.eth_block,
    vr.node_type,
    vr.spid,
    vr.comet_pubkey,
    vr.voting_power
FROM etl_validator_registrations_v2 vr
JOIN etl_addresses a ON vr.address_id = a.id
JOIN etl_transactions_v2 t ON vr.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Get validator deregistrations by tx hash using normalized schema
-- name: GetValidatorDeregistrationsByTxHash :many
SELECT 
    vd.comet_address,
    vd.comet_pubkey
FROM etl_validator_deregistrations_v2 vd
JOIN etl_transactions_v2 t ON vd.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Get SLA rollups by tx hash using normalized schema
-- name: GetSlaRollupsByTxHash :many
SELECT 
    sr.timestamp,
    sr.block_start,
    sr.block_end
FROM etl_sla_rollups_v2 sr
JOIN etl_transactions_v2 t ON sr.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Get SLA node reports by tx hash using normalized schema
-- name: GetSlaNodeReportsByTxHash :many
SELECT 
    a.address,
    snr.num_blocks_proposed
FROM etl_sla_node_reports_v2 snr
JOIN etl_addresses a ON snr.address_id = a.id
JOIN etl_sla_rollups_v2 sr ON snr.sla_rollup_id = sr.id
JOIN etl_transactions_v2 t ON sr.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Search functionality using normalized schema
-- name: SearchUnified :many
SELECT 
    'transaction' as type,
    t.tx_hash as id,
    'Transaction ' || SUBSTRING(t.tx_hash, 1, 8) || '...' as title,
    t.tx_type || ' at block ' || b.block_height as subtitle
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
WHERE t.tx_hash ILIKE '%' || $1 || '%'
UNION ALL
SELECT 
    'block' as type,
    b.block_height::text as id,
    'Block ' || b.block_height as title,
    'Proposed by ' || SUBSTRING(b.proposer_address, 1, 8) || '...' as subtitle
FROM etl_blocks b
WHERE b.block_height::text ILIKE '%' || $1 || '%'
UNION ALL
SELECT 
    'account' as type,
    a.address as id,
    SUBSTRING(a.address, 1, 8) || '...' as title,
    'Address' as subtitle
FROM etl_addresses a
WHERE a.address ILIKE '%' || $1 || '%'
ORDER BY type, id
LIMIT $2;

-- Search addresses
-- name: SearchAddress :many
SELECT DISTINCT address
FROM etl_addresses
WHERE address ILIKE '%' || $1 || '%'
LIMIT 10;

-- Search validator registrations
-- name: SearchValidatorRegistration :many
SELECT DISTINCT a.address
FROM etl_validator_registrations_v2 vr
JOIN etl_addresses a ON vr.address_id = a.id
WHERE a.address ILIKE '%' || $1 || '%'
   OR vr.comet_address ILIKE '%' || $1 || '%'
   OR vr.endpoint ILIKE '%' || $1 || '%'
LIMIT 10;

-- Get relation types by address (placeholder for compatibility)
-- name: GetRelationTypesByAddress :many
SELECT DISTINCT me.entity_type as relation_type
FROM etl_manage_entities_v2 me
JOIN etl_addresses a ON me.address_id = a.id
WHERE a.address = $1;

-- Get storage proofs by tx hash using normalized schema
-- name: GetStorageProofsByTxHash :many
SELECT 
    a.address,
    sp.height,
    sp.prover_addresses,
    sp.cid,
    sp.proof_signature
FROM etl_storage_proofs_v2 sp
JOIN etl_addresses a ON sp.address_id = a.id
JOIN etl_transactions_v2 t ON sp.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Get storage proof verifications by tx hash using normalized schema
-- name: GetStorageProofVerificationsByTxHash :many
SELECT 
    spv.height,
    spv.proof
FROM etl_storage_proof_verifications_v2 spv
JOIN etl_transactions_v2 t ON spv.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Get releases by tx hash using normalized schema
-- name: GetReleasesByTxHash :many
SELECT 
    r.release_data
FROM etl_releases_v2 r
JOIN etl_transactions_v2 t ON r.transaction_id = t.id
WHERE t.tx_hash = $1;

-- Get transactions by address using normalized schema (complex CTE query)
-- name: GetTransactionsByAddress :many
WITH address_transactions AS (
    -- Play transactions
    SELECT 
        t.tx_hash,
        t.tx_type,
        b.block_height,
        t.tx_index as index,
        a.address,
        'play' as relation_type,
        b.block_time
    FROM etl_transactions_v2 t
    JOIN etl_blocks b ON t.block_id = b.id
    JOIN etl_plays_v2 p ON p.transaction_id = t.id
    JOIN etl_addresses a ON p.address_id = a.id
    WHERE LOWER(a.address) = LOWER($1)
    
    UNION ALL
    
    -- Manage entity transactions
    SELECT 
        t.tx_hash,
        t.tx_type,
        b.block_height,
        t.tx_index as index,
        a.address,
        me.action || '_' || me.entity_type as relation_type,
        b.block_time
    FROM etl_transactions_v2 t
    JOIN etl_blocks b ON t.block_id = b.id
    JOIN etl_manage_entities_v2 me ON me.transaction_id = t.id
    JOIN etl_addresses a ON me.address_id = a.id
    WHERE LOWER(a.address) = LOWER($1)
    
    UNION ALL
    
    -- Validator registration transactions
    SELECT 
        t.tx_hash,
        t.tx_type,
        b.block_height,
        t.tx_index as index,
        a.address,
        'validator_registration' as relation_type,
        b.block_time
    FROM etl_transactions_v2 t
    JOIN etl_blocks b ON t.block_id = b.id
    JOIN etl_validator_registrations_v2 vr ON vr.transaction_id = t.id
    JOIN etl_addresses a ON vr.address_id = a.id
    WHERE LOWER(a.address) = LOWER($1)
    
    UNION ALL
    
    -- Validator deregistration transactions (by comet_address)
    SELECT 
        t.tx_hash,
        t.tx_type,
        b.block_height,
        t.tx_index as index,
        vd.comet_address as address,
        'validator_deregistration' as relation_type,
        b.block_time
    FROM etl_transactions_v2 t
    JOIN etl_blocks b ON t.block_id = b.id
    JOIN etl_validator_deregistrations_v2 vd ON vd.transaction_id = t.id
    WHERE LOWER(vd.comet_address) = LOWER($1)
    
    UNION ALL
    
    -- Storage proof transactions
    SELECT 
        t.tx_hash,
        t.tx_type,
        b.block_height,
        t.tx_index as index,
        a.address,
        'storage_proof' as relation_type,
        b.block_time
    FROM etl_transactions_v2 t
    JOIN etl_blocks b ON t.block_id = b.id
    JOIN etl_storage_proofs_v2 sp ON sp.transaction_id = t.id
    JOIN etl_addresses a ON sp.address_id = a.id
    WHERE LOWER(a.address) = LOWER($1)
    
    UNION ALL
    
    -- SLA node report transactions
    SELECT 
        t.tx_hash,
        t.tx_type,
        b.block_height,
        t.tx_index as index,
        a.address,
        'sla_node_report' as relation_type,
        b.block_time
    FROM etl_transactions_v2 t
    JOIN etl_blocks b ON t.block_id = b.id
    JOIN etl_sla_rollups_v2 sr ON sr.transaction_id = t.id
    JOIN etl_sla_node_reports_v2 snr ON snr.sla_rollup_id = sr.id
    JOIN etl_addresses a ON snr.address_id = a.id
    WHERE LOWER(a.address) = LOWER($1)
)
SELECT 
    tx_hash,
    tx_type,
    block_height,
    index,
    address,
    relation_type,
    block_time
FROM address_transactions
WHERE ($4::text = '' OR relation_type = $4)
    AND ($5::timestamp IS NULL OR block_time >= $5)
    AND ($6::timestamp IS NULL OR block_time <= $6)
ORDER BY block_height DESC, index DESC
LIMIT $2 OFFSET $3;

-- Statistics queries using PostgreSQL views
-- These queries leverage database views for efficient stats calculation

-- Get overall transaction statistics
-- name: GetTransactionStats :one
SELECT * FROM v_transaction_stats;

-- Get transaction type breakdown for last 24h
-- name: GetTransactionTypeBreakdown24h :many
SELECT * FROM v_transaction_type_breakdown_24h;

-- Get plays statistics
-- name: GetPlaysStats :one
SELECT * FROM v_plays_stats;

-- Get validator statistics
-- name: GetValidatorStats :one
SELECT * FROM v_validator_stats;

-- Get network rates (BPS/TPS) based on latest SLA rollup
-- name: GetNetworkRates :one
SELECT 
    COALESCE(ROUND(blocks_per_second)::int, 0) as blocks_per_second,
    COALESCE(ROUND(transactions_per_second)::int, 0) as transactions_per_second,
    COALESCE(block_count, 0) as block_count,
    COALESCE(transaction_count, 0) as transaction_count,
    start_time,
    end_time
FROM v_network_rates;

-- Get latest block information
-- name: GetLatestBlockInfo :one
SELECT * FROM v_latest_block_info;

-- Get top tracks in last 24h
-- name: GetTopTracks24h :many
SELECT * FROM v_top_tracks_24h
LIMIT $1;

-- Get geographic distribution of plays
-- name: GetPlaysLocationDistribution24h :many
SELECT * FROM v_plays_by_location_24h
LIMIT $1;

-- Get entity type statistics for last 24h
-- name: GetEntityTypeStats24h :many
SELECT * FROM v_entity_type_stats_24h;

-- Get sync status by comparing latest indexed vs chain height
-- name: GetSyncStatus :one
SELECT 
    lbi.latest_indexed_height,
    lbi.latest_block_time,
    CASE 
        WHEN lbi.latest_indexed_height < $1 - 100 THEN true 
        ELSE false 
    END as is_syncing,
    $1 as latest_chain_height,
    $1 - lbi.latest_indexed_height as block_delta
FROM v_latest_block_info lbi;
