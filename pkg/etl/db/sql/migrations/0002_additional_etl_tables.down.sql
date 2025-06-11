-- Drop additional ETL tables

-- Drop indexes for etl_releases
drop index if exists etl_releases_release_data_gin_idx;
drop index if exists etl_releases_tx_hash_idx;
drop index if exists etl_releases_block_height_idx;

-- Drop indexes for etl_storage_proof_verifications
drop index if exists etl_storage_proof_verifications_height_idx;
drop index if exists etl_storage_proof_verifications_tx_hash_idx;
drop index if exists etl_storage_proof_verifications_block_height_idx;

-- Drop indexes for etl_storage_proofs
drop index if exists etl_storage_proofs_address_idx;
drop index if exists etl_storage_proofs_height_idx;
drop index if exists etl_storage_proofs_tx_hash_idx;
drop index if exists etl_storage_proofs_block_height_idx;

-- Drop indexes for etl_validator_misbehavior_deregistrations
drop index if exists etl_validator_misbehavior_deregistrations_comet_address_idx;
drop index if exists etl_validator_misbehavior_deregistrations_tx_hash_idx;
drop index if exists etl_validator_misbehavior_deregistrations_block_height_idx;

-- Drop indexes for etl_sla_node_reports
drop index if exists etl_sla_node_reports_tx_hash_idx;
drop index if exists etl_sla_node_reports_block_height_idx;
drop index if exists etl_sla_node_reports_address_idx;
drop index if exists etl_sla_node_reports_sla_rollup_id_idx;

-- Drop indexes for etl_sla_rollups
drop index if exists etl_sla_rollups_block_range_idx;
drop index if exists etl_sla_rollups_timestamp_idx;
drop index if exists etl_sla_rollups_tx_hash_idx;
drop index if exists etl_sla_rollups_block_height_idx;

-- Drop indexes for etl_validator_registrations_legacy
drop index if exists etl_validator_registrations_legacy_comet_address_idx;
drop index if exists etl_validator_registrations_legacy_tx_hash_idx;
drop index if exists etl_validator_registrations_legacy_block_height_idx;

-- Drop tables (order matters due to foreign keys)
drop table if exists etl_sla_node_reports;
drop table if exists etl_sla_rollups;
drop table if exists etl_validator_registrations_legacy;
drop table if exists etl_validator_misbehavior_deregistrations;
drop table if exists etl_storage_proofs;
drop table if exists etl_storage_proof_verifications;
drop table if exists etl_releases; 
