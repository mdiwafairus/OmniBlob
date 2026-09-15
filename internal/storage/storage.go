package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"pwni-file-sync/internal/config"
	"pwni-file-sync/pkg/fileutil"
)

var safeFilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

type StorageService struct {
	rootPath     string
	legacyPath   string
	shardingType string
}

func NewStorageService(cfg *config.StorageConfig) (*StorageService, error) {
	// Ensure root storage directory exists
	if err := fileutil.EnsureDir(cfg.RootPath); err != nil {
		return nil, fmt.Errorf("init storage root: %w", err)
	}

	return &StorageService{
		rootPath:     cfg.RootPath,
		legacyPath:   cfg.LegacyPath,
		shardingType: cfg.ShardingType,
	}, nil
}

// GenerateStoragePath creates path following production format:
// 1. Explicit directory: <bucket>/<directory>/<filename>
// 2. With referensi_id/UUID: <bucket>/<module>/<referensi_id>/<filename>
// 3. Direct in module: <bucket>/<module>/<filename>
func (s *StorageService) GenerateStoragePath(bucket string, module string, directory string, referensiID string, filename string) string {
	if bucket == "" || bucket == "files" {
		bucket = "uploads"
	}
	if module == "" {
		module = "general"
	}
	module = strings.ToLower(strings.TrimSpace(module))

	cleanName := safeFilenameRegex.ReplaceAllString(filename, "_")
	if cleanName == "" {
		cleanName = "file"
	}

	// 1. Explicit directory provided from client (e.g. "lapor_diri/uuid" or "<bucket>/lapor_diri/uuid" or "pelayanan")
	if directory != "" {
		cleanDir := strings.Trim(filepath.ToSlash(directory), "/")
		cleanDir = strings.TrimPrefix(cleanDir, "uploads/")
		return filepath.ToSlash(filepath.Join(bucket, cleanDir, cleanName))
	}

	// 2. Subfolder by UUID / Referensi ID (e.g. <bucket>/lapor_diri/<uuid>/<filename>)
	if referensiID != "" {
		cleanRef := safeFilenameRegex.ReplaceAllString(referensiID, "_")
		return filepath.ToSlash(filepath.Join(bucket, module, cleanRef, cleanName))
	}

	// 3. Direct in module folder (e.g. <bucket>/lapor_diri/<filename>)
	return filepath.ToSlash(filepath.Join(bucket, module, cleanName))
}

// GenerateShardedPath creates a subpath like: "<bucket>/lapordiri/2026/08/123_document.pdf"
func (s *StorageService) GenerateShardedPath(bucket string, module string, directory string, binID int64, filename string, createDate time.Time) string {
	if bucket == "" || bucket == "files" {
		bucket = "uploads"
	}

	if module == "" {
		module = "general"
	}
	module = strings.ToLower(strings.TrimSpace(module))

	if createDate.IsZero() {
		createDate = time.Now()
	}

	cleanName := safeFilenameRegex.ReplaceAllString(filename, "_")
	if cleanName == "" {
		cleanName = "file"
	}

	var prefixedFilename string
	if binID > 0 {
		prefixedFilename = fmt.Sprintf("%d_%s", binID, cleanName)
	} else {
		prefixedFilename = fmt.Sprintf("%d_%s", time.Now().UnixNano(), cleanName)
	}

	year := createDate.Format("2006")
	month := createDate.Format("01")

	// If this is a legacy migration and it has a directory structure, preserve it!
	if module == "legacy_scan" && directory != "" {
		// Example output: uploads/PWNI/subfolder/2026/08/123_file.txt
		cleanDir := filepath.ToSlash(directory)
		return filepath.ToSlash(filepath.Join(bucket, cleanDir, year, month, prefixedFilename))
	}

	return filepath.ToSlash(filepath.Join(bucket, module, year, month, prefixedFilename))
}

// Save streams data from reader to destination file and computes SHA256 checksum.
func (s *StorageService) Save(ctx context.Context, bucket string, customPath string, module string, directory string, referensiID string, binID int64, filename string, r io.Reader) (string, string, int64, error) {
	var relPath string
	if customPath != "" {
		relPath = filepath.ToSlash(strings.TrimPrefix(customPath, "/"))
	} else if directory == "date" {
		relPath = s.GenerateShardedPath(bucket, module, directory, binID, filename, time.Now())
	} else {
		relPath = s.GenerateStoragePath(bucket, module, directory, referensiID, filename)
	}

	absPath := filepath.Join(s.rootPath, filepath.FromSlash(relPath))

	if err := fileutil.EnsureDir(filepath.Dir(absPath)); err != nil {
		return "", "", 0, fmt.Errorf("ensure dir: %w", err)
	}

	file, err := os.Create(absPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	multiWriter := io.MultiWriter(file, hash)

	size, err := io.Copy(multiWriter, r)
	if err != nil {
		_ = os.Remove(absPath)
		return "", "", 0, fmt.Errorf("stream copy file: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	return relPath, checksum, size, nil
}

// Open opens a file, first checking the new sharded rootPath, then falling back to legacyPath.
func (s *StorageService) Open(ctx context.Context, relOrFullPath string) (*os.File, os.FileInfo, string, error) {
	cleanRel := filepath.FromSlash(strings.TrimPrefix(relOrFullPath, "/"))

	// 1. Try New Sharded Root Path
	newAbsPath := filepath.Join(s.rootPath, cleanRel)
	if fileutil.Exists(newAbsPath) {
		f, err := os.Open(newAbsPath)
		if err == nil {
			info, errStat := f.Stat()
			if errStat == nil && !info.IsDir() {
				return f, info, newAbsPath, nil
			}
			f.Close()
		}
	}

	// 2. Try Direct Absolute/Relative Path
	if fileutil.Exists(relOrFullPath) {
		f, err := os.Open(relOrFullPath)
		if err == nil {
			info, errStat := f.Stat()
			if errStat == nil && !info.IsDir() {
				return f, info, relOrFullPath, nil
			}
			f.Close()
		}
	}

	// 3. Fallback to Legacy Path
	if s.legacyPath != "" {
		legacyAbsPath := filepath.Join(s.legacyPath, cleanRel)
		if fileutil.Exists(legacyAbsPath) {
			f, err := os.Open(legacyAbsPath)
			if err == nil {
				info, errStat := f.Stat()
				if errStat == nil && !info.IsDir() {
					return f, info, legacyAbsPath, nil
				}
				f.Close()
			}
		}

		// Also check basename in legacy root if relPath had subdirectories
		baseLegacy := filepath.Join(s.legacyPath, filepath.Base(cleanRel))
		if fileutil.Exists(baseLegacy) {
			f, err := os.Open(baseLegacy)
			if err == nil {
				info, errStat := f.Stat()
				if errStat == nil && !info.IsDir() {
					return f, info, baseLegacy, nil
				}
				f.Close()
			}
		}
	}

	return nil, nil, "", os.ErrNotExist
}

// MigrateLegacyFile copies or moves a file from legacy storage to the new sharded path.
func (s *StorageService) MigrateLegacyFile(ctx context.Context, bucket string, legacyRelPath string, module string, directory string, binID int64, filename string, createDate time.Time) (string, string, int64, error) {
	srcFile, info, srcPath, err := s.Open(ctx, legacyRelPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("find legacy file %s: %w", legacyRelPath, err)
	}
	defer srcFile.Close()

	// Use original modified time from legacy file instead of current time
	originalModTime := info.ModTime()

	destRelPath := s.GenerateShardedPath(bucket, module, directory, binID, filename, originalModTime)
	destAbsPath := filepath.Join(s.rootPath, filepath.FromSlash(destRelPath))

	// If source is already at destination, just compute SHA256 and return
	if srcPath == destAbsPath {
		checksum, _ := fileutil.SHA256FromFile(destAbsPath)
		return destRelPath, checksum, info.Size(), nil
	}

	if err := fileutil.EnsureDir(filepath.Dir(destAbsPath)); err != nil {
		return "", "", 0, err
	}

	destFile, err := os.Create(destAbsPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("create dest: %w", err)
	}
	defer destFile.Close()

	hash := sha256.New()
	multiWriter := io.MultiWriter(destFile, hash)

	size, err := io.Copy(multiWriter, srcFile)
	if err != nil {
		_ = os.Remove(destAbsPath)
		return "", "", 0, fmt.Errorf("copy legacy file: %w", err)
	}
	
	// Preserve original modified timestamp!
	_ = os.Chtimes(destAbsPath, originalModTime, originalModTime)

	checksum := hex.EncodeToString(hash.Sum(nil))
	return destRelPath, checksum, size, nil
}

func (s *StorageService) Delete(ctx context.Context, relOrFullPath string) error {
	cleanRel := filepath.FromSlash(strings.TrimPrefix(relOrFullPath, "/"))
	var actualPath string
	if filepath.IsAbs(cleanRel) {
		actualPath = cleanRel
	} else {
		actualPath = filepath.Join(s.rootPath, cleanRel)
	}
	return os.Remove(actualPath)
}

