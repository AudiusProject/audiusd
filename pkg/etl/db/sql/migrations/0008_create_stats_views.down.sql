-- Rollback: Drop statistics views
-- Migration 0008 Down: Remove stats views

DROP VIEW IF EXISTS v_entity_type_stats_24h;
DROP VIEW IF EXISTS v_plays_by_location_24h;
DROP VIEW IF EXISTS v_top_tracks_24h;
DROP VIEW IF EXISTS v_latest_block_info;
DROP VIEW IF EXISTS v_network_rates;
DROP VIEW IF EXISTS v_validator_stats;
DROP VIEW IF EXISTS v_plays_stats;
DROP VIEW IF EXISTS v_transaction_type_breakdown_24h;
DROP VIEW IF EXISTS v_transaction_stats; 
