-- Migration 0015: Fix validator uptime summary to work with materialized views
-- The v_validator_uptime_summary view was broken after migration 0014 replaced the views it depends on

-- Drop the old view that depends on the replaced views
DROP VIEW IF EXISTS v_validator_uptime_summary;

-- Create an updated view that works with materialized views from migration 0014
-- This provides a lightweight summary for dashboard and monitoring
CREATE OR REPLACE VIEW v_validator_uptime_summary AS
WITH recent_rollups AS (
    -- Get the last 5 rollups from the materialized view
    SELECT 
        id, 
        start_block, 
        end_block, 
        date_finalized, 
        block_quota,
        avg_block_time
    FROM mv_sla_rollup 
    ORDER BY timestamp DESC 
    LIMIT 5
),
sla_status AS (
    SELECT 
        mvsr.node,
        rr.id as rollup_id,
        rr.date_finalized,
        mvsr.blocks_proposed,
        rr.block_quota,
        mvsr.challenges_received,
        mvsr.challenges_failed,
        -- Calculate SLA pass/fail status
        CASE 
            WHEN mvsr.blocks_proposed = 0 THEN 'offline'
            WHEN rr.block_quota > 0 AND mvsr.challenges_received >= 0 THEN
                CASE 
                    WHEN (mvsr.blocks_proposed::float / rr.block_quota >= 0.8) 
                         AND (CASE WHEN mvsr.challenges_received > 0 
                                  THEN (1.0 - (mvsr.challenges_failed::float / mvsr.challenges_received)) >= 0.8 
                                  ELSE true END) 
                    THEN 'pass'
                    ELSE 'fail'
                END
            ELSE 'unknown'
        END as sla_status
    FROM mv_sla_rollup_score mvsr
    JOIN recent_rollups rr ON mvsr.sla_id = rr.id
)
SELECT 
    node,
    rollup_id,
    date_finalized,
    sla_status,
    -- For debugging/tooltips, include basic metrics
    blocks_proposed,
    block_quota,
    challenges_received,
    challenges_failed
FROM sla_status
ORDER BY date_finalized DESC, node;

-- Add comment for maintenance
COMMENT ON VIEW v_validator_uptime_summary IS 'Lightweight validator uptime summary using materialized views. Refreshes automatically when mv_sla_rollup and mv_sla_rollup_score are refreshed.'; 
