package logger

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pwni-file-sync/pkg/license"
)

type AuditEntry struct {
	Timestamp     string `json:"timestamp"` // Monotonic clock string
	RemoteAddr    string `json:"remote_addr"`
	XForwardedFor string `json:"x_forwarded_for,omitempty"`
	Action        string `json:"action"`
	Reason        string `json:"reason"`
	PreviousHash  string `json:"previous_hash"`
	CurrentHash   string `json:"current_hash"`
}

type AuditLogger struct {
	mu       sync.Mutex
	writer   *DailyWriter
	lastHash string
}

func getLatestAuditFile(logDir string) string {
	files, err := filepath.Glob(filepath.Join(logDir, "audit-*.log"))
	if err != nil || len(files) == 0 {
		return ""
	}
	// Glob returns sorted paths, so YYYY-MM-DD will sort naturally
	return files[len(files)-1]
}

func NewAuditLogger(logDir string) (*AuditLogger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create audit log dir: %v", err)
	}

	lastHash := "0000000000000000000000000000000000000000000000000000000000000000" // Genesis block hash
	latestFile := getLatestAuditFile(logDir)
	
	if latestFile != "" {
		file, err := os.Open(latestFile)
		if err == nil {
			scanner := bufio.NewScanner(file)
			var lastLine string
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" {
					lastLine = line
				}
			}
			file.Close()
			if lastLine != "" {
				var lastEntry AuditEntry
				if err := json.Unmarshal([]byte(lastLine), &lastEntry); err == nil && lastEntry.CurrentHash != "" {
					lastHash = lastEntry.CurrentHash
				}
			}
		}
	}

	return &AuditLogger{
		writer:   NewDailyWriter(logDir, "audit"),
		lastHash: lastHash,
	}, nil
}

func (a *AuditLogger) LogConnection(remoteAddr, xForwardedFor, action, reason string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry := AuditEntry{
		Timestamp:     license.GetMonotonicTime().Format(time.RFC3339Nano),
		RemoteAddr:    remoteAddr,
		XForwardedFor: xForwardedFor,
		Action:        action,
		Reason:        reason,
		PreviousHash:  a.lastHash,
	}

	// Compute hash: SHA256(JSON without CurrentHash + PreviousHash)
	rawJSON, _ := json.Marshal(entry) // Marshal without CurrentHash (since it's empty)
	hashData := string(rawJSON) + a.lastHash
	hash := sha256.Sum256([]byte(hashData))
	currentHashStr := hex.EncodeToString(hash[:])

	entry.CurrentHash = currentHashStr
	
	finalJSON, _ := json.Marshal(entry)
	
	_, err := a.writer.Write(append(finalJSON, '\n'))
	
	a.lastHash = currentHashStr
	return err
}

func (a *AuditLogger) Close() error {
	return a.writer.Close()
}
