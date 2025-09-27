package server

import (
	"sort"
	"strings"

	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	v1beta1 "github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	"google.golang.org/protobuf/proto"
)

// TransactionTypeOrder defines the processing order for different transaction types.
// The index in the slice determines the priority (earlier = higher priority).
// This ensures dependencies are handled correctly (e.g., create before update/delete).
var TransactionTypeOrder = []interface{}{
	// === V1 Transactions (processed first) ===

	// Validator operations (registrations before deregistrations)
	(*corev1.SignedTransaction_ValidatorRegistration)(nil), // Legacy validator registration
	(*corev1.Attestation_ValidatorRegistration)(nil),       // Attestation-based registration

	// Storage proofs (must exist before verification)
	(*corev1.SignedTransaction_StorageProof)(nil),

	// Entity management - handled with special logic for action ordering
	(*corev1.SignedTransaction_ManageEntity)(nil),

	// Track plays
	(*corev1.SignedTransaction_Plays)(nil),

	// DDEX Release operations (ERN)
	(*corev1.SignedTransaction_Release)(nil),

	// Reward operations (in dependency order)
	(*corev1.RewardMessage_Create)(nil), // Rewards must exist before update/delete
	(*corev1.RewardMessage_Update)(nil),
	(*corev1.RewardMessage_Delete)(nil),

	// Post-creation operations
	(*corev1.SignedTransaction_StorageProofVerification)(nil), // Verifies existing storage proofs
	(*corev1.SignedTransaction_SlaRollup)(nil),                // SLA rollup operations

	// Deregistrations (should be last to allow cleanup)
	(*corev1.SignedTransaction_ValidatorDeregistration)(nil), // Misbehavior-based deregistration
	(*corev1.Attestation_ValidatorDeregistration)(nil),       // Attestation-based deregistration

	// === V2 Transactions (processed after V1) ===
	(*v1beta1.Transaction)(nil), // All V2 transactions
}

// getV1TransactionPriority returns the priority for a v1 transaction.
// Lower numbers = higher priority (processed first).
func getV1TransactionPriority(tx *corev1.SignedTransaction) float64 {
	// Default to lowest priority
	defaultPriority := float64(len(TransactionTypeOrder) + 100)

	for i, txType := range TransactionTypeOrder {
		if matches := matchesTransactionType(tx, txType); matches {
			basePriority := float64(i)
			subPriority := getTransactionSubPriority(tx)
			return basePriority + subPriority
		}
	}

	return defaultPriority
}

// getV2TransactionPriority returns the priority for a v2 transaction.
func getV2TransactionPriority() float64 {
	for i, txType := range TransactionTypeOrder {
		if _, ok := txType.(*v1beta1.Transaction); ok {
			return float64(i)
		}
	}
	return float64(len(TransactionTypeOrder) + 100)
}

// getTransactionSubPriority returns a fractional sub-priority for transactions with variants.
// This allows ordering within the same transaction type (e.g., Create before Update before Delete).
func getTransactionSubPriority(tx *corev1.SignedTransaction) float64 {
	switch t := tx.Transaction.(type) {
	case *corev1.SignedTransaction_ManageEntity:
		return getManageEntitySubPriority(t.ManageEntity)
	case *corev1.SignedTransaction_Reward:
		return getRewardSubPriority(t.Reward)
	case *corev1.SignedTransaction_Attestation:
		return getAttestationSubPriority(t.Attestation)
	default:
		return 0.0 // No sub-priority for other types
	}
}

// getManageEntitySubPriority returns sub-priority for ManageEntity actions.
func getManageEntitySubPriority(entity *corev1.ManageEntityLegacy) float64 {
	if entity == nil {
		return 0.9
	}
	action := entity.Action
	if strings.EqualFold(action, "Create") {
		return 0.0
	} else if strings.EqualFold(action, "Update") {
		return 0.1
	} else if strings.EqualFold(action, "Delete") {
		return 0.2
	}
	return 0.9 // Other actions
}

// getRewardSubPriority returns sub-priority for Reward actions.
func getRewardSubPriority(reward *corev1.RewardMessage) float64 {
	if reward == nil {
		return 0.9
	}
	switch reward.Action.(type) {
	case *corev1.RewardMessage_Create:
		return 0.0
	case *corev1.RewardMessage_Update:
		return 0.1
	case *corev1.RewardMessage_Delete:
		return 0.2
	default:
		return 0.9
	}
}

// getAttestationSubPriority returns sub-priority for Attestation types.
func getAttestationSubPriority(attestation *corev1.Attestation) float64 {
	if attestation == nil {
		return 0.9
	}
	switch attestation.Body.(type) {
	case *corev1.Attestation_ValidatorRegistration:
		return 0.0
	case *corev1.Attestation_ValidatorDeregistration:
		return 0.1
	default:
		return 0.9
	}
}

// matchesTransactionType checks if a SignedTransaction matches the given type
func matchesTransactionType(tx *corev1.SignedTransaction, txType interface{}) bool {
	switch txType.(type) {
	// Direct SignedTransaction types
	case *corev1.SignedTransaction_ValidatorRegistration:
		_, ok := tx.Transaction.(*corev1.SignedTransaction_ValidatorRegistration)
		return ok
	case *corev1.SignedTransaction_ValidatorDeregistration:
		_, ok := tx.Transaction.(*corev1.SignedTransaction_ValidatorDeregistration)
		return ok
	case *corev1.SignedTransaction_StorageProof:
		_, ok := tx.Transaction.(*corev1.SignedTransaction_StorageProof)
		return ok
	case *corev1.SignedTransaction_StorageProofVerification:
		_, ok := tx.Transaction.(*corev1.SignedTransaction_StorageProofVerification)
		return ok
	case *corev1.SignedTransaction_SlaRollup:
		_, ok := tx.Transaction.(*corev1.SignedTransaction_SlaRollup)
		return ok
	case *corev1.SignedTransaction_ManageEntity:
		_, ok := tx.Transaction.(*corev1.SignedTransaction_ManageEntity)
		return ok
	case *corev1.SignedTransaction_Plays:
		_, ok := tx.Transaction.(*corev1.SignedTransaction_Plays)
		return ok
	case *corev1.SignedTransaction_Release:
		_, ok := tx.Transaction.(*corev1.SignedTransaction_Release)
		return ok

	// Attestation subtypes
	case *corev1.Attestation_ValidatorRegistration:
		if att, ok := tx.Transaction.(*corev1.SignedTransaction_Attestation); ok {
			_, hasReg := att.Attestation.Body.(*corev1.Attestation_ValidatorRegistration)
			return hasReg
		}
		return false
	case *corev1.Attestation_ValidatorDeregistration:
		if att, ok := tx.Transaction.(*corev1.SignedTransaction_Attestation); ok {
			_, hasDereg := att.Attestation.Body.(*corev1.Attestation_ValidatorDeregistration)
			return hasDereg
		}
		return false

	// Reward message subtypes
	case *corev1.RewardMessage_Create:
		if reward, ok := tx.Transaction.(*corev1.SignedTransaction_Reward); ok {
			_, isCreate := reward.Reward.Action.(*corev1.RewardMessage_Create)
			return isCreate
		}
		return false
	case *corev1.RewardMessage_Update:
		if reward, ok := tx.Transaction.(*corev1.SignedTransaction_Reward); ok {
			_, isUpdate := reward.Reward.Action.(*corev1.RewardMessage_Update)
			return isUpdate
		}
		return false
	case *corev1.RewardMessage_Delete:
		if reward, ok := tx.Transaction.(*corev1.SignedTransaction_Reward); ok {
			_, isDelete := reward.Reward.Action.(*corev1.RewardMessage_Delete)
			return isDelete
		}
		return false

	default:
		return false
	}
}

// OrderTransactionBytes sorts raw transaction bytes to ensure operations are processed in the correct dependency order.
// This works with raw transaction bytes from various sources (mempool, rollups, deregistrations, etc.).
func OrderTransactionBytes(txBytes [][]byte) [][]byte {
	if len(txBytes) <= 1 {
		return txBytes
	}

	// Create wrapper structs to track bytes with their priorities
	type txWithPriority struct {
		bytes    []byte
		priority float64
	}

	txsWithPriority := make([]txWithPriority, 0, len(txBytes))

	for _, bytes := range txBytes {
		priority := getTransactionBytePriority(bytes)
		txsWithPriority = append(txsWithPriority, txWithPriority{
			bytes:    bytes,
			priority: priority,
		})
	}

	// Sort by priority (stable sort preserves order for same priority)
	sort.SliceStable(txsWithPriority, func(i, j int) bool {
		return txsWithPriority[i].priority < txsWithPriority[j].priority
	})

	// Extract the sorted bytes
	result := make([][]byte, len(txsWithPriority))
	for i, tx := range txsWithPriority {
		result[i] = tx.bytes
	}

	return result
}

// getTransactionBytePriority returns the priority for raw transaction bytes.
func getTransactionBytePriority(txBytes []byte) float64 {
	// Try to unmarshal as v2 transaction first
	var v2tx v1beta1.Transaction
	if err := proto.Unmarshal(txBytes, &v2tx); err == nil && v2tx.Envelope != nil {
		// It's a v2 transaction
		return getV2TransactionPriority()
	}

	// Try to unmarshal as v1 transaction
	var v1tx corev1.SignedTransaction
	if err := proto.Unmarshal(txBytes, &v1tx); err == nil {
		return getV1TransactionPriority(&v1tx)
	}

	// If we can't determine the type, return lowest priority
	return float64(len(TransactionTypeOrder) + 100)
}
