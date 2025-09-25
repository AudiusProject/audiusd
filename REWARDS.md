# Reward System Documentation

## Overview

The reward system allows for programmatic creation, updating, and deletion of rewards on the Audius blockchain. Each reward has claim authorities who can authorize reward operations and issue reward attestations.

## Reward Operations

All reward operations use deterministic signatures with block height expiry for security and replay protection.

### Message Types

#### CreateReward
Creates a new reward with the following fields:
- `reward_id`: Unique identifier for the reward
- `name`: Human-readable name
- `amount`: Reward amount in wei
- `claim_authorities`: List of authorized addresses and their names
- `deadline_block_height`: Block height after which transaction expires
- `signature`: Signature over deterministic reward data

#### UpdateReward
Updates an existing reward:
- `address`: The deployed reward address
- `name`: Updated name
- `amount`: Updated amount in wei
- `claim_authorities`: Updated list of authorities
- `deadline_block_height`: Block height after which transaction expires
- `signature`: Signature over deterministic update data

#### DeleteReward
Deletes a reward:
- `address`: The deployed reward address
- `deadline_block_height`: Block height after which transaction expires
- `signature`: Signature over deterministic delete data

## Signature Generation

All reward operations require signatures over deterministic data to ensure security and prevent replay attacks.

### Signature Data Format

**CreateReward**: Hash of `reward_id|name|amount|claim_authorities_json|deadline_block_height`

**UpdateReward**: Hash of `address|name|amount|claim_authorities_json|deadline_block_height`

**DeleteReward**: Hash of `address|deadline_block_height`

Where:
- `claim_authorities_json` is a sorted JSON array of "address:name" strings
- All data is joined with `|` separator
- The final signature data is a SHA256 hash of the concatenated string, hex encoded

### Creating Signatures

1. Create the deterministic signature data string
2. Hash it with SHA256 and hex encode
3. Convert hex string to bytes
4. Sign the bytes using Ethereum-style signing (EthSign)
5. Include the signature in the reward message

Example in Go:
```go
import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "sort"
)

func createCreateRewardSignature(privateKey *ecdsa.PrivateKey, createReward *CreateReward) (string, error) {
    // Sort authorities for deterministic ordering
    authorities := make([]string, len(createReward.ClaimAuthorities))
    for i, auth := range createReward.ClaimAuthorities {
        authorities[i] = fmt.Sprintf("%s:%s", auth.Address, auth.Name)
    }
    sort.Strings(authorities)

    authoritiesJson, _ := json.Marshal(authorities)
    data := fmt.Sprintf("%s|%s|%d|%s|%d",
        createReward.RewardId,
        createReward.Name,
        createReward.Amount,
        string(authoritiesJson),
        createReward.DeadlineBlockHeight)

    // Hash the data
    hash := sha256.Sum256([]byte(data))
    hexData := hex.EncodeToString(hash[:])

    // Convert back to bytes and sign
    dataBytes, _ := hex.DecodeString(hexData)
    return common.EthSign(privateKey, dataBytes)
}
```

## Transaction Flow

### 1. Create Transaction
```go
// Create the reward message
createReward := &corev1.CreateReward{
    RewardId: "my_reward_123",
    Name:     "Test Reward",
    Amount:   100,
    ClaimAuthorities: []*corev1.ClaimAuthority{
        {Address: oracleAddr, Name: "Oracle"},
    },
    DeadlineBlockHeight: currentHeight + 100,
    Signature:          signature, // Created using deterministic signing
}

// Wrap in reward message
rewardMsg := &corev1.RewardMessage{
    Action: &corev1.RewardMessage_Create{
        Create: createReward,
    },
}

// Create signed transaction
signedTx := &corev1.SignedTransaction{
    Transaction: &corev1.SignedTransaction_Reward{
        Reward: rewardMsg,
    },
}

// Send via SendTransaction RPC
response, err := client.SendTransaction(ctx, &corev1.SendTransactionRequest{
    Transaction: signedTx,
})
```

### 2. ABCI Processing

When the transaction is processed in the ABCI layer:

1. **Signature Validation**: The deterministic signature data is recreated and the signature is verified
2. **Expiry Check**: Current block height is compared to deadline_block_height
3. **Authorization**: For updates/deletes, verify signer is in existing claim authorities
4. **Address Generation**: For creates, generate deterministic reward address
5. **Storage**: Store reward data in database

### 3. Authorization Rules

- **Create**: Any valid signature can create a reward (signer becomes initial authority)
- **Update**: Only existing claim authorities can update a reward
- **Delete**: Only existing claim authorities can delete a reward
- Address comparisons are case-insensitive using `strings.EqualFold`

## Security Features

### Replay Protection
- Each transaction includes a `deadline_block_height`
- Transactions are rejected if current height > deadline
- Prevents signature reuse across different time periods

### Deterministic Signatures
- Signature data is generated deterministically from stable fields
- Immune to protobuf schema evolution
- Consistent across implementations

### Authorization
- Only claim authorities can modify existing rewards
- Signature verification ensures transaction authenticity
- Case-insensitive address matching prevents casing issues

## Querying Rewards

Use the existing `GetRewards` and `GetReward` RPCs:

```go
// Get all active rewards
rewards, err := client.GetRewards(ctx, &corev1.GetRewardsRequest{})

// Get specific reward by address
reward, err := client.GetReward(ctx, &corev1.GetRewardRequest{
    Address: rewardAddress,
})
```

## Error Handling

Common errors:
- `ErrRewardExpired`: Transaction deadline has passed
- `ErrRewardSignatureInvalid`: Invalid signature or signature verification failed
- Authorization errors: Signer not in claim authorities for updates/deletes

## Migration from Legacy RPC

The dedicated `CreateReward`, `UpdateReward`, and `DeleteReward` RPCs have been removed. All reward operations now go through the standard `SendTransaction` flow with proper signature validation.

This provides:
- ✅ Better security through deterministic signatures
- ✅ Replay protection via block height expiry
- ✅ Consistent transaction processing
- ✅ Future-proof design independent of protobuf evolution