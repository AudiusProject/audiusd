-- Drop pagination performance indexes

DROP INDEX IF EXISTS etl_blocks_proposer_height;
DROP INDEX IF EXISTS etl_transactions_recent_optimization;
DROP INDEX IF EXISTS etl_transactions_block_height_count;
DROP INDEX IF EXISTS etl_blocks_height_lookup;
DROP INDEX IF EXISTS etl_blocks_pagination_covering;
DROP INDEX IF EXISTS etl_blocks_block_height_desc_pagination;
DROP INDEX IF EXISTS etl_transactions_id_block_height_desc;
DROP INDEX IF EXISTS etl_transactions_id_desc_covering; 
