-- get latest indexed block height
-- name: GetLatestIndexedBlock :one
select block_height
from etl_blocks
order by id desc
limit 1;

-- name: GetIndexedBlock :one
select *
from etl_blocks
where block_height = $1;

-- name: GetLatestBlocks :many
select *
from etl_blocks
order by block_height desc
limit $1 offset $2;

-- name: GetTotalBlocksCount :one
select count(*) as total
from etl_blocks;

-- name: GetLatestTransactions :many
select t.*, b.block_time
from etl_transactions t
join etl_blocks b on t.block_height = b.block_height
order by t.id desc
limit $1 offset $2;

-- name: GetTotalTransactionsCount :one
select count(*) as total
from etl_transactions;

-- name: GetBlockRangeByTime :one
select min(block_height) as start_block,
    max(block_height) as end_block
from etl_blocks
where block_time between $1 and $2;

-- name: GetPlaysByAddress :many
select address,
    track_id,
    extract(
        epoch
        from played_at
    )::bigint as timestamp,
    city,
    country,
    region,
    block_height,
    tx_hash
from etl_plays
where address = $1
    and block_height between $2 and $3
order by played_at desc
limit $4 offset $5;

-- name: GetPlaysByTrack :many
select address,
    track_id,
    extract(
        epoch
        from played_at
    )::bigint as timestamp,
    city,
    country,
    region,
    block_height,
    tx_hash
from etl_plays
where track_id = $1
    and block_height between $2 and $3
order by played_at desc
limit $4 offset $5;

-- name: GetPlays :many
select address,
    track_id,
    extract(
        epoch
        from played_at
    )::bigint as timestamp,
    city,
    country,
    region,
    block_height,
    tx_hash
from etl_plays
where block_height between $1 and $2
order by played_at desc
limit $3 offset $4;

-- get total count of plays with filtering
-- name: GetPlaysCount :one
select count(*) as total
from etl_plays
where (
        $1::text is null
        or address = $1
    )
    and (
        $2::text is null
        or track_id = $2
    )
    and (
        $3::timestamp is null
        or $4::timestamp is null
        or played_at between $3 and $4
    );

-- get play count by track
-- name: GetPlayCountByTrack :one
select count(*) as play_count
from etl_plays
where track_id = $1;

-- get play count by address
-- name: GetPlayCountByAddress :one
select count(*) as play_count
from etl_plays
where address = $1;

-- get validator registrations (deduplicated by address, keeping latest)
-- name: GetValidatorRegistrations :many
select distinct on (address) address,
    endpoint,
    comet_address,
    comet_pubkey,
    eth_block,
    node_type,
    spid,
    voting_power,
    block_height,
    tx_hash
from etl_validator_registrations
order by address, block_height desc;

-- get validator deregistrations
-- name: GetValidatorDeregistrations :many
select comet_address,
    comet_pubkey,
    block_height,
    tx_hash
from etl_validator_deregistrations;

-- name: GetPlaysByLocation :many
select tx_hash,
    address,
    track_id,
    played_at,
    city,
    region,
    country,
    created_at
from etl_plays
where (
        nullif($1, '')::text is null
        or lower(city) = lower($1)
    )
    and (
        nullif($2, '')::text is null
        or lower(region) = lower($2)
    )
    and (
        nullif($3, '')::text is null
        or lower(country) = lower($3)
    )
order by played_at desc
limit $4;

-- name: GetAvailableCities :many
select city,
    region,
    country,
    count(*) as play_count
from etl_plays
where city is not null
    and (
        nullif($1, '')::text is null
        or lower(country) = lower($1)
    )
    and (
        nullif($2, '')::text is null
        or lower(region) = lower($2)
    )
group by city,
    region,
    country
order by count(*) desc
limit $3;

-- name: GetAvailableRegions :many
select region,
    country,
    count(*) as play_count
from etl_plays
where region is not null
    and (
        nullif($1, '')::text is null
        or lower(country) = lower($1)
    )
group by region,
    country
order by count(*) desc
limit $2;

-- name: GetAvailableCountries :many
select country,
    count(*) as play_count
from etl_plays
where country is not null
group by country
order by count(*) desc
limit $1;

-- name: GetBlockTransactions :many
select * from etl_transactions
where block_height = $1;

-- name: SearchBlockHeight :many
select block_height
from etl_blocks
where block_height::text % $1
    and similarity(block_height::text, $1) > 0.4
    and block_height::text like $1 || '%'
order by similarity(block_height::text, $1) desc;

-- name: SearchTxHash :many
select tx_hash
from etl_transactions
where tx_hash % $1
    and similarity(tx_hash, $1) > 0.4
    and tx_hash like $1 || '%'
order by similarity(tx_hash, $1) desc;

-- name: SearchAddress :many
select address
from etl_manage_entities
where address % $1
    and similarity(address, $1) > 0.4
    and address like $1 || '%'
order by similarity(address, $1) desc;

-- name: SearchValidatorRegistration :many
select address
from etl_validator_registrations
where address % $1
    and similarity(address, $1) > 0.4
    and address like $1 || '%'
order by similarity(address, $1) desc;

-- name: GetTransaction :one
select t.*, b.block_time, b.proposer_address
from etl_transactions t
join etl_blocks b on t.block_height = b.block_height
where t.tx_hash = $1;

-- name: GetPlaysByTxHash :many
select address,
    track_id,
    extract(
        epoch
        from played_at
    )::bigint as timestamp,
    city,
    country,
    region,
    block_height,
    tx_hash
from etl_plays
where tx_hash = $1;

-- name: GetManageEntitiesByTxHash :many
select address,
    entity_type,
    entity_id,
    action,
    metadata,
    signature,
    signer,
    nonce,
    block_height,
    tx_hash
from etl_manage_entities
where tx_hash = $1;

-- name: GetValidatorRegistrationsByTxHash :many
select address,
    comet_address,
    comet_pubkey,
    eth_block,
    node_type,
    spid,
    voting_power,
    block_height,
    tx_hash
from etl_validator_registrations
where tx_hash = $1;

-- name: GetValidatorDeregistrationsByTxHash :many
select comet_address,
    comet_pubkey,
    block_height,
    tx_hash
from etl_validator_deregistrations
where tx_hash = $1;

-- name: GetSlaRollupsByTxHash :many
select block_start,
    block_end,
    timestamp,
    block_height,
    tx_hash
from etl_sla_rollups
where tx_hash = $1;

-- name: GetSlaNodeReportsByTxHash :many
select sla_rollup_id,
    address,
    num_blocks_proposed,
    block_height,
    tx_hash
from etl_sla_node_reports
where tx_hash = $1;

-- name: GetStorageProofsByTxHash :many
select address,
    height,
    prover_addresses,
    cid,
    proof_signature,
    block_height,
    tx_hash
from etl_storage_proofs
where tx_hash = $1;

-- name: GetStorageProofVerificationsByTxHash :many
select height,
    proof,
    block_height,
    tx_hash
from etl_storage_proof_verifications
where tx_hash = $1;

-- name: GetReleasesByTxHash :many
select release_data,
    block_height,
    tx_hash
from etl_releases
where tx_hash = $1;

-- Dashboard Statistics Queries

-- name: GetActiveValidatorsCount :one
select count(distinct r.comet_address) as count
from etl_validator_registrations r
left join etl_validator_deregistrations d on r.comet_address = d.comet_address
where d.comet_address is null;

-- name: GetRecentProposers :many
select distinct proposer_address
from etl_blocks
order by block_height desc
limit $1;

-- name: GetBlocksPerSecond :one
select case 
    when extract(epoch from (max(block_time) - min(block_time))) > 0 
    then (count(*) - 1)::float / extract(epoch from (max(block_time) - min(block_time)))
    else 0.0
end as bps
from etl_blocks
where block_time >= now() - interval '1 hour';

-- name: GetTransactionsPerSecond :one
select case 
    when extract(epoch from (max(b.block_time) - min(b.block_time))) > 0 
    then count(t.*)::float / extract(epoch from (max(b.block_time) - min(b.block_time)))
    else 0.0
end as tps
from etl_transactions t
join etl_blocks b on t.block_height = b.block_height
where b.block_time >= now() - interval '1 hour';

-- name: GetTransactionTypeBreakdown :many
select tx_type as type,
    count(*) as count
from etl_transactions t
join etl_blocks b on t.block_height = b.block_height
where b.block_time >= now() - interval '24 hours'
group by tx_type
order by count(*) desc;

-- name: GetLatestSLARollup :one
select * from etl_sla_rollups order by block_height desc limit 1;

-- name: GetBlocks :many
select * from etl_blocks where block_height between $1 and $2 order by block_height desc;

-- name: GetTransactionsCount :one
select count(*) from etl_transactions where block_height between $1 and $2;

-- name: GetTransactionsCountTimeRange :one
select count(*) as total
from etl_transactions t
join etl_blocks b on t.block_height = b.block_height
where b.block_time between $1 and $2;

-- Alternative subquery approach for transaction count by time range
-- name: GetTransactionsCountTimeRangeSubquery :one
select count(*) as total
from etl_transactions
where block_height in (
    select block_height 
    from etl_blocks 
    where block_time between $1 and $2
);
