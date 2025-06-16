-- Migration 0017: Dashboard Stats Materialized Views
-- Create materialized views for expensive dashboard statistics to improve performance

-- Materialized view for transaction statistics
CREATE MATERIALIZED VIEW mv_dashboard_transaction_stats AS
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
    COUNT(CASE WHEN b.block_time >= lbt.latest_time - INTERVAL '30 days' THEN 1 END) as total_transactions_30d,
    lbt.latest_time as calculated_at
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
CROSS JOIN latest_block_time lbt
GROUP BY lbt.latest_time;

-- Create unique index for materialized view refresh
CREATE UNIQUE INDEX mv_dashboard_transaction_stats_unique_idx ON mv_dashboard_transaction_stats (calculated_at);

-- Materialized view for transaction type breakdown (last 24h)
CREATE MATERIALIZED VIEW mv_dashboard_transaction_breakdown AS
WITH latest_block_time AS (
    SELECT MAX(block_time) as latest_time
    FROM etl_blocks
)
SELECT 
    t.tx_type as type,
    COUNT(*) as count,
    lbt.latest_time as calculated_at
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id
CROSS JOIN latest_block_time lbt
WHERE b.block_time >= lbt.latest_time - INTERVAL '24 hours'
GROUP BY t.tx_type, lbt.latest_time
ORDER BY count DESC;

-- Create index for the materialized view
CREATE INDEX mv_dashboard_transaction_breakdown_time_idx ON mv_dashboard_transaction_breakdown (calculated_at);

-- Materialized view for validator statistics
CREATE MATERIALIZED VIEW mv_dashboard_validator_stats AS
SELECT 
    COUNT(DISTINCT vr.comet_address) as total_registered_validators,
    COUNT(DISTINCT CASE WHEN vr.comet_address NOT IN (
        SELECT vd.comet_address FROM etl_validator_deregistrations_v2 vd
    ) THEN vr.comet_address END) as active_validators,
    COUNT(DISTINCT vd.comet_address) as deregistered_validators,
    NOW() as calculated_at
FROM etl_validator_registrations_v2 vr
LEFT JOIN etl_validator_deregistrations_v2 vd ON vr.comet_address = vd.comet_address;

-- Create unique index for the materialized view
CREATE UNIQUE INDEX mv_dashboard_validator_stats_unique_idx ON mv_dashboard_validator_stats (calculated_at);

-- Materialized view for network rates (based on latest SLA rollup)
CREATE MATERIALIZED VIEW mv_dashboard_network_rates AS
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
    end_time,
    NOW() as calculated_at
FROM sla_blocks;

-- Create unique index for the materialized view
CREATE UNIQUE INDEX mv_dashboard_network_rates_unique_idx ON mv_dashboard_network_rates (calculated_at);

-- Create a function to refresh all dashboard materialized views
CREATE OR REPLACE FUNCTION refresh_dashboard_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_dashboard_transaction_stats;
    REFRESH MATERIALIZED VIEW mv_dashboard_transaction_breakdown;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_dashboard_validator_stats;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_dashboard_network_rates;
END;
$$ LANGUAGE plpgsql;

-- Add comment for maintenance
COMMENT ON FUNCTION refresh_dashboard_materialized_views() IS 'Refreshes all dashboard materialized views. Should be called every few minutes via cron or trigger.';

-- Initial population of the materialized views
REFRESH MATERIALIZED VIEW mv_dashboard_transaction_stats;
REFRESH MATERIALIZED VIEW mv_dashboard_transaction_breakdown;
REFRESH MATERIALIZED VIEW mv_dashboard_validator_stats;
REFRESH MATERIALIZED VIEW mv_dashboard_network_rates; 
