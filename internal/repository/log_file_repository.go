package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"pwni-file-sync/internal/entity"

	"github.com/jackc/pgx/v4/pgxpool"
)

type logRepository struct {
	db *pgxpool.Pool
}

func NewLogRepository(db *pgxpool.Pool) LogRepository {
	return &logRepository{db: db}
}

func (r *logRepository) GetLastBinID(ctx context.Context, server string) (int64, error) {
	const query = `SELECT COALESCE(MAX(last_bin_id), 0) FROM log_file_rsync WHERE server = $1`

	var lastBinID int64
	row := r.db.QueryRow(ctx, query, server)
	if err := row.Scan(&lastBinID); err != nil {
		return 0, fmt.Errorf("GetLastBinID: %w", err)
	}

	return lastBinID, nil
}

func (r *logRepository) Insert(ctx context.Context, log entity.LogFileRsync) error {
	binExecutedJSON, err := json.Marshal(log.BinExecuted)
	if err != nil {
		return fmt.Errorf("marshal bin executed: %w", err)
	}

	const query = `INSERT INTO log_file_rsync (server, last_bin_id, bin_executed, error_logs) VALUES ($1, $2, $3, $4)`

	_, err = r.db.Exec(ctx, query, log.Server, log.LastBinID, binExecutedJSON, log.ErrorLogs)
	if err != nil {
		return fmt.Errorf("Insert: %w", err)
	}

	return nil
}
