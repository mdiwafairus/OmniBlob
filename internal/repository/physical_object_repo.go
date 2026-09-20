package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"pwni-file-sync/internal/entity"
)

type physicalObjectRepository struct {
	db *pgxpool.Pool
}

func NewPhysicalObjectRepository(db *pgxpool.Pool) PhysicalObjectRepository {
	return &physicalObjectRepository{db: db}
}

func (r *physicalObjectRepository) GetByHash(ctx context.Context, hash string) (*entity.PhysicalObject, error) {
	const query = `
		SELECT content_hash, size, storage_path, first_stored_at
		FROM physical_objects
		WHERE content_hash = $1
	`
	var obj entity.PhysicalObject
	err := r.db.QueryRow(ctx, query, hash).Scan(
		&obj.ContentHash, &obj.Size, &obj.StoragePath, &obj.FirstStoredAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Return nil if not found
		}
		return nil, fmt.Errorf("GetByHash: %w", err)
	}
	return &obj, nil
}

func (r *physicalObjectRepository) Insert(ctx context.Context, obj *entity.PhysicalObject) error {
	const query = `
		INSERT INTO physical_objects (content_hash, size, storage_path, first_stored_at)
		VALUES ($1, $2, $3, COALESCE($4, CURRENT_TIMESTAMP))
		ON CONFLICT (content_hash) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, obj.ContentHash, obj.Size, obj.StoragePath, obj.FirstStoredAt)
	if err != nil {
		return fmt.Errorf("Insert PhysicalObject: %w", err)
	}
	return nil
}
