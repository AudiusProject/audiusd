-- Revert network rates view back to original float types
-- Migration 0010 Down: Revert network rates view

-- Drop and recreate the view with original float types
DROP VIEW IF EXISTS v_network_rates;

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
