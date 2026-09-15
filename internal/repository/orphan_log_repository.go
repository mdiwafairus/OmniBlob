package repository

import (
	"context"
	"fmt"

	"pwni-file-sync/internal/entity"

	"github.com/jackc/pgx/v4/pgxpool"
)

type orphanLogRepository struct {
	db *pgxpool.Pool
}

func NewOrphanLogRepository(db *pgxpool.Pool) OrphanLogRepository {
	return &orphanLogRepository{db: db}
}

func (r *orphanLogRepository) Insert(ctx context.Context, log *entity.OrphanFileLog) error {
	const query = `
		INSERT INTO orphan_file_log (original_path, quarantine_path, size, action, reason)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query, log.OriginalPath, log.QuarantinePath, log.Size, log.Action, log.Reason)
	if err != nil {
		return fmt.Errorf("OrphanLogRepository.Insert: %w", err)
	}
	return nil
}
