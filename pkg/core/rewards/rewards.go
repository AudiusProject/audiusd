package rewards

import (
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

const WAUDIO_DECIMALS = 8 // adjust this based on your token setup

var (
	ErrInvalidBase64Input       = errors.New("invalid base64 input")
	ErrInvalidJSON              = errors.New("failed to unmarshal JSON into struct")
	ErrRemarshalFailed          = errors.New("failed to re-marshal claim struct")
	ErrSchemaValidationFailed   = errors.New("claim_schema validation failed")
	ErrCanonicalizationFailed   = errors.New("failed to canonicalize JSON")
	ErrNotCanonicalJSON         = errors.New("input JSON is not canonicalized")
	ErrInvalidSignatureHex      = errors.New("invalid signature hex")
	ErrInvalidSignatureLength   = errors.New("invalid signature length")
	ErrSignatureRecoveryFailed  = errors.New("failed to recover public key")
	ErrCanonicalDecodeFailed    = errors.New("failed to decode canonical base64")
	ErrSigningFailed            = errors.New("failed to sign reward claim")
	ErrMarshalAttestationFailed = errors.New("failed to marshal attestation")
	ErrAmountMismatch           = errors.New("amounts not matching")
	ErrUnauthorizedSigner       = errors.New("not signed by correct key")
	ErrClaimNotValidReward      = errors.New("claim not valid reward")
)

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

func NewRewardService(config *config.Config, logger *common.Logger) (*RewardService, error) {
	rewardSchemaData, rewardsData, err := getEnvFiles(config.Environment)
	if err != nil {
		return nil, err
	}

	rewardSchema, err := jsonschema.CompileString("reward_schema.json", string(rewardSchemaData))
	if err != nil {
		logger.Errorf("could not compile reward_schema.json schema: %v", err)
		return nil, err
	}

	var rawRewards []any
	if err := json.Unmarshal(rewardsData, &rawRewards); err != nil {
		logger.Errorf("could not parse rewards.json: %v", err)
		return nil, err
	}

	var rewards []*Reward
	for _, raw := range rawRewards {
		if err := rewardSchema.Validate(raw); err != nil {
			logger.Errorf("invalid reward in rewards.json: %v", err)
			return nil, err
		}

		rawBytes, err := json.Marshal(raw)
		if err != nil {
			logger.Errorf("could not re-marshal reward: %v", err)
			return nil, err
		}

		var reward Reward
		if err := json.Unmarshal(rawBytes, &reward); err != nil {
			logger.Errorf("could not unmarshal reward into struct: %v", err)
			return nil, err
		}
		rewards = append(rewards, &reward)
	}

	return &RewardService{
		config:       config,
		logger:       logger,
		rewards:      rewards,
		rewardSchema: rewardSchema,
	}, nil
}

func (rs *RewardService) AttestRewardClaim(data, signature string) (owner string, attestation string, err error) {
	return "", "", nil
}

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

	sep := []byte("_")
	joined := append(userBytes, sep...)
	joined = append(joined, amountBytes...)
	joined = append(joined, sep...)
	joined = append(joined, combinedIDBytes...)
	joined = append(joined, sep...)
	joined = append(joined, oracleBytes...)

	return joined, nil
}

func (rs *RewardService) SignAttestation(attestationBytes []byte) (string, error) {
	privateKey := rs.config.EthereumKey

	hash := crypto.Keccak256(attestationBytes)
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign hash: %w", err)
	}

	return "0x" + hex.EncodeToString(signature), nil
}

// Helpers

func uint64Pow(a, b int) uint64 {
	result := uint64(1)
	for range b {
		result *= uint64(a)
	}
	return result
}

func toBytesHex(hexStr string) ([]byte, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	return hex.DecodeString(hexStr)
}
