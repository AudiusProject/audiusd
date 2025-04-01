package rewards

import (
	_ "embed"
	"fmt"
)

//go:embed schemas/claim_schema.json
var claimSchema []byte

//go:embed schemas/attestation_schema.json
var attestationSchema []byte

//go:embed schemas/reward_schema.json
var rewardSchema []byte

// Repeat for rewards data
//
//go:embed dev/rewards.json
var rewardsDev []byte

//go:embed stage/rewards.json
var rewardsStage []byte

//go:embed prod/rewards.json
var rewardsProd []byte

func getEnvFiles(env string) (c []byte, a []byte, r []byte, re []byte, err error) {
	switch env {
	case "dev":
		return claimSchema, attestationSchema, rewardSchema, rewardsDev, nil
	case "stage":
		return claimSchema, attestationSchema, rewardSchema, rewardsStage, nil
	case "prod":
		return claimSchema, attestationSchema, rewardSchema, rewardsProd, nil
	default:
		return nil, nil, nil, nil, fmt.Errorf("unknown environment: %s", env)
	}
}
