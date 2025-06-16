-- Performance indexes for validator pages and uptime queries

-- Indexes for v_sla_rollup view performance
-- The view orders by date_finalized DESC, so we need this index
CREATE INDEX IF NOT EXISTS idx_etl_sla_rollups_v2_timestamp_desc ON etl_sla_rollups_v2 (timestamp DESC);

-- Indexes for v_sla_rollup_score view performance
-- The view joins on sla_rollup_id frequently
CREATE INDEX IF NOT EXISTS idx_etl_sla_node_reports_v2_sla_rollup_id ON etl_sla_node_reports_v2 (sla_rollup_id);

-- Indexes for address lookups (used in many validator queries)
CREATE INDEX IF NOT EXISTS idx_etl_addresses_address_hash ON etl_addresses USING hash (address);

-- Composite index for validator registrations (address + block height for DISTINCT ON queries)
CREATE INDEX IF NOT EXISTS idx_etl_validator_registrations_v2_address_block ON etl_validator_registrations_v2 (address_id, transaction_id DESC);

-- Index for validator deregistrations comet_address lookups
CREATE INDEX IF NOT EXISTS idx_etl_validator_deregistrations_v2_comet_address_hash ON etl_validator_deregistrations_v2 USING hash (comet_address);

-- Index for transaction -> block joins (used in many queries)
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_block_id_tx_index ON etl_transactions_v2 (block_id, tx_index);

-- Index for block time ordering (used in many time-based queries)
CREATE INDEX IF NOT EXISTS idx_etl_blocks_block_time_desc ON etl_blocks (block_time DESC);

-- Composite index for storage proofs (address + height for challenge calculations)
CREATE INDEX IF NOT EXISTS idx_etl_storage_proofs_v2_address_height ON etl_storage_proofs_v2 (address_id, height);

-- Index for storage proof height ranges (used in SLA rollup challenge calculations)
CREATE INDEX IF NOT EXISTS idx_etl_storage_proofs_v2_height_range ON etl_storage_proofs_v2 (height);

-- Index for endpoint filtering in validator registrations
CREATE INDEX IF NOT EXISTS idx_etl_validator_registrations_v2_endpoint_lower ON etl_validator_registrations_v2 (LOWER(endpoint));

-- Composite index for the GetAllRegisteredValidatorsWithEndpoints query
CREATE INDEX IF NOT EXISTS idx_etl_validator_registrations_v2_address_id_desc ON etl_validator_registrations_v2 (address_id, id DESC); 
