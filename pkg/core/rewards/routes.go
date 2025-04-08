package rewards

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (rs *RewardService) GetRewards(c echo.Context) error {
	return c.JSON(http.StatusOK, rs.Rewards)
}

func (rs *RewardService) AttestReward(c echo.Context) error {
	userWallet := c.QueryParam("user_wallet")
	if userWallet == "" {
		return c.JSON(http.StatusBadRequest, "user_wallet is required")
	}
	challengeId := c.QueryParam("reward_id")
	if challengeId == "" {
		return c.JSON(http.StatusBadRequest, "reward_id is required")
	}
	challengeSpecifier := c.QueryParam("specifier")
	if challengeSpecifier == "" {
		return c.JSON(http.StatusBadRequest, "specifier is required")
	}
	oracleAddress := c.QueryParam("oracle_address")
	if oracleAddress == "" {
		return c.JSON(http.StatusBadRequest, "oracle_address is required")
	}
	signature := c.QueryParam("signature")
	if signature == "" {
		return c.JSON(http.StatusBadRequest, "signature is required")
	}

	res := map[string]any{
		"owner":       "node address",
		"attestation": "test attestation",
	}
	return c.JSON(http.StatusOK, res)
}
