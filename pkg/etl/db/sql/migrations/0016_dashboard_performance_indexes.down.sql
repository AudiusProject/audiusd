-- Migration 0016 Down: Drop Dashboard Performance Indexes

DROP INDEX IF EXISTS idx_etl_transactions_v2_block_id_covering;
DROP INDEX IF EXISTS idx_etl_blocks_height_desc_latest;
DROP INDEX IF EXISTS idx_etl_validator_deregistrations_v2_comet_address;
DROP INDEX IF EXISTS idx_etl_validator_registrations_v2_comet_address;
DROP INDEX IF EXISTS idx_etl_transactions_v2_block_type;
DROP INDEX IF EXISTS idx_etl_transactions_v2_created_at_stats;
DROP INDEX IF EXISTS idx_etl_blocks_block_time_dashboard;
DROP INDEX IF EXISTS idx_etl_transactions_v2_id_desc_dashboard; 
