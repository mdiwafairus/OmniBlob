package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"

	"pwni-file-sync/internal/config"
)

func NewPostgres(cfg *config.Config, log *zerolog.Logger) (*pgxpool.Pool, error) {
	if log != nil {
		log.Info().
			Str("host", cfg.Database.Host).
			Int("port", cfg.Database.Port).
			Str("database", cfg.Database.Name).
			Str("user", cfg.Database.Username).
			Str("sslmode", cfg.Database.SSLMode).
			Msg("Attempting connection to PostgreSQL database...")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		if log != nil {
			log.Error().Err(err).Msg("Failed to parse PostgreSQL connection string")
		}
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	// Connection Pool Configuration
	poolConfig.MaxConns = int32(cfg.Database.MaxOpenConn)
	poolConfig.MinConns = int32(cfg.Database.MinOpenConn)
	if cfg.Database.MaxIdleTime > 0 {
		poolConfig.MaxConnIdleTime = time.Duration(cfg.Database.MaxIdleTime) * time.Minute
	}
	if cfg.Database.MaxLifeTime > 0 {
		poolConfig.MaxConnLifetime = time.Duration(cfg.Database.MaxLifeTime) * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		if log != nil {
			log.Error().Err(err).
				Str("host", cfg.Database.Host).
				Int("port", cfg.Database.Port).
				Str("database", cfg.Database.Name).
				Msg("Database connection failed (PostgreSQL might be offline or unreachable)")
		}
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	// Verify database is responsive with Ping
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		if log != nil {
			log.Error().Err(err).
				Str("host", cfg.Database.Host).
				Int("port", cfg.Database.Port).
				Str("database", cfg.Database.Name).
				Msg("Database ping failed (PostgreSQL is not responding)")
		}
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if log != nil {
		log.Info().
			Str("host", cfg.Database.Host).
			Int("port", cfg.Database.Port).
			Str("database", cfg.Database.Name).
			Int("max_conns", cfg.Database.MaxOpenConn).
			Int("min_conns", cfg.Database.MinOpenConn).
			Msg("Connected successfully to PostgreSQL database")
	}

	// Auto-migrate tables and columns
	if err := AutoMigrate(ctx, pool, log); err != nil {
		if log != nil {
			log.Warn().Err(err).Msg("Database auto-migration encountered warning")
		}
	}

	return pool, nil
}

func AutoMigrate(ctx context.Context, pool *pgxpool.Pool, log *zerolog.Logger) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS log_file_rsync (
			id BIGSERIAL PRIMARY KEY,
			server VARCHAR(100) NOT NULL,
			last_bin_id BIGINT DEFAULT 0,
			bin_executed JSONB,
			error_logs TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS binary_file (
			bin_id BIGSERIAL PRIMARY KEY,
			referensi_id VARCHAR(100),
			module VARCHAR(100) DEFAULT '',
			directory VARCHAR(255) DEFAULT '',
			file_name VARCHAR(255),
			path VARCHAR(500),
			size BIGINT DEFAULT 0,
			mime_type VARCHAR(100) DEFAULT '',
			checksum VARCHAR(64) DEFAULT '',
			flag VARCHAR(10) DEFAULT '1',
			create_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS module VARCHAR(100) DEFAULT '';`,
		`ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS directory VARCHAR(255) DEFAULT '';`,
		`ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS size BIGINT DEFAULT 0;`,
		`ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS mime_type VARCHAR(100) DEFAULT '';`,
		`ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS checksum VARCHAR(64) DEFAULT '';`,
		`ALTER TABLE binary_file ADD COLUMN IF NOT EXISTS flag VARCHAR(10) DEFAULT '1';`,
		`CREATE INDEX IF NOT EXISTS idx_binary_file_referensi_id ON binary_file (referensi_id);`,
		`CREATE INDEX IF NOT EXISTS idx_binary_file_module ON binary_file (module);`,
		`CREATE INDEX IF NOT EXISTS idx_log_file_rsync_server ON log_file_rsync (server);`,
	}

	for _, q := range queries {
		queryCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := pool.Exec(queryCtx, q)
		cancel()
		if err != nil {
			return fmt.Errorf("execute migration query: %w", err)
		}
	}

	if log != nil {
		log.Info().Msg("Database schema verified / auto-migrated successfully (tables & indexes ready)")
	}
	return nil
}
