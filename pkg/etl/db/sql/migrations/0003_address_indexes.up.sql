-- Add indexes on lowercase address columns for case-insensitive queries

-- Index for etl_plays.address (used in play transactions)
CREATE INDEX idx_etl_plays_address_lower ON etl_plays (LOWER(address));

-- Index for etl_manage_entities.address (used in manage entity transactions)
CREATE INDEX idx_etl_manage_entities_address_lower ON etl_manage_entities (LOWER(address));

-- Index for etl_validator_registrations.address (used in validator registration transactions)
CREATE INDEX idx_etl_validator_registrations_address_lower ON etl_validator_registrations (LOWER(address));

-- Index for etl_validator_deregistrations.comet_address (used in validator deregistration transactions)
CREATE INDEX idx_etl_validator_deregistrations_comet_address_lower ON etl_validator_deregistrations (LOWER(comet_address));

-- Index for etl_storage_proofs.address (used in storage proof transactions)
CREATE INDEX idx_etl_storage_proofs_address_lower ON etl_storage_proofs (LOWER(address));

-- Index for etl_sla_node_reports.address (used in SLA node report transactions)
CREATE INDEX idx_etl_sla_node_reports_address_lower ON etl_sla_node_reports (LOWER(address)); 
