-- get latest indexed block height
-- name: GetLatestIndexedBlock :one
select block_height 
from etl_blocks 
order by id desc 
limit 1;

-- name: GetBlockRangeByTime :one
select
  min(block_height) as start_block,
  max(block_height) as end_block
from etl_blocks
where block_time between $1 and $2;


-- name: GetPlaysByAddress :many
select 
    address,
    track_id,
    extract(epoch from played_at)::bigint as timestamp,
    city,
    country,
    region,
    block_height,
    tx_hash
from etl_plays
where 
    address = $1
    and block_height between $2 and $3
order by played_at desc
limit $4 offset $5;


-- name: GetPlaysByTrack :many
select 
    address,
    track_id,
    extract(epoch from played_at)::bigint as timestamp,
    city,
    country,
    region,
    block_height,
    tx_hash
from etl_plays
where 
    track_id = $1
    and block_height between $2 and $3
order by played_at desc
limit $4 offset $5;

-- name: GetPlays :many
select 
    address,
    track_id,
    extract(epoch from played_at)::bigint as timestamp,
    city,
    country,
    region,
    block_height,
    tx_hash
from etl_plays
where 
    block_height between $1 and $2
order by played_at desc
limit $3 offset $4;


-- get total count of plays with filtering
-- name: GetPlaysCount :one
select count(*) as total
from etl_plays
where 
    ($1::text is null or address = $1)
    and ($2::text is null or track_id = $2)
    and ($3::timestamp is null or $4::timestamp is null or played_at between $3 and $4);

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

-- get validator registrations
-- name: GetValidatorRegistrations :many
select 
    address,
    comet_address,
    comet_pubkey,
    eth_block,
    node_type,
    spid,
    voting_power,
    block_height,
    tx_hash
from etl_validator_registrations;

-- get validator deregistrations
-- name: GetValidatorDeregistrations :many
select 
    comet_address,
    comet_pubkey,
    block_height,
    tx_hash
from etl_validator_deregistrations;
