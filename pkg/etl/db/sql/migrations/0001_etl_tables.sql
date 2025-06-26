create table if not exists etl_addresses(
  id serial primary key,
  address text not null,
  pub_key bytea,
  first_seen_block_height bigint,
  created_at timestamp not null
);


create table if not exists etl_transactions(
  id serial primary key,
  tx_hash text not null,
  block_height bigint not null,
  tx_index integer not null,
  tx_type text not null,
  created_at timestamp not null
);

create table if not exists etl_blocks(
  id serial primary key,
  proposer_address text not null,
  block_height bigint not null,
  block_time timestamp not null
);

create table if not exists etl_plays(
  id serial primary key,
  user_id text not null,
  track_id text not null,
  city text not null,
  region text not null,
  country text not null,
  played_at timestamp not null,
  block_height bigint not null,
  tx_hash text not null,
  listened_at timestamp not null,
  recorded_at timestamp not null
);

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
  created_at timestamp not null
);

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
  tx_hash text not null
);

create table if not exists etl_validator_deregistrations(
  id serial primary key,
  comet_address text not null,
  comet_pubkey bytea not null,
  block_height bigint not null,
  tx_hash text not null
);

create table if not exists etl_sla_rollups(
  id serial primary key,
  block_start bigint not null,
  block_end bigint not null,
  block_height bigint not null,
  validator_count integer not null,
  block_quota integer not null,
  tx_hash text not null,
  created_at timestamp not null
);

create table if not exists etl_sla_node_reports(
  id serial primary key,
  sla_rollup_id integer not null references etl_sla_rollups(id),
  address text not null,
  num_blocks_proposed integer not null,
  challenges_received integer not null,
  challenges_failed integer not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp not null
);

create table if not exists etl_validator_misbehavior_deregistrations(
  id serial primary key,
  comet_address text not null,
  pub_key bytea not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp not null
);

create table if not exists etl_storage_proofs(
  id serial primary key,
  height bigint not null,
  address text not null,
  prover_addresses text[] not null,
  cid text not null,
  proof_signature bytea,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp not null
);

create table if not exists etl_storage_proof_verifications(
  id serial primary key,
  height bigint not null,
  proof bytea not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp not null
);

create table if not exists etl_validators(
  id serial primary key,
  address text not null,
  endpoint text not null,
  comet_address text not null,
  node_type text not null,
  spid text not null,
  voting_power bigint not null,
  status text not null,
  registered_at bigint not null,
  deregistered_at bigint,
  created_at timestamp not null,
  updated_at timestamp not null
);
