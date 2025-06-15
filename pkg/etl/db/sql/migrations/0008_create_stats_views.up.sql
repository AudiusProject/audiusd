-- Create PostgreSQL views for statistics calculations
-- Migration 0008: Stats Views (using latest block timestamp as reference)

-- View for transaction counts by time periods (relative to latest block)
CREATE OR REPLACE VIEW v_transaction_stats AS
WITH latest_block_time AS (
    SELECT MAX(block_time) as latest_time
    FROM etl_blocks
)
SELECT 
    COUNT(*) as total_transactions,
    COUNT(CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '24 hours' THEN 1 END) as total_transactions_24h,
    COUNT(CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '48 hours' 
               AND b.block_time < lbt.latest_time - INTERVAL '24 hours' THEN 1 END) as total_transactions_previous_24h,
    COUNT(CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '7 days' THEN 1 END) as total_transactions_7d,
    COUNT(CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '30 days' THEN 1 END) as total_transactions_30d
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
CROSS JOIN latest_block_time lbt;

-- View for transaction type breakdown (last 24 hours relative to latest block)
CREATE OR REPLACE VIEW v_transaction_type_breakdown_24h AS
WITH latest_block_time AS (
    SELECT MAX(block_time) as latest_time
    FROM etl_blocks
)
SELECT 
    t.tx_type as type,
    COUNT(*) as count
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
CROSS JOIN latest_block_time lbt
WHERE b.block_time >= lbt.latest_time - INTERVAL '24 hours'
GROUP BY t.tx_type
ORDER BY count DESC;

-- View for total plays statistics (relative to latest block)
CREATE OR REPLACE VIEW v_plays_stats AS
WITH latest_block_time AS (
    SELECT MAX(block_time) as latest_time
    FROM etl_blocks
)
SELECT 
    COUNT(*) as total_plays,
    COUNT(CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '24 hours' THEN 1 END) as total_plays_24h,
    COUNT(CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '7 days' THEN 1 END) as total_plays_7d,
    COUNT(CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '30 days' THEN 1 END) as total_plays_30d,
    COUNT(DISTINCT p.address_id) as unique_players_all_time,
    COUNT(DISTINCT CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '24 hours' THEN p.address_id END) as unique_players_24h
FROM etl_plays_v2 p
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
CROSS JOIN latest_block_time lbt;

-- View for validator statistics
CREATE OR REPLACE VIEW v_validator_stats AS
SELECT 
    COUNT(DISTINCT vr.comet_address) as total_registered_validators,
    COUNT(DISTINCT CASE WHEN vr.comet_address NOT IN (
        SELECT vd.comet_address FROM etl_validator_deregistrations_v2 vd
    ) THEN vr.comet_address END) as active_validators,
    COUNT(DISTINCT vd.comet_address) as deregistered_validators
FROM etl_validator_registrations_v2 vr
LEFT JOIN etl_validator_deregistrations_v2 vd ON vr.comet_address = vd.comet_address;

-- View for block and transaction rates (based on latest SLA rollup)
CREATE OR REPLACE VIEW v_network_rates AS
WITH latest_sla AS (
    SELECT 
        sr.block_start,
        sr.block_end,
        sr.timestamp as sla_timestamp
    FROM etl_sla_rollups_v2 sr
    JOIN etl_transactions_v2 t ON sr.transaction_id = t.id
    JOIN etl_blocks b ON t.block_id = b.id
    ORDER BY sr.timestamp DESC
    LIMIT 1
),
sla_blocks AS (
    SELECT 
        COUNT(*) as block_count,
        MIN(b.block_time) as start_time,
        MAX(b.block_time) as end_time,
        COUNT(t.id) as transaction_count
    FROM etl_blocks b
    JOIN latest_sla ls ON b.block_height BETWEEN ls.block_start AND ls.block_end
    LEFT JOIN etl_transactions_v2 t ON t.block_id = b.id
)
SELECT 
    CASE 
        WHEN EXTRACT(EPOCH FROM (end_time - start_time)) > 0 
        THEN block_count::float / EXTRACT(EPOCH FROM (end_time - start_time))
        ELSE 0 
    END as blocks_per_second,
    CASE 
        WHEN EXTRACT(EPOCH FROM (end_time - start_time)) > 0 
        THEN transaction_count::float / EXTRACT(EPOCH FROM (end_time - start_time))
        ELSE 0 
    END as transactions_per_second,
    block_count,
    transaction_count,
    start_time,
    end_time
FROM sla_blocks;

-- View for latest block information
CREATE OR REPLACE VIEW v_latest_block_info AS
SELECT 
    block_height as latest_indexed_height,
    block_time as latest_block_time,
    proposer_address as latest_proposer
FROM etl_blocks
ORDER BY block_height DESC
LIMIT 1;

-- View for top tracks by play count (last 24h relative to latest block)
CREATE OR REPLACE VIEW v_top_tracks_24h AS
WITH latest_block_time AS (
    SELECT MAX(block_time) as latest_time
    FROM etl_blocks
)
SELECT 
    p.track_id,
    COUNT(*) as play_count,
    COUNT(DISTINCT p.address_id) as unique_players
FROM etl_plays_v2 p
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
CROSS JOIN latest_block_time lbt
WHERE b.block_time >= lbt.latest_time - INTERVAL '24 hours'
GROUP BY p.track_id
ORDER BY play_count DESC
LIMIT 100;

-- View for geographic distribution of plays (last 24h relative to latest block)
CREATE OR REPLACE VIEW v_plays_by_location_24h AS
WITH latest_block_time AS (
    SELECT MAX(block_time) as latest_time
    FROM etl_blocks
)
SELECT 
    p.country,
    p.region,
    p.city,
    COUNT(*) as play_count
FROM etl_plays_v2 p
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
CROSS JOIN latest_block_time lbt
WHERE b.block_time >= lbt.latest_time - INTERVAL '24 hours'
    AND p.country IS NOT NULL
GROUP BY p.country, p.region, p.city
ORDER BY play_count DESC
LIMIT 1000;

-- View for entity type distribution (manage entities, last 24h relative to latest block)
CREATE OR REPLACE VIEW v_entity_type_stats_24h AS
WITH latest_block_time AS (
    SELECT MAX(block_time) as latest_time
    FROM etl_blocks
)
SELECT 
    me.entity_type,
    me.action,
    COUNT(*) as count
FROM etl_manage_entities_v2 me
JOIN etl_transactions_v2 t ON me.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
CROSS JOIN latest_block_time lbt
WHERE b.block_time >= lbt.latest_time - INTERVAL '24 hours'
GROUP BY me.entity_type, me.action
ORDER BY count DESC; 
