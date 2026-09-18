package service

import (
	"context"
	"time"

	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/repository"

	"github.com/rs/zerolog"
)

type ReconciliationService struct {
	cfg        *config.ReconciliationConfig
	rootPath   string
	binaryRepo repository.BinaryFileRepository
	logger     zerolog.Logger
	stopChan   chan struct{}
}

func NewReconciliationService(
	cfg *config.ReconciliationConfig,
	storageCfg *config.StorageConfig,
	binaryRepo repository.BinaryFileRepository,
	logger zerolog.Logger,
) *ReconciliationService {
	return &ReconciliationService{
		cfg:        cfg,
		rootPath:   storageCfg.RootPath,
		binaryRepo: binaryRepo,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

func (s *ReconciliationService) StartBackgroundReconciliation(ctx context.Context) {
	if !s.cfg.Enabled {
		s.logger.Info().Msg("Background reconciliation job is disabled in config")
		return
	}

	interval := time.Duration(s.cfg.IntervalSec) * time.Second
	if interval < 60*time.Second {
		interval = 60 * time.Second
	}

	s.logger.Info().
		Int("batch_size", s.cfg.BatchSize).
		Dur("interval", interval).
		Msg("Starting background reconciliation worker...")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.runReconciliationBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Stopping reconciliation worker...")
			return
		case <-s.stopChan:
			s.logger.Info().Msg("Reconciliation worker received stop signal")
			return
		case <-ticker.C:
			s.runReconciliationBatch(ctx)
		}
	}
}

func (s *ReconciliationService) Stop() {
	close(s.stopChan)
}

func (s *ReconciliationService) runReconciliationBatch(ctx context.Context) {
	batchSize := s.cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	// This is a placeholder for actual DB scanning logic.
	// Normally we would query DB for files with Flag == "M" 
	// and verify their existence and MD5 on disk.
	s.logger.Info().Msg("Reconciliation batch running (metadata vs physical checksum)")
}
