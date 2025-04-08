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
	rewardID := c.QueryParam("reward_id")
	if rewardID == "" {
		return c.JSON(http.StatusBadRequest, "reward_id is required")
	}
	specifier := c.QueryParam("specifier")
	if specifier == "" {
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

	reward, err := rs.GetRewardById(rewardID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	claimDataHash := GetClaimDataHash(userWallet, rewardID, specifier, oracleAddress)
	valid := CompareClaimHash(userWallet, rewardID, specifier, oracleAddress, claimDataHash)
	if !valid {
		return c.JSON(http.StatusUnauthorized, "invalid claim data hash")
	}
	recoveredWallet, err := RecoverWalletFromSignature(claimDataHash, signature)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	if !slices.Contains(reward.ClaimWallets, strings.ToUpper(recoveredWallet)) {
		return c.JSON(http.StatusUnauthorized, fmt.Sprintf("wallet %s is not authorized to claim reward %s", recoveredWallet, rewardID))
	}

	// construct attestation bytes
	attestationBytes, err := GetAttestationBytes(userWallet, rewardID, specifier, oracleAddress, reward.Amount)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

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
