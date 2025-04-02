package rewards

import (
	_ "embed"
	"fmt"
)

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

func getEnvFiles(env string) (r []byte, re []byte, err error) {
	switch env {
	case "dev":
		return rewardSchema, rewardsDev, nil
	case "stage":
		return rewardSchema, rewardsStage, nil
	case "prod":
		return rewardSchema, rewardsProd, nil
	default:
		return nil, nil, fmt.Errorf("unknown environment: %s", env)
	}
}
