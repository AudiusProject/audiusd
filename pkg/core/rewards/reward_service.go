package rewards

import (
	"strings"

	"github.com/AudiusProject/audiusd/pkg/core/config"
)

type RewardService struct {
	Config  *config.Config
	Rewards []Reward
}

func NewRewardService(config *config.Config) *RewardService {
	// Create a deep copy of BaseRewards
	rewards := make([]Reward, len(BaseRewards))
	copy(rewards, BaseRewards)

	// Get the appropriate pubkeys and reward extensions based on environment
	var pubkeys []string
	var extensions []Reward
	switch config.Environment {
	case "dev":
		pubkeys = DevPubkeys
		extensions = DevRewardExtensions
	case "stage":
		pubkeys = StagePubkeys
		extensions = StageRewardExtensions
	case "prod":
		pubkeys = ProdPubkeys
		extensions = ProdRewardExtensions
	}

	// Capitalize pubkeys
	for i := range pubkeys {
		pubkeys[i] = strings.ToUpper(pubkeys[i])
	}

	// Assign pubkeys to all base rewards
	for i := range rewards {
		rewards[i].ClaimWallets = pubkeys
	}

	// Add environment-specific rewards
	if len(extensions) > 0 {
		// Create a copy of extensions to avoid modifying the original
		extendedRewards := make([]Reward, len(extensions))
		copy(extendedRewards, extensions)

		// Assign pubkeys to extended rewards
		for i := range extendedRewards {
			extendedRewards[i].ClaimWallets = pubkeys
		}

		// Append extended rewards to base rewards
		rewards = append(rewards, extendedRewards...)
	}

	return &RewardService{
		Config:  config,
		Rewards: rewards,
	}
}

type Reward struct {
	ClaimWallets []string `json:"claim_wallets"`
	Amount       uint64   `json:"amount"`
	RewardId     string   `json:"reward_id"`
	Name         string   `json:"name"`
}
