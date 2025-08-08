create table if not exists eth_staked(
    address text primary key,
    total_staked bigint not null
);
