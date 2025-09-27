# Transaction Ordering System

The Audius blockchain uses a sophisticated transaction ordering system to ensure that operations are processed in the correct dependency order. This prevents issues like trying to update an entity before it's created, or deleting a reward that doesn't exist yet.

## Overview

The transaction ordering system works with raw transaction bytes (`[][]byte`) and assigns each transaction a floating-point priority. Transactions are then sorted by priority, with lower numbers processed first.

**Location**: `pkg/core/server/transaction_ordering.go`

## Priority Structure

The system uses a two-level priority structure:

1. **Base Priority** (integer): Determines the main transaction type order
2. **Sub-Priority** (fractional): Orders variants within the same transaction type

Final priority = Base Priority + Sub-Priority (e.g., 3.1 = base 3 + sub 0.1)

## Transaction Type Order

```go
var TransactionTypeOrder = []interface{}{
    // Priority 0-1: Validator Operations
    (*corev1.SignedTransaction_ValidatorRegistration)(nil),     // 0.x
    (*corev1.Attestation_ValidatorRegistration)(nil),          // 1.x

    // Priority 2: Storage Operations
    (*corev1.SignedTransaction_StorageProof)(nil),             // 2.0

    // Priority 3: Entity Management
    (*corev1.SignedTransaction_ManageEntity)(nil),             // 3.x

    // Priority 4: Track Operations
    (*corev1.SignedTransaction_Plays)(nil),                    // 4.0

    // Priority 5: DDEX Operations
    (*corev1.SignedTransaction_Release)(nil),                  // 5.0

    // Priority 6-8: Reward Operations
    (*corev1.RewardMessage_Create)(nil),                       // 6.x
    (*corev1.RewardMessage_Update)(nil),                       // 7.x
    (*corev1.RewardMessage_Delete)(nil),                       // 8.x

    // Priority 9-10: Post-Creation Operations
    (*corev1.SignedTransaction_StorageProofVerification)(nil), // 9.0
    (*corev1.SignedTransaction_SlaRollup)(nil),                // 10.0

    // Priority 11-12: Deregistrations
    (*corev1.SignedTransaction_ValidatorDeregistration)(nil),  // 11.0
    (*corev1.Attestation_ValidatorDeregistration)(nil),       // 12.x

    // Priority 13: V2 Transactions
    (*v1beta1.Transaction)(nil),                               // 13.0
}
```

## Sub-Priority System

### ManageEntity Actions
- **Create**: 3.0 (base 3 + sub 0.0)
- **Update**: 3.1 (base 3 + sub 0.1)
- **Delete**: 3.2 (base 3 + sub 0.2)

### Reward Actions
- **Create**: 6.0 (base 6 + sub 0.0)
- **Update**: 7.1 (base 7 + sub 0.1)
- **Delete**: 8.2 (base 8 + sub 0.2)

### Attestation Types
- **Registration**: x.0 (sub 0.0)
- **Deregistration**: x.1 (sub 0.1)

## Example Transaction Order

Given these scrambled transactions:
```
- Reward Delete (address: "reward1")
- Entity Delete (id: 1)
- V2 Transaction
- Entity Update (id: 2)
- Reward Create (id: "reward3")
- Entity Create (id: 3)
- Validator Registration
```

They would be ordered as:
```
1. Validator Registration         (priority 0.0)
2. Entity Create (id: 3)         (priority 3.0)
3. Entity Update (id: 2)         (priority 3.1)
4. Entity Delete (id: 1)         (priority 3.2)
5. Reward Create (id: "reward3") (priority 6.0)
6. Reward Delete (address: "reward1") (priority 8.2)
7. V2 Transaction                (priority 13.0)
```

## Adding New Transaction Types

### 1. Add to TransactionTypeOrder

Add your new transaction type to the `TransactionTypeOrder` slice at the appropriate position:

```go
var TransactionTypeOrder = []interface{}{
    // ... existing types ...
    (*corev1.SignedTransaction_YourNewType)(nil), // Insert at desired priority
    // ... rest of types ...
}
```

### 2. Add Matching Logic

Update `matchesTransactionType` function:

```go
func matchesTransactionType(tx *corev1.SignedTransaction, txType interface{}) bool {
    switch txType.(type) {
    // ... existing cases ...
    case *corev1.SignedTransaction_YourNewType:
        _, ok := tx.Transaction.(*corev1.SignedTransaction_YourNewType)
        return ok
    // ... rest of cases ...
    }
}
```

### 3. Add Sub-Priority (Optional)

If your transaction type has variants that need sub-ordering:

1. **Add to getTransactionSubPriority**:
```go
func getTransactionSubPriority(tx *corev1.SignedTransaction) float64 {
    switch t := tx.Transaction.(type) {
    // ... existing cases ...
    case *corev1.SignedTransaction_YourNewType:
        return getYourNewTypeSubPriority(t.YourNewType)
    // ... rest of cases ...
    }
}
```

2. **Implement sub-priority function**:
```go
func getYourNewTypeSubPriority(yourType *corev1.YourNewType) float64 {
    if yourType == nil {
        return 0.9
    }
    switch yourType.Action.(type) {
    case *corev1.YourNewType_Create:
        return 0.0
    case *corev1.YourNewType_Update:
        return 0.1
    case *corev1.YourNewType_Delete:
        return 0.2
    default:
        return 0.9
    }
}
```

## Testing

### Unit Tests

The comprehensive test in `transaction_ordering_comprehensive_test.go` verifies the complete ordering:

```go
func TestTransactionOrderingComprehensive(t *testing.T) {
    // Creates scrambled transactions of all types
    scrambledTxs := [][]byte{ /* all transaction types */ }

    // Orders them
    orderedTxs := OrderTransactionBytes(scrambledTxs)

    // Verifies correct order
    expectedSequence := []string{
        "validator_registration",
        "attestation_registration",
        "storage_proof",
        "entity_create",
        "entity_update",
        "entity_delete",
        "track_plays",
        "ddex_release",
        "reward_create",
        "reward_update",
        "reward_delete",
        // ... etc
    }

    assert.Equal(t, expectedSequence, actualSequence)
}
```

### Adding Test Cases

When adding new transaction types, update the test:

1. Add the new transaction to `scrambledTxs`
2. Add the expected type string to `expectedSequence`
3. Add type detection logic to `getTransactionTypeFromBytes`

## Common Patterns

### Create-Update-Delete Pattern
For transaction types with lifecycle operations:
- Create: sub-priority 0.0
- Update: sub-priority 0.1
- Delete: sub-priority 0.2

### Registration-Deregistration Pattern
For transaction types with registration:
- Registration: sub-priority 0.0
- Deregistration: sub-priority 0.1

## Debugging

To debug transaction ordering issues:

1. **Check the priority**: Add logging to see computed priorities
2. **Verify matching**: Ensure `matchesTransactionType` correctly identifies your transaction
3. **Test sub-priorities**: Verify sub-priority calculations return expected values
4. **Run comprehensive test**: Ensure your changes don't break existing ordering

## Integration

The ordering system integrates with the consensus mechanism:

- **PrepareProposal**: Called in `abci.go` to order transactions before proposing a block
- **Input**: Raw transaction bytes from mempool, SLA rollups, deregistrations
- **Output**: Ordered transaction bytes ready for block proposal

```go
func (s *Server) PrepareProposal(req *abcitypes.RequestPrepareProposal) (*abcitypes.ResponsePrepareProposal, error) {
    // ... collect transactions from various sources ...

    // Order all transactions by dependency
    orderedTxs := OrderTransactionBytes(proposalTxs)

    // ... return ordered transactions ...
}
```