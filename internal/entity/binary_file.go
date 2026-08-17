package entity

import "time"

type BinaryFile struct {
	BinID       int64     `json:"bin_id" db:"bin_id"`
	ReferensiID string    `json:"referensi_id" db:"referensi_id"`
	Module      string    `json:"module" db:"module"`
	Directory   string    `json:"directory" db:"directory"`
	FileName    string    `json:"file_name" db:"file_name"`
	Path        string    `json:"path" db:"path"`
	Size        int64     `json:"size" db:"size"`
	MimeType    string    `json:"mime_type" db:"mime_type"`
	Checksum    string    `json:"checksum" db:"checksum"`
	Flag        string    `json:"flag" db:"flag"` // e.g. "1" = active, "0" = deleted, "M" = migrated
	CreateDate  time.Time `json:"create_date" db:"create_date"`
}
