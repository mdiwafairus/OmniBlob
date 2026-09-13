package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"pwni-file-sync/internal/api"
	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/database"
	"pwni-file-sync/internal/logger"
	"pwni-file-sync/internal/repository"
	"pwni-file-sync/internal/service"
	"pwni-file-sync/internal/storage"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
)

func main() {
	configFlag := flag.String("config", "configs/config.yaml", "Path to configuration file")
	portFlag := flag.Int("port", 0, "Override server port")
	maxUploadFlag := flag.Int("max-upload", 0, "Override max upload size in MB")
	scanLegacy := flag.Bool("scan-legacy", false, "Scan legacy_path and automatically populate the database for migration")
	flag.Parse()

	log := logger.NewLogger()
	log.Info().Msg("Starting OmniBlob (pwni-file-sync) Object Storage & Sync Service...")

	// 1. Load Configuration
	configPath := *configFlag
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Str("config_path", configPath).Msg("Failed to load configuration file")
	}

	// Apply CLI overrides
	if *portFlag != 0 {
		cfg.Server.Port = *portFlag
	}
	if *maxUploadFlag != 0 {
		cfg.Server.MaxUploadSizeMB = *maxUploadFlag
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

	if *scanLegacy {
		runLegacyScanner(dbPool, cfg.Storage.LegacyPath, &log)
		return
	}

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
	dashboardRepo := repository.NewDashboardRepository(dbPool)

	// Context for graceful background jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5. Start Background File Migration Worker (reorganizing legacy NFS files locally)
	migrationService := service.NewMigrationService(&cfg.Migration, storageService, binaryRepo, logRepo, log)
	go migrationService.StartBackgroundMigration(ctx)

	// 5.5 Initialize Audit Logger
	auditLogger, err := logger.NewAuditLogger("logs")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Audit Logger")
	}
	defer auditLogger.Close()

	// 6. Initialize HTTP API Server
	handler := api.NewHandler(storageService, binaryRepo, log, cfg.Server.MaxUploadSizeMB)
	dashboardHandler := api.NewDashboardHandler(dashboardRepo, &cfg.Server, &cfg.Migration, &cfg.Storage, log)
	explorerHandler := api.NewExplorerHandler(&cfg.Storage, &cfg.Migration, log)
	router := api.NewRouter(handler, dashboardHandler, explorerHandler, &cfg.Server, &cfg.Security, auditLogger, log)
	httpServer := api.NewServer(&cfg.Server, &cfg.Security, auditLogger, router, log)

	// Run HTTP Server in a separate goroutine
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Fatal().Err(err).Msg("HTTP Server encountered a fatal error")
		}
	}()

	log.Info().
		Int("port", cfg.Server.Port).
		Msg("pwni-file-sync is fully running and ready to handle file requests from application servers.")

	fmt.Printf("\n========================================================\n")
	fmt.Printf("🚀 OmniBlob Storage Node & API Server is RUNNING\n")
	fmt.Printf("📊 Dashboard (if built) is accessible at: http://localhost:%d/\n", cfg.Server.Port)
	fmt.Printf("========================================================\n\n")

	// 7. Wait for OS termination signals (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("Received termination signal, shutting down services gracefully...")

	// Cancel background workers
	cancel()
	migrationService.Stop()

	// Graceful HTTP shutdown with 10s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP Server shutdown error")
	}

	log.Info().Msg("pwni-file-sync service stopped cleanly.")
}

func runLegacyScanner(dbPool *pgxpool.Pool, legacyPath string, log *zerolog.Logger) {
	log.Info().Str("legacy_path", legacyPath).Msg("Starting Auto-Discovery Legacy File Scanner...")
	
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		log.Error().Msg("Legacy path does not exist. Cannot scan.")
		return
	}

	ctx := context.Background()

	var count int
	err := filepath.WalkDir(legacyPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			relPath, err := filepath.Rel(legacyPath, path)
			if err != nil {
				return nil
			}

			// Clean path for database storage (use forward slashes universally)
			relPath = filepath.ToSlash(relPath)
			filename := filepath.Base(relPath)
			dir := filepath.Dir(relPath)
			if dir == "." {
				dir = ""
			}
			
			// Try to insert
			_, err = dbPool.Exec(ctx, 
				`INSERT INTO binary_file (referensi_id, module, directory, file_name, path, flag) 
				 VALUES ($1, 'legacy_scan', $2, $3, $4, '1')
				 ON CONFLICT DO NOTHING`,
				 "auto_"+relPath, dir, filename, relPath)
			
			if err != nil {
				var exists bool
				_ = dbPool.QueryRow(ctx, "SELECT true FROM binary_file WHERE path = $1 LIMIT 1", relPath).Scan(&exists)
				if !exists {
					_, err = dbPool.Exec(ctx, 
						`INSERT INTO binary_file (referensi_id, module, directory, file_name, path, flag) 
						 VALUES ($1, 'legacy_scan', $2, $3, $4, '1')`,
						 "auto_"+relPath, dir, filename, relPath)
					if err == nil {
						count++
					}
				}
			} else {
				count++
			}
		}
		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Scanner encountered an error")
	}

	log.Info().Int("total_files_queued", count).Msg("Auto-Discovery complete. You can now start OmniBlob normally to begin migration.")
}
