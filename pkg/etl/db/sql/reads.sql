-- Normalized read queries for ETL database
-- Uses the new schema with proper JOIN operations for efficiency

-- get latest indexed block height
-- name: GetLatestIndexedBlock :one
SELECT block_height
FROM etl_blocks
ORDER BY id DESC
LIMIT 1;

-- name: GetTransactionsByBlockHeightCursor :many
select * from etl_transactions
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetPlaysByBlockHeightCursor :many
select * from etl_plays
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetManageEntitiesByBlockHeightCursor :many
select * from etl_manage_entities
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetValidatorRegistrationsByBlockHeightCursor :many
select * from etl_validator_registrations
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetValidatorDeregistrationsByBlockHeightCursor :many
select * from etl_validator_deregistrations
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetSlaRollupsByBlockHeightCursor :many
select * from etl_sla_rollups
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetSlaNodeReportsByBlockHeightCursor :many
select * from etl_sla_node_reports
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetValidatorMisbehaviorDeregistrationsByBlockHeightCursor :many
select * from etl_validator_misbehavior_deregistrations
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetStorageProofsByBlockHeightCursor :many
select * from etl_storage_proofs
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetStorageProofVerificationsByBlockHeightCursor :many
select * from etl_storage_proof_verifications
where block_height > $1 or (block_height = $1 and id > $2)
order by block_height, id
limit $3;

-- name: GetBlockRangeFirst :one
select id, proposer_address, block_height, block_time 
from etl_blocks
where block_time >= $1 and block_time <= $2
order by block_time
limit 1;

-- name: GetBlockRangeLast :one
select id, proposer_address, block_height, block_time 
from etl_blocks
where block_time >= $1 and block_time <= $2
order by block_time desc
limit 1;
