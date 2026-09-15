package main

import (
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

	// 2. Check if there's a valid explicit config key (dummy check for now, or you can use license pkg)
	if configKey != "" && configKey != "CHANGE_ME_SECRET_KEY" && len(configKey) > 10 {
		// In a real system, you would call license.LoadLicense and VerifySignature here.
		// For now, if they provide a key, we assume they are activated.
		fmt.Println("dYs? Enterprise License Key detected. Thank you for your purchase!")
		return nil
	}

	// 3. Handle 3-Month Free Trial
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
			return fmt.Errorf("trial file is corrupted. Please provide a valid license key.")
		}
	}

	// 4. Check expiration
	expirationDate := trial.FirstRun.AddDate(0, 3, 0) // 3 Months
	if now.After(expirationDate) {
		return fmt.Errorf("FATAL: Your 3-month free trial has expired (Expired on %s). Please provide a valid license_key in config.yaml to continue using OmniBlob.", expirationDate.Format("2006-01-02"))
	}

	daysLeft := int(expirationDate.Sub(now).Hours() / 24)
	fmt.Printf("[TRIAL ACTIVE] You have %d days left in your free trial.\n", daysLeft)

	return nil
}
