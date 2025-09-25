package integration_tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/integration_tests/utils"
	"github.com/google/uuid"
	protob "google.golang.org/protobuf/proto"
)

func TestRewardsLifecycle(t *testing.T) {
	ctx := context.Background()
	sdk := utils.DiscoveryOne

	// Wait for devnet to be ready
	if err := utils.WaitForDevnetHealthy(30 * time.Second); err != nil {
		t.Fatalf("Devnet not ready: %v", err)
	}

	t.Run("Create, Update, Delete, and Query Rewards", func(t *testing.T) {
		// Test 1: Create a new reward using the dedicated RPC
		rewardID := "test_reward_" + uuid.NewString()

		createReq := &corev1.CreateRewardRequest{
			RewardId: rewardID,
			Name:     "Test Integration Reward",
			Amount:   100,
			ClaimAuthorities: []*corev1.ClaimAuthority{
				{
					Address: "0x1234567890123456789012345678901234567890",
					Name:    "Test Oracle",
				},
				{
					Address: "0x0987654321098765432109876543210987654321",
					Name:    "Backup Oracle",
				},
			},
			Signature: "placeholder_signature_" + uuid.NewString(), // In real implementation, this would be a proper signature
		}

		createRes, err := sdk.Core.CreateReward(ctx, connect.NewRequest(createReq))
		if err != nil {
			t.Fatalf("Failed to create reward: %v", err)
		}

		rewardAddress := createRes.Msg.Address
		createTxHash := createRes.Msg.TxHash
		t.Logf("Created reward at address: %s", rewardAddress)
		t.Logf("Created reward transaction hash: %s", createTxHash)

		// Wait for transaction to be processed
		time.Sleep(3 * time.Second)

		// Test 2: Query all rewards to find our created reward
		getAllRewardsRes, err := sdk.Core.GetRewards(ctx, connect.NewRequest(&corev1.GetRewardsRequest{}))
		if err != nil {
			t.Fatalf("Failed to get all rewards: %v", err)
		}

		var createdReward *corev1.GetRewardResponse
		for _, reward := range getAllRewardsRes.Msg.Rewards {
			if reward.RewardId == rewardID {
				createdReward = reward
				break
			}
		}

		if createdReward == nil {
			t.Fatalf("Created reward not found in rewards list")
		}

		// Verify the address matches what was returned from CreateReward
		if createdReward.Address != rewardAddress {
			t.Errorf("Expected address %s, got %s", rewardAddress, createdReward.Address)
		}

		// Verify reward details
		if createdReward.Name != "Test Integration Reward" {
			t.Errorf("Expected name 'Test Integration Reward', got %s", createdReward.Name)
		}
		if createdReward.Amount != 100 {
			t.Errorf("Expected amount 100, got %d", createdReward.Amount)
		}
		if len(createdReward.ClaimAuthorities) != 2 {
			t.Errorf("Expected 2 claim authorities, got %d", len(createdReward.ClaimAuthorities))
		}

		// Test 3: Query specific reward by address
		getRewardRes, err := sdk.Core.GetReward(ctx, connect.NewRequest(&corev1.GetRewardRequest{
			Address: rewardAddress,
		}))
		if err != nil {
			t.Fatalf("Failed to get specific reward: %v", err)
		}

		if getRewardRes.Msg.RewardId != rewardID {
			t.Errorf("Expected reward_id %s, got %s", rewardID, getRewardRes.Msg.RewardId)
		}

		// Test 4: Update the reward using the dedicated RPC
		updateReq := &corev1.UpdateRewardRequest{
			Address: rewardAddress,
			Name:    "Updated Test Reward",
			Amount:  150,
			ClaimAuthorities: []*corev1.ClaimAuthority{
				{
					Address: "0x1234567890123456789012345678901234567890",
					Name:    "Test Oracle",
				},
				// Added a new authority
				{
					Address: "0x1111111111111111111111111111111111111111",
					Name:    "New Authority",
				},
			},
			Signature: "placeholder_signature_" + uuid.NewString(), // Should be signed by claim authority
		}

		updateRes, err := sdk.Core.UpdateReward(ctx, connect.NewRequest(updateReq))
		if err != nil {
			t.Fatalf("Failed to update reward: %v", err)
		}

		updateTxHash := updateRes.Msg.TxHash
		t.Logf("Updated reward transaction hash: %s", updateTxHash)

		// Wait for update to be processed
		time.Sleep(3 * time.Second)

		// Verify update was applied
		getUpdatedRewardRes, err := sdk.Core.GetReward(ctx, connect.NewRequest(&corev1.GetRewardRequest{
			Address: rewardAddress,
		}))
		if err != nil {
			t.Fatalf("Failed to get updated reward: %v", err)
		}

		if getUpdatedRewardRes.Msg.Name != "Updated Test Reward" {
			t.Errorf("Expected updated name 'Updated Test Reward', got %s", getUpdatedRewardRes.Msg.Name)
		}
		if getUpdatedRewardRes.Msg.Amount != 150 {
			t.Errorf("Expected updated amount 150, got %d", getUpdatedRewardRes.Msg.Amount)
		}

		// Test 5: Delete the reward using the dedicated RPC
		deleteReq := &corev1.DeleteRewardRequest{
			Address:   rewardAddress,
			Signature: "placeholder_signature_" + uuid.NewString(), // Should be signed by claim authority
		}

		deleteRes, err := sdk.Core.DeleteReward(ctx, connect.NewRequest(deleteReq))
		if err != nil {
			t.Fatalf("Failed to delete reward: %v", err)
		}

		deleteTxHash := deleteRes.Msg.TxHash
		t.Logf("Deleted reward transaction hash: %s", deleteTxHash)

		// Test 6: Verify transactions can be retrieved
		time.Sleep(2 * time.Second)

		// Just verify that the transactions can be retrieved by hash
		createTxRes, err := sdk.Core.GetTransaction(ctx, connect.NewRequest(&corev1.GetTransactionRequest{
			TxHash: createTxHash,
		}))
		if err != nil {
			t.Fatalf("Failed to retrieve create transaction: %v", err)
		}
		if createTxRes.Msg.Transaction.Hash != createTxHash {
			t.Errorf("Expected create transaction hash %s, got %s", createTxHash, createTxRes.Msg.Transaction.Hash)
		}

		updateTxRes, err := sdk.Core.GetTransaction(ctx, connect.NewRequest(&corev1.GetTransactionRequest{
			TxHash: updateTxHash,
		}))
		if err != nil {
			t.Fatalf("Failed to retrieve update transaction: %v", err)
		}
		if updateTxRes.Msg.Transaction.Hash != updateTxHash {
			t.Errorf("Expected update transaction hash %s, got %s", updateTxHash, updateTxRes.Msg.Transaction.Hash)
		}

		deleteTxRes, err := sdk.Core.GetTransaction(ctx, connect.NewRequest(&corev1.GetTransactionRequest{
			TxHash: deleteTxHash,
		}))
		if err != nil {
			t.Fatalf("Failed to retrieve delete transaction: %v", err)
		}
		if deleteTxRes.Msg.Transaction.Hash != deleteTxHash {
			t.Errorf("Expected delete transaction hash %s, got %s", deleteTxHash, deleteTxRes.Msg.Transaction.Hash)
		}

		t.Logf("Successfully completed rewards lifecycle test")
		t.Logf("Create TX: %s", createTxHash)
		t.Logf("Update TX: %s", updateTxHash)
		t.Logf("Delete TX: %s", deleteTxHash)
		t.Logf("Reward Address: %s", rewardAddress)
	})
}

func TestRewardTransactionHashing(t *testing.T) {
	t.Run("should produce consistent transaction hashes", func(t *testing.T) {
		rewardMsg := &corev1.RewardMessage{
			Action: &corev1.RewardMessage_Create{
				Create: &corev1.CreateReward{
					RewardId: "test_hash_reward",
					Name:     "Test Hash Reward",
					Amount:   50,
					ClaimAuthorities: []*corev1.ClaimAuthority{
						{
							Address: "0x1234567890123456789012345678901234567890",
							Name:    "Test Oracle",
						},
					},
				},
			},
		}

		tx := &corev1.SignedTransaction{
			Signature: "test_signature",
			Transaction: &corev1.SignedTransaction_Reward{
				Reward: rewardMsg,
			},
		}

		// Marshal twice to ensure consistent hashing
		txBytes1, err := protob.Marshal(tx)
		if err != nil {
			t.Fatalf("Failed to marshal transaction first time: %v", err)
		}

		txBytes2, err := protob.Marshal(tx)
		if err != nil {
			t.Fatalf("Failed to marshal transaction second time: %v", err)
		}

		txHash1 := common.ToTxHashFromBytes(txBytes1)
		txHash2 := common.ToTxHashFromBytes(txBytes2)

		if txHash1 != txHash2 {
			t.Errorf("Transaction hashes should be consistent: %s != %s", txHash1, txHash2)
		}

		t.Logf("Consistent transaction hash: %s", txHash1)
	})
}