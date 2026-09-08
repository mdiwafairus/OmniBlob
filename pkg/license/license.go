package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/gibson042/canonicaljson-go"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrLicenseExpired   = errors.New("license expired")
	ErrLicenseNotActive = errors.New("license not yet active")
	ErrInvalidHardware  = errors.New("hardware fingerprint mismatch")
)

type LicenseStatus string

const (
	StatusValid       LicenseStatus = "VALID"
	StatusGracePeriod LicenseStatus = "GRACE_PERIOD"
	StatusExpired     LicenseStatus = "EXPIRED"
)

type Entitlements struct {
	Features []string       `json:"features"`
	Quotas   map[string]int `json:"quotas"`
	Tier     string         `json:"tier"`
}

type Payload struct {
	KeyID               string       `json:"key_id"`
	LicenseID           string       `json:"license_id"`
	CustomerID          string       `json:"customer_id"`
	DeploymentID        string       `json:"deployment_id"`
	HardwareFingerprint string       `json:"hardware_fingerprint"`
	IssuedAt            time.Time    `json:"issued_at"`
	NotBefore           time.Time    `json:"not_before"`
	ExpiresAt           time.Time    `json:"expires_at"`
	GracePeriodDays     int          `json:"grace_period_days"`
	Entitlements        Entitlements `json:"entitlements"`
}

type License struct {
	Payload   Payload `json:"payload"`
	Signature string  `json:"signature"`
}

func GenerateLicense(payload Payload, privateKey ed25519.PrivateKey) (*License, error) {
	canonicalData, err := canonicaljson.Marshal(payload)
	if err != nil {
		return nil, err
	}

	sig := ed25519.Sign(privateKey, canonicalData)
	encodedSig := base64.StdEncoding.EncodeToString(sig)

	return &License{
		Payload:   payload,
		Signature: encodedSig,
	}, nil
}

func (l *License) VerifySignature(publicKey ed25519.PublicKey) error {
	canonicalData, err := canonicaljson.Marshal(l.Payload)
	if err != nil {
		return err
	}

	sigBytes, err := base64.StdEncoding.DecodeString(l.Signature)
	if err != nil {
		return ErrInvalidSignature
	}

	if !ed25519.Verify(publicKey, canonicalData, sigBytes) {
		return ErrInvalidSignature
	}

	return nil
}

func (l *License) CheckStatus(currentTime time.Time) LicenseStatus {
	if currentTime.Before(l.Payload.NotBefore) {
		// Even if not active yet, let's treat it as expired or a separate state.
		// Standard status state machine: if before NotBefore, we'll say EXPIRED (STOP).
		return StatusExpired
	}

	if currentTime.After(l.Payload.ExpiresAt) {
		gracePeriodEnd := l.Payload.ExpiresAt.AddDate(0, 0, l.Payload.GracePeriodDays)
		if currentTime.After(gracePeriodEnd) {
			return StatusExpired
		}
		return StatusGracePeriod
	}

	return StatusValid
}

func SaveLicense(l *License, filename string) error {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func LoadLicense(filename string) (*License, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var l License
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	return &l, nil
}
