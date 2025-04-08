package rewards

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

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

	reward, err := rs.GetRewardById(challengeId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	claimDataHash := GetClaimDataHash(userWallet, challengeId, challengeSpecifier, oracleAddress)
	recoveredWallet, err := RecoverWalletFromSignature(claimDataHash, signature)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	if !slices.Contains(reward.ClaimWallets, strings.ToUpper(recoveredWallet)) {
		return c.JSON(http.StatusUnauthorized, fmt.Sprintf("wallet %s is not authorized to claim reward %s", recoveredWallet, challengeId))
	}

	// construct attestation bytes
	attestationBytes := []byte(fmt.Sprintf("%s_%s_%s_%s", userWallet, challengeId, challengeSpecifier, oracleAddress))

	owner, attestation, err := rs.SignAttestation(attestationBytes)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	res := map[string]any{
		"owner":       owner,
		"attestation": attestation,
	}
	return c.JSON(http.StatusOK, res)
}
