-- +migrate Up
create table if not exists core_plays(
    rowid serial primary key,
    user_id text not null,
    track_id text not null,

    -- future fields
    listener_address text not null,
    cid text not null,

    city text not null,
    country text not null,
    region text not null,
    signer text not null,
    signature text not null,
    block bigint not null,
    created_at timestamptz not null
)

-- +migrate Down
drop table if exists core_plays;
