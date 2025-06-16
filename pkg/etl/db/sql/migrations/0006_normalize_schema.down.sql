-- Rollback schema normalization
-- Migration 0006 Down: Remove normalized tables and views

-- Drop views first
DROP VIEW IF EXISTS etl_manage_entities_with_details;
DROP VIEW IF EXISTS etl_plays_with_details;
DROP VIEW IF EXISTS etl_transactions_with_block;

-- Drop indexes
DROP INDEX IF EXISTS idx_etl_transactions_v2_tx_hash_prefix;
DROP INDEX IF EXISTS idx_etl_transactions_v2_tx_hash_trgm;
DROP INDEX IF EXISTS idx_etl_addresses_address_prefix;
DROP INDEX IF EXISTS idx_etl_addresses_address_trgm;

DROP INDEX IF EXISTS idx_etl_sla_node_reports_v2_address_id;
DROP INDEX IF EXISTS idx_etl_sla_node_reports_v2_sla_rollup_id;
DROP INDEX IF EXISTS idx_etl_sla_rollups_v2_timestamp;
DROP INDEX IF EXISTS idx_etl_sla_rollups_v2_transaction_id;

DROP INDEX IF EXISTS idx_etl_validator_deregistrations_v2_comet_address;
DROP INDEX IF EXISTS idx_etl_validator_deregistrations_v2_transaction_id;
DROP INDEX IF EXISTS idx_etl_validator_registrations_v2_comet_address;
DROP INDEX IF EXISTS idx_etl_validator_registrations_v2_address_id;
DROP INDEX IF EXISTS idx_etl_validator_registrations_v2_transaction_id;

DROP INDEX IF EXISTS idx_etl_manage_entities_v2_entity_id;
DROP INDEX IF EXISTS idx_etl_manage_entities_v2_entity_type;
DROP INDEX IF EXISTS idx_etl_manage_entities_v2_address_id;
DROP INDEX IF EXISTS idx_etl_manage_entities_v2_transaction_id;

DROP INDEX IF EXISTS idx_etl_plays_v2_played_at;
DROP INDEX IF EXISTS idx_etl_plays_v2_track_id;
DROP INDEX IF EXISTS idx_etl_plays_v2_address_id;
DROP INDEX IF EXISTS idx_etl_plays_v2_transaction_id;

DROP INDEX IF EXISTS idx_etl_transactions_v2_created_at;
DROP INDEX IF EXISTS idx_etl_transactions_v2_tx_type;
DROP INDEX IF EXISTS idx_etl_transactions_v2_block_id;
DROP INDEX IF EXISTS idx_etl_transactions_v2_tx_hash;

DROP INDEX IF EXISTS idx_etl_addresses_first_seen;
DROP INDEX IF EXISTS idx_etl_addresses_address;

-- Drop normalized tables (in dependency order)
DROP TABLE IF EXISTS etl_validator_misbehavior_deregistrations_v2;
DROP TABLE IF EXISTS etl_validator_registrations_legacy_v2;
DROP TABLE IF EXISTS etl_releases_v2;
DROP TABLE IF EXISTS etl_storage_proof_verifications_v2;
DROP TABLE IF EXISTS etl_storage_proofs_v2;
DROP TABLE IF EXISTS etl_sla_node_reports_v2;
DROP TABLE IF EXISTS etl_sla_rollups_v2;
DROP TABLE IF EXISTS etl_validator_deregistrations_v2;
DROP TABLE IF EXISTS etl_validator_registrations_v2;
DROP TABLE IF EXISTS etl_manage_entities_v2;
DROP TABLE IF EXISTS etl_plays_v2;
DROP TABLE IF EXISTS etl_transactions_v2;
DROP TABLE IF EXISTS etl_addresses; 
