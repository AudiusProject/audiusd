-- +migrate Up

create table if not exists core_decoded_tx (
    id bigserial primary key,
    block_height bigint not null,
    tx_index integer not null,
    tx_hash text not null,
    tx_type text not null,
    tx_data jsonb not null,
    created_at timestamp with time zone not null,
    unique(block_height, tx_index),
    unique(tx_hash)
);

create index if not exists core_decoded_tx_block_height_idx on core_decoded_tx(block_height);
create index if not exists core_decoded_tx_tx_hash_idx on core_decoded_tx(tx_hash);
create index if not exists core_decoded_tx_tx_type_idx on core_decoded_tx(tx_type);

-- +migrate Down
drop table if exists core_decoded_tx;
