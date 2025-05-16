-- +migrate Up
alter table sla_rollups
add column funding_round int;

create index if not exists idx_sla_rollups_funding_round on sla_rollups(funding_round);

create table if not exists funding_round_sla_results(
  id serial primary key,
  funding_round int not null,
  block_number int not null,
  comet_address text not null,
  eth_address text not null,
  total_sla_rollups int not null,
  sla_misses int not null,
  total_challenges int not null,
  failed_challengs int not null,
  reward_proportion float not null,
  finalized boolean not null
);

create index if not exists idx_funding_round_sla_results on funding_round_sla_results(funding_round desc);

-- +migrate Down
alter table sla_rollups
drop column funding_round;

drop index if exists idx_sla_rollups_funding_round;

drop table if exists funding_round_sla_results;
drop index if exists idx_funding_round_sla_results;
