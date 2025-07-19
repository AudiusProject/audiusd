-- +migrate Up
create table if not exists core_ern (
    id bigserial primary key,
    address text not null,
    sender text not null,
    nonce bigint not null,
    message_control_type smallint not null,
    raw_message bytea not null,
    block_height bigint not null
);

create index if not exists idx_core_ern_address on core_ern (address);
create index if not exists idx_core_ern_nonce on core_ern (nonce);
create index if not exists idx_core_ern_block_height on core_ern (block_height);
create index if not exists idx_core_ern_message_control_type on core_ern (message_control_type);
create index if not exists idx_core_ern_sender on core_ern (sender);

create table if not exists core_mead(
    id bigserial primary key,
    address text not null,
    sender text not null,
    nonce bigint not null,
    message_control_type smallint not null,
    ern_address text not null,
    raw_message bytea not null,
    block_height bigint not null,
);

create index if not exists idx_core_mead_address on core_mead (address);
create index if not exists idx_core_mead_nonce on core_mead (nonce);
create index if not exists idx_core_mead_block_height on core_mead (block_height);
create index if not exists idx_core_mead_message_control_type on core_mead (message_control_type);
create index if not exists idx_core_mead_sender on core_mead (sender);

create table if not exists core_pie(
    id bigserial primary key,
    address text not null,
    sender text not null,
    nonce bigint not null,
    message_control_type smallint not null,
    ern_address text not null,
    raw_message bytea not null,
    block_height bigint not null,
);

create index if not exists idx_core_pie_address on core_pie (address);
create index if not exists idx_core_pie_nonce on core_pie (nonce);
create index if not exists idx_core_pie_block_height on core_pie (block_height);
create index if not exists idx_core_pie_message_control_type on core_pie (message_control_type);
create index if not exists idx_core_pie_sender on core_pie (sender);

-- +migrate Down
drop index if exists idx_core_mead_address;
drop index if exists idx_core_mead_nonce;
drop index if exists idx_core_mead_block_height;
drop index if exists idx_core_mead_message_control_type;
drop index if exists idx_core_mead_sender;

drop table if exists core_mead;

drop index if exists idx_core_pie_address;
drop index if exists idx_core_pie_nonce;
drop index if exists idx_core_pie_block_height;
drop index if exists idx_core_pie_message_control_type;
drop index if exists idx_core_pie_sender;

drop table if exists core_pie;

drop index if exists idx_core_ern_address;
drop index if exists idx_core_ern_nonce;
drop index if exists idx_core_ern_block_height;
drop index if exists idx_core_ern_message_control_type;
drop index if exists idx_core_ern_sender;

drop table if exists core_ern;
