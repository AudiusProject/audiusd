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
  proposer_address text not null,
  block_height bigint not null,
  block_time timestamp not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_blocks_block_height_idx on etl_blocks(block_height);

create index if not exists etl_blocks_block_time_idx on etl_blocks(block_time);

create table if not exists etl_validator_registrations(
  id serial primary key,
  address text not null,
  endpoint text not null,
  comet_address text not null,
  eth_block text not null,
  node_type text not null,
  spid text not null,
  comet_pubkey bytea not null,
  voting_power bigint not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_validator_registrations_block_height_idx on etl_validator_registrations(block_height);

create index if not exists etl_validator_registrations_tx_hash_idx on etl_validator_registrations(tx_hash);

create table if not exists etl_validator_deregistrations(
  id serial primary key,
  comet_address text not null,
  comet_pubkey bytea not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_validator_deregistrations_block_height_idx on etl_validator_deregistrations(block_height);

create index if not exists etl_validator_deregistrations_tx_hash_idx on etl_validator_deregistrations(tx_hash);

create table if not exists etl_accounts(
  id serial primary key,
  address text not null,
  pubkey bytea not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_accounts_address_idx on etl_accounts(address);

create extension if not exists pg_trgm;

create index if not exists etl_accounts_address_trgm on etl_accounts using gin (address gin_trgm_ops);

create index if not exists etl_transactions_tx_hash_trgm on etl_transactions using gin (tx_hash gin_trgm_ops);

create index if not exists etl_validators_address_trgm on etl_validators using gin (address gin_trgm_ops);

create table if not exists etl_transactions(
  id serial primary key,
  tx_hash text not null,
  block_height bigint not null,
  index bigint not null,
  tx_type text not null,
  tx_data bytea not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_transactions_tx_hash_idx on etl_transactions(tx_hash);

create index if not exists etl_transactions_block_height_idx on etl_transactions(block_height);

create index if not exists etl_transactions_tx_type_idx on etl_transactions(tx_type);