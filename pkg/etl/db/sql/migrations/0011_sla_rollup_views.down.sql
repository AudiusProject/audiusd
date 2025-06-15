-- Drop SLA rollup views
-- Migration 0011 Down: Remove SLA rollup views

DROP VIEW IF EXISTS v_sla_rollup_score;
DROP VIEW IF EXISTS v_sla_rollup; 
