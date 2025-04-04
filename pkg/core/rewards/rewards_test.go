package rewards

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"testing"

	"log/slog"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	trackUploadPrivKey = mustPrivateKeyFromHex("a57b1cab53462acec8fbd5afa21045780fd2afcbf63c4c288d60d51a00794009")
)

func mustPrivateKeyFromHex(hexKey string) *ecdsa.PrivateKey {
	bytes, err := hex.DecodeString(hexKey)
	if err != nil {
		log.Fatalf("failed to decode hex: %v", err)
	}
	privKey, err := crypto.ToECDSA(bytes)
	if err != nil {
		log.Fatalf("failed to convert to ECDSA: %v", err)
	}
	return privKey
}

func TestTestPrivateKey(t *testing.T) {
	require.NotNil(t, trackUploadPrivKey)
}

func TestAttestRewardClaim(t *testing.T) {
	// Setup test config and logger
	cfg := &config.Config{
		Environment:   "dev",
		WalletAddress: "0x24D50c19297592d5d13BEFf90A5a60E63db58c30", // matches pubkey from rewards.json
	}
	logger := common.NewLogger(&slog.HandlerOptions{})

	// Create reward service
	rs := NewRewardService(cfg, logger)

	// Test data
	encodedUserId := "mEx6RYQ"
	challengeId := "fp" // from rewards.json
	challengeSpecifier := "37364e80"
	oracleAddress := "0x00b6462e955dA5841b6D9e1E2529B830F00f31Bf"

	// Create a valid signature for the claim
	claimData := fmt.Sprintf("%s_%s_%s_%s", encodedUserId, challengeId, challengeSpecifier, oracleAddress)
	hash := crypto.Keccak256([]byte(claimData))

	// Create a private key for signing (this would normally be the oracle's key)
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	// Sign the hash
	signature, err := crypto.Sign(hash, privateKey)
	assert.NoError(t, err)

	// Convert signature to hex string
	signatureHex := hex.EncodeToString(signature)

	// Call AttestRewardClaim
	claimSigner, attestationSigner, attestation, err := rs.AttestRewardClaim(
		encodedUserId,
		challengeId,
		challengeSpecifier,
		oracleAddress,
		signatureHex,
	)

	// Verify results
	assert.NoError(t, err)
	assert.NotEmpty(t, claimSigner)
	assert.Equal(t, cfg.WalletAddress, attestationSigner)
	assert.NotEmpty(t, attestation)

	// Verify the attestation contains the correct amount (2 for "fp" reward)
	attestationObj := &Attestation{
		Amount:             "2", // from rewards.json
		OracleAddress:      oracleAddress,
		UserAddress:        claimSigner,
		ChallengeID:        challengeId,
		ChallengeSpecifier: challengeSpecifier,
	}
	expectedBytes, err := attestationObj.GetAttestationBytes()
	assert.NoError(t, err)

	// The attestation should be a signature of these bytes
	// We can verify it was signed by the service's wallet
	attestationBytes, err := hex.DecodeString(strings.TrimPrefix(attestation, "0x"))
	assert.NoError(t, err)

	// Recover the public key from the attestation
	recoveredPub, err := crypto.SigToPub(expectedBytes, attestationBytes)
	assert.NoError(t, err)
	recoveredAddr := crypto.PubkeyToAddress(*recoveredPub)
	assert.Equal(t, cfg.WalletAddress, recoveredAddr.String())
}
