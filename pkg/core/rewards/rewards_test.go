package rewards_test

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log"
	"testing"

	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/AudiusProject/audiusd/pkg/core/rewards"
	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/ethereum/go-ethereum/crypto"
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

func generateSignedClaim(t *testing.T, privKey *ecdsa.PrivateKey, claim rewards.RewardClaim) (string, string) {
	t.Helper()

	claimJSON, err := json.Marshal(claim)
	require.NoError(t, err)

	canonicalJSON, err := jsoncanonicalizer.Transform(claimJSON)
	require.NoError(t, err)

	hash := sha256.Sum256(canonicalJSON)
	sigBytes, err := crypto.Sign(hash[:], privKey)
	require.NoError(t, err)

	claimBase64 := base64.StdEncoding.EncodeToString(claimJSON) // still base64 encode original JSON for API call
	signature := hex.EncodeToString(sigBytes)
	return claimBase64, signature
}

func TestAttestRewardClaim_DevEnv(t *testing.T) {
	logger := common.NewLogger(nil)

	// Use the dev embedded rewards.json so match a real pubkey from that
	nodeKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	testClaim := rewards.RewardClaim{
		ID:        "track-upload",
		Amount:    3,
		Specifier: "1234",
	}

	claimDataB64, claimSignature := generateSignedClaim(t, trackUploadPrivKey, testClaim)

	cfg := &config.Config{
		EthereumKey: nodeKey,
		Environment: "dev",
	}

	service, err := rewards.NewRewardService(cfg, logger)
	require.NoError(t, err)

	attestationB64, sigHex, err := service.AttestRewardClaim(claimDataB64, claimSignature)
	require.NoError(t, err)
	require.NotEmpty(t, attestationB64)
	require.NotEmpty(t, sigHex)

	decoded, err := base64.StdEncoding.DecodeString(attestationB64)
	require.NoError(t, err)

	var att rewards.RewardAttestation
	err = json.Unmarshal(decoded, &att)
	require.NoError(t, err)
	require.Equal(t, testClaim.ID, att.ID)
	require.Equal(t, testClaim.Amount, att.Amount)
	require.Equal(t, testClaim.Specifier, att.Specifier)
	require.Equal(t, sigHex, att.Signature)
}
