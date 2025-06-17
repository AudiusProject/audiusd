-- Migration 0014: Optimize SLA Rollup Views for Better Performance
-- The current v_sla_rollup and v_sla_rollup_score views are causing 2+ second delays

-- Step 1: Create materialized views for expensive calculations
-- These will be refreshed periodically instead of calculated on every query

-- Optimized SLA rollup view with pre-calculated values
CREATE MATERIALIZED VIEW mv_sla_rollup AS
SELECT 
    sr.id,
    (sr.block_end - sr.block_start + 1) as total_blocks,
    -- Pre-calculate avg block time more efficiently
    COALESCE(
        CASE 
            WHEN sr.block_end > sr.block_start AND block_times.time_diff > 0 THEN
                block_times.time_diff / (sr.block_end - sr.block_start)
            ELSE 0 
        END, 0
    ) as avg_block_time,
    sr.block_start as start_block,
    sr.block_end as end_block,
    -- Pre-calculate block quota
    CASE 
        WHEN validator_counts.validator_count > 0 THEN
            (sr.block_end - sr.block_start + 1) / validator_counts.validator_count
        ELSE 0
    END as block_quota,
    t.tx_hash as tx,
    b.block_time as date_finalized,
    sr.timestamp
FROM etl_sla_rollups_v2 sr
JOIN etl_transactions_v2 t ON sr.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
-- Pre-calculate time differences efficiently with a single query per rollup
LEFT JOIN (
    SELECT 
        sr_times.id,
        EXTRACT(EPOCH FROM (
            MAX(b_times.block_time) - MIN(b_times.block_time)
        ))::float as time_diff
    FROM etl_sla_rollups_v2 sr_times
    JOIN etl_blocks b_times ON b_times.block_height BETWEEN sr_times.block_start AND sr_times.block_end
    GROUP BY sr_times.id
) block_times ON block_times.id = sr.id
-- Pre-calculate validator counts
LEFT JOIN (
    SELECT 
        snr_count.sla_rollup_id,
        COUNT(DISTINCT snr_count.address_id) as validator_count
    FROM etl_sla_node_reports_v2 snr_count
    GROUP BY snr_count.sla_rollup_id
) validator_counts ON validator_counts.sla_rollup_id = sr.id;

-- Create indexes on the materialized view
CREATE UNIQUE INDEX mv_sla_rollup_id_idx ON mv_sla_rollup (id);
CREATE INDEX mv_sla_rollup_timestamp_idx ON mv_sla_rollup (timestamp DESC);
CREATE INDEX mv_sla_rollup_date_finalized_idx ON mv_sla_rollup (date_finalized DESC);

-- Optimized SLA rollup score materialized view
-- This eliminates the expensive CROSS JOIN and multiple subqueries
CREATE MATERIALIZED VIEW mv_sla_rollup_score AS
SELECT 
    snr.num_blocks_proposed as blocks_proposed,
    COALESCE(challenge_stats.challenges_received, 0) as challenges_received,
    COALESCE(challenge_stats.challenges_failed, 0) as challenges_failed,
    sr.id as sla_id,
    a.address as node,
    sr.timestamp
FROM etl_sla_node_reports_v2 snr
JOIN etl_sla_rollups_v2 sr ON snr.sla_rollup_id = sr.id
JOIN etl_addresses a ON snr.address_id = a.id
-- Pre-aggregate challenge statistics more efficiently
LEFT JOIN (
    SELECT 
        sr_challenge.id as sla_rollup_id,
        a_challenge.address,
        COUNT(DISTINCT sp_challenge.height) as challenges_received,
        COUNT(DISTINCT sp_challenge.height) - COUNT(DISTINCT CASE WHEN sp_submitted.height IS NOT NULL THEN sp_challenge.height END) as challenges_failed
    FROM etl_sla_rollups_v2 sr_challenge
    JOIN etl_storage_proofs_v2 sp_challenge ON sp_challenge.height BETWEEN sr_challenge.block_start AND sr_challenge.block_end
    JOIN etl_addresses a_challenge ON a_challenge.address = ANY(sp_challenge.prover_addresses)
    LEFT JOIN etl_storage_proofs_v2 sp_submitted ON sp_submitted.height = sp_challenge.height 
        AND sp_submitted.address_id = a_challenge.id
    GROUP BY sr_challenge.id, a_challenge.address
) challenge_stats ON challenge_stats.sla_rollup_id = sr.id AND challenge_stats.address = a.address;

-- Create indexes on the materialized view
CREATE INDEX mv_sla_rollup_score_sla_id_idx ON mv_sla_rollup_score (sla_id);
CREATE INDEX mv_sla_rollup_score_node_idx ON mv_sla_rollup_score (node);
CREATE INDEX mv_sla_rollup_score_timestamp_idx ON mv_sla_rollup_score (timestamp DESC);
CREATE INDEX mv_sla_rollup_score_node_timestamp_idx ON mv_sla_rollup_score (node, timestamp DESC);

-- Step 2: Replace the slow views with fast ones that use materialized views
DROP VIEW IF EXISTS v_sla_rollup CASCADE;
DROP VIEW IF EXISTS v_sla_rollup_score CASCADE;

-- Fast replacement view for v_sla_rollup
CREATE VIEW v_sla_rollup AS
SELECT 
    id,
    total_blocks,
    avg_block_time,
    start_block,
    end_block,
    block_quota,
    tx,
    date_finalized
FROM mv_sla_rollup
ORDER BY timestamp DESC;

-- Fast replacement view for v_sla_rollup_score  
CREATE VIEW v_sla_rollup_score AS
SELECT 
    blocks_proposed,
    challenges_received,
    challenges_failed,
    sla_id,
    node
FROM mv_sla_rollup_score
ORDER BY timestamp DESC, node;

-- Step 3: Create a function to refresh materialized views
-- This should be called periodically (e.g., every few minutes) via cron job or background task
CREATE OR REPLACE FUNCTION refresh_sla_rollup_materialized_views()
RETURNS void AS $$
BEGIN
    -- Refresh in dependency order
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_sla_rollup;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_sla_rollup_score;
END;
$$ LANGUAGE plpgsql;

-- Step 4: Create a lightweight summary view for dashboard queries
-- This avoids complex JOINs for simple dashboard stats
CREATE MATERIALIZED VIEW mv_sla_rollup_dashboard_stats AS
SELECT 
    sr.id,
    sr.avg_block_time,
    sr.start_block,
    sr.end_block,
    sr.date_finalized,
    sr.timestamp,
    -- Pre-calculate commonly needed stats
    ROW_NUMBER() OVER (ORDER BY sr.timestamp DESC) as rollup_sequence
FROM mv_sla_rollup sr;

-- Index for fast dashboard queries
CREATE UNIQUE INDEX mv_sla_rollup_dashboard_stats_id_idx ON mv_sla_rollup_dashboard_stats (id);
CREATE INDEX mv_sla_rollup_dashboard_stats_timestamp_idx ON mv_sla_rollup_dashboard_stats (timestamp DESC);
CREATE INDEX mv_sla_rollup_dashboard_stats_sequence_idx ON mv_sla_rollup_dashboard_stats (rollup_sequence);

-- Comments for maintenance
COMMENT ON MATERIALIZED VIEW mv_sla_rollup IS 'Optimized materialized view for SLA rollup data. Refresh periodically with refresh_sla_rollup_materialized_views()';
COMMENT ON MATERIALIZED VIEW mv_sla_rollup_score IS 'Optimized materialized view for SLA rollup scores. Refresh periodically with refresh_sla_rollup_materialized_views()';
COMMENT ON MATERIALIZED VIEW mv_sla_rollup_dashboard_stats IS 'Lightweight view for dashboard stats. Refresh with mv_sla_rollup.';
COMMENT ON FUNCTION refresh_sla_rollup_materialized_views() IS 'Call this function periodically to refresh SLA rollup materialized views. Recommended: every 5-10 minutes.'; 
