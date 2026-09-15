package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/entity"
	"pwni-file-sync/internal/repository"
	"pwni-file-sync/internal/storage"

	"github.com/rs/zerolog"
)

type MigrationService struct {
	cfg         *config.MigrationConfig
	storageRepo *storage.StorageService
	binaryRepo  repository.BinaryFileRepository
	logRepo     repository.LogRepository
	logger      zerolog.Logger
	stopChan    chan struct{}
}

func NewMigrationService(
	cfg *config.MigrationConfig,
	storageRepo *storage.StorageService,
	binaryRepo repository.BinaryFileRepository,
	logRepo repository.LogRepository,
	logger zerolog.Logger,
) *MigrationService {
	return &MigrationService{
		cfg:         cfg,
		storageRepo: storageRepo,
		binaryRepo:  binaryRepo,
		logRepo:     logRepo,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

// StartBackgroundMigration runs the continuous migration loop in the background.
func (s *MigrationService) StartBackgroundMigration(ctx context.Context) {
	if !s.cfg.Enabled {
		s.logger.Info().Msg("Background file migration is disabled in config")
		return
	}

	interval := time.Duration(s.cfg.IntervalSec) * time.Second
	if interval < 2*time.Second {
		interval = 10 * time.Second
	}

	s.logger.Info().
		Int("batch_size", s.cfg.BatchSize).
		Dur("interval", interval).
		Msg("Starting background file migration worker on storage server...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial run
	s.runMigrationBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Stopping background migration worker...")
			return
		case <-s.stopChan:
			s.logger.Info().Msg("Migration worker received stop signal")
			return
		case <-ticker.C:
			s.runMigrationBatch(ctx)
		}
	}
}

func (s *MigrationService) Stop() {
	close(s.stopChan)
}

func (s *MigrationService) runMigrationBatch(ctx context.Context) {
	serverName := "local-storage-134"

	// 1. Get last checkpoint ID
	lastBinID, err := s.logRepo.GetLastBinID(ctx, serverName)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to retrieve last_bin_id checkpoint, defaulting to 0")
		lastBinID = 0
	}

	// 2. Fetch next batch of pending files
	batchSize := s.cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	pendingFiles, err := s.binaryRepo.GetPendingFiles(ctx, lastBinID, batchSize)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to query pending files from database")
		return
	}

	if len(pendingFiles) == 0 {
		return // No new files to migrate
	}

	s.logger.Info().
		Int("count", len(pendingFiles)).
		Int64("from_bin_id", lastBinID).
		Msg("Processing file migration batch to sharded storage...")

	workers := s.cfg.WorkerCount
	if workers <= 0 {
		workers = 2
	}
	if workers > 8 {
		workers = 8 // Safety cap agar tidak membebani I/O disk
	}

	var mu sync.Mutex
	var executedIDs []int64
	var highestBinID int64 = lastBinID
	var errorCount int

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, file := range pendingFiles {
		if file.BinID > highestBinID {
			highestBinID = file.BinID
		}

		// If file is already marked migrated and exists in new sharded format, skip
		if file.Flag == "M" && strings.Contains(file.Path, "/") {
			mu.Lock()
			executedIDs = append(executedIDs, file.BinID)
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire worker slot

		go func(f entity.BinaryFile) {
			defer wg.Done()
			defer func() { <-sem }() // Release worker slot

			// Try resolving legacy source path
			legacyPath := f.Path
			if legacyPath == "" {
				legacyPath = filepath.Join(f.Directory, f.FileName)
				if f.Directory == "" {
					legacyPath = filepath.Join(f.Module, f.FileName)
				}
			}

			// Migrate local file to sharded layout
			// RESOLUSI: Menggunakan 8 parameter ("uploads" sebagai bucket dan f.Directory)
			newRelPath, checksum, size, err := s.storageRepo.MigrateLegacyFile(
				ctx, "uploads", legacyPath, f.Module, f.Directory, f.BinID, f.FileName, f.CreateDate,
			)

			if err != nil {
				s.logger.Error().Err(err).Int64("bin_id", f.BinID).Str("path", legacyPath).Msg("Failed to migrate file (Skipping)")
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}

			// Update database record with new sharded path and checksum
			if err := s.binaryRepo.UpdatePathAndStatus(ctx, f.BinID, newRelPath, checksum, size, "M"); err != nil {
				s.logger.Error().Err(err).Int64("bin_id", f.BinID).Msg("Failed to update migrated path in database")
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}

			// Add metadata sidecar for data recovery (Orphaned Data Prevention)
			f.Path = newRelPath
			f.Checksum = checksum
			f.Size = size
			f.Flag = "M"
			destAbsPath := filepath.Join(s.storageRepo.RootPath(), filepath.FromSlash(newRelPath))
			_ = s.storageRepo.SaveMetadataSidecar(destAbsPath, f)

			mu.Lock()
			executedIDs = append(executedIDs, f.BinID)
			mu.Unlock()
		}(file)
	}

	// Wait for current batch workers to finish
	wg.Wait()

	// 3. Record checkpoint to log_file_rsync
	if len(executedIDs) > 0 || highestBinID > lastBinID {
		logEntry := entity.LogFileRsync{
			Server:      serverName,
			LastBinID:   highestBinID,
			BinExecuted: executedIDs,
			ErrorLogs:   fmt.Sprintf("Batch completed. Migrated: %d, Errors/Skipped: %d", len(executedIDs), errorCount),
		}
		_ = s.logRepo.Insert(ctx, logEntry)
	}

	s.logger.Info().
		Int("migrated", len(executedIDs)).
		Int("errors", errorCount).
		Int64("checkpoint_bin_id", highestBinID).
		Int("active_workers", workers).
		Msg("Batch migration completed successfully")
}