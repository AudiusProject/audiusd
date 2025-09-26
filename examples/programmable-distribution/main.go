package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"net/http"

	"github.com/AudiusProject/audiusd/pkg/sdk"
	"github.com/ethereum/go-ethereum/crypto"
)

type GeolocationHandler struct {
	privateKey        *ecdsa.PrivateKey
	allowedCity       string
	auds              *sdk.AudiusdSDK
	ernAddress        string
	resourceAddresses []string
	releaseAddresses  []string
}

func NewGeolocationHandler(privateKey *ecdsa.PrivateKey, allowedCity string, auds *sdk.AudiusdSDK) *GeolocationHandler {
	return &GeolocationHandler{
		privateKey:        privateKey,
		allowedCity:       allowedCity,
		auds:              auds,
		ernAddress:        "",
		resourceAddresses: []string{},
		releaseAddresses:  []string{},
	}
}

func main() {
	ctx := context.Background()

	fmt.Println("🎵 Programmable Distribution Demo")
	fmt.Println("================================")

	// Generate a demo private key for the filtering service
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	auds := sdk.NewAudiusdSDK("validator.audius.co")
	auds.Init(ctx)
	auds.SetPrivKey(privateKey)

	// Create geolocation-based filtering handler
	// This simulates the Cloudflare Worker logic
	handler := NewGeolocationHandler(privateKey, "Bozeman", auds)

	// Upload a demo track in the background
	go func() {
		fmt.Println("📀 Uploading demo track with programmable distribution...")
		if err := uploadTrackExample(ctx, auds, handler); err != nil {
			fmt.Printf("⚠️  Track upload simulation failed (expected in demo): %v\n", err)
			fmt.Println("In a real implementation, this would:")
			fmt.Println("  1. Upload the track to mediorum")
			fmt.Println("  2. Create an ERN with the track CID")
			fmt.Println("  3. Associate the filtering service URL with the track")
		}
	}()

	fmt.Println("\n🔐 Programmable Distribution Concept:")
	fmt.Println("  • Track owner deploys filtering logic (like the HTTP service above)")
	fmt.Println("  • When users request stream access, they must pass the owner's filters")
	fmt.Println("  • Owner can implement any logic: geolocation, payment, time-based, etc.")
	fmt.Println("  • Only approved requests get valid streaming signatures")

	// Start HTTP server for the filtering service (main thread)
	mux := http.NewServeMux()
	mux.Handle("/stream-access", handler)

	fmt.Println("\n🌐 Starting geolocation filtering service on :8080")
	fmt.Println("Test URLs:")
	fmt.Println("  ✅ Allowed:  curl 'http://localhost:8080/stream-access?city=Bozeman'")
	fmt.Println("  ❌ Blocked:  curl 'http://localhost:8080/stream-access?city=Seattle'")
	fmt.Println("\n⏳ Server running... Press Ctrl+C to exit")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
