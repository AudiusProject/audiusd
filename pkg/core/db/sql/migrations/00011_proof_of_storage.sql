-- +migrate Up

-- incomplete: challenge has not finialized or passed deadline yet
-- complete: challenge has been finalized and is valid
-- invalid: challenge was completed but results are inconclusive
-- fault: verifier failed to properly evaluate and share results
create type challenge_status as enum ('incomplete', 'complete', 'invalid', 'fault');
create table pos_challenges(
  id serial primary key,
  block_height bigint not null unique,
  verifier_address text not null,
  cid text,
  status challenge_status not null default 'incomplete'
);

-- incomplete: challenge has not finialized or passed deadline yet
-- pass: node passed the challenge
-- fail: node failed the challenge
-- exempt: challenge was deemed inconclusive or verifier faulted
create type proof_status as enum ('incomplete', 'pass', 'fail', 'exempt');
create table storage_proofs(
  id serial primary key,
  block_height bigint not null,
  address text not null,
  encrypted_proof text,
  decrypted_proof text,
  status proof_status not null default 'incomplete',
  unique (address, block_height)
);

create index idx_block_height on pos_challenges(block_height desc);
create index idx_block_height on storage_proofs(block_height desc);

-- +migrate Down
drop table if exists pos_challenges;
drop table if exists storage_proofs;
drop type if exists challenge_status;
drop type if exists proof_status;
