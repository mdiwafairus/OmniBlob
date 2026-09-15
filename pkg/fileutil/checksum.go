package fileutil

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// SHA256FromBytes computes the SHA256 hex string of a byte slice.
func SHA256FromBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SHA256FromFile computes SHA256 hex string directly by streaming a file from disk.
func SHA256FromFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
