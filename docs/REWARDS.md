# Programmatic Rewards Implementation

This document describes the implementation of programmatic rewards on the Audius chain, allowing rewards to be deployed, updated, and deleted on-chain with deterministic addresses.

## Overview

The programmatic rewards system allows rewards to be managed entirely on-chain through transactions, similar to how smart contracts work on Ethereum. Each reward gets a deterministic address that can be used to reference and modify it later.

## Architecture

### Core Components

1. **Protocol Buffers** (`proto/core/v1/types.proto`)
   - `RewardMessage` with Create/Update/Delete actions as a oneof
   - Added to `SignedTransaction` as transaction type 1009
   - `GetRewardRequest/Response` for querying rewards

2. **Database Schema** (`migrations/00026_programmatic_rewards.sql`)
   - `core_rewards` table with proper indexing
   - `claim_authorities` as text array with GIN index for efficient queries

3. **Server Implementation** (`pkg/core/server/rewards.go`)
   - Transaction finalization handlers for each operation
   - Authorization checks using claim authorities
   - Deterministic address generation

4. **ABCI Integration** (`pkg/core/server/abci.go`)
   - Transaction validation and routing
   - Signature verification and sender recovery

## Transaction Types

### 1. CreateReward
Creates a new reward with a deterministic address.

```protobuf
message CreateReward {
  string reward_id = 1;
  string name = 2;
  uint64 amount = 3;
  repeated ClaimAuthority claim_authorities = 4;
}
```

**Process:**
1. Generate deterministic address using `common.CreateAddress()`
2. Store reward with claim authorities
3. No authorization required (creator becomes authority)

### 2. UpdateReward
Updates an existing reward by its deployed address.

```protobuf
message UpdateReward {
  string address = 1; // The deployed reward address
  string name = 2;
  uint64 amount = 3;
  repeated ClaimAuthority claim_authorities = 4;
}
```

**Process:**
1. Verify sender is in existing reward's claim authorities
2. Update reward data with new values
3. Authorization: Only claim authorities can update

### 3. DeleteReward
Marks a reward as deleted by its deployed address.

```protobuf
message DeleteReward {
  string address = 1; // The deployed reward address
}
```

**Process:**
1. Verify sender is in existing reward's claim authorities
2. Record deletion transaction
3. Authorization: Only claim authorities can delete

## Security Model

### Signature Verification
```go
// Recover sender address from signature
rewardBytes, err := proto.Marshal(msg.GetReward())
_, sender, err := common.EthRecover(msg.Signature, rewardBytes)
```

### Authorization Checks
```go
// Check if sender is authorized (for Update/Delete)
authorized := false
for _, auth := range existingReward.ClaimAuthorities {
    if auth == sender {
        authorized = true
        break
    }
}
```

## Deterministic Addresses

Rewards use the same deterministic address generation as ERNs:

```go
rewardAddress := common.CreateAddress(createReward, chainID, blockHeight, txHash)
```

This generates addresses similar to Ethereum's CREATE2 opcode:
- **Content Hash**: Keccak256 of the serialized reward message
- **Salt**: Derived from txHash, chainID, and blockHeight
- **Final Address**: Last 20 bytes of Keccak256(0xff || chainID || saltHash || contentHash)

## Database Schema

```sql
create table core_rewards (
    id bigserial primary key,
    address text not null,
    index bigint not null,
    tx_hash text not null,
    sender text not null,
    reward_id text not null,
    name text not null,
    amount bigint not null,
    claim_authorities text[] default '{}',
    raw_message bytea not null,
    block_height bigint not null,
    created_at timestamp with time zone default now(),
    updated_at timestamp with time zone default now()
);

-- Efficient indexing
create index idx_core_rewards_address on core_rewards (address);
create index idx_core_rewards_reward_id on core_rewards (reward_id);
create index idx_core_rewards_claim_authorities on core_rewards using gin (claim_authorities);
```

## API Endpoints

### GetRewards
Returns all active rewards (latest version of each reward).

```protobuf
rpc GetRewards(GetRewardsRequest) returns (GetRewardsResponse)
```

### GetReward
Returns a specific reward by its deployed address.

```protobuf
rpc GetReward(GetRewardRequest) returns (GetRewardResponse)
```

### GetRewardAttestation
Generates attestations for reward claims, now integrated with programmatic rewards.

```protobuf
rpc GetRewardAttestation(GetRewardAttestationRequest) returns (GetRewardAttestationResponse)
```

**Updated Request Format:**
```protobuf
message GetRewardAttestationRequest {
  string eth_recipient_address = 1;
  string reward_address = 2; // The deployed reward address (instead of reward_id)
  string specifier = 3;
  string oracle_address = 4;
  string signature = 5;
  uint64 amount = 6;
}
```

**Attestation Process:**
1. **Reward Validation**: Looks up the programmatic reward by its deployed address
2. **Authorization Check**: Verifies the oracle_address is in the reward's claim_authorities
3. **Claim Processing**: Uses the reward's stored reward_id for traditional reward claim flow
4. **Attestation Generation**: Returns signed attestation for the reward claim

This bridges the gap between the new programmatic rewards system and the existing reward claiming infrastructure.

## Dual API Architecture

The implementation provides two complementary APIs to ensure backwards compatibility:

### gRPC/Connect API (`pkg/core/server/connect.go`)
**Modern, Feature-Complete Interface**

- **Programmatic Rewards**: Use `reward_address` parameter to reference deployed rewards
- **Legacy Rewards**: Use `reward_id` parameter for hardcoded rewards (backwards compatible)
- **Unified Response**: Both programmatic and legacy rewards in `GetRewards()` response
- **Authorization**: Validates oracle permissions for both reward types

```protobuf
// New programmatic reward attestation
GetRewardAttestation({
  reward_address: "0xABC123...",
  oracle_address: "0xOracle1",
  // ... other params
})

// Legacy hardcoded reward attestation
GetRewardAttestation({
  reward_id: "weekly_bonus",
  oracle_address: "0xOracle1",
  // ... other params
})
```

### REST API (`pkg/core/server/http.go`)
**Legacy-Compatible Interface**

- **Legacy Only**: Uses existing `reward_id` parameter unchanged
- **Hardcoded Rewards**: Only works with original hardcoded rewards system
- **Zero Changes**: Existing integrations continue working without modification

```http
GET /core/rewards/attestation?
  reward_id=weekly_bonus&
  oracle_address=0xOracle1&
  // ... other params
```

### Migration Strategy

1. **Phase 1** - **Zero Disruption**:
   - Existing REST integrations continue unchanged
   - Deploy programmatic rewards via gRPC

2. **Phase 2** - **Gradual Migration**:
   - New services use gRPC API for programmatic rewards
   - Legacy services remain on REST API

3. **Phase 3** - **Full Migration** (Optional):
   - Migrate legacy services to gRPC for unified experience
   - Deprecate REST endpoints when ready

## Complete Reward Lifecycle

### 1. Reward Deployment
```go
// Deploy a new reward on-chain
rewardTx := &corev1.SignedTransaction{
    Signature: signature,
    Transaction: &corev1.SignedTransaction_Reward{
        Reward: &corev1.RewardMessage{
            Action: &corev1.RewardMessage_Create{
                Create: &corev1.CreateReward{
                    RewardId: "weekly_listener_bonus",
                    Name: "Weekly Listener Bonus",
                    Amount: 50,
                    ClaimAuthorities: []*corev1.ClaimAuthority{
                        {Address: "0xOracle1", Name: "Primary Oracle"},
                        {Address: "0xOracle2", Name: "Backup Oracle"},
                    },
                },
            },
        },
    },
}
// Result: Reward gets deterministic address 0xABC123...
```

### 2. Reward Discovery
```go
// Query rewards by authority
rewards, _ := client.GetRewardsByClaimAuthority("0xOracle1")

// Or get specific reward
reward, _ := client.GetReward(&corev1.GetRewardRequest{
    Address: "0xABC123...",
})
```

### 3. Reward Claiming
```http
GET /api/v1/rewards/attestation?
  eth_recipient_address=0xUser123&
  reward_address=0xABC123...&
  specifier=weekly_2024_01&
  oracle_address=0xOracle1&
  signature=0xSig456...&
  amount=50
```

**Response:**
```json
{
  "owner": "0xValidator789",
  "attestation": "0xAttestation..."
}
```

### 4. Reward Updates
```go
// Update reward (only claim authorities can do this)
updateTx := &corev1.SignedTransaction{
    Signature: oracleSignature,
    Transaction: &corev1.SignedTransaction_Reward{
        Reward: &corev1.RewardMessage{
            Action: &corev1.RewardMessage_Update{
                Update: &corev1.UpdateReward{
                    Address: "0xABC123...", // Deployed reward address
                    Name: "Updated Weekly Bonus",
                    Amount: 75, // Increased amount
                    ClaimAuthorities: [...], // Can add/remove authorities
                },
            },
        },
    },
}
```

## SQL Queries

### Reading Rewards
- `GetReward`: Get specific reward by address
- `GetRewardByID`: Get reward by original reward_id
- `GetActiveRewards`: Get latest version of all rewards
- `GetRewardsByClaimAuthority`: Find rewards where address is a claim authority

### Writing Rewards
- `InsertCoreReward`: Insert new reward transaction (Create/Update/Delete)

## Usage Examples

### Creating a Reward
```go
rewardMsg := &corev1.RewardMessage{
    Action: &corev1.RewardMessage_Create{
        Create: &corev1.CreateReward{
            RewardId: "weekly_bonus",
            Name: "Weekly Bonus Reward",
            Amount: 100,
            ClaimAuthorities: []*corev1.ClaimAuthority{
                {Address: "0x123...", Name: "Admin"},
            },
        },
    },
}
```

### Updating a Reward
```go
rewardMsg := &corev1.RewardMessage{
    Action: &corev1.RewardMessage_Update{
        Update: &corev1.UpdateReward{
            Address: "0xabc...", // Deployed reward address
            Name: "Updated Weekly Bonus",
            Amount: 150,
            ClaimAuthorities: []*corev1.ClaimAuthority{
                {Address: "0x123...", Name: "Admin"},
                {Address: "0x456...", Name: "Moderator"},
            },
        },
    },
}
```

### Querying Rewards by Authority
```sql
SELECT * FROM core_rewards
WHERE '0x123...' = ANY(claim_authorities)
ORDER BY block_height DESC;
```

## Transaction Flow

1. **Validation** (`validateBlockTx`):
   - Basic transaction structure validation
   - Ensure RewardMessage is not nil

2. **Finalization** (`finalizeTransaction`):
   - Recover sender address from signature
   - Route to appropriate handler (Create/Update/Delete)
   - Perform authorization checks
   - Store transaction in database

3. **Storage**:
   - All operations store complete transaction history
   - Latest state determined by highest block_height per address

## Benefits

1. **Deterministic Addresses**: Rewards behave like smart contracts with predictable addresses
2. **Efficient Querying**: GIN index on claim_authorities enables fast authority-based queries
3. **Full Audit Trail**: Complete transaction history preserved
4. **Secure Authorization**: Cryptographic signature verification + authority checks
5. **Consistent Patterns**: Follows same patterns as ERN, MEAD, PIE implementations
6. **Integrated Attestation**: Seamless integration with existing reward claiming infrastructure
7. **Oracle Authorization**: Only authorized claim authorities can generate reward attestations
8. **Backward Compatibility**: Existing reward claiming flows work with programmatic rewards

## Testing

A comprehensive integration test is available at `pkg/integration_tests/12_rewards_test.go` which tests:

1. **Complete Lifecycle**: Create → Query → Update → Delete rewards
2. **Transaction Validation**: Verifies all transactions are stored and retrievable
3. **Deterministic Addressing**: Tests that rewards get predictable addresses
4. **Authorization Flow**: Validates claim authority checks (when signatures are properly implemented)
5. **Hash Consistency**: Ensures transaction hashing is deterministic

### Running Tests
```bash
cd pkg/integration_tests
go test -run TestRewardsLifecycle
go test -run TestRewardTransactionHashing
```

## Implementation Notes

- Rewards are never actually "deleted" - deletion is recorded as a transaction
- All reward modifications create new database entries (append-only)
- Query the latest state using `ORDER BY block_height DESC LIMIT 1`
- Claim authorities are stored as text arrays for efficient PostgreSQL operations
- Integration tests use placeholder signatures - production requires proper cryptographic signatures