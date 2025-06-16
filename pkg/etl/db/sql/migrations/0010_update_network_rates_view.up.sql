-- Update network rates view to return proper numeric types
-- Migration 0010: Update network rates view for decimal precision

-- Drop and recreate the view with proper numeric types
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
        THEN ROUND((block_count::float / EXTRACT(EPOCH FROM (end_time - start_time)))::numeric, 2)
        ELSE 0.00::numeric 
    END as blocks_per_second,
    CASE 
        WHEN EXTRACT(EPOCH FROM (end_time - start_time)) > 0 
        THEN ROUND((transaction_count::float / EXTRACT(EPOCH FROM (end_time - start_time)))::numeric, 2)
        ELSE 0.00::numeric 
    END as transactions_per_second,
    block_count,
    transaction_count,
    start_time,
    end_time
FROM sla_blocks; 
