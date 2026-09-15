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
	keyHex := flag.String("key", "", "Ed25519 Private Key in hex format")
	outPath := flag.String("out", "omni.license", "Output file path")
	flag.Parse()

	if *keyHex == "" {
		fmt.Println("Error: -key is required")
		os.Exit(1)
	}

	privBytes, err := hex.DecodeString(*keyHex)
	if err != nil {
		fmt.Printf("Error decoding private key: %v\n", err)
		os.Exit(1)
	}

	if len(privBytes) != ed25519.PrivateKeySize {
		fmt.Printf("Error: invalid private key size. Expected %d, got %d\n", ed25519.PrivateKeySize, len(privBytes))
		os.Exit(1)
	}

	privKey := ed25519.PrivateKey(privBytes)

	// In a real app, this payload would be populated from user input or a DB.
	now := time.Now().Truncate(time.Second)
	payload := license.Payload{
		KeyID:           "master-key-01",
		LicenseID:       "lic-001",
		CustomerID:      "cust-omni-01",
		DeploymentID:    "dep-cloud-01",
		IssuedAt:        now,
		NotBefore:       now,
		ExpiresAt:       now.AddDate(1, 0, 0), // 1 year validity
		GracePeriodDays: 14,
		Entitlements: license.Entitlements{
			Features: []string{"advanced_analytics", "sso"},
			Quotas: map[string]int{
				"max_users": 50,
			},
			Tier: "professional",
		},
	}

	lic, err := license.GenerateLicense(payload, privKey)
	if err != nil {
		fmt.Printf("Error generating license: %v\n", err)
		os.Exit(1)
	}

	if err := license.SaveLicense(lic, *outPath); err != nil {
		fmt.Printf("Error saving license: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("License successfully generated at %s\n", *outPath)
	
	// Just print public key for testing
	pubKey := privKey.Public().(ed25519.PublicKey)
	fmt.Printf("Public Key (Hex) to use for verification: %x\n", pubKey)
}
