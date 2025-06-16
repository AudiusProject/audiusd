-- Drop performance optimization indexes

DROP INDEX IF EXISTS etl_transactions_id_desc;
DROP INDEX IF EXISTS etl_validator_deregistrations_comet_address;
DROP INDEX IF EXISTS etl_validator_registrations_comet_address_block_height;
DROP INDEX IF EXISTS etl_blocks_height_desc_covering;
DROP INDEX IF EXISTS etl_transactions_tx_type_block_height;
DROP INDEX IF EXISTS etl_blocks_block_time_height;
DROP INDEX IF EXISTS etl_transactions_block_height_id_desc; 
