package repository

import (
	"context"
	"fmt"
	"time"

	"pwni-file-sync/internal/entity"

	"github.com/jackc/pgx/v4/pgxpool"
)

type binaryFileRepository struct {
	db *pgxpool.Pool
}

func NewBinaryFileRepository(db *pgxpool.Pool) BinaryFileRepository {
	return &binaryFileRepository{db: db}
}

func (r *binaryFileRepository) GetByID(ctx context.Context, binID int64) (*entity.BinaryFile, error) {
	const query = `
		SELECT bin_id, 
		       COALESCE(referensi_id, ''), 
		       COALESCE(module, ''), 
		       COALESCE(directory, ''), 
		       COALESCE(file_name, ''), 
		       COALESCE(path, ''), 
		       COALESCE(size, 0), 
		       COALESCE(mime_type, ''), 
		       COALESCE(checksum, ''), 
		       COALESCE(flag, '1'), 
		       COALESCE(create_date, NOW())
		FROM binary_file
		WHERE bin_id = $1
		LIMIT 1
	`
	var f entity.BinaryFile
	row := r.db.QueryRow(ctx, query, binID)
	err := row.Scan(
		&f.BinID, &f.ReferensiID, &f.Module, &f.Directory,
		&f.FileName, &f.Path, &f.Size, &f.MimeType,
		&f.Checksum, &f.Flag, &f.CreateDate,
	)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	return &f, nil
}

func (r *binaryFileRepository) GetByReferensiID(ctx context.Context, referensiID string, module string) ([]entity.BinaryFile, error) {
	query := `
		SELECT bin_id, 
		       COALESCE(referensi_id, ''), 
		       COALESCE(module, ''), 
		       COALESCE(directory, ''), 
		       COALESCE(file_name, ''), 
		       COALESCE(path, ''), 
		       COALESCE(size, 0), 
		       COALESCE(mime_type, ''), 
		       COALESCE(checksum, ''), 
		       COALESCE(flag, '1'), 
		       COALESCE(create_date, NOW())
		FROM binary_file
		WHERE referensi_id = $1 AND ($2 = '' OR module = $2)
		ORDER BY bin_id DESC
	`
	rows, err := r.db.Query(ctx, query, referensiID, module)
	if err != nil {
		return nil, fmt.Errorf("GetByReferensiID: %w", err)
	}
	defer rows.Close()

	var files []entity.BinaryFile
	for rows.Next() {
		var f entity.BinaryFile
		if err := rows.Scan(
			&f.BinID, &f.ReferensiID, &f.Module, &f.Directory,
			&f.FileName, &f.Path, &f.Size, &f.MimeType,
			&f.Checksum, &f.Flag, &f.CreateDate,
		); err != nil {
			return nil, fmt.Errorf("scan binary file: %w", err)
		}
		files = append(files, f)
	}
	return files, nil
}

func (r *binaryFileRepository) GetPendingFiles(ctx context.Context, lastBinID int64, limit int) ([]entity.BinaryFile, error) {
	if limit <= 0 {
		limit = 100
	}
	const query = `
		SELECT bin_id, 
		       COALESCE(referensi_id, ''), 
		       COALESCE(module, ''), 
		       COALESCE(directory, ''), 
		       COALESCE(file_name, ''), 
		       COALESCE(path, ''), 
		       COALESCE(size, 0), 
		       COALESCE(mime_type, ''), 
		       COALESCE(checksum, ''), 
		       COALESCE(flag, '1'), 
		       COALESCE(create_date, NOW())
		FROM binary_file
		WHERE bin_id > $1
		ORDER BY bin_id ASC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, lastBinID, limit)
	if err != nil {
		return nil, fmt.Errorf("GetPendingFiles: %w", err)
	}
	defer rows.Close()

	var files []entity.BinaryFile
	for rows.Next() {
		var f entity.BinaryFile
		if err := rows.Scan(
			&f.BinID, &f.ReferensiID, &f.Module, &f.Directory,
			&f.FileName, &f.Path, &f.Size, &f.MimeType,
			&f.Checksum, &f.Flag, &f.CreateDate,
		); err != nil {
			return nil, fmt.Errorf("scan pending file: %w", err)
		}
		files = append(files, f)
	}
	return files, nil
}

func (r *binaryFileRepository) Insert(ctx context.Context, file *entity.BinaryFile) (int64, error) {
	if file.CreateDate.IsZero() {
		file.CreateDate = time.Now()
	}
	if file.Flag == "" {
		file.Flag = "1"
	}

	// Auto-generate next bin_id if not explicitly provided
	if file.BinID <= 0 {
		var nextID int64
		err := r.db.QueryRow(ctx, "SELECT COALESCE(MAX(bin_id), 0) + 1 FROM binary_file").Scan(&nextID)
		if err != nil || nextID <= 0 {
			nextID = time.Now().Unix()
		}
		file.BinID = nextID
	}

	const query = `
		INSERT INTO binary_file (bin_id, referensi_id, module, directory, file_name, path, size, mime_type, checksum, flag, create_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING bin_id
	`
	var newID int64
	err := r.db.QueryRow(
		ctx, query,
		file.BinID, file.ReferensiID, file.Module, file.Directory, file.FileName,
		file.Path, file.Size, file.MimeType, file.Checksum, file.Flag, file.CreateDate,
	).Scan(&newID)

	if err != nil {
		return 0, fmt.Errorf("Insert binary_file: %w", err)
	}
	file.BinID = newID
	return newID, nil
}

func (r *binaryFileRepository) UpdatePathAndStatus(ctx context.Context, binID int64, path string, checksum string, size int64, flag string) error {
	const query = `
		UPDATE binary_file 
		SET path = $1, checksum = $2, size = $3, flag = $4
		WHERE bin_id = $5
	`
	_, err := r.db.Exec(ctx, query, path, checksum, size, flag, binID)
	if err != nil {
		return fmt.Errorf("UpdatePathAndStatus: %w", err)
	}
	return nil
}
