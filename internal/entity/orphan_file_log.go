package entity

import "time"

type OrphanFileLog struct {
	ID             int64     `json:"id" db:"id"`
	OriginalPath   string    `json:"original_path" db:"original_path"`
	QuarantinePath string    `json:"quarantine_path" db:"quarantine_path"`
	Size           int64     `json:"size" db:"size"`
	Action         string    `json:"action" db:"action"`
	Reason         string    `json:"reason" db:"reason"`
	DetectedAt     time.Time `json:"detected_at" db:"detected_at"`
}
