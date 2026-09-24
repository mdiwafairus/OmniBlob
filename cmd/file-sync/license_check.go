package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"pwni-file-sync/pkg/license"
)

type TrialStatus struct {
	FirstRun time.Time `json:"first_run"`
}

func enforceLicensing(configKey string) error {
	now := time.Now()
	
	// 1. Prevent clock tampering using the existing watermark system
	err := license.UpdateWatermark("omni_watermark.json", now)
	if err == license.ErrClockTampered {
		return fmt.Errorf("FATAL: System clock tampering detected. The clock was rolled back. Execution halted.")
	} else if err != nil {
		// Ignore first run error or permission error, just proceed
	}

	// 2. Check for cryptographically signed Enterprise License (omni.license)
	licFile := "omni.license"
	if _, err := os.Stat(licFile); err == nil {
		// Public Key (Hardcoded for OmniBlob Enterprise)
		pubHex := "da592e95e339a014a66ca5a2b5c82d06e2e9902bfa286bcbf654e7a3cf9402c8"
		pubBytes, _ := hex.DecodeString(pubHex)
		
		lic, err := license.LoadLicense(licFile)
		if err != nil {
			return fmt.Errorf("FATAL: Failed to read %s: %v", licFile, err)
		}
		
		if err := lic.VerifySignature(pubBytes); err != nil {
			return fmt.Errorf("FATAL: License signature invalid or tampered! (%v)", err)
		}
		
		lastSeen, _ := license.GetWatermark("omni_watermark.json")
		status := lic.CheckStatus(now, lastSeen)
		if status == license.StatusExpired {
			return fmt.Errorf("FATAL: Enterprise License has EXPIRED!")
		}
		
		if status == license.StatusGracePeriod {
			fmt.Printf("dY~? WARNING: License expired, but you are in grace period.\n")
		} else {
			fmt.Printf("dYs? Enterprise License (%s) Valid & Active. Thank you!\n", lic.Payload.Entitlements.Tier)
		}
		return nil
	}

	// 3. Handle 3-Month Free Trial if no license file exists
	trialFile := "omni_trial.json"
	var trial TrialStatus
	
	data, err := os.ReadFile(trialFile)
	if err != nil {
		if os.IsNotExist(err) {
			// First run ever!
			trial.FirstRun = now
			b, _ := json.Marshal(trial)
			os.WriteFile(trialFile, b, 0644)
			fmt.Println("dY~? First run detected. Your 3-month free trial starts now!")
		} else {
			return fmt.Errorf("failed to read trial status: %v", err)
		}
	} else {
		if err := json.Unmarshal(data, &trial); err != nil {
			return fmt.Errorf("trial file is corrupted. Please provide a valid omni.license.")
		}
	}

	// 4. Check expiration
	expirationDate := trial.FirstRun.AddDate(0, 3, 0) // 3 Months
	if now.After(expirationDate) {
		return fmt.Errorf("FATAL: Your 3-month free trial has expired (Expired on %s). Please purchase a valid omni.license to continue.", expirationDate.Format("2006-01-02"))
	}

	daysLeft := int(expirationDate.Sub(now).Hours() / 24)
	fmt.Printf("[TRIAL ACTIVE] You have %d days left in your free trial.\n", daysLeft)

	return nil
}
