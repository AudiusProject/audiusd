-- Drop indexes on lowercase address columns

DROP INDEX IF EXISTS idx_etl_plays_address_lower;
DROP INDEX IF EXISTS idx_etl_manage_entities_address_lower;
DROP INDEX IF EXISTS idx_etl_validator_registrations_address_lower;
DROP INDEX IF EXISTS idx_etl_validator_deregistrations_comet_address_lower;
DROP INDEX IF EXISTS idx_etl_storage_proofs_address_lower;
DROP INDEX IF EXISTS idx_etl_sla_node_reports_address_lower; 
