-- insert a new play record
-- name: InsertPlay :one
insert into etl_plays (
        address,
        track_id,
        city,
        region,
        country,
        played_at,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning *;

-- insert multiple play records with batch size control
-- name: InsertPlays :many
insert into etl_plays (
        address,
        track_id,
        city,
        region,
        country,
        played_at,
        block_height,
        tx_hash
    )
values (
        unnest($1::text []),
        unnest($2::text []),
        unnest($3::text []),
        unnest($4::text []),
        unnest($5::text []),
        unnest($6::timestamp []),
        unnest($7::bigint []),
        unnest($8::text [])
    ) on conflict do nothing
returning *;

-- insert a new block record
-- name: InsertBlock :one
insert into etl_blocks (
        proposer_address,
        block_height,
        block_time
    )
values ($1, $2, $3)
returning *;

-- delete plays by block height range (useful for reindexing)
-- name: DeletePlaysByBlockRange :exec
delete from etl_plays
where block_height between $1 and $2;

-- insert a new manage entity record
-- name: InsertManageEntity :one
insert into etl_manage_entities (
        address,
        entity_type,
        entity_id,
        action,
        metadata,
        signature,
        signer,
        nonce,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
returning *;

-- insert multiple manage entity records with batch size control
-- name: InsertManageEntities :many
insert into etl_manage_entities (
        address,
        entity_type,
        entity_id,
        action,
        metadata,
        signature,
        signer,
        nonce,
        block_height,
        tx_hash
    )
values (
        unnest($1::text []),
        unnest($2::text []),
        unnest($3::bigint []),
        unnest($4::text []),
        unnest($5::text []),
        unnest($6::text []),
        unnest($7::text []),
        unnest($8::text []),
        unnest($9::bigint []),
        unnest($10::text [])
    ) on conflict do nothing
returning *;

-- delete manage entities by block height range (useful for reindexing)
-- name: DeleteManageEntitiesByBlockRange :exec
delete from etl_manage_entities
where block_height between $1 and $2;

-- insert a new validator registration record
-- name: InsertValidatorRegistration :one
insert into etl_validator_registrations (
        address,
        endpoint,
        comet_address,
        eth_block,
        node_type,
        spid,
        comet_pubkey,
        voting_power,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
returning *;

-- insert a new validator deregistration record
-- name: InsertValidatorDeregistration :one
insert into etl_validator_deregistrations (
        comet_address,
        comet_pubkey,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4)
returning *;

-- insert a new transaction record
-- name: InsertTransaction :one
insert into etl_transactions (
        tx_hash,
        block_height,
        index,
        tx_type
    )
values ($1, $2, $3, $4)
returning *;

-- insert a new legacy validator registration record
-- name: InsertValidatorRegistrationLegacy :one
insert into etl_validator_registrations_legacy (
        endpoint,
        comet_address,
        eth_block,
        node_type,
        sp_id,
        pub_key,
        power,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
returning *;

-- insert a new SLA rollup record
-- name: InsertSlaRollup :one
insert into etl_sla_rollups (
        timestamp,
        block_start,
        block_end,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4, $5)
returning *;

-- insert a new SLA node report record
-- name: InsertSlaNodeReport :one
insert into etl_sla_node_reports (
        sla_rollup_id,
        address,
        num_blocks_proposed,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4, $5)
returning *;

-- insert a new validator misbehavior deregistration record
-- name: InsertValidatorMisbehaviorDeregistration :one
insert into etl_validator_misbehavior_deregistrations (
        comet_address,
        pub_key,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4)
returning *;

-- insert a new storage proof record
-- name: InsertStorageProof :one
insert into etl_storage_proofs (
        height,
        address,
        prover_addresses,
        cid,
        proof_signature,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4, $5, $6, $7)
returning *;

-- insert a new storage proof verification record
-- name: InsertStorageProofVerification :one
insert into etl_storage_proof_verifications (
        height,
        proof,
        block_height,
        tx_hash
    )
values ($1, $2, $3, $4)
returning *;

-- insert a new release record
-- name: InsertRelease :one
insert into etl_releases (
        release_data,
        block_height,
        tx_hash
    )
values ($1, $2, $3)
returning *;
