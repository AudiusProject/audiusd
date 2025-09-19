package address

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Generator generates deterministic addresses for entities
type Generator struct {
	chainID string
	height  int64
	txHash  string
}

// New creates a new address generator
func New(chainID string, height int64, txHash string) *Generator {
	return &Generator{
		chainID: chainID,
		height:  height,
		txHash:  txHash,
	}
}

// Create generates a deterministic address from components
func (g *Generator) Create(components ...string) string {
	// Build deterministic string from stable components
	parts := []string{
		g.chainID,
		fmt.Sprintf("%d", g.height),
		g.txHash,
	}
	parts = append(parts, components...)

	data := strings.Join(parts, ":")
	hash := sha256.Sum256([]byte(data))

	// Return as ethereum-style address (0x + first 20 bytes of hash)
	return "0x" + hex.EncodeToString(hash[:20])
}

// ERN generates address for ERN using message ID
func (g *Generator) ERN(messageID string) string {
	return g.Create("ern", messageID)
}

// Party generates address for Party using party reference
func (g *Generator) Party(partyReference string) string {
	return g.Create("party", partyReference)
}

// Resource generates address for Resource using resource reference
func (g *Generator) Resource(resourceReference string) string {
	return g.Create("resource", resourceReference)
}

// Release generates address for Release using release reference
func (g *Generator) Release(releaseReference string) string {
	return g.Create("release", releaseReference)
}

// Deal generates address for Deal using deal reference
func (g *Generator) Deal(dealReference string) string {
	return g.Create("deal", dealReference)
}