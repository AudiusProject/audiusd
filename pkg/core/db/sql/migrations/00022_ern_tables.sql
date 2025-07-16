-- +migrate Up

-- Core ERN message storage with raw protobuf
create table if not exists ern_messages (
    id bigserial primary key,
    address text not null unique,
    tx_hash text not null,
    block_height bigint not null,
    sender_address text not null,
    raw_ern_message bytea not null -- Serialized NewReleaseMessage protobuf
);

-- links releases contained in an ern message to the ern message
create table if not exists ern_release_addresses (
    id bigserial primary key,
    address text not null unique,
    ern_address text not null references ern_messages(address)
);

-- links sound recordings contained in an ern message to the ern message
create table if not exists ern_sound_recording_addresses (
    id bigserial primary key,
    address text not null unique,
    ern_address text not null references ern_messages(address)
);
