package rewards

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	canonical "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

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

type RewardClaim struct {
	ID        string `json:"id"`
	Amount    uint   `json:"amount"`
	Specifier string `json:"specifier"`
}

type Reward struct {
	ID      string   `json:"id"`
	Amount  uint     `json:"amount"`
	Pubkeys []string `json:"pubkeys"`
}

type RewardAttestation struct {
	ID        string `json:"id"`
	Amount    uint   `json:"amount"`
	Specifier string `json:"specifier"`
	Signature string `json:"signature"`
}

type RewardService struct {
	config *config.Config
	logger *common.Logger

	rewards []*Reward

	claimSchema       *jsonschema.Schema
	attestationSchema *jsonschema.Schema
	rewardSchema      *jsonschema.Schema
}

func NewRewardService(config *config.Config, logger *common.Logger) (*RewardService, error) {
	claimSchemaData, attestationSchemaData, rewardSchemaData, rewardsData, err := getEnvFiles(config.Environment)
	if err != nil {
		return nil, err
	}

	claimSchema, err := jsonschema.CompileString("claim_schema.json", string(claimSchemaData))
	if err != nil {
		logger.Errorf("could not compile claim_schema.json schema: %v", err)
		return nil, err
	}

	attestationSchema, err := jsonschema.CompileString("attestation_schema.json", string(attestationSchemaData))
	if err != nil {
		logger.Errorf("could not compile attestation_schema.json schema: %v", err)
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
		config:            config,
		logger:            logger,
		rewards:           rewards,
		claimSchema:       claimSchema,
		attestationSchema: attestationSchema,
		rewardSchema:      rewardSchema,
	}, nil
}

func (rs *RewardService) AttestRewardClaim(data, signature string) (string, string, error) {
	rewardClaim, err := rs.ParseRewardClaim(data)
	if err != nil {
		return "", "", err
	}

	addr, canonicalB64, err := rs.RecoverSigner(data, signature)
	if err != nil {
		return "", "", err
	}

	if err := rs.ValidateRewardClaim(rewardClaim, addr); err != nil {
		return "", "", err
	}

	canonicalJSON, err := base64.StdEncoding.DecodeString(canonicalB64)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrCanonicalDecodeFailed, err)
	}

	hash := sha256.Sum256(canonicalJSON)
	privKey := rs.config.EthereumKey
	sigBytes, err := crypto.Sign(hash[:], privKey)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrSigningFailed, err)
	}

	attestation := RewardAttestation{
		ID:        rewardClaim.ID,
		Amount:    rewardClaim.Amount,
		Specifier: rewardClaim.Specifier,
		Signature: hex.EncodeToString(sigBytes),
	}

	attJSON, err := json.Marshal(attestation)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrMarshalAttestationFailed, err)
	}

	attestationB64 := base64.StdEncoding.EncodeToString(attJSON)
	return attestationB64, attestation.Signature, nil
}

func (rs *RewardService) ParseRewardClaim(data string) (*RewardClaim, error) {
	jsonBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBase64Input, err)
	}

	var claim RewardClaim
	if err := json.Unmarshal(jsonBytes, &claim); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	structBytes, err := json.Marshal(claim)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRemarshalFailed, err)
	}

	var forValidation any
	if err := json.Unmarshal(structBytes, &forValidation); err != nil {
		return nil, fmt.Errorf("unexpected: could not prepare for schema validation: %w", err)
	}

	if err := rs.claimSchema.Validate(forValidation); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchemaValidationFailed, err)
	}

	return &claim, nil
}

func (rs *RewardService) RecoverSigner(dataB64, signatureHex string) (string, string, error) {
	jsonBytes, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidBase64Input, err)
	}

	canonicalJSON, err := canonical.Transform(jsonBytes)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrCanonicalizationFailed, err)
	}

	if !slices.Equal(canonicalJSON, jsonBytes) {
		return "", "", ErrNotCanonicalJSON
	}

	hash := sha256.Sum256(canonicalJSON)
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrInvalidSignatureHex, err)
	}

	if len(sigBytes) != 65 {
		return "", "", fmt.Errorf("%w: expected 65 bytes, got %d", ErrInvalidSignatureLength, len(sigBytes))
	}

	pubKey, err := crypto.SigToPub(hash[:], sigBytes)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrSignatureRecoveryFailed, err)
	}

	recoveredAddress := crypto.PubkeyToAddress(*pubKey).Hex()
	canonicalB64 := base64.StdEncoding.EncodeToString(canonicalJSON)
	return recoveredAddress, canonicalB64, nil
}

func (rs *RewardService) ValidateRewardClaim(claim *RewardClaim, recoveredSigner string) error {
	for _, reward := range rs.rewards {
		if reward.ID != claim.ID {
			continue
		}

		if claim.Amount != reward.Amount {
			return ErrAmountMismatch
		}

		if !slices.Contains(reward.Pubkeys, recoveredSigner) {
			return ErrUnauthorizedSigner
		}

		return nil
	}
	return fmt.Errorf("%w: %s", ErrClaimNotValidReward, claim.ID)
}
