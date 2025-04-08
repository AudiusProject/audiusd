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
	rewards := []Reward{}
	if config.Environment == "dev" {
		rewards = append(rewards, DevRewards...)
	} else if config.Environment == "stage" {
		rewards = append(rewards, StageRewards...)
	} else if config.Environment == "prod" {
		rewards = append(rewards, ProdRewards...)
	}

	// caplitalize pubkeys in rewards
	for i, reward := range rewards {
		for j, claimWallet := range reward.ClaimWallets {
			rewards[i].ClaimWallets[j] = strings.ToUpper(claimWallet)
		}
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
