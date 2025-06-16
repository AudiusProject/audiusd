-- Create an efficient view for validator uptime summary (just pass/fail status)
-- This is much faster than fetching all detailed SLA rollup data
CREATE OR REPLACE VIEW v_validator_uptime_summary AS
WITH recent_rollups AS (
    -- Get the last 5 rollups
    SELECT id, start_block, end_block, date_finalized, block_quota 
    FROM v_sla_rollup 
    ORDER BY date_finalized DESC 
    LIMIT 5
),
sla_status AS (
    SELECT 
        vsr.node,
        rr.id as rollup_id,
        rr.date_finalized,
        vsr.blocks_proposed,
        rr.block_quota,
        vsr.challenges_received,
        vsr.challenges_failed,
        -- Calculate SLA pass/fail status
        CASE 
            WHEN vsr.blocks_proposed = 0 THEN 'offline'
            WHEN rr.block_quota > 0 AND vsr.challenges_received >= 0 THEN
                CASE 
                    WHEN (vsr.blocks_proposed::float / rr.block_quota >= 0.8) 
                         AND (CASE WHEN vsr.challenges_received > 0 
                                  THEN (1.0 - (vsr.challenges_failed::float / vsr.challenges_received)) >= 0.8 
                                  ELSE true END) 
                    THEN 'pass'
                    ELSE 'fail'
                END
            ELSE 'unknown'
        END as sla_status
    FROM v_sla_rollup_score vsr
    JOIN recent_rollups rr ON vsr.sla_id = rr.id
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
