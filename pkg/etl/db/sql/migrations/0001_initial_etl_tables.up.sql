create table if not exists etl_plays(
  id serial primary key,
  address text not null,
  track_id text not null,
  city text not null,
  region text not null,
  country text not null,
  played_at timestamp not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_plays_address_idx on etl_plays(address);
create index if not exists etl_plays_track_id_idx on etl_plays(track_id);
create index if not exists etl_plays_played_at_idx on etl_plays(played_at);
create index if not exists etl_plays_block_height_idx on etl_plays(block_height);
create index if not exists etl_plays_tx_hash_idx on etl_plays(tx_hash);

create table if not exists etl_manage_entities(
  id serial primary key,
  address text not null,
  entity_type text not null,
  entity_id bigint not null,
  action text not null,
  metadata text,
  signature text not null,
  signer text not null,
  nonce text not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_manage_entities_block_height_idx on etl_manage_entities(block_height);
create index if not exists etl_manage_entities_tx_hash_idx on etl_manage_entities(tx_hash);

create table if not exists etl_blocks(
  id serial primary key,
  block_height bigint not null,
  block_time timestamp not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_blocks_block_height_idx on etl_blocks(block_height);
create index if not exists etl_blocks_block_time_idx on etl_blocks(block_time);

