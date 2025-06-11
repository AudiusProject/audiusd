-- Additional ETL tables for missing transaction types

-- Table for legacy validator registrations (ValidatorRegistrationLegacy)
create table if not exists etl_validator_registrations_legacy(
  id serial primary key,
  endpoint text not null,
  comet_address text not null,
  eth_block text not null,
  node_type text not null,
  sp_id text not null,
  pub_key bytea not null,
  power bigint not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_validator_registrations_legacy_block_height_idx on etl_validator_registrations_legacy(block_height);
create index if not exists etl_validator_registrations_legacy_tx_hash_idx on etl_validator_registrations_legacy(tx_hash);
create index if not exists etl_validator_registrations_legacy_comet_address_idx on etl_validator_registrations_legacy(comet_address);

-- Table for SLA rollups
create table if not exists etl_sla_rollups(
  id serial primary key,
  timestamp timestamp not null,
  block_start bigint not null,
  block_end bigint not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_sla_rollups_block_height_idx on etl_sla_rollups(block_height);
create index if not exists etl_sla_rollups_tx_hash_idx on etl_sla_rollups(tx_hash);
create index if not exists etl_sla_rollups_timestamp_idx on etl_sla_rollups(timestamp);
create index if not exists etl_sla_rollups_block_range_idx on etl_sla_rollups(block_start, block_end);

-- Table for SLA node reports (part of SLA rollup)
create table if not exists etl_sla_node_reports(
  id serial primary key,
  sla_rollup_id integer not null references etl_sla_rollups(id),
  address text not null,
  num_blocks_proposed integer not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_sla_node_reports_sla_rollup_id_idx on etl_sla_node_reports(sla_rollup_id);
create index if not exists etl_sla_node_reports_address_idx on etl_sla_node_reports(address);
create index if not exists etl_sla_node_reports_block_height_idx on etl_sla_node_reports(block_height);
create index if not exists etl_sla_node_reports_tx_hash_idx on etl_sla_node_reports(tx_hash);

-- Table for validator misbehavior deregistrations
create table if not exists etl_validator_misbehavior_deregistrations(
  id serial primary key,
  comet_address text not null,
  pub_key bytea not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_validator_misbehavior_deregistrations_block_height_idx on etl_validator_misbehavior_deregistrations(block_height);
create index if not exists etl_validator_misbehavior_deregistrations_tx_hash_idx on etl_validator_misbehavior_deregistrations(tx_hash);
create index if not exists etl_validator_misbehavior_deregistrations_comet_address_idx on etl_validator_misbehavior_deregistrations(comet_address);

-- Table for storage proofs
create table if not exists etl_storage_proofs(
  id serial primary key,
  height bigint not null,
  address text not null,
  prover_addresses text[] not null,
  cid text not null,
  proof_signature bytea,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_storage_proofs_block_height_idx on etl_storage_proofs(block_height);
create index if not exists etl_storage_proofs_tx_hash_idx on etl_storage_proofs(tx_hash);
create index if not exists etl_storage_proofs_height_idx on etl_storage_proofs(height);
create index if not exists etl_storage_proofs_address_idx on etl_storage_proofs(address);

-- Table for storage proof verifications
create table if not exists etl_storage_proof_verifications(
  id serial primary key,
  height bigint not null,
  proof bytea not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_storage_proof_verifications_block_height_idx on etl_storage_proof_verifications(block_height);
create index if not exists etl_storage_proof_verifications_tx_hash_idx on etl_storage_proof_verifications(tx_hash);
create index if not exists etl_storage_proof_verifications_height_idx on etl_storage_proof_verifications(height);

-- Table for releases (DDEX)
create table if not exists etl_releases(
  id serial primary key,
  release_data jsonb not null,
  block_height bigint not null,
  tx_hash text not null,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp
);

create index if not exists etl_releases_block_height_idx on etl_releases(block_height);
create index if not exists etl_releases_tx_hash_idx on etl_releases(tx_hash);
create index if not exists etl_releases_release_data_gin_idx on etl_releases using gin (release_data); 
