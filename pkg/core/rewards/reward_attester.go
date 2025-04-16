package rewards

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/ethereum/go-ethereum/crypto"
)

type RewardAttester struct {
	EthereumAddress string
	EthereumKey     *ecdsa.PrivateKey
	Rewards         []Reward
}

func (rs *RewardAttester) Validate(claim RewardClaim) error {
	reward, err := rs.getRewardById(claim.RewardID)
	if err != nil {
		return err
	}

	if claim.RecipientEthAddress == "" {
		return fmt.Errorf("missing recipient eth address")
	}

	if claim.Specifier == "" {
		return fmt.Errorf("missing specifier")
	}

	if claim.Amount != reward.Amount {
		return fmt.Errorf("amount does not match reward amount")
	}

	// TODO: Check oracle is registered, maybe validate lengths of inputs?

	return nil
}

func (rs *RewardAttester) Authenticate(claim RewardClaim, signature string) error {
	reward, err := rs.getRewardById(claim.RewardID)
	if err != nil {
		return err
	}

	recoveredSigner, err := recoverSigner(claim, signature)
	if err != nil {
		return err
	}

	if !ValidClaimAuthority(reward.ClaimAuthorities, recoveredSigner) {
		return fmt.Errorf("address %s is not a claim authority for reward %s", recoveredSigner, reward.RewardId)
	}

	return nil
}

func (rs *RewardAttester) Attest(claim RewardClaim) (message []byte, signature string, err error) {
	claimData, err := claim.Compile()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get attestation bytes: %w", err)
	}

	hash := crypto.Keccak256(claimData)

	signatureBytes, err := crypto.Sign(hash, rs.EthereumKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign hash: %w", err)
	}

	return claimData, "0x" + hex.EncodeToString(signatureBytes), nil
}

func NewRewardAttester(config *config.Config) *RewardAttester {
	// Create a deep copy of BaseRewards
	rewards := make([]Reward, len(BaseRewards))
	copy(rewards, BaseRewards)

	// Get the appropriate pubkeys and reward extensions based on environment
	var pubkeys []ClaimAuthority
	var extensions []Reward
	switch config.Environment {
	case "dev":
		pubkeys = DevClaimAuthorities
		extensions = DevRewardExtensions
	case "stage":
		pubkeys = StageClaimAuthorities
		extensions = StageRewardExtensions
	case "prod":
		pubkeys = ProdClaimAuthorities
		extensions = ProdRewardExtensions
	}

	// Assign pubkeys to all base rewards
	for i := range rewards {
		rewards[i].ClaimAuthorities = pubkeys
	}

	// Add environment-specific rewards
	if len(extensions) > 0 {
		// Create a copy of extensions to avoid modifying the original
		extendedRewards := make([]Reward, len(extensions))
		copy(extendedRewards, extensions)

		// Assign pubkeys to extended rewards
		for i := range extendedRewards {
			extendedRewards[i].ClaimAuthorities = pubkeys
		}

		// Append extended rewards to base rewards
		rewards = append(rewards, extendedRewards...)
	}

	// Get owner address from private key
	address := ""
	if config.EthereumKey != nil {
		pubKey := config.EthereumKey.Public()
		pubKeyECDSA, _ := pubKey.(*ecdsa.PublicKey)
		address = crypto.PubkeyToAddress(*pubKeyECDSA).Hex()
	}

	return &RewardAttester{
		EthereumAddress: address,
		EthereumKey:     config.EthereumKey,
		Rewards:         rewards,
	}
}

func (rs *RewardAttester) getRewardById(rewardID string) (*Reward, error) {
	for _, reward := range rs.Rewards {
		if reward.RewardId == rewardID {
			return &reward, nil
		}
	}
	return nil, fmt.Errorf("reward %s not found", rewardID)
}

func recoverSigner(claim RewardClaim, signature string) (string, error) {
	claimData, err := claim.Compile()
	if err != nil {
		return "", fmt.Errorf("failed to get attestation bytes: %w", err)
	}

	// Remove any existing 0x prefix
	sigHex := strings.TrimPrefix(signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode signature: %w", err)
	}

	hash := crypto.Keccak256(claimData)
	recoveredWallet, err := crypto.SigToPub(hash[:], sigBytes)
	if err != nil {
		return "", fmt.Errorf("failed to recover wallet from signature: %w", err)
	}

	return crypto.PubkeyToAddress(*recoveredWallet).String(), nil
}
