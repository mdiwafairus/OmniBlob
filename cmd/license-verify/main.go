package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	"pwni-file-sync/pkg/license"
)

func main() {
	pubHex := flag.String("pub", "", "Ed25519 Public Key in hex format")
	licPath := flag.String("lic", "omni.license", "Path to license file")
	flag.Parse()

	if *pubHex == "" {
		fmt.Println("Error: -pub is required")
		os.Exit(1)
	}

	pubBytes, err := hex.DecodeString(*pubHex)
	if err != nil {
		fmt.Printf("Error decoding public key: %v\n", err)
		os.Exit(1)
	}

	if len(pubBytes) != ed25519.PublicKeySize {
		fmt.Printf("Error: invalid public key size. Expected %d, got %d\n", ed25519.PublicKeySize, len(pubBytes))
		os.Exit(1)
	}

	pubKey := ed25519.PublicKey(pubBytes)

	// Load license
	lic, err := license.LoadLicense(*licPath)
	if err != nil {
		fmt.Printf("Error loading license: %v\n", err)
		os.Exit(1)
	}

	// Verify signature
	if err := lic.VerifySignature(pubKey); err != nil {
		fmt.Printf("Signature validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Signature is VALID.")

	// Check status
	status := lic.CheckStatus(time.Now())
	fmt.Printf("License Status: %s\n", status)

	switch status {
	case license.StatusValid:
		fmt.Println("System State: RUN - License is active and valid.")
	case license.StatusGracePeriod:
		fmt.Println("System State: WARNING - License is expired but within grace period.")
	case license.StatusExpired:
		fmt.Println("System State: STOP - License is expired or not active.")
	}

	fmt.Printf("\nLicense Details:\n")
	fmt.Printf("- Customer: %s\n", lic.Payload.CustomerID)
	fmt.Printf("- Tier: %s\n", lic.Payload.Entitlements.Tier)
	fmt.Printf("- Features: %v\n", lic.Payload.Entitlements.Features)
	fmt.Printf("- Quotas: %v\n", lic.Payload.Entitlements.Quotas)
}
