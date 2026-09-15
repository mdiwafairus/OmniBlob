package license

import (
	"encoding/json"
	"os"
	"time"
)

// Watermark stores the monotonic last seen time to prevent clock rewinding.
type Watermark struct {
	LastSeenTime time.Time `json:"last_seen_time"`
}

// UpdateWatermark writes the current time to the watermark file.
// It returns ErrClockTampered if the current time is behind the recorded last seen time.
func UpdateWatermark(filename string, currentTime time.Time) error {
	var wm Watermark
	data, err := os.ReadFile(filename)
	if err == nil {
		if err := json.Unmarshal(data, &wm); err == nil {
			if currentTime.Before(wm.LastSeenTime) {
				return ErrClockTampered
			}
		}
	}

	wm.LastSeenTime = currentTime
	newData, err := json.Marshal(wm)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, newData, 0644)
}

// GetWatermark reads the last seen time from the watermark file.
// Returns a zero time.Time if the file doesn't exist.
func GetWatermark(filename string) (time.Time, error) {
	var wm Watermark
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	if err := json.Unmarshal(data, &wm); err != nil {
		return time.Time{}, err
	}
	return wm.LastSeenTime, nil
}

// GetMonotonicTime returns a monotonic timestamp that is guaranteed to not go backwards.
// It checks against a global/local watermark file.
func GetMonotonicTime() time.Time {
	now := time.Now()
	// Ignore errors, if tampered, we still return the wall clock but it's recorded
	_ = UpdateWatermark("audit_watermark.json", now)
	last, _ := GetWatermark("audit_watermark.json")
	if last.After(now) {
		return last // Clock was rewound, return the monotonic peak
	}
	return now
}
