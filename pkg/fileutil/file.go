package fileutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Exists checks if a file or directory exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// EnsureDir creates the directory and any parent directories if they don't exist.
func EnsureDir(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("ensure dir %s: %w", dirPath, err)
	}
	return nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src file: %w", err)
	}
	defer sourceFile.Close()

	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}

	return destFile.Sync()
}

// MoveFile attempts to rename (move) a file, falling back to copy+delete if across volumes.
func MoveFile(src, dst string) error {
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Fallback to copy and delete if Rename fails (e.g. cross-device link)
	if err := CopyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}
