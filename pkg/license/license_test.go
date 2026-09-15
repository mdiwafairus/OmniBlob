package license

import (
	"crypto/ed25519"
	"testing"
	"time"
)

func TestLicenseGenerationAndVerification(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	now := time.Now().Truncate(time.Second) // JSON marshal/unmarshal handles timestamps slightly differently with nano precision usually, so truncate
	payload := Payload{
		KeyID:           "key-001",
		LicenseID:       "lic-123",
		CustomerID:      "cust-456",
		DeploymentID:    "dep-789",
		IssuedAt:        now,
		NotBefore:       now.Add(-1 * time.Hour),
		ExpiresAt:       now.Add(24 * time.Hour),
		GracePeriodDays: 7,
		Entitlements: Entitlements{
			Features: []string{"pro", "api"},
			Quotas: map[string]int{
				"users": 100,
			},
			Tier: "enterprise",
		},
	}

	lic, err := GenerateLicense(payload, priv)
	if err != nil {
		t.Fatalf("GenerateLicense failed: %v", err)
	}

	if err := lic.VerifySignature(pub); err != nil {
		t.Fatalf("VerifySignature failed: %v", err)
	}

	// Tamper the payload
	lic.Payload.Entitlements.Tier = "hacked"
	if err := lic.VerifySignature(pub); err == nil {
		t.Fatalf("VerifySignature should have failed for tampered payload")
	}
}

func TestLicenseStatus(t *testing.T) {
	now := time.Now()
	payload := Payload{
		NotBefore:       now.Add(-2 * time.Hour),
		ExpiresAt:       now.Add(2 * time.Hour),
		GracePeriodDays: 5,
	}
	lic := &License{Payload: payload}

	// Valid status
	if status := lic.CheckStatus(now, time.Time{}); status != StatusValid {
		t.Errorf("Expected StatusValid, got %v", status)
	}

	// Grace period status
	graceTime := now.Add(24 * time.Hour * 3) // +3 days, still within 5 days grace period
	if status := lic.CheckStatus(graceTime, time.Time{}); status != StatusGracePeriod {
		t.Errorf("Expected StatusGracePeriod, got %v", status)
	}

	// Expired status (past grace period)
	expiredTime := now.Add(24 * time.Hour * 6) // +6 days, past 5 days grace period
	if status := lic.CheckStatus(expiredTime, time.Time{}); status != StatusExpired {
		t.Errorf("Expected StatusExpired, got %v", status)
	}

	// Expired status (before NotBefore)
	earlyTime := now.Add(-3 * time.Hour) // Before NotBefore
	if status := lic.CheckStatus(earlyTime, time.Time{}); status != StatusExpired {
		t.Errorf("Expected StatusExpired for early check, got %v", status)
	}

	// Clock tampered status
	lastSeen := now.Add(1 * time.Hour)
	if status := lic.CheckStatus(now, lastSeen); status != StatusClockTampered {
		t.Errorf("Expected StatusClockTampered, got %v", status)
	}
}
