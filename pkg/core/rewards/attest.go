package rewards

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

func (rs *RewardService) GetRewardById(rewardID string) (*Reward, error) {
	for _, reward := range rs.Rewards {
		if reward.RewardId == rewardID {
			return &reward, nil
		}
	}
	return nil, fmt.Errorf("reward %s not found", rewardID)
}

func (rs *RewardService) SignAttestation(attestationBytes []byte) (owner string, attestation string, err error) {
	privateKey := rs.Config.EthereumKey
	pubKey := privateKey.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return "", "", fmt.Errorf("failed to cast public key to ecdsa.PublicKey")
	}
	owner = crypto.PubkeyToAddress(*pubKeyECDSA).String()

	// Apply Ethereum message prefix and hash
	prefixedHash := accounts.TextHash(attestationBytes)

	// Sign the prefixed hash
	signature, err := crypto.Sign(prefixedHash, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign attestation: %w", err)
	}

	return owner, "0x" + hex.EncodeToString(signature), nil
}

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

func GetClaimDataHash(userWallet, challengeId, challengeSpecifier, oracleAddress string) []byte {
	claimData := fmt.Sprintf("%s_%s_%s_%s", userWallet, challengeId, challengeSpecifier, oracleAddress)
	return crypto.Keccak256([]byte(claimData))
}

func SignClaimDataHash(hash []byte, privateKey *ecdsa.PrivateKey) (string, error) {
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign hash: %w", err)
	}
	return "0x" + hex.EncodeToString(signature), nil
}
