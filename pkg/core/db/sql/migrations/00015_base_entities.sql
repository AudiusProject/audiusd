-- +migrate Up

create table if not exists users(
  rowid bigserial primary key,
  pubkey bytea not null
);

create table if not exists tracks(
  rowid bigserial primary key,
  cid text not null
);

create table if not exists follows(
  rowid bigserial primary key,
  follower bytea not null,
  following bytea not null
);

create table if not exists saves(
  rowid bigserial primary key,
  user bytea not null,
  track cid text not null
);

create table if not exists reposts(
  rowid bigserial primary key,
  user bytea not null,
  track text not null
);

create table if not exists plays(
  rowid bigserial primary key,
  track text not null
);

-- +migrate Down
