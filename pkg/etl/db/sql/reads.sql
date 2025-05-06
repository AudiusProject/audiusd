-- get latest indexed block height
-- name: GetLatestIndexedBlock :one
select block_height 
from etl_latest_indexed_block 
order by id desc 
limit 1;

-- get plays with filtering, pagination, and ordering
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
    ($1::text is null or address = $1)
    and ($2::text is null or track_id = $2)
    and ($3::timestamp is null or $4::timestamp is null or played_at between $3 and $4)
order by 
    case 
        when $5 = 'played_at' and $6 = 'asc' then played_at
        when $5 = 'block_height' and $6 = 'asc' then block_height
    end asc,
    case 
        when $5 = 'played_at' and $6 = 'desc' then played_at
        when $5 = 'block_height' and $6 = 'desc' then block_height
    end desc
limit $7 offset $8;

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
