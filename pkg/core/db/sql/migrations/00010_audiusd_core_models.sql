-- +migrate Up
create table if not exists audiusd_blocks (
  rowid bigserial primary key,
  height bigint not null,
  hash text not null,
  chain_id text not null,
  created_at timestamp not null,
  proposer text not null,

  unique (height, chain_id)
);

create index on audiusd_blocks (height, chain_id);
create index on audiusd_blocks (created_at);
create index on audiusd_blocks (proposer);

create table if not exists audiusd_txs (
  rowid bigserial primary key,
  hash text not null,
  signed_transaction bytea not null,
  tx_type text not null,
  chain_id text not null,
  
  block_height bigint not null,
  constraint fk_audiusd_txs_block_height foreign key (block_height, chain_id) references audiusd_blocks(height, chain_id) on delete cascade,

  created_at timestamp not null,
  constraint fk_audiusd_txs_created_at foreign key (created_at) references audiusd_blocks(created_at) on delete cascade,

  unique (hash, chain_id)
);

create index on audiusd_txs (hash, chain_id);
create index on audiusd_txs (block_height, chain_id);
create index on audiusd_txs (tx_type);

-- +migrate Down
drop index on audiusd_blocks (height, chain_id);
drop index on audiusd_blocks (created_at);
drop index on audiusd_blocks (proposer);

drop index on audiusd_txs (hash, chain_id);
drop index on audiusd_txs (block_height, chain_id);
drop index on audiusd_txs (tx_type);

drop table if exists audiusd_txs;
drop table if exists audiusd_blocks;
