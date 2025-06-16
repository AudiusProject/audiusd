-- Performance optimization indexes for dashboard queries

-- Composite index for GetLatestTransactions join performance
-- This will help the frequent JOIN between transactions and blocks
CREATE INDEX IF NOT EXISTS etl_transactions_block_height_id_desc 
ON etl_transactions (block_height, id DESC);

-- Composite index for time-based queries on blocks  
-- This will help with time range filtering and aggregations
CREATE INDEX IF NOT EXISTS etl_blocks_block_time_height 
ON etl_blocks (block_time, block_height);

-- Composite index for transaction type breakdown queries
-- This will speed up the tx_type aggregations with time filtering
CREATE INDEX IF NOT EXISTS etl_transactions_tx_type_block_height 
ON etl_transactions (tx_type, block_height);

-- Covering index for latest blocks query (includes proposer for dashboard)
CREATE INDEX IF NOT EXISTS etl_blocks_height_desc_covering 
ON etl_blocks (block_height DESC) INCLUDE (proposer_address, block_time);

-- Index for validator count queries (active validators)
CREATE INDEX IF NOT EXISTS etl_validator_registrations_comet_address_block_height 
ON etl_validator_registrations (comet_address, block_height DESC);

CREATE INDEX IF NOT EXISTS etl_validator_deregistrations_comet_address 
ON etl_validator_deregistrations (comet_address);

-- Additional index to optimize ORDER BY id DESC queries
CREATE INDEX IF NOT EXISTS etl_transactions_id_desc 
ON etl_transactions (id DESC);
