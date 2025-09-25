package integration_tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	corev1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/integration_tests/utils"
	"github.com/ethereum/go-ethereum/crypto"
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
		// Generate random private keys for claim authorities
		oracleKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate oracle key: %v", err)
		}
		oracleAddr := common.PrivKeyToAddress(oracleKey)

		backupKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate backup key: %v", err)
		}
		backupAddr := common.PrivKeyToAddress(backupKey)

		// Get current block height for deadline
		status, err := sdk.Core.GetStatus(ctx, connect.NewRequest(&corev1.GetStatusRequest{}))
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}
		currentHeight := status.Msg.ChainInfo.CurrentHeight
		deadlineHeight := currentHeight + 100 // 100 block buffer

		// Test 1: Create a new reward using SendTransaction
		rewardID := "test_reward_" + uuid.NewString()

		// Create the reward with signature and deadline
		createReward := &corev1.CreateReward{
			RewardId: rewardID,
			Name:     "Test Integration Reward",
			Amount:   100,
			ClaimAuthorities: []*corev1.ClaimAuthority{
				{
					Address: oracleAddr,
					Name:    "Test Oracle",
				},
				{
					Address: backupAddr,
					Name:    "Backup Oracle",
				},
			},
			DeadlineBlockHeight: deadlineHeight,
		}

		// Sign the create reward using deterministic signing
		createSignature, err := common.SignCreateReward(oracleKey, createReward)
		if err != nil {
			t.Fatalf("Failed to sign create reward: %v", err)
		}
		createReward.Signature = createSignature

		// Create the reward message
		createRewardMsg := &corev1.RewardMessage{
			Action: &corev1.RewardMessage_Create{
				Create: createReward,
			},
		}

		// Create signed transaction
		signedTx := &corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Reward{
				Reward: createRewardMsg,
			},
		}

		// Send via SendTransaction
		createRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(&corev1.SendTransactionRequest{
			Transaction: signedTx,
		}))
		if err != nil {
			t.Fatalf("Failed to create reward: %v", err)
		}

		createTxHash := createRes.Msg.Transaction.Hash
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

		rewardAddress := createdReward.Address
		t.Logf("Created reward at address: %s", rewardAddress)

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

		// Test 4: Update the reward using SendTransaction
		// Generate a new authority key
		newAuthorityKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate new authority key: %v", err)
		}
		newAuthorityAddr := common.PrivKeyToAddress(newAuthorityKey)

		// Create the update reward with signature and deadline
		updateReward := &corev1.UpdateReward{
			Address: rewardAddress,
			Name:    "Updated Test Reward",
			Amount:  150,
			ClaimAuthorities: []*corev1.ClaimAuthority{
				{
					Address: oracleAddr,
					Name:    "Test Oracle",
				},
				{
					Address: newAuthorityAddr,
					Name:    "New Authority",
				},
			},
			DeadlineBlockHeight: deadlineHeight,
		}

		// Sign the update reward using deterministic signing
		updateSignature, err := common.SignUpdateReward(oracleKey, updateReward)
		if err != nil {
			t.Fatalf("Failed to sign update reward: %v", err)
		}
		updateReward.Signature = updateSignature

		// Create the reward message
		updateRewardMsg := &corev1.RewardMessage{
			Action: &corev1.RewardMessage_Update{
				Update: updateReward,
			},
		}

		// Create signed transaction
		updateSignedTx := &corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Reward{
				Reward: updateRewardMsg,
			},
		}

		// Send via SendTransaction
		updateRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(&corev1.SendTransactionRequest{
			Transaction: updateSignedTx,
		}))
		if err != nil {
			t.Fatalf("Failed to update reward: %v", err)
		}

		updateTxHash := updateRes.Msg.Transaction.Hash
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

		// Test 5: Delete the reward using SendTransaction
		// Create the delete reward with signature and deadline
		deleteReward := &corev1.DeleteReward{
			Address:             rewardAddress,
			DeadlineBlockHeight: deadlineHeight,
		}

		// Sign the delete reward using deterministic signing
		deleteSignature, err := common.SignDeleteReward(oracleKey, deleteReward)
		if err != nil {
			t.Fatalf("Failed to sign delete reward: %v", err)
		}
		deleteReward.Signature = deleteSignature

		// Create the reward message
		deleteRewardMsg := &corev1.RewardMessage{
			Action: &corev1.RewardMessage_Delete{
				Delete: deleteReward,
			},
		}

		// Create signed transaction
		deleteSignedTx := &corev1.SignedTransaction{
			Transaction: &corev1.SignedTransaction_Reward{
				Reward: deleteRewardMsg,
			},
		}

		// Send via SendTransaction
		deleteRes, err := sdk.Core.SendTransaction(ctx, connect.NewRequest(&corev1.SendTransactionRequest{
			Transaction: deleteSignedTx,
		}))
		if err != nil {
			t.Fatalf("Failed to delete reward: %v", err)
		}

		deleteTxHash := deleteRes.Msg.Transaction.Hash
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
