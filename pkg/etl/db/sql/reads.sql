-- get latest indexed block height
-- name: GetLatestIndexedBlock :one
SELECT block_height
FROM etl_blocks
ORDER BY id DESC
LIMIT 1;

-- name: GetTotalTransactions :one
select id from etl_transactions order by id desc limit 1;

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

-- Transaction content queries by hash
-- name: GetPlaysByTxHash :many
select * from etl_plays
where tx_hash = $1;

-- name: GetManageEntityByTxHash :one
select * from etl_manage_entities
where tx_hash = $1;

-- name: GetValidatorRegistrationByTxHash :one
select * from etl_validator_registrations
where tx_hash = $1;

-- name: GetValidatorDeregistrationByTxHash :one
select * from etl_validator_deregistrations
where tx_hash = $1;

-- name: GetSlaRollupByTxHash :one
select * from etl_sla_rollups
where tx_hash = $1;

-- name: GetStorageProofByTxHash :one
select * from etl_storage_proofs
where tx_hash = $1;

-- name: GetStorageProofVerificationByTxHash :one
select * from etl_storage_proof_verifications
where tx_hash = $1;

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

-- name: GetBlocksByPage :many
select * from etl_blocks
order by block_height desc
limit $1 offset $2;

-- name: GetBlockByHeight :one
select * from etl_blocks
where block_height = $1;

-- name: GetTransactionByHash :one  
select * from etl_transactions
where tx_hash = $1;

-- name: GetTransactionsByPage :many
select * from etl_transactions
order by block_height desc, tx_index desc
limit $1 offset $2;

-- name: GetActiveValidators :many
select * from etl_validators
where status = 'active'
order by voting_power desc
limit $1 offset $2;

-- name: GetValidatorRegistrations :many
select vr.*, v.endpoint, v.node_type, v.spid, v.voting_power, v.status
from etl_validator_registrations vr
left join etl_validators v on v.comet_address = vr.comet_address
order by vr.block_height desc
limit $1 offset $2;

-- name: GetValidatorDeregistrations :many
select vd.*, v.endpoint, v.node_type, v.spid, v.voting_power, v.status
from etl_validator_deregistrations vd
left join etl_validators v on v.comet_address = vd.comet_address
order by vd.block_height desc
limit $1 offset $2;

-- name: GetValidatorByAddress :one
select * from etl_validators
where address = $1 or comet_address = $1;

-- name: GetSlaNodeReportsByAddress :many
select * from etl_sla_node_reports
where address = $1
order by block_height desc
limit $2;

-- name: GetValidatorsForSlaRollup :many
select v.*, snr.num_blocks_proposed, snr.challenges_received, snr.challenges_failed
from etl_validators v
left join etl_sla_node_reports snr on v.comet_address = snr.address and snr.sla_rollup_id = $1
where v.status = 'active'
order by v.voting_power desc;

-- name: GetAllActiveValidatorsWithRecentRollups :many
select v.*, snr.sla_rollup_id, snr.num_blocks_proposed, snr.challenges_received, snr.challenges_failed, snr.block_height as report_block_height, snr.created_at as report_created_at
from etl_validators v
left join etl_sla_node_reports snr on v.comet_address = snr.address
where v.status = 'active'
order by v.voting_power desc, snr.sla_rollup_id desc;

-- name: GetSlaRollupsWithPagination :many
select * from etl_sla_rollups
order by id desc
limit $1 offset $2;

-- name: GetBlockTransactionCount :one
select count(*) from etl_transactions
where block_height = $1;

-- name: GetActiveValidatorCount :one
select count(*) from etl_validators
where status = 'active';
