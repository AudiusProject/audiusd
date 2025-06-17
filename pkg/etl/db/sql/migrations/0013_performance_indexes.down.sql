-- Drop performance indexes for validator pages and uptime queries

DROP INDEX IF EXISTS idx_etl_validator_registrations_v2_address_id_desc;
DROP INDEX IF EXISTS idx_etl_validator_registrations_v2_endpoint_lower;
DROP INDEX IF EXISTS idx_etl_storage_proofs_v2_height_range;
DROP INDEX IF EXISTS idx_etl_storage_proofs_v2_address_height;
DROP INDEX IF EXISTS idx_etl_blocks_block_time_desc;
DROP INDEX IF EXISTS idx_etl_transactions_v2_block_id_tx_index;
DROP INDEX IF EXISTS idx_etl_validator_deregistrations_v2_comet_address_hash;
DROP INDEX IF EXISTS idx_etl_validator_registrations_v2_address_block;
DROP INDEX IF EXISTS idx_etl_addresses_address_hash;
DROP INDEX IF EXISTS idx_etl_sla_node_reports_v2_sla_rollup_id;
DROP INDEX IF EXISTS idx_etl_sla_rollups_v2_timestamp_desc; 
