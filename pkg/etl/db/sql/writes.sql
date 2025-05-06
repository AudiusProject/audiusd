-- insert a new play record
-- name: InsertPlay :one
insert into etl_plays (
    address,
    track_id,
    city,
    region,
    country,
    played_at,
    block_height,
    tx_hash
) values (
    $1, $2, $3, $4, $5, $6, $7, $8
) returning *;

-- insert multiple play records with batch size control
-- name: InsertPlays :many
insert into etl_plays (
    address,
    track_id,
    city,
    region,
    country,
    played_at,
    block_height,
    tx_hash
) values (
    unnest($1::text[]),
    unnest($2::text[]),
    unnest($3::text[]),
    unnest($4::text[]),
    unnest($5::text[]),
    unnest($6::timestamp[]),
    unnest($7::bigint[]),
    unnest($8::text[])
)
on conflict do nothing
returning *;

-- update latest indexed block
-- name: UpdateLatestIndexedBlock :one
insert into etl_latest_indexed_block (block_height)
values ($1)
returning *;

-- delete plays by block height range (useful for reindexing)
-- name: DeletePlaysByBlockRange :exec
delete from etl_plays
where block_height between $1 and $2;
