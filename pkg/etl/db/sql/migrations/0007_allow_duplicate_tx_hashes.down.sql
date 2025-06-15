-- Rollback: Restore unique constraint on tx_hash
-- Migration 0007 Down: Remove composite constraint and restore original unique constraint

-- Drop the composite unique constraint
ALTER TABLE etl_transactions_v2 DROP CONSTRAINT IF EXISTS etl_transactions_v2_tx_hash_block_id_key;

-- Drop the composite index
DROP INDEX IF EXISTS idx_etl_transactions_v2_tx_hash_block_id;

-- Restore the original unique constraint on tx_hash
-- Note: This might fail if there are actually duplicate tx_hashes in the database
ALTER TABLE etl_transactions_v2 ADD CONSTRAINT etl_transactions_v2_tx_hash_key UNIQUE (tx_hash); 
