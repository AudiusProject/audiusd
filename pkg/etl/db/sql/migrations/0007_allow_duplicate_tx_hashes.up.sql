-- Allow duplicate transaction hashes across different blocks
-- Migration 0007: Remove unique constraint on tx_hash and add composite unique constraint

-- Drop the existing unique constraint on tx_hash
ALTER TABLE etl_transactions_v2 DROP CONSTRAINT IF EXISTS etl_transactions_v2_tx_hash_key;

-- Add a composite unique constraint on (tx_hash, block_id) to ensure 
-- the same transaction hash can appear in different blocks but not duplicated within the same block
ALTER TABLE etl_transactions_v2 ADD CONSTRAINT etl_transactions_v2_tx_hash_block_id_key 
    UNIQUE (tx_hash, block_id);

-- Update the index to support the new constraint pattern
DROP INDEX IF EXISTS idx_etl_transactions_v2_tx_hash;
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_tx_hash_block_id ON etl_transactions_v2(tx_hash, block_id);

-- Keep the individual tx_hash index for search purposes  
CREATE INDEX IF NOT EXISTS idx_etl_transactions_v2_tx_hash ON etl_transactions_v2(tx_hash); 
