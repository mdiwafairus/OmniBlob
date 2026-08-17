package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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

	var executedIDs []int64
	var highestBinID int64 = lastBinID
	var errorCount int

	for _, file := range pendingFiles {
		if file.BinID > highestBinID {
			highestBinID = file.BinID
		}

		// If file is already marked migrated and exists in new sharded format, skip
		if file.Flag == "M" && strings.Contains(file.Path, "/") {
			executedIDs = append(executedIDs, file.BinID)
			continue
		}

		// Try resolving legacy source path
		legacyPath := file.Path
		if legacyPath == "" {
			legacyPath = filepath.Join(file.Directory, file.FileName)
			if file.Directory == "" {
				legacyPath = filepath.Join(file.Module, file.FileName)
			}
		}

		// Migrate local file to sharded layout
		newRelPath, checksum, size, err := s.storageRepo.MigrateLegacyFile(
			ctx, legacyPath, file.Module, file.BinID, file.FileName, file.CreateDate,
		)

		if err != nil {
			// Physical file might not exist yet on disk
			errorCount++
			continue
		}

		// Update database record with new sharded path and checksum
		if err := s.binaryRepo.UpdatePathAndStatus(ctx, file.BinID, newRelPath, checksum, size, "M"); err != nil {
			s.logger.Error().Err(err).Int64("bin_id", file.BinID).Msg("Failed to update migrated path in database")
			errorCount++
			continue
		}

		executedIDs = append(executedIDs, file.BinID)
	}

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
		Msg("Batch migration completed successfully")
}
