package integration_tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/AudiusProject/audiusd/pkg/api/core/v1"
	"github.com/AudiusProject/audiusd/pkg/common"
	"github.com/AudiusProject/audiusd/pkg/integration_tests/utils"
	"github.com/AudiusProject/audiusd/pkg/sdk"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestRewardsLifecycle(t *testing.T) {
	ctx := context.Background()
	nodeUrl := utils.DiscoveryOneRPC

	// Wait for devnet to be ready
	if err := utils.WaitForDevnetHealthy(30 * time.Second); err != nil {
		t.Fatalf("Devnet not ready: %v", err)
	}

	t.Run("Create, Update, Delete, and Query Rewards", func(t *testing.T) {
		// Generate random private keys for claim authorities
		creatorKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate oracle key: %v", err)
		}
		creatorAddr := common.PrivKeyToAddress(creatorKey)
		creator := sdk.NewAudiusdSDK(nodeUrl)
		creator.SetPrivKey(creatorKey)

		updaterKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate backup key: %v", err)
		}
		updaterAddr := common.PrivKeyToAddress(updaterKey)
		updater := sdk.NewAudiusdSDK(nodeUrl)
		updater.SetPrivKey(updaterKey)

		deleterKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate backup key: %v", err)
		}
		deleterAddr := common.PrivKeyToAddress(deleterKey)
		deleter := sdk.NewAudiusdSDK(nodeUrl)
		deleter.SetPrivKey(deleterKey)

		t.Logf("creator key: %s", creatorAddr)
		t.Logf("updater key: %s", updaterAddr)
		t.Logf("deleter key: %s", deleterAddr)

		// Step 2: Create three rewards with different claim authorities
		// Reward 1: creator and updater as claim authorities
		reward1, err := creator.Rewards.CreateReward(ctx, &v1.CreateReward{
			RewardId: "reward1",
			Name:     "Test Reward 1",
			Amount:   1000,
			ClaimAuthorities: []*v1.ClaimAuthority{
				{Address: creatorAddr, Name: "Creator"},
				{Address: updaterAddr, Name: "Updater"},
			},
			DeadlineBlockHeight: 999999,
		})
		if err != nil {
			t.Fatalf("Failed to create reward1: %v", err)
		}
		t.Logf("Created reward1 at address: %s", reward1.Address)

		// Reward 2: only creator as claim authority
		reward2, err := creator.Rewards.CreateReward(ctx, &v1.CreateReward{
			RewardId: "reward2",
			Name:     "Test Reward 2",
			Amount:   2000,
			ClaimAuthorities: []*v1.ClaimAuthority{
				{Address: creatorAddr, Name: "Creator"},
			},
			DeadlineBlockHeight: 999999,
		})
		if err != nil {
			t.Fatalf("Failed to create reward2: %v", err)
		}
		t.Logf("Created reward2 at address: %s", reward2.Address)

		// Reward 3: creator and deleter as claim authorities
		reward3, err := creator.Rewards.CreateReward(ctx, &v1.CreateReward{
			RewardId: "reward3",
			Name:     "Test Reward 3",
			Amount:   3000,
			ClaimAuthorities: []*v1.ClaimAuthority{
				{Address: creatorAddr, Name: "Creator"},
				{Address: deleterAddr, Name: "Deleter"},
			},
			DeadlineBlockHeight: 999999,
		})
		if err != nil {
			t.Fatalf("Failed to create reward3: %v", err)
		}
		t.Logf("Created reward3 at address: %s", reward3.Address)

		// Step 3: Query GetRewards for each user and verify correct rewards show up
		// Creator should see all three rewards
		creatorRewards, err := creator.Rewards.GetRewards(ctx, creatorAddr)
		if err != nil {
			t.Fatalf("Failed to get creator rewards: %v", err)
		}
		if len(creatorRewards.Rewards) != 3 {
			t.Fatalf("Expected creator to have 3 rewards, got %d", len(creatorRewards.Rewards))
		}
		t.Logf("Creator has %d rewards", len(creatorRewards.Rewards))

		// Updater should see only reward1
		updaterRewards, err := updater.Rewards.GetRewards(ctx, updaterAddr)
		if err != nil {
			t.Fatalf("Failed to get updater rewards: %v", err)
		}
		if len(updaterRewards.Rewards) != 1 {
			t.Fatalf("Expected updater to have 1 reward, got %d", len(updaterRewards.Rewards))
		}
		if updaterRewards.Rewards[0].Address != reward1.Address {
			t.Fatalf("Expected updater to have reward1, got different reward")
		}
		t.Logf("Updater has %d rewards", len(updaterRewards.Rewards))

		// Deleter should see only reward3
		deleterRewards, err := deleter.Rewards.GetRewards(ctx, deleterAddr)
		if err != nil {
			t.Fatalf("Failed to get deleter rewards: %v", err)
		}
		if len(deleterRewards.Rewards) != 1 {
			t.Fatalf("Expected deleter to have 1 reward, got %d", len(deleterRewards.Rewards))
		}
		if deleterRewards.Rewards[0].Address != reward3.Address {
			t.Fatalf("Expected deleter to have reward3, got different reward")
		}
		t.Logf("Deleter has %d rewards", len(deleterRewards.Rewards))

		// Step 4: Updater updates reward1 to remove creator as claim authority
		_, err = updater.Rewards.UpdateReward(ctx, &v1.UpdateReward{
			Address: reward1.Address,
			Name:    "Test Reward 1 Updated",
			Amount:  1500,
			ClaimAuthorities: []*v1.ClaimAuthority{
				{Address: updaterAddr, Name: "Updater"},
			},
			DeadlineBlockHeight: 999999,
		})
		if err != nil {
			t.Fatalf("Failed to update reward1: %v", err)
		}
		t.Logf("Updated reward1 to remove creator")

		// Step 5: Test GetRewards after update
		// Creator should now see only 2 rewards (reward2 and reward3)
		creatorRewardsAfterUpdate, err := creator.Rewards.GetRewards(ctx, creatorAddr)
		if err != nil {
			t.Fatalf("Failed to get creator rewards after update: %v", err)
		}
		if len(creatorRewardsAfterUpdate.Rewards) != 2 {
			t.Fatalf("Expected creator to have 2 rewards after update, got %d", len(creatorRewardsAfterUpdate.Rewards))
		}
		t.Logf("Creator has %d rewards after update", len(creatorRewardsAfterUpdate.Rewards))

		// Updater should still see 1 reward (the updated reward1)
		updaterRewardsAfterUpdate, err := updater.Rewards.GetRewards(ctx, updaterAddr)
		if err != nil {
			t.Fatalf("Failed to get updater rewards after update: %v", err)
		}
		if len(updaterRewardsAfterUpdate.Rewards) != 1 {
			t.Fatalf("Expected updater to have 1 reward after update, got %d", len(updaterRewardsAfterUpdate.Rewards))
		}
		if updaterRewardsAfterUpdate.Rewards[0].Address != reward1.Address {
			t.Fatalf("Expected updater to still have reward1 after update")
		}
		t.Logf("Updater has %d rewards after update", len(updaterRewardsAfterUpdate.Rewards))

		// Step 6: Creator attempts to update reward1 and should fail
		// First, let's verify what the current state of reward1 is
		currentReward1, err := creator.Rewards.GetReward(ctx, reward1.Address)
		if err != nil {
			t.Fatalf("Failed to get current reward1 state: %v", err)
		}
		t.Logf("Current reward1 claim authorities: %v", currentReward1.ClaimAuthorities)
		t.Logf("Creator address: %s", creatorAddr)
		t.Logf("Updater address: %s", updaterAddr)

		_, err = creator.Rewards.UpdateReward(ctx, &v1.UpdateReward{
			Address: reward1.Address,
			Name:    "Should Fail Update",
			Amount:  9999,
			ClaimAuthorities: []*v1.ClaimAuthority{
				{Address: creatorAddr, Name: "Creator"},
			},
			DeadlineBlockHeight: 999999,
		})
		if err == nil {
			t.Fatalf("Expected creator update to fail, but it succeeded")
		}
		t.Logf("Creator correctly failed to update reward1: %v", err)

		// Step 7: Deleter deletes reward3
		_, err = deleter.Rewards.DeleteReward(ctx, &v1.DeleteReward{
			Address:             reward3.Address,
			DeadlineBlockHeight: 999999,
		})
		if err != nil {
			t.Fatalf("Failed to delete reward3: %v", err)
		}
		t.Logf("Deleter successfully deleted reward3")

		// Step 8: Verify reward3 no longer shows up in relevant GetRewards queries
		// Creator should now see only 1 reward (reward2)
		creatorRewardsAfterDelete, err := creator.Rewards.GetRewards(ctx, creatorAddr)
		if err != nil {
			t.Fatalf("Failed to get creator rewards after delete: %v", err)
		}
		if len(creatorRewardsAfterDelete.Rewards) != 1 {
			t.Fatalf("Expected creator to have 1 reward after delete, got %d", len(creatorRewardsAfterDelete.Rewards))
		}
		if creatorRewardsAfterDelete.Rewards[0].Address != reward2.Address {
			t.Fatalf("Expected creator to have only reward2 after delete")
		}
		t.Logf("Creator has %d rewards after delete", len(creatorRewardsAfterDelete.Rewards))

		// Deleter should now see 0 rewards
		deleterRewardsAfterDelete, err := deleter.Rewards.GetRewards(ctx, deleterAddr)
		if err != nil {
			t.Fatalf("Failed to get deleter rewards after delete: %v", err)
		}
		if len(deleterRewardsAfterDelete.Rewards) != 0 {
			t.Fatalf("Expected deleter to have 0 rewards after delete, got %d", len(deleterRewardsAfterDelete.Rewards))
		}
		t.Logf("Deleter has %d rewards after delete", len(deleterRewardsAfterDelete.Rewards))

		t.Logf("All reward lifecycle tests passed successfully!")
	})
}
