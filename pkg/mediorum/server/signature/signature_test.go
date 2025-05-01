package signature

import (
	"net/url"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestSignature(t *testing.T) {
	example := `%7B%22data%22%3A%20%22%7B%5C%22trackId%5C%22%3A%201485%2C%20%5C%22cid%5C%22%3A%20%5C%22QmdGpDEBq6v6Kv9H61HbeVqyiPo7iBe12tVtkhNig6ipWp%5C%22%2C%20%5C%22timestamp%5C%22%3A%201681484247930%2C%20%5C%22userId%5C%22%3A%2050419%2C%20%5C%22shouldCache%5C%22%3A%201%7D%22%2C%20%22signature%22%3A%20%220x54e5daff013068dfe10f9e360ca39b8cda8497652a6b029e71656ea538d541187c07e6241e8d06c9ea95df01152c3b8d87f2aeb28814fdce0c13978a884bf4fa1b%22%7D`
	value, err := url.QueryUnescape(example)
	assert.NoError(t, err)

	data, err := ParseFromQueryString(value)
	assert.NoError(t, err)
	// fmt.Printf("%+v \n", data)
	assert.Equal(t, data.SignerWallet, "0x5E98cBEEAA2aCEDEc0833AC3D1634E2A7aE0f3c2")
}

func TestGenerateAndParseSignature(t *testing.T) {
	// Create test data
	data := SignatureData{
		TrackId:     1485,
		Cid:         "QmdGpDEBq6v6Kv9H61HbeVqyiPo7iBe12tVtkhNig6ipWp",
		Timestamp:   1681484247930,
		UserID:      50419,
		ShouldCache: 1,
	}

	// Generate a new private key for testing
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	// Generate the signature
	envelope, err := GenerateSignature(data, privateKey)
	assert.NoError(t, err)

	// Parse the signature back
	parsed, err := ParseFromQueryString(envelope)
	assert.NoError(t, err)

	// Verify the parsed data matches the original
	assert.Equal(t, data.TrackId, parsed.Data.TrackId)
	assert.Equal(t, data.Cid, parsed.Data.Cid)
	assert.Equal(t, data.Timestamp, parsed.Data.Timestamp)
	assert.Equal(t, data.UserID, parsed.Data.UserID)
	assert.Equal(t, data.ShouldCache, parsed.Data.ShouldCache)

	// Verify the signer's address matches the private key we used
	expectedAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	assert.Equal(t, expectedAddress.String(), parsed.SignerWallet)
}
