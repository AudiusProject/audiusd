package rewards

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	canonical "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/santhosh-tekuri/jsonschema/v5"
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

	// Decode canonical JSON for signing
	canonicalJSON, err := base64.StdEncoding.DecodeString(canonicalB64)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode canonical base64: %w", err)
	}

	hash := sha256.Sum256(canonicalJSON)

	privKey := rs.config.EthereumKey
	sigBytes, err := crypto.Sign(hash[:], privKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign reward claim: %w", err)
	}

	// Build RewardAttestation struct
	attestation := RewardAttestation{
		ID:        rewardClaim.ID,
		Amount:    rewardClaim.Amount,
		Specifier: rewardClaim.Specifier,
		Signature: hex.EncodeToString(sigBytes),
	}

	// Encode to JSON, then base64
	attJSON, err := json.Marshal(attestation)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal attestation: %w", err)
	}

	attestationB64 := base64.StdEncoding.EncodeToString(attJSON)

	return attestationB64, attestation.Signature, nil
}

func (rs *RewardService) ParseRewardClaim(data string) (*RewardClaim, error) {
	jsonBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 input: %w", err)
	}

	var claim RewardClaim
	if err := json.Unmarshal(jsonBytes, &claim); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into struct: %w", err)
	}

	structBytes, err := json.Marshal(claim)
	if err != nil {
		return nil, fmt.Errorf("failed to re-marshal claim struct: %w", err)
	}

	var forValidation any
	if err := json.Unmarshal(structBytes, &forValidation); err != nil {
		return nil, fmt.Errorf("unexpected: could not prepare for schema validation: %w", err)
	}
	if err := rs.claimSchema.Validate(forValidation); err != nil {
		return nil, fmt.Errorf("claim_schema validation failed: %w", err)
	}

	return &claim, nil
}

func (rs *RewardService) RecoverSigner(dataB64, signatureHex string) (string, string, error) {
	jsonBytes, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 input: %w", err)
	}

	// canonicalize the JSON, out of order fields result in different hashes
	canonicalJSON, err := canonical.Transform(jsonBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to canonicalize JSON: %w", err)
	}

	hash := sha256.Sum256(canonicalJSON)

	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return "", "", fmt.Errorf("invalid signature hex: %w", err)
	}
	if len(sigBytes) != 65 {
		return "", "", fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(sigBytes))
	}

	pubKey, err := crypto.SigToPub(hash[:], sigBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to recover public key: %w", err)
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
			return fmt.Errorf("amounts not matching")
		}

		spew.Dump(reward.Pubkeys, recoveredSigner)

		if !slices.Contains(reward.Pubkeys, recoveredSigner) {
			return fmt.Errorf("not signed by correct key")
		}

		return nil
	}
	return fmt.Errorf("claim %s not valid reward", claim.ID)
}
