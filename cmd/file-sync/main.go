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

	// 1.5 Enforce Licensing & Trial System
	if err := enforceLicensing(cfg.Server.ApiKey); err != nil {
		fmt.Printf("\n========================================================\n")
		fmt.Printf("%v\n", err)
		fmt.Printf("========================================================\n\n")
		log.Fatal().Msg("Licensing enforcement failed. Halting.")
	}

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
	// dashboardRepo := repository.NewDashboardRepository(dbPool)

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

	// Start Background Reconciliation Worker (checking DB checksum vs physical file)
	reconciliationService := service.NewReconciliationService(&cfg.Reconciliation, &cfg.Storage, binaryRepo, log)
	go reconciliationService.StartBackgroundReconciliation(ctx)

	// 6. Initialize HTTP API Server (MERGE CONFLICT RESOLVED)
	handler := api.NewHandler(storageService, binaryRepo, log, cfg.Server.MaxUploadSizeMB, cfg.Server.AllowedExtensions, cfg.Auth.AccessKey, cfg.Auth.SecretKey)
	dashboardService := service.NewDashboardService(binaryRepo, logRepo)
	
	var clientNames []string
	for _, client := range cfg.Server.Clients {
		clientNames = append(clientNames, client.Name)
	}
	dashboardHandler := api.NewDashboardHandler(dashboardService, &cfg.Storage, clientNames)
	explorerHandler := api.NewExplorerHandler(&cfg.Storage, &cfg.Migration, log)
	
	router := api.NewRouter(handler, dashboardHandler, explorerHandler, &cfg.Server, &cfg.Security, auditLogger, cfg.Auth.SecretKey, log, dbPool)
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

	protocol := "http"
	if cfg.Server.TLSEnabled {
		protocol = "https"
	}

	fmt.Printf("\n========================================================\n")
	fmt.Printf("🚀 OmniBlob Storage Node & API Server is RUNNING\n")
	fmt.Printf("📊 Dashboard (if built) is accessible at: %s://localhost:%d/\n", protocol, cfg.Server.Port)
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
			// Skip files/folders with permission errors instead of aborting the whole scan
			log.Warn().Err(err).Str("path", path).Msg("Skipping inaccessible path")
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			relPath, err := filepath.Rel(legacyPath, path)
			if err != nil {
				return nil
			}

			// Extract original ModTime to fix Dashboard year
			info, err := d.Info()
			modTime := time.Now()
			size := int64(0)
			if err == nil {
				modTime = info.ModTime()
				size = info.Size()
			}

			// Clean path for database storage (use forward slashes universally)
			relPath = filepath.ToSlash(relPath)
			filename := filepath.Base(relPath)
			dir := filepath.Dir(relPath)
			if dir == "." {
				dir = ""
			}
			
			// Check if file already exists in DB to prevent duplicates
			// (Since the table might not have a UNIQUE constraint on path)
			var exists bool
			_ = dbPool.QueryRow(ctx, "SELECT true FROM binary_file WHERE path = $1 AND module = 'legacy_scan' LIMIT 1", relPath).Scan(&exists)
			
			if !exists {
				// Try to insert WITH create_date (ModTime) and size
				_, err = dbPool.Exec(ctx, 
					`INSERT INTO binary_file (referensi_id, module, directory, file_name, path, flag, create_date, size) 
					 VALUES ($1, 'legacy_scan', $2, $3, $4, '1', $5, $6)`,
					 "auto_"+relPath, dir, filename, relPath, modTime, size)
				
				if err == nil {
					count++
				} else {
					log.Warn().Err(err).Str("file", filename).Msg("Database rejected file insertion")
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Scanner finished with some errors")
	}

	log.Info().Int("total_files_queued", count).Msg("Auto-Discovery complete. You can now start OmniBlob normally to begin migration.")
}