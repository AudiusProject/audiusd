-- Additional indexes for transactions and blocks page performance

-- Covering index for transactions page - includes all fields needed to avoid table lookups
CREATE INDEX IF NOT EXISTS etl_transactions_id_desc_covering 
ON etl_transactions (id DESC) INCLUDE (tx_hash, block_height, index, tx_type, created_at, updated_at);

-- Optimized index for the JOIN in GetLatestTransactions - different column order
CREATE INDEX IF NOT EXISTS etl_transactions_id_block_height_desc 
ON etl_transactions (id DESC, block_height);

-- Index for blocks pagination - covers the ORDER BY block_height DESC with LIMIT/OFFSET
CREATE INDEX IF NOT EXISTS etl_blocks_block_height_desc_pagination 
ON etl_blocks (block_height DESC, id);

-- Covering index for etl_blocks to include all commonly accessed fields
CREATE INDEX IF NOT EXISTS etl_blocks_pagination_covering 
ON etl_blocks (block_height DESC) INCLUDE (id, proposer_address, block_time, created_at, updated_at);

-- Index to optimize block height lookups (for the blockHeights map in transactions page)
CREATE INDEX IF NOT EXISTS etl_blocks_height_lookup 
ON etl_blocks (block_height) INCLUDE (block_time);

-- Index for transaction counts per block (if needed for block details)
CREATE INDEX IF NOT EXISTS etl_transactions_block_height_count 
ON etl_transactions (block_height) INCLUDE (id);

-- Index for recent transactions with better JOIN performance
CREATE INDEX IF NOT EXISTS etl_transactions_recent_optimization 
ON etl_transactions (id DESC, tx_hash, block_height, tx_type);

-- Index for proposer lookups (validator pages)
CREATE INDEX IF NOT EXISTS etl_blocks_proposer_height 
ON etl_blocks (proposer_address, block_height DESC); 
