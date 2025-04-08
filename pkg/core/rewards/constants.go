package rewards

var (
	DevRewards = []Reward{
		{
			ClaimWallets: []string{"0x1234567890123456789012345678901234567890"},
			Amount:       1000000000000000000,
			RewardId:     "dev-reward-1",
		},
	}

	StageRewards = []Reward{
		{
			ClaimWallets: []string{"0x1234567890123456789012345678901234567890"},
			Amount:       1000000000000000000,
			RewardId:     "stage-reward-1",
		},
	}

	ProdRewards = []Reward{
		{
			ClaimWallets: []string{"0x1234567890123456789012345678901234567890"},
			Amount:       1000000000000000000,
			RewardId:     "prod-reward-1",
		},
	}
)
