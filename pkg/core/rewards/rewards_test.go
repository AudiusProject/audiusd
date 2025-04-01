package rewards_test

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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

func generateCanonicalSignedClaim(t *testing.T, privKey *ecdsa.PrivateKey, claim rewards.RewardClaim) (string, string) {
	raw, err := json.Marshal(claim)
	require.NoError(t, err)

	canonicalJSON, err := jsoncanonicalizer.Transform(raw)
	require.NoError(t, err)

	hash := sha256.Sum256(canonicalJSON)
	sig, err := crypto.Sign(hash[:], privKey)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(canonicalJSON), hex.EncodeToString(sig)
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

	claimBase64 := base64.StdEncoding.EncodeToString(canonicalJSON)
	signature := hex.EncodeToString(sigBytes)
	return claimBase64, signature
}

func TestAttestRewardClaim(t *testing.T) {
	logger := common.NewLogger(nil)

	// Use the dev embedded rewards.json so match a real pubkey from that
	nodeKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	testClaim := rewards.RewardClaim{
		ID:        "u",
		Amount:    1,
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

func newTestService(t *testing.T) *rewards.RewardService {
	cfg := &config.Config{
		EthereumKey: trackUploadPrivKey,
		Environment: "dev",
	}
	logger := common.NewLogger(nil)
	svc, err := rewards.NewRewardService(cfg, logger)
	require.NoError(t, err)
	return svc
}

func TestError_InvalidBase64(t *testing.T) {
	svc := newTestService(t)
	_, _, err := svc.AttestRewardClaim("!notbase64!", "abc123")
	require.ErrorIs(t, err, rewards.ErrInvalidBase64Input)
}

func TestError_InvalidJSON(t *testing.T) {
	svc := newTestService(t)
	input := base64.StdEncoding.EncodeToString([]byte("{invalid"))
	_, _, err := svc.AttestRewardClaim(input, "abc123")
	require.ErrorIs(t, err, rewards.ErrInvalidJSON)
}

func TestError_NotCanonicalJSON(t *testing.T) {
	svc := newTestService(t)
	claim := rewards.RewardClaim{
		ID:        "u",
		Amount:    3,
		Specifier: "1234",
	}
	raw, _ := json.Marshal(claim)
	data := base64.StdEncoding.EncodeToString(raw)
	_, _, err := svc.AttestRewardClaim(data, "00")
	require.ErrorIs(t, err, rewards.ErrNotCanonicalJSON)
}

func TestError_InvalidSignatureHex(t *testing.T) {
	svc := newTestService(t)
	claim := rewards.RewardClaim{ID: "u", Amount: 3, Specifier: "1234"}
	data, _ := generateCanonicalSignedClaim(t, trackUploadPrivKey, claim)
	_, _, err := svc.AttestRewardClaim(data, "ZZZ")
	require.ErrorIs(t, err, rewards.ErrInvalidSignatureHex)
}

func TestError_InvalidSignatureLength(t *testing.T) {
	svc := newTestService(t)
	claim := rewards.RewardClaim{ID: "u", Amount: 3, Specifier: "1234"}
	data, _ := generateCanonicalSignedClaim(t, trackUploadPrivKey, claim)
	_, _, err := svc.AttestRewardClaim(data, "abcd")
	require.ErrorIs(t, err, rewards.ErrInvalidSignatureLength)
}

func TestError_ClaimNotValidReward(t *testing.T) {
	svc := newTestService(t)
	claim := rewards.RewardClaim{ID: "does-not-exist", Amount: 1, Specifier: "x"}
	data, sig := generateCanonicalSignedClaim(t, trackUploadPrivKey, claim)
	_, _, err := svc.AttestRewardClaim(data, sig)
	require.True(t, errors.Is(err, rewards.ErrClaimNotValidReward))
}

func TestError_AmountMismatch(t *testing.T) {
	svc := newTestService(t)
	claim := rewards.RewardClaim{ID: "u", Amount: 999, Specifier: "1234"}
	data, sig := generateCanonicalSignedClaim(t, trackUploadPrivKey, claim)
	_, _, err := svc.AttestRewardClaim(data, sig)
	require.ErrorIs(t, err, rewards.ErrAmountMismatch)
}

func TestError_UnauthorizedSigner(t *testing.T) {
	svc := newTestService(t)

	// valid claim, but signer is not in pubkeys for this reward
	otherKey, _ := crypto.GenerateKey()
	claim := rewards.RewardClaim{ID: "u", Amount: 1, Specifier: "1234"}
	data, sig := generateCanonicalSignedClaim(t, otherKey, claim)
	_, _, err := svc.AttestRewardClaim(data, sig)
	require.ErrorIs(t, err, rewards.ErrUnauthorizedSigner)
}
