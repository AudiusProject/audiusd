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

create table if not exists etl_latest_indexed_block(
  id serial primary key,
  block_height bigint not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_latest_indexed_block_block_height_idx on etl_latest_indexed_block(block_height);

create table if not exists etl_blocks(
  id serial primary key,
  height bigint not null,
  time timestamp not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_blocks_height_idx on etl_blocks(height);
create index if not exists etl_blocks_time_idx on etl_blocks(time);
