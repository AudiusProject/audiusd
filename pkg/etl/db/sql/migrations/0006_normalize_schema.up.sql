-- Normalize ETL schema by removing duplication and establishing proper FK relationships
-- Migration 0006: Schema Normalization

-- First, create the new normalized tables

-- Addresses table for deduplication
CREATE TABLE IF NOT EXISTS etl_addresses (
    id SERIAL PRIMARY KEY,
    address TEXT NOT NULL UNIQUE,
    first_seen_block_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Updated transactions table with foreign key to blocks
CREATE TABLE IF NOT EXISTS etl_transactions_v2 (
    id SERIAL PRIMARY KEY,
    tx_hash TEXT NOT NULL UNIQUE,
    block_id INTEGER NOT NULL,
    tx_index INTEGER NOT NULL,
    tx_type TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (block_id) REFERENCES etl_blocks(id) ON DELETE CASCADE
);

-- Normalized plays table (remove duplicated fields)
CREATE TABLE IF NOT EXISTS etl_plays_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    address_id INTEGER NOT NULL,
    track_id TEXT NOT NULL,
    city TEXT,
    region TEXT,
    country TEXT,
    played_at TIMESTAMP NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE,
    FOREIGN KEY (address_id) REFERENCES etl_addresses(id)
);

-- Normalized manage entities table
CREATE TABLE IF NOT EXISTS etl_manage_entities_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    address_id INTEGER NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id BIGINT NOT NULL,
    action TEXT NOT NULL,
    metadata TEXT,
    signature TEXT NOT NULL,
    signer_address_id INTEGER NOT NULL,
    nonce TEXT NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE,
    FOREIGN KEY (address_id) REFERENCES etl_addresses(id),
    FOREIGN KEY (signer_address_id) REFERENCES etl_addresses(id)
);

-- Normalized validator registrations table
CREATE TABLE IF NOT EXISTS etl_validator_registrations_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    address_id INTEGER NOT NULL,
    endpoint TEXT NOT NULL,
    comet_address TEXT NOT NULL,
    eth_block TEXT NOT NULL,
    node_type TEXT NOT NULL,
    spid TEXT NOT NULL,
    comet_pubkey BYTEA NOT NULL,
    voting_power BIGINT NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE,
    FOREIGN KEY (address_id) REFERENCES etl_addresses(id)
);

-- Normalized validator deregistrations table
CREATE TABLE IF NOT EXISTS etl_validator_deregistrations_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    comet_address TEXT NOT NULL,
    comet_pubkey BYTEA NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE
);

-- Normalized SLA rollups table
CREATE TABLE IF NOT EXISTS etl_sla_rollups_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    block_start BIGINT NOT NULL,
    block_end BIGINT NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE
);

-- Normalized SLA node reports table
CREATE TABLE IF NOT EXISTS etl_sla_node_reports_v2 (
    id SERIAL PRIMARY KEY,
    sla_rollup_id INTEGER NOT NULL,
    address_id INTEGER NOT NULL,
    num_blocks_proposed INTEGER NOT NULL,
    FOREIGN KEY (sla_rollup_id) REFERENCES etl_sla_rollups_v2(id) ON DELETE CASCADE,
    FOREIGN KEY (address_id) REFERENCES etl_addresses(id)
);

-- Normalized storage proofs table
CREATE TABLE IF NOT EXISTS etl_storage_proofs_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    height BIGINT NOT NULL,
    address_id INTEGER NOT NULL,
    prover_addresses TEXT[] NOT NULL,
    cid TEXT NOT NULL,
    proof_signature BYTEA NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE,
    FOREIGN KEY (address_id) REFERENCES etl_addresses(id)
);

-- Normalized storage proof verifications table
CREATE TABLE IF NOT EXISTS etl_storage_proof_verifications_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    height BIGINT NOT NULL,
    proof BYTEA NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE
);

-- Normalized releases table
CREATE TABLE IF NOT EXISTS etl_releases_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    release_data BYTEA NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE
);

-- Normalized validator registrations legacy table
CREATE TABLE IF NOT EXISTS etl_validator_registrations_legacy_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    endpoint TEXT NOT NULL,
    comet_address TEXT NOT NULL,
    eth_block TEXT NOT NULL,
    node_type TEXT NOT NULL,
    sp_id TEXT NOT NULL,
    pub_key BYTEA NOT NULL,
    power BIGINT NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE
);

-- Normalized validator misbehavior deregistrations table
CREATE TABLE IF NOT EXISTS etl_validator_misbehavior_deregistrations_v2 (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    comet_address TEXT NOT NULL,
    pub_key BYTEA NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES etl_transactions_v2(id) ON DELETE CASCADE
);

-- Create indexes for optimal performance
CREATE INDEX IF NOT EXISTS idx_etl_addresses_address ON etl_addresses(address);
CREATE INDEX IF NOT EXISTS idx_etl_addresses_first_seen ON etl_addresses(first_seen_block_id);

CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_tx_hash ON etl_transactions_v2(tx_hash);
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_block_id ON etl_transactions_v2(block_id);
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_tx_type ON etl_transactions_v2(tx_type);
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_created_at ON etl_transactions_v2(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_etl_plays_v2_transaction_id ON etl_plays_v2(transaction_id);
CREATE INDEX IF NOT EXISTS idx_etl_plays_v2_address_id ON etl_plays_v2(address_id);
CREATE INDEX IF NOT EXISTS idx_etl_plays_v2_track_id ON etl_plays_v2(track_id);
CREATE INDEX IF NOT EXISTS idx_etl_plays_v2_played_at ON etl_plays_v2(played_at DESC);

CREATE INDEX IF NOT EXISTS idx_etl_manage_entities_v2_transaction_id ON etl_manage_entities_v2(transaction_id);
CREATE INDEX IF NOT EXISTS idx_etl_manage_entities_v2_address_id ON etl_manage_entities_v2(address_id);
CREATE INDEX IF NOT EXISTS idx_etl_manage_entities_v2_entity_type ON etl_manage_entities_v2(entity_type);
CREATE INDEX IF NOT EXISTS idx_etl_manage_entities_v2_entity_id ON etl_manage_entities_v2(entity_id);

CREATE INDEX IF NOT EXISTS idx_etl_validator_registrations_v2_transaction_id ON etl_validator_registrations_v2(transaction_id);
CREATE INDEX IF NOT EXISTS idx_etl_validator_registrations_v2_address_id ON etl_validator_registrations_v2(address_id);
CREATE INDEX IF NOT EXISTS idx_etl_validator_registrations_v2_comet_address ON etl_validator_registrations_v2(comet_address);

CREATE INDEX IF NOT EXISTS idx_etl_validator_deregistrations_v2_transaction_id ON etl_validator_deregistrations_v2(transaction_id);
CREATE INDEX IF NOT EXISTS idx_etl_validator_deregistrations_v2_comet_address ON etl_validator_deregistrations_v2(comet_address);

CREATE INDEX IF NOT EXISTS idx_etl_sla_rollups_v2_transaction_id ON etl_sla_rollups_v2(transaction_id);
CREATE INDEX IF NOT EXISTS idx_etl_sla_rollups_v2_timestamp ON etl_sla_rollups_v2(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_etl_sla_node_reports_v2_sla_rollup_id ON etl_sla_node_reports_v2(sla_rollup_id);
CREATE INDEX IF NOT EXISTS idx_etl_sla_node_reports_v2_address_id ON etl_sla_node_reports_v2(address_id);

-- Trigram indexes for search functionality
CREATE INDEX IF NOT EXISTS idx_etl_addresses_address_trgm ON etl_addresses USING gin (address gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_etl_addresses_address_prefix ON etl_addresses (address text_pattern_ops);

CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_tx_hash_trgm ON etl_transactions_v2 USING gin (tx_hash gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_tx_hash_prefix ON etl_transactions_v2 (tx_hash text_pattern_ops);

-- Create a view that maintains compatibility with existing queries
-- This allows gradual migration of application code
CREATE VIEW etl_transactions_with_block AS
SELECT 
    t.id,
    t.tx_hash,
    b.block_height,
    t.tx_index as index,
    t.tx_type,
    b.block_time,
    b.proposer_address,
    t.created_at
FROM etl_transactions_v2 t
JOIN etl_blocks b ON t.block_id = b.id;

CREATE VIEW etl_plays_with_details AS
SELECT 
    p.id,
    a.address,
    p.track_id,
    p.city,
    p.region,
    p.country,
    p.played_at,
    t.tx_hash,
    b.block_height,
    b.block_time
FROM etl_plays_v2 p
JOIN etl_addresses a ON p.address_id = a.id
JOIN etl_transactions_v2 t ON p.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id;

CREATE VIEW etl_manage_entities_with_details AS
SELECT 
    me.id,
    a.address,
    me.entity_type,
    me.entity_id,
    me.action,
    me.metadata,
    me.signature,
    sa.address as signer,
    me.nonce,
    t.tx_hash,
    b.block_height,
    b.block_time
FROM etl_manage_entities_v2 me
JOIN etl_addresses a ON me.address_id = a.id
JOIN etl_addresses sa ON me.signer_address_id = sa.id
JOIN etl_transactions_v2 t ON me.transaction_id = t.id
JOIN etl_blocks b ON t.block_id = b.id; 
