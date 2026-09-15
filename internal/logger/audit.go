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
	file     *os.File
	lastHash string
}

func NewAuditLogger(logDir string) (*AuditLogger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create audit log dir: %v", err)
	}

	logPath := filepath.Join(logDir, "audit.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open audit log file: %v", err)
	}

	// Read last line to get previous hash
	lastHash := "0000000000000000000000000000000000000000000000000000000000000000" // Genesis block hash
	scanner := bufio.NewScanner(file)
	var lastLine string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lastLine = line
		}
	}
	if lastLine != "" {
		var lastEntry AuditEntry
		if err := json.Unmarshal([]byte(lastLine), &lastEntry); err == nil && lastEntry.CurrentHash != "" {
			lastHash = lastEntry.CurrentHash
		}
	}

	return &AuditLogger{
		file:     file,
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
	
	if _, err := a.file.WriteString(string(finalJSON) + "\n"); err != nil {
		return err
	}
	// Sync immediately to ensure audit integrity
	_ = a.file.Sync()

	a.lastHash = currentHashStr
	return nil
}

func (a *AuditLogger) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}
