-- +migrate Up

create table if not exists core_decoded_tx (
    id bigserial primary key,
    block_height bigint not null,
    tx_index integer not null,
    tx_hash text not null,
    tx_type text not null,
    created_at timestamp with time zone not null,

    -- Common fields for all transactions
    signature text,
    request_id text,

    -- ValidatorRegistration specific fields
    validator_endpoint text,
    validator_comet_address text,
    validator_eth_block text,
    validator_node_type text,
    validator_sp_id text,
    validator_pub_key bytea,
    validator_power bigint,

    -- ValidatorDeregistration specific fields
    deregistration_comet_address text,
    deregistration_pub_key bytea,

    -- SlaRollup specific fields
    sla_timestamp timestamp with time zone,
    sla_block_start bigint,
    sla_block_end bigint,
    sla_reports jsonb, -- Array of node reports

    -- StorageProof specific fields
    storage_proof_height bigint,
    storage_proof_address text,
    storage_proof_prover_addresses text[],
    storage_proof_cid text,
    storage_proof_signature bytea,

    -- StorageProofVerification specific fields
    storage_verification_height bigint,
    storage_verification_proof bytea,

    -- ManageEntityLegacy specific fields
    manage_entity_user_id bigint,
    manage_entity_type text,
    manage_entity_id bigint,
    manage_entity_action text,
    manage_entity_metadata text,
    manage_entity_signature text,
    manage_entity_signer text,
    manage_entity_nonce text,

    unique(block_height, tx_index),
    unique(tx_hash)
);

-- Create a separate table for track plays
create table if not exists core_decoded_tx_plays (
    id bigserial primary key,
    tx_hash text not null references core_decoded_tx(tx_hash),
    user_id text not null,
    track_id text not null,
    timestamp timestamp with time zone not null,
    signature text,
    city text,
    region text,
    country text
);

create index if not exists core_decoded_tx_block_height_idx on core_decoded_tx(block_height);
create index if not exists core_decoded_tx_tx_hash_idx on core_decoded_tx(tx_hash);
create index if not exists core_decoded_tx_tx_type_idx on core_decoded_tx(tx_type);
create index if not exists core_decoded_tx_validator_comet_address_idx on core_decoded_tx(validator_comet_address);
create index if not exists core_decoded_tx_storage_proof_address_idx on core_decoded_tx(storage_proof_address);
create index if not exists core_decoded_tx_manage_entity_user_id_idx on core_decoded_tx(manage_entity_user_id);

-- Add indexes for commonly queried play fields
create index if not exists core_decoded_tx_plays_tx_hash_idx on core_decoded_tx_plays(tx_hash);
create index if not exists core_decoded_tx_plays_user_id_idx on core_decoded_tx_plays(user_id);
create index if not exists core_decoded_tx_plays_track_id_idx on core_decoded_tx_plays(track_id);
create index if not exists core_decoded_tx_plays_timestamp_idx on core_decoded_tx_plays(timestamp);
create index if not exists core_decoded_tx_plays_country_idx on core_decoded_tx_plays(country);

-- +migrate Down
drop table if exists core_decoded_tx_plays;
drop table if exists core_decoded_tx;
