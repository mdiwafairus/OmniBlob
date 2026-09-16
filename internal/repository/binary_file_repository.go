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

func (r *binaryFileRepository) GetByFileName(ctx context.Context, fileName string) (*entity.BinaryFile, error) {
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
		WHERE file_name = $1
		ORDER BY bin_id DESC
		LIMIT 1
	`
	var f entity.BinaryFile
	row := r.db.QueryRow(ctx, query, fileName)
	err := row.Scan(
		&f.BinID, &f.ReferensiID, &f.Module, &f.Directory,
		&f.FileName, &f.Path, &f.Size, &f.MimeType,
		&f.Checksum, &f.Flag, &f.CreateDate,
	)
	if err != nil {
		return nil, fmt.Errorf("GetByFileName: %w", err)
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

func (r *binaryFileRepository) GetMigrationStats(ctx context.Context) (*entity.MigrationStats, error) {
	const query = `
		SELECT 
			COUNT(*) as total_files,
			SUM(CASE WHEN flag = 'M' THEN 1 ELSE 0 END) as migrated_files,
			COALESCE(SUM(CASE WHEN flag = 'M' THEN size ELSE 0 END), 0) as total_migrated_bytes
		FROM binary_file
	`
	var stats entity.MigrationStats
	err := r.db.QueryRow(ctx, query).Scan(&stats.TotalFiles, &stats.MigratedFiles, &stats.TotalMigratedBytes)
	if err != nil {
		return nil, fmt.Errorf("GetMigrationStats: %w", err)
	}
	return &stats, nil
}

func (r *binaryFileRepository) GetExtensionStats(ctx context.Context) ([]entity.ExtensionStat, error) {
	const query = `
		SELECT 
			LOWER(SUBSTRING(file_name FROM '\.([^\.]+)$')) as extension,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size_bytes
		FROM binary_file
		WHERE file_name LIKE '%.%'
		GROUP BY extension
		ORDER BY size_bytes DESC
		LIMIT 50
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetExtensionStats: %w", err)
	}
	defer rows.Close()

	var stats []entity.ExtensionStat
	for rows.Next() {
		var stat entity.ExtensionStat
		if err := rows.Scan(&stat.Extension, &stat.Count, &stat.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan extension stat: %w", err)
		}
		stats = append(stats, stat)
	}
	return stats, nil
}

func (r *binaryFileRepository) GetTopLargeFiles(ctx context.Context, limit int) ([]entity.LargeFile, error) {
	if limit <= 0 {
		limit = 100
	}
	const query = `
		SELECT 
			bin_id,
			file_name,
			COALESCE(path, '') as path,
			COALESCE(size, 0) as size_bytes,
			COALESCE(LOWER(SUBSTRING(file_name FROM '\.([^\.]+)$')), '') as extension
		FROM binary_file
		ORDER BY size DESC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("GetTopLargeFiles: %w", err)
	}
	defer rows.Close()

	var files []entity.LargeFile
	for rows.Next() {
		var f entity.LargeFile
		if err := rows.Scan(&f.BinID, &f.FileName, &f.Path, &f.SizeBytes, &f.Extension); err != nil {
			return nil, fmt.Errorf("scan large file: %w", err)
		}
		files = append(files, f)
	}
	return files, nil
}

func (r *binaryFileRepository) GetDataQualityStats(ctx context.Context) (*entity.DataQualityStats, error) {
	const query = `
		WITH DuplicateHashes AS (
			SELECT 
				checksum, 
				COUNT(*) as count, 
				MAX(size) as file_size, 
				MAX(file_name) as example_name
			FROM binary_file
			WHERE checksum != '' AND checksum IS NOT NULL
			GROUP BY checksum
			HAVING COUNT(*) > 1
		)
		SELECT 
			checksum, 
			count, 
			file_size,
			(count - 1) * file_size as wasted_bytes,
			example_name
		FROM DuplicateHashes
		ORDER BY wasted_bytes DESC
		LIMIT 100
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetDataQualityStats groups: %w", err)
	}
	defer rows.Close()

	var stats entity.DataQualityStats
	for rows.Next() {
		var g entity.DuplicateGroup
		if err := rows.Scan(&g.Checksum, &g.Count, &g.SizeBytes, &g.WastedBytes, &g.ExampleName); err != nil {
			return nil, fmt.Errorf("scan duplicate group: %w", err)
		}
		stats.DuplicateGroups = append(stats.DuplicateGroups, g)
	}
	
	const totalQuery = `
		WITH DuplicateHashes AS (
			SELECT COUNT(*) as count, MAX(size) as file_size
			FROM binary_file
			WHERE checksum != '' AND checksum IS NOT NULL
			GROUP BY checksum
			HAVING COUNT(*) > 1
		)
		SELECT 
			COALESCE(SUM(count - 1), 0) as total_duplicates,
			COALESCE(SUM((count - 1) * file_size), 0) as total_wasted
		FROM DuplicateHashes
	`
	err = r.db.QueryRow(ctx, totalQuery).Scan(&stats.TotalDuplicateFiles, &stats.TotalWastedBytes)
	if err != nil {
		return nil, fmt.Errorf("GetDataQualityStats totals: %w", err)
	}

	return &stats, nil
}

func (r *binaryFileRepository) ExistsByPath(ctx context.Context, path string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT true FROM binary_file WHERE path = $1 LIMIT 1", path).Scan(&exists)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}



func (r *binaryFileRepository) GetModuleStats(ctx context.Context) ([]entity.ModuleStat, error) {
	query := `
		SELECT 
			COALESCE(NULLIF(module, ''), 'unknown') as mod,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size_bytes
		FROM binary_file
		GROUP BY mod
		ORDER BY size_bytes DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil { return nil, err }
	defer rows.Close()

	var stats []entity.ModuleStat
	for rows.Next() {
		var s entity.ModuleStat
		if err := rows.Scan(&s.Module, &s.Count, &s.SizeBytes); err != nil { return nil, err }
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *binaryFileRepository) GetYearlyStats(ctx context.Context) ([]entity.YearStat, error) {
	query := `
		SELECT 
			EXTRACT(YEAR FROM create_date) as year,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size_bytes
		FROM binary_file
		WHERE create_date IS NOT NULL
		GROUP BY year
		ORDER BY year ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil { return nil, err }
	defer rows.Close()

	var stats []entity.YearStat
	for rows.Next() {
		var s entity.YearStat
		if err := rows.Scan(&s.Year, &s.Count, &s.SizeBytes); err != nil { return nil, err }
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *binaryFileRepository) GetMonthlyStats(ctx context.Context) ([]entity.MonthlyStat, error) {
	query := `
		SELECT 
			EXTRACT(YEAR FROM create_date) as year,
			EXTRACT(MONTH FROM create_date) as month,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size_bytes
		FROM binary_file
		WHERE create_date IS NOT NULL
		GROUP BY year, month
		ORDER BY year ASC, month ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil { return nil, err }
	defer rows.Close()

	var stats []entity.MonthlyStat
	for rows.Next() {
		var s entity.MonthlyStat
		if err := rows.Scan(&s.Year, &s.Month, &s.Count, &s.SizeBytes); err != nil { return nil, err }
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *binaryFileRepository) GetJobHistory(ctx context.Context, clientID string) ([]entity.JobHistory, int64, error) {
	var totalJobs int64
	var history []entity.JobHistory

	countQuery := "SELECT COUNT(*) FROM log_file_rsync"
	err := r.db.QueryRow(ctx, countQuery).Scan(&totalJobs)
	if err != nil { return []entity.JobHistory{}, 0, nil }

	query := `
		SELECT 
			id::VARCHAR, 
			'Background Migration' as name, 
			'Migrasi' as type, 
			0 as volume_bytes, 
			TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI') as date, 
			CASE WHEN error_logs IS NULL OR error_logs = '' THEN 'Success' ELSE 'Failed' END as status
		FROM log_file_rsync
		ORDER BY id DESC LIMIT 20
	
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil { return []entity.JobHistory{}, 0, nil }
	defer rows.Close()

	for rows.Next() {
		var j entity.JobHistory
		if err := rows.Scan(&j.ID, &j.Name, &j.Type, &j.VolumeBytes, &j.Date, &j.Status); err != nil { return nil, 0, err }
		
		j.Duration = "< 1m"
		history = append(history, j)
	}
	
	if history == nil { history = []entity.JobHistory{} }
	
	return history, totalJobs, nil
}
