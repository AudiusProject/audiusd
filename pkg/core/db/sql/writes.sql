-- name: UpsertAppState :exec
insert into core_app_state (block_height, app_hash)
values ($1, $2);

-- name: InsertRegisteredNode :exec
insert into core_validators(pub_key, endpoint, eth_address, comet_address, comet_pub_key, eth_block, node_type, sp_id)
values ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: DeleteRegisteredNode :exec
delete from core_validators
where comet_address = $1;

-- name: UpsertSlaRollupReport :exec
with updated as (
    update sla_node_reports 
    set blocks_proposed = blocks_proposed + 1
    where address = $1 and sla_rollup_id is null
    returning *
)
insert into sla_node_reports (address, blocks_proposed, sla_rollup_id)
select $1, 1, null
where not exists (select 1 from updated);

-- name: ClearUncommittedSlaNodeReports :exec
delete from sla_node_reports
where sla_rollup_id is null;

-- name: CommitSlaNodeReport :exec
insert into sla_node_reports (sla_rollup_id, address, blocks_proposed)
values ($1, $2, $3);

-- name: CommitSlaRollup :one
insert into sla_rollups (time, tx_hash, block_start, block_end)
values ($1, $2, $3, $4)
returning id;

-- name: InsertTxStat :exec
insert into core_tx_stats (tx_type, tx_hash, block_height, created_at)
values ($1, $2, $3, $4)
on conflict (tx_hash) do nothing;

-- name: StoreBlock :exec
insert into core_blocks (height, chain_id, hash, proposer, created_at)
values ($1, $2, $3, $4, $5);

-- name: StoreTransaction :exec
insert into core_transactions (block_id, index, tx_hash, transaction, created_at)
values ($1, $2, $3, $4, $5);

-- name: InsertStorageProofPeers :exec
insert into storage_proof_peers (block_height, prover_addresses)
values ($1, $2);

-- name: InsertStorageProof :exec
insert into storage_proofs (block_height, address, cid, proof_signature, prover_addresses)
values ($1, $2, $3, $4, $5);

-- name: UpdateStorageProof :exec
update storage_proofs 
set proof = $1, status = $2
where block_height = $3 and address = $4;

-- name: InsertFailedStorageProof :exec
insert into storage_proofs (block_height, address, status)
values ($1, $2, 'fail');

-- name: InsertDecodedTx :exec
insert into core_etl_tx (
    block_height,
    tx_index,
    tx_hash,
    tx_type,
    tx_data,
    created_at
) values (
    $1, $2, $3, $4, $5, $6
);

-- name: InsertDecodedPlay :exec
insert into core_etl_tx_plays (
    tx_hash,
    user_id,
    track_id,
    played_at,
    signature,
    city,
    region,
    country,
    created_at
) values (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: InsertDecodedValidatorRegistration :exec
insert into core_etl_tx_validator_registration (
    tx_hash,
    endpoint,
    comet_address,
    eth_block,
    node_type,
    sp_id,
    pub_key,
    power,
    created_at
) values (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: InsertDecodedValidatorDeregistration :exec
insert into core_etl_tx_validator_deregistration (
    tx_hash,
    comet_address,
    pub_key,
    created_at
) values (
    $1, $2, $3, $4
);

-- name: InsertDecodedSlaRollup :exec
insert into core_etl_tx_sla_rollup (
    tx_hash,
    block_start,
    block_end,
    timestamp,
    created_at
) values (
    $1, $2, $3, $4, $5
);

-- name: InsertDecodedStorageProof :exec
insert into core_etl_tx_storage_proof (
    tx_hash,
    height,
    address,
    cid,
    proof_signature,
    prover_addresses,
    created_at
) values (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: InsertDecodedStorageProofVerification :exec
insert into core_etl_tx_storage_proof_verification (
    tx_hash,
    height,
    proof,
    created_at
) values (
    $1, $2, $3, $4
);

-- name: InsertDecodedManageEntity :exec
insert into core_etl_tx_manage_entity (
    tx_hash,
    user_id,
    entity_type,
    entity_id,
    action,
    metadata,
    signature,
    signer,
    nonce,
    created_at
) values (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);
