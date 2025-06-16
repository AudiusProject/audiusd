-- Migration 0014 Down: Revert SLA Rollup View Optimizations

-- Drop the new optimized structures
DROP FUNCTION IF EXISTS refresh_sla_rollup_materialized_views();
DROP MATERIALIZED VIEW IF EXISTS mv_sla_rollup_dashboard_stats;
DROP VIEW IF EXISTS v_sla_rollup CASCADE;
DROP VIEW IF EXISTS v_sla_rollup_score CASCADE;
DROP MATERIALIZED VIEW IF EXISTS mv_sla_rollup_score;
DROP MATERIALIZED VIEW IF EXISTS mv_sla_rollup;

-- Restore the original views from migration 0011
-- SLA rollup view with rollup-level aggregated data
CREATE OR REPLACE VIEW v_sla_rollup AS
SELECT 
    sr.id,
    (sr.block_end - sr.block_start + 1) as total_blocks,
    CASE 
        WHEN sr.block_end > sr.block_start THEN
            EXTRACT(EPOCH FROM (
                (SELECT MAX(b.block_time) FROM etl_blocks b WHERE b.block_height BETWEEN sr.block_start AND sr.block_end) -
                (SELECT MIN(b.block_time) FROM etl_blocks b WHERE b.block_height BETWEEN sr.block_start AND sr.block_end)
            ))::float / (sr.block_end - sr.block_start)
        ELSE 0 
    END as avg_block_time,
    sr.block_start as start_block,
    sr.block_end as end_block,
    -- Calculate block quota as (total blocks) / (number of active validators at the time)
    CASE 
        WHEN (
            SELECT COUNT(DISTINCT snr.address_id) 
            FROM etl_sla_node_reports_v2 snr 
            WHERE snr.sla_rollup_id = sr.id
        ) > 0 THEN
            (sr.block_end - sr.block_start + 1) / (
                SELECT COUNT(DISTINCT snr.address_id) 
                FROM etl_sla_node_reports_v2 snr 
                WHERE snr.sla_rollup_id = sr.id
            )
        ELSE 0
    END as block_quota,
    t.tx_hash as tx,
    b.block_time as date_finalized
FROM etl_sla_rollups_v2 sr
JOIN etl_transactions_v2 t ON sr.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id
ORDER BY sr.timestamp DESC;

-- SLA rollup score view with validator-specific data for each rollup
CREATE OR REPLACE VIEW v_sla_rollup_score AS
SELECT 
    snr.num_blocks_proposed as blocks_proposed,
    -- Challenges received: count distinct storage proof heights where this validator was expected to provide proof in this rollup period
    COALESCE(challenge_counts.challenges_received, 0) as challenges_received,
    -- Challenges failed: challenges received minus challenges where they actually provided a proof
    COALESCE(challenge_counts.challenges_failed, 0) as challenges_failed,
    sr.id as sla_id,
    a.address as node
FROM etl_sla_node_reports_v2 snr
JOIN etl_sla_rollups_v2 sr ON snr.sla_rollup_id = sr.id
JOIN etl_addresses a ON snr.address_id = a.id
LEFT JOIN (
    -- For each rollup and validator, count challenges received and failed
    SELECT 
        sr_inner.id as sla_rollup_id,
        a_inner.address,
        -- Count distinct heights where this address appears in prover_addresses (expected to prove)
        COUNT(DISTINCT sp.height) as challenges_received,
        -- Count distinct heights where they were expected but didn't submit a proof
        COUNT(DISTINCT sp.height) - COUNT(DISTINCT sp_submitted.height) as challenges_failed
    FROM etl_sla_rollups_v2 sr_inner
    CROSS JOIN etl_addresses a_inner
    -- Find storage proofs in this rollup's block range where this address was expected to provide proof
    JOIN etl_storage_proofs_v2 sp ON sp.height BETWEEN sr_inner.block_start AND sr_inner.block_end 
        AND a_inner.address = ANY(sp.prover_addresses)
    -- Check if this address actually submitted a proof at that height
    LEFT JOIN etl_storage_proofs_v2 sp_submitted ON sp_submitted.height = sp.height 
        AND sp_submitted.address_id = a_inner.id
    GROUP BY sr_inner.id, a_inner.address
    HAVING COUNT(DISTINCT sp.height) > 0  -- Only include addresses that had challenges
) challenge_counts ON challenge_counts.sla_rollup_id = sr.id AND challenge_counts.address = a.address
ORDER BY sr.timestamp DESC, a.address; 
