package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pwni-file-sync/internal/api"
	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/database"
	"pwni-file-sync/internal/logger"
	"pwni-file-sync/internal/repository"
	"pwni-file-sync/internal/service"
	"pwni-file-sync/internal/storage"
)

func main() {
	log := logger.NewLogger()
	log.Info().Msg("Starting pwni-file-sync Object Storage & Sync Service...")

	// 1. Load Configuration
	configPath := "configs/config.yaml"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Str("config_path", configPath).Msg("Failed to load configuration file")
	}
	log.Info().Str("config_path", configPath).Msg("Configuration loaded successfully")

	// 2. Initialize Database Connection & Healthcheck
	dbPool, err := database.NewPostgres(cfg, &log)
	if err != nil {
		log.Fatal().Err(err).Msg("Database initialization failed. Please check your database connection or service status.")
	}
	defer func() {
		log.Info().Msg("Closing database connection pool...")
		dbPool.Close()
	}()

	// 3. Initialize Storage Engine
	storageService, err := storage.NewStorageService(&cfg.Storage)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Storage Engine")
	}
	log.Info().
		Str("root_path", cfg.Storage.RootPath).
		Str("legacy_path", cfg.Storage.LegacyPath).
		Msg("Storage Engine initialized successfully")

	// 4. Initialize Repositories
	binaryRepo := repository.NewBinaryFileRepository(dbPool)
	logRepo := repository.NewLogRepository(dbPool)

	// Context for graceful background jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5. Start Background File Migration Worker (reorganizing legacy NFS files locally)
	migrationService := service.NewMigrationService(&cfg.Migration, storageService, binaryRepo, logRepo, log)
	go migrationService.StartBackgroundMigration(ctx)

	// Start Background Reconciliation Worker (checking DB checksum vs physical file)
	reconciliationService := service.NewReconciliationService(&cfg.Reconciliation, &cfg.Storage, binaryRepo, log)
	go reconciliationService.StartBackgroundReconciliation(ctx)

	// 6. Initialize HTTP API Server
	handler := api.NewHandler(storageService, binaryRepo, log, cfg.Server.MaxUploadSizeMB, cfg.Server.AllowedExtensions)
	router := api.NewRouter(handler, &cfg.Server, log, dbPool)
	httpServer := api.NewServer(&cfg.Server, router, log)

	// Run HTTP Server in a separate goroutine
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Fatal().Err(err).Msg("HTTP Server encountered a fatal error")
		}
	}()

	log.Info().
		Int("port", cfg.Server.Port).
		Msg("pwni-file-sync is fully running and ready to handle file requests from application servers.")

	// 7. Wait for OS termination signals (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("Received termination signal, shutting down services gracefully...")

	// Cancel background workers
	cancel()
	migrationService.Stop()
	reconciliationService.Stop()

	// Graceful HTTP shutdown with 10s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP Server shutdown error")
	}

	log.Info().Msg("pwni-file-sync service stopped cleanly.")
}
