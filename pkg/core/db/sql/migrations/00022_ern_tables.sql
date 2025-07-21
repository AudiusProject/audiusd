-- +migrate Up
create table if not exists core_ern (
    id bigserial primary key,
    address text not null,
    sender text not null,
    nonce bigint not null,
    message_control_type smallint not null,
    party_addresses text[] not null,
    resource_addresses text[] not null,
    release_addresses text[] not null,
    deal_addresses text[] not null,
    raw_message bytea not null,
    block_height bigint not null
);

create index if not exists idx_core_ern_address on core_ern (address);
create index if not exists idx_core_ern_nonce on core_ern (nonce);
create index if not exists idx_core_ern_block_height on core_ern (block_height);
create index if not exists idx_core_ern_message_control_type on core_ern (message_control_type);
create index if not exists idx_core_ern_sender on core_ern (sender);

create index if not exists idx_core_ern_party_addresses_gin on core_ern using gin (party_addresses);
create index if not exists idx_core_ern_resource_addresses_gin on core_ern using gin (resource_addresses);
create index if not exists idx_core_ern_release_addresses_gin on core_ern using gin (release_addresses);
create index if not exists idx_core_ern_deal_addresses_gin on core_ern using gin (deal_addresses);

create table if not exists core_mead(
    id bigserial primary key,
    address text not null,
    sender text not null,
    nonce bigint not null,
    message_control_type smallint not null,
    resource_addresses text[] not null,
    release_addresses text[] not null,
    raw_message bytea not null,
    block_height bigint not null
);

create index if not exists idx_core_mead_address on core_mead (address);
create index if not exists idx_core_mead_nonce on core_mead (nonce);
create index if not exists idx_core_mead_block_height on core_mead (block_height);
create index if not exists idx_core_mead_message_control_type on core_mead (message_control_type);
create index if not exists idx_core_mead_sender on core_mead (sender);

create index if not exists idx_core_mead_resource_addresses_gin on core_mead using gin (resource_addresses);
create index if not exists idx_core_mead_release_addresses_gin on core_mead using gin (release_addresses);

create table if not exists core_pie(
    id bigserial primary key,
    address text not null,
    sender text not null,
    nonce bigint not null,
    message_control_type smallint not null,
    party_addresses text[] not null,
    raw_message bytea not null,
    block_height bigint not null
);

create index if not exists idx_core_pie_address on core_pie (address);
create index if not exists idx_core_pie_nonce on core_pie (nonce);
create index if not exists idx_core_pie_block_height on core_pie (block_height);
create index if not exists idx_core_pie_message_control_type on core_pie (message_control_type);
create index if not exists idx_core_pie_sender on core_pie (sender);

create index if not exists idx_core_pie_party_addresses_gin on core_pie using gin (party_addresses);

-- +migrate Down
drop index if exists idx_core_mead_address;
drop index if exists idx_core_mead_nonce;
drop index if exists idx_core_mead_block_height;
drop index if exists idx_core_mead_message_control_type;
drop index if exists idx_core_mead_sender;
drop index if exists idx_core_mead_resource_addresses_gin;
drop index if exists idx_core_mead_release_addresses_gin;

drop table if exists core_mead;

drop index if exists idx_core_pie_address;
drop index if exists idx_core_pie_nonce;
drop index if exists idx_core_pie_block_height;
drop index if exists idx_core_pie_message_control_type;
drop index if exists idx_core_pie_sender;
drop index if exists idx_core_pie_party_addresses_gin;

drop table if exists core_pie;

drop index if exists idx_core_ern_address;
drop index if exists idx_core_ern_nonce;
drop index if exists idx_core_ern_block_height;
drop index if exists idx_core_ern_message_control_type;
drop index if exists idx_core_ern_sender;
drop index if exists idx_core_ern_party_addresses_gin;
drop index if exists idx_core_ern_resource_addresses_gin;
drop index if exists idx_core_ern_release_addresses_gin;
drop index if exists idx_core_ern_deal_addresses_gin;

drop table if exists core_ern;
