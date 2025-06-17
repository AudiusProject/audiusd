-- Migration 0016: Dashboard Performance Indexes
-- Simple indexes to speed up dashboard queries without complex materialized views

-- Index for latest transactions query (most common dashboard query)
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_id_desc_dashboard 
ON etl_transactions_v2 (id DESC) 
INCLUDE (tx_hash, tx_type, tx_index, block_id);

-- Index for transaction stats by block time (for time-based filtering)
CREATE INDEX IF NOT EXISTS idx_etl_blocks_block_time_dashboard 
ON etl_blocks (block_time DESC) 
INCLUDE (id, block_height);

-- Index to speed up transaction counting for stats
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_created_at_stats 
ON etl_transactions_v2 (created_at DESC) 
INCLUDE (block_id, tx_type);

-- Index for transaction type breakdown queries
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_block_type 
ON etl_transactions_v2 (block_id, tx_type);

-- Index for validator count queries
CREATE INDEX IF NOT EXISTS idx_etl_validator_registrations_v2_comet_address 
ON etl_validator_registrations_v2 (comet_address) 
INCLUDE (address_id);

-- Index for validator deregistrations lookup
CREATE INDEX IF NOT EXISTS idx_etl_validator_deregistrations_v2_comet_address 
ON etl_validator_deregistrations_v2 (comet_address);

-- Index for latest block info
CREATE INDEX IF NOT EXISTS idx_etl_blocks_height_desc_latest 
ON etl_blocks (block_height DESC) 
INCLUDE (block_time, proposer_address);

-- Composite index for JOIN between transactions and blocks (heavily used)
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_block_id_covering 
ON etl_transactions_v2 (block_id) 
INCLUDE (id, tx_hash, tx_type, tx_index, created_at); 
