package rewards

import (
	"crypto/ecdsa"
	"encoding/hex"
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var (
	privKey = mustPrivateKeyFromHex("d09ba371c359f10f22ccda12fd26c598c7921bda3220c9942174562bc6a36fe8")
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
	require.NotNil(t, privKey)
}

func TestRecovery(t *testing.T) {
	// sourced from http://audius-protocol-discovery-provider-1/v1/full/challenges/fp/attest?oracle=0xF0D5BC18421fa04D0a2A2ef540ba5A9f04014BE3&specifier=96509ed&user_id=4OWaod
	hash := GetClaimDataHash("4OWaod", "fp", "96509ed", "0xF0D5BC18421fa04D0a2A2ef540ba5A9f04014BE3")
	signature, err := SignClaimDataHash(hash, privKey)
	require.NoError(t, err)
	require.Equal(t, "0x638ec25893b1eb45b8d4c649a936fb6a3ebd8b261075f958a7fb00e4e76d378538d87772d33ef379af8e367e3592b010194190e36a0d4ddaa3e09182758fd52801", signature)
	wallet, err := RecoverWalletFromSignature(hash, signature)
	require.NoError(t, err)
	require.Equal(t, "0x73EB6d82CFB20bA669e9c178b718d770C49BB52f", wallet)
}
