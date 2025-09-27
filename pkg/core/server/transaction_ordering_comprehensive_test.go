package server

import (
	"testing"

	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	v1beta1 "github.com/AudiusProject/audiusd/pkg/api/core/v1beta1"
	ddexv1beta1 "github.com/AudiusProject/audiusd/pkg/api/ddex/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestTransactionOrderingComprehensive(t *testing.T) {
	// Create all transaction types in scrambled order
	scrambledTxs := [][]byte{
		// V2 transaction (should be last)
		marshal(&v1beta1.Transaction{
			Envelope: &v1beta1.Envelope{
				Header: &v1beta1.EnvelopeHeader{},
			},
		}),

		// Reward Delete (should be after reward create/update)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Reward{
				Reward: &corev1.RewardMessage{
					Action: &corev1.RewardMessage_Delete{
						Delete: &corev1.DeleteReward{Address: "reward1"},
					},
				},
			},
		}),

		// Entity Delete (should be after entity create/update)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_ManageEntity{
				ManageEntity: &corev1.ManageEntityLegacy{
					Action:   "Delete",
					EntityId: 1,
				},
			},
		}),

		// Storage Proof Verification (should be after storage proofs)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_StorageProofVerification{
				StorageProofVerification: &corev1.StorageProofVerification{
					Height: 100,
				},
			},
		}),

		// Validator Deregistration (should be last among V1)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_ValidatorDeregistration{
				ValidatorDeregistration: &corev1.ValidatorMisbehaviorDeregistration{
					CometAddress: "validator1",
				},
			},
		}),

		// Entity Update (should be after entity create)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_ManageEntity{
				ManageEntity: &corev1.ManageEntityLegacy{
					Action:   "Update",
					EntityId: 2,
				},
			},
		}),

		// Reward Update (should be after reward create)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Reward{
				Reward: &corev1.RewardMessage{
					Action: &corev1.RewardMessage_Update{
						Update: &corev1.UpdateReward{Address: "reward2"},
					},
				},
			},
		}),

		// SLA Rollup (should be after most operations)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_SlaRollup{
				SlaRollup: &corev1.SlaRollup{
					BlockStart: 100,
				},
			},
		}),

		// Track Plays (should be after entity management)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Plays{
				Plays: &corev1.TrackPlays{},
			},
		}),

		// Entity Create (should be early)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_ManageEntity{
				ManageEntity: &corev1.ManageEntityLegacy{
					Action:   "Create",
					EntityId: 3,
				},
			},
		}),

		// Reward Create (should be after entities)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Reward{
				Reward: &corev1.RewardMessage{
					Action: &corev1.RewardMessage_Create{
						Create: &corev1.CreateReward{RewardId: "reward3"},
					},
				},
			},
		}),

		// Storage Proof (should be early)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_StorageProof{
				StorageProof: &corev1.StorageProof{
					Height: 100,
				},
			},
		}),

		// DDEX Release (should be after storage but before rewards)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Release{
				Release: &ddexv1beta1.NewReleaseMessage{},
			},
		}),

		// Attestation Deregistration (should be last among attestations)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Attestation{
				Attestation: &corev1.Attestation{
					Body: &corev1.Attestation_ValidatorDeregistration{
						ValidatorDeregistration: &corev1.ValidatorDeregistration{
							CometAddress: "validator2",
						},
					},
				},
			},
		}),

		// Validator Registration (should be first)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_ValidatorRegistration{
				ValidatorRegistration: &corev1.ValidatorRegistrationLegacy{
					CometAddress: "validator3",
				},
			},
		}),

		// Attestation Registration (should be early)
		marshal(&corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Attestation{
				Attestation: &corev1.Attestation{
					Body: &corev1.Attestation_ValidatorRegistration{
						ValidatorRegistration: &corev1.ValidatorRegistration{
							CometAddress: "validator4",
						},
					},
				},
			},
		}),
	}

	// Order the transactions
	orderedTxs := OrderTransactionBytes(scrambledTxs)

	// Verify we have the same number of transactions
	assert.Equal(t, len(scrambledTxs), len(orderedTxs))

	// Extract transaction types in order to verify the expected sequence
	var sequence []string
	for _, txBytes := range orderedTxs {
		txType := getTransactionTypeFromBytes(txBytes)
		sequence = append(sequence, txType)
	}

	// Expected order based on our TransactionTypeOrder
	expectedSequence := []string{
		"validator_registration",        // Validator registrations first
		"attestation_registration",     // Attestation registrations
		"storage_proof",                // Storage proofs
		"entity_create",                // Entity creates
		"entity_update",                // Entity updates
		"entity_delete",                // Entity deletes
		"track_plays",                  // Track plays
		"ddex_release",                 // DDEX releases
		"reward_create",                // Reward creates
		"reward_update",                // Reward updates
		"reward_delete",                // Reward deletes
		"storage_proof_verification",   // Storage proof verifications
		"sla_rollup",                   // SLA rollups
		"validator_deregistration",     // Validator deregistrations
		"attestation_deregistration",   // Attestation deregistrations
		"v2_transaction",               // V2 transactions last
	}

	assert.Equal(t, expectedSequence, sequence, "Transactions should be in the correct dependency order")
}

// Helper function to marshal any protobuf message
func marshal(msg proto.Message) []byte {
	bytes, _ := proto.Marshal(msg)
	return bytes
}

// Helper function to determine transaction type from bytes
func getTransactionTypeFromBytes(txBytes []byte) string {
	// Try V2 first
	var v2tx v1beta1.Transaction
	if err := proto.Unmarshal(txBytes, &v2tx); err == nil && v2tx.Envelope != nil {
		return "v2_transaction"
	}

	// Try V1
	var v1tx corev1.SignedTransaction
	if err := proto.Unmarshal(txBytes, &v1tx); err == nil {
		switch t := v1tx.Transaction.(type) {
		case *corev1.SignedTransaction_ValidatorRegistration:
			return "validator_registration"
		case *corev1.SignedTransaction_ValidatorDeregistration:
			return "validator_deregistration"
		case *corev1.SignedTransaction_StorageProof:
			return "storage_proof"
		case *corev1.SignedTransaction_StorageProofVerification:
			return "storage_proof_verification"
		case *corev1.SignedTransaction_SlaRollup:
			return "sla_rollup"
		case *corev1.SignedTransaction_ManageEntity:
			if t.ManageEntity != nil {
				switch t.ManageEntity.Action {
				case "Create":
					return "entity_create"
				case "Update":
					return "entity_update"
				case "Delete":
					return "entity_delete"
				}
			}
			return "entity_other"
		case *corev1.SignedTransaction_Plays:
			return "track_plays"
		case *corev1.SignedTransaction_Release:
			return "ddex_release"
		case *corev1.SignedTransaction_Attestation:
			if t.Attestation.GetValidatorRegistration() != nil {
				return "attestation_registration"
			} else if t.Attestation.GetValidatorDeregistration() != nil {
				return "attestation_deregistration"
			}
			return "attestation_other"
		case *corev1.SignedTransaction_Reward:
			if t.Reward != nil {
				switch t.Reward.Action.(type) {
				case *corev1.RewardMessage_Create:
					return "reward_create"
				case *corev1.RewardMessage_Update:
					return "reward_update"
				case *corev1.RewardMessage_Delete:
					return "reward_delete"
				}
			}
			return "reward_other"
		}
	}

	return "unknown"
}