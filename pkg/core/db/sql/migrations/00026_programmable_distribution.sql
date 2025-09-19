-- +migrate Up
create table if not exists core_uploads(
  id bigserial primary key,
  uploader_address text not null,
  cid text not null,
  upid text not null,
  upload_signature text not null,
  tx_hash text not null,
  block_height bigint not null
);

-- +migrate Down
drop table if exists core_uploads;
