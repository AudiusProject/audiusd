package rewards

import (
	"bytes"
	"crypto/ecdsa"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"slices"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/labstack/echo/v4"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

const WAUDIO_DECIMALS = 8 // adjust this based on your token setup

type Reward struct {
	ID      string   `json:"id"`
	Amount  uint     `json:"amount"`
	Pubkeys []string `json:"pubkeys"`
}

type RewardService struct {
	config *config.Config
	logger *common.Logger

	rewards      []*Reward
	rewardSchema *jsonschema.Schema
}

func NewRewardService(config *config.Config, logger *common.Logger) *RewardService {
	rewardSchemaData, rewardsData, err := getEnvFiles(config.Environment)
	if err != nil {
		logger.Errorf("could not get env files: %v", err)
		return &RewardService{
			config: config,
			logger: logger,
		}
	}

	rewardSchema, err := jsonschema.CompileString("reward_schema.json", string(rewardSchemaData))
	if err != nil {
		logger.Errorf("could not compile reward_schema.json schema: %v", err)
		return &RewardService{
			config: config,
			logger: logger,
		}
	}

	var rawRewards []any
	if err := json.Unmarshal(rewardsData, &rawRewards); err != nil {
		logger.Errorf("could not parse rewards.json: %v", err)
		return &RewardService{
			config: config,
			logger: logger,
		}
	}

	var rewards []*Reward
	for _, raw := range rawRewards {
		if err := rewardSchema.Validate(raw); err != nil {
			logger.Errorf("invalid reward in rewards.json: %v", err)
			continue
		}

		rawBytes, err := json.Marshal(raw)
		if err != nil {
			logger.Errorf("could not re-marshal reward: %v", err)
			continue
		}

		var reward Reward
		if err := json.Unmarshal(rawBytes, &reward); err != nil {
			logger.Errorf("could not unmarshal reward into struct: %v", err)
			continue
		}
		rewards = append(rewards, &reward)
	}

	return &RewardService{
		config:       config,
		logger:       logger,
		rewards:      rewards,
		rewardSchema: rewardSchema,
	}
}

// GetClaimDataHash constructs and hashes the claim data from its components
func GetClaimDataHash(userWallet, challengeId, challengeSpecifier, oracleAddress string) []byte {
	claimData := fmt.Sprintf("%s_%s_%s_%s", userWallet, challengeId, challengeSpecifier, oracleAddress)
	return crypto.Keccak256([]byte(claimData))
}

// RecoverWalletFromSignature recovers the wallet address from a signature and its corresponding hash
func RecoverWalletFromSignature(hash []byte, signature string) (string, error) {
	// Remove any existing 0x prefix
	sigHex := strings.TrimPrefix(signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode signature: %w", err)
	}

	recoveredWallet, err := crypto.SigToPub(hash[:], sigBytes)
	if err != nil {
		return "", fmt.Errorf("failed to recover wallet from signature: %w", err)
	}

	return crypto.PubkeyToAddress(*recoveredWallet).String(), nil
}

func (rs *RewardService) AttestReward(c echo.Context) error {
	rs.logger.Infof("AttestReward %v", c.QueryParams())
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

	_, attestationSigner, attestation, err := rs.AttestRewardClaim(userWallet, challengeId, challengeSpecifier, oracleAddress, signature)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"owner":       attestationSigner,
		"attestation": attestation,
	})
}

func (rs *RewardService) AttestRewardClaim(userWallet, challengeId, challengeSpecifier, oracleAddress, signature string) (claimSigner string, attestationSigner string, attestation string, err error) {
	// construct the claim data and recover the signer wallet from the signature
	hash := GetClaimDataHash(userWallet, challengeId, challengeSpecifier, oracleAddress)
	rs.logger.Infof("signature: %s, hash: %s", signature, hash)
	claimWallet, err := RecoverWalletFromSignature(hash, signature)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to recover wallet: %w", err)
	}

	// get reward object from rewards array, use method to get reward by id
	reward, err := rs.GetRewardById(challengeId)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get reward: %w", err)
	}

	// compare the claim wallet to the reward pubkeys, early out if match is found.
	// return error if no match is found
	found := slices.Contains(reward.Pubkeys, claimWallet)
	if !found {
		return "", "", "", fmt.Errorf("claim wallet %s does not match any reward pubkeys %v", claimWallet, reward.Pubkeys)
	}

	// create attestation object
	attestationObj := &Attestation{
		Amount:             fmt.Sprintf("%d", reward.Amount),
		OracleAddress:      oracleAddress,
		UserAddress:        claimWallet,
		ChallengeID:        challengeId,
		ChallengeSpecifier: challengeSpecifier,
	}

	// get attestation bytes
	attestationBytes, err := attestationObj.GetAttestationBytes()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get attestation bytes: %w", err)
	}
	// Sign the attestation
	signedAttestation, err := rs.SignAttestation(attestationBytes)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to sign attestation: %w", err)
	}

	return claimWallet, rs.config.WalletAddress, signedAttestation, nil
}

// iterate through rewards array and return the reward object that matches the challengeId
func (rs *RewardService) GetRewardById(challengeId string) (*Reward, error) {
	for _, reward := range rs.rewards {
		if reward.ID == challengeId {
			return reward, nil
		}
	}
	return nil, fmt.Errorf("reward not found")
}

// structure used by the node to sign the attestation that is responded to the GET attest route
type Attestation struct {
	Amount             string
	OracleAddress      string
	UserAddress        string
	ChallengeID        string
	ChallengeSpecifier string
}

// String returns the formatted string representation.
func (a *Attestation) String() string {
	return fmt.Sprintf(
		"%s_%s_%s_%s",
		a.UserAddress,
		a.Amount,
		a.getCombinedID(),
		a.OracleAddress,
	)
}

func (a *Attestation) getCombinedID() string {
	return fmt.Sprintf("%s:%s", a.ChallengeID, a.ChallengeSpecifier)
}

func (a *Attestation) getEncodedAmount() ([]byte, error) {
	// Parse amount as integer
	var amtInt uint64
	_, err := fmt.Sscan(a.Amount, &amtInt)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	amt := amtInt * uint64Pow(10, WAUDIO_DECIMALS)
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, amt)
	return buf, nil
}

func (a *Attestation) GetAttestationBytes() ([]byte, error) {
	userBytes, err := toBytesHex(a.UserAddress)
	if err != nil {
		return nil, err
	}

	oracleBytes, err := toBytesHex(a.OracleAddress)
	if err != nil {
		return nil, err
	}

	combinedIDBytes := []byte(a.getCombinedID())

	amountBytes, err := a.getEncodedAmount()
	if err != nil {
		return nil, err
	}

	joined := bytes.Join([][]byte{
		userBytes,
		amountBytes,
		combinedIDBytes,
		oracleBytes,
	}, []byte("_"))

	return joined, nil
}

func (rs *RewardService) SignAttestation(attestationBytes []byte) (string, error) {
	privateKey := rs.config.EthereumKey // should be *ecdsa.PrivateKey

	// Apply Ethereum message prefix and hash
	prefixedHash := accounts.TextHash(attestationBytes)

	// Sign the prefixed hash
	signature, err := crypto.Sign(prefixedHash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign attestation: %w", err)
	}

	// Return hex-encoded signature
	return "0x" + hex.EncodeToString(signature), nil
}

// SignClaimDataHash signs a claim data hash with the provided private key
func SignClaimDataHash(hash []byte, privateKey *ecdsa.PrivateKey) (string, error) {
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign hash: %w", err)
	}
	return "0x" + hex.EncodeToString(signature), nil
}

// Helpers

func uint64Pow(a, b int) uint64 {
	result := uint64(1)
	for i := 0; i < b; i++ {
		result *= uint64(a)
	}
	return result
}

func toBytesHex(hexStr string) ([]byte, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	return hex.DecodeString(hexStr)
}
