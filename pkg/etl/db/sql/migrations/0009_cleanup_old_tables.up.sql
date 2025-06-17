-- Drop old denormalized tables and indexes
-- Migration 0009: Cleanup old tables after normalization

-- Drop trigram indexes first
DROP INDEX IF EXISTS etl_blocks_block_height_trgm;
DROP INDEX IF EXISTS etl_blocks_block_height_prefix;
DROP INDEX IF EXISTS etl_manage_entities_address_trgm;
DROP INDEX IF EXISTS etl_manage_entities_address_prefix;
DROP INDEX IF EXISTS etl_transactions_tx_hash_trgm;
DROP INDEX IF EXISTS etl_transactions_tx_hash_prefix;
DROP INDEX IF EXISTS etl_validator_registrations_address_trgm;
DROP INDEX IF EXISTS etl_validator_registrations_address_prefix;

-- Drop old denormalized tables (keeping etl_blocks and etl_addresses as they're part of the normalized schema)
DROP TABLE IF EXISTS etl_plays CASCADE;
DROP TABLE IF EXISTS etl_manage_entities CASCADE;
DROP TABLE IF EXISTS etl_transactions CASCADE;
DROP TABLE IF EXISTS etl_validator_registrations CASCADE;
DROP TABLE IF EXISTS etl_validator_deregistrations CASCADE;
DROP TABLE IF EXISTS etl_validator_registrations_legacy CASCADE;
DROP TABLE IF EXISTS etl_sla_node_reports CASCADE;
DROP TABLE IF EXISTS etl_sla_rollups CASCADE;
DROP TABLE IF EXISTS etl_validator_misbehavior_deregistrations CASCADE;
DROP TABLE IF EXISTS etl_storage_proofs CASCADE;
DROP TABLE IF EXISTS etl_storage_proof_verifications CASCADE;
DROP TABLE IF EXISTS etl_releases CASCADE;

-- Drop old compatibility views if they exist
DROP VIEW IF EXISTS etl_transactions_with_block;
DROP VIEW IF EXISTS etl_plays_with_details;
DROP VIEW IF EXISTS etl_manage_entities_with_details;

-- Note: We keep etl_blocks and etl_addresses as they are core to the normalized schema
-- We also keep the _v2 tables and the new statistics views 
