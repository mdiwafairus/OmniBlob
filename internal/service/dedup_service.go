package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pwni-file-sync/internal/repository"
	"pwni-file-sync/pkg/fileutil"

	"github.com/rs/zerolog"
)

type DedupConfig struct {
	StorageRoot     string
	QuarantineRoot  string
	DryRun          bool
	GracePeriodDays int
	MaxMoveLimit    int
}

type DedupService struct {
	cfg        *DedupConfig
	binaryRepo repository.BinaryFileRepository
	logger     zerolog.Logger
}

func NewDedupService(cfg *DedupConfig, binaryRepo repository.BinaryFileRepository, logger zerolog.Logger) *DedupService {
	if cfg.GracePeriodDays <= 0 {
		cfg.GracePeriodDays = 1 // Default safety: 24 hours
	}
	if cfg.MaxMoveLimit <= 0 {
		cfg.MaxMoveLimit = 100 // Default safety limit
	}
	if cfg.QuarantineRoot == "" {
		cfg.QuarantineRoot = filepath.Join(cfg.StorageRoot, "_quarantine")
	}

	return &DedupService{
		cfg:        cfg,
		binaryRepo: binaryRepo,
		logger:     logger,
	}
}

// FindAndQuarantineOrphans scans the storage root, finds orphan files, and moves them to quarantine.
func (s *DedupService) FindAndQuarantineOrphans(ctx context.Context) error {
	s.logger.Info().
		Bool("dry_run", s.cfg.DryRun).
		Int("grace_period_days", s.cfg.GracePeriodDays).
		Int("max_move_limit", s.cfg.MaxMoveLimit).
		Str("storage_root", s.cfg.StorageRoot).
		Str("quarantine_root", s.cfg.QuarantineRoot).
		Msg("Starting Orphan File Detection (Option B: Move to Quarantine)...")

	if !s.cfg.DryRun {
		if err := fileutil.EnsureDir(s.cfg.QuarantineRoot); err != nil {
			return fmt.Errorf("failed to create quarantine directory: %w", err)
		}
	}

	var orphanFiles []string
	cutoffTime := time.Now().AddDate(0, 0, -s.cfg.GracePeriodDays)

	// Step 1: Scan files
	err := filepath.Walk(s.cfg.StorageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Warn().Err(err).Str("path", path).Msg("Error accessing path during scan")
			return nil
		}

		if info.IsDir() {
			// Skip quarantine folder to avoid infinite recursion
			if strings.HasPrefix(path, s.cfg.QuarantineRoot) {
				return filepath.SkipDir
			}
			return nil
		}

		// Guardrail 1: Grace Period (Skip recent files)
		if info.ModTime().After(cutoffTime) {
			return nil
		}

		// Check if it exists in the database
		relPath, err := filepath.Rel(s.cfg.StorageRoot, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		exists, err := s.binaryRepo.ExistsByPath(ctx, relPath)
		if err != nil {
			s.logger.Error().Err(err).Str("path", relPath).Msg("Database check failed")
			return nil
		}

		if !exists {
			orphanFiles = append(orphanFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("file scan failed: %w", err)
	}

	s.logger.Info().Int("orphan_count", len(orphanFiles)).Msg("Scan complete.")

	if len(orphanFiles) == 0 {
		return nil
	}

	// Guardrail 2: Max Move Limit / Circuit Breaker
	if len(orphanFiles) > s.cfg.MaxMoveLimit {
		s.logger.Warn().
			Int("orphan_count", len(orphanFiles)).
			Int("limit", s.cfg.MaxMoveLimit).
			Msg("Circuit Breaker Triggered: Orphan file count exceeds maximum allowed limit. Aborting quarantine process.")
		return fmt.Errorf("orphan count %d exceeds safety limit %d", len(orphanFiles), s.cfg.MaxMoveLimit)
	}

	// Step 2: Move to Quarantine (or just log if DryRun)
	movedCount := 0
	for _, orphanPath := range orphanFiles {
		relPath, _ := filepath.Rel(s.cfg.StorageRoot, orphanPath)
		targetPath := filepath.Join(s.cfg.QuarantineRoot, relPath)

		// Guardrail 3: Dry-Run Mode
		if s.cfg.DryRun {
			s.logger.Info().Str("file", relPath).Msg("[DRY RUN] Would move file to quarantine")
			continue
		}

		// Ensure target directory exists inside quarantine
		if err := fileutil.EnsureDir(filepath.Dir(targetPath)); err != nil {
			s.logger.Error().Err(err).Str("target", targetPath).Msg("Failed to create target quarantine directory")
			continue
		}

		// Option B: Move to quarantine
		if err := os.Rename(orphanPath, targetPath); err != nil {
			s.logger.Error().Err(err).Str("file", orphanPath).Msg("Failed to move file to quarantine")
			continue
		}

		movedCount++
		s.logger.Info().Str("file", relPath).Msg("Moved orphan file to quarantine")
	}

	if s.cfg.DryRun {
		s.logger.Info().Msg("Dry run completed. No files were actually moved.")
	} else {
		s.logger.Info().Int("moved_count", movedCount).Msg("Quarantine process completed successfully.")
	}

	return nil
}
