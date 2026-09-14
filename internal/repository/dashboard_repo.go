package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

type DashboardRepository interface {
	GetSummaryStats(ctx context.Context) (DashboardSummary, error)
	GetExtensionStats(ctx context.Context) ([]ExtensionStat, error)
	GetTopLargeFiles(ctx context.Context) ([]LargeFile, error)
	GetDuplicateStats(ctx context.Context) (DataQualityStats, error)
	GetModuleStats(ctx context.Context) ([]ModuleStat, error)
	GetYearlyStats(ctx context.Context) ([]YearStat, error)
	GetMonthlyStats(ctx context.Context) ([]MonthlyStat, error)
	GetJobHistory(ctx context.Context, clientID string) ([]JobHistory, int64, error)
}

type dashboardRepository struct {
	db *pgxpool.Pool
}

func NewDashboardRepository(db *pgxpool.Pool) DashboardRepository {
	return &dashboardRepository{db: db}
}

type JobHistory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	VolumeBytes int64  `json:"volume_bytes"`
	Date        string `json:"date"`
	Duration    string `json:"duration"`
	Status      string `json:"status"`
}

type DashboardSummary struct {
	TotalDataMigratedBytes  int64        `json:"total_data_migrated_bytes"`
	TotalFilesMigrated      int64        `json:"total_files_migrated"`
	TotalPendingFiles       int64        `json:"total_pending_files"`
	OverallProgressPercent  float64      `json:"overall_progress_percent"`
	StaleFilesOver1Year     int64        `json:"stale_files_over_1_year"`
	TotalLifetimeMigrations int64        `json:"total_lifetime_migrations"`
	MigrationHistory        []JobHistory `json:"migration_history"`
}

type ExtensionStat struct {
	Extension string `json:"extension"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
}

type ModuleStat struct {
	Module    string `json:"module"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
}

type YearStat struct {
	Year      int    `json:"year"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
}

type LargeFile struct {
	BinID     int64  `json:"bin_id"`
	FileName  string `json:"file_name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Extension string `json:"extension"`
}

type DuplicateGroup struct {
	Checksum    string `json:"checksum"`
	Count       int64  `json:"count"`
	SizeBytes   int64  `json:"size_bytes"`
	WastedBytes int64  `json:"wasted_bytes"`
	ExampleName string `json:"example_name"`
}

type DataQualityStats struct {
	TotalDuplicateFiles int64            `json:"total_duplicate_files"`
	TotalWastedBytes    int64            `json:"total_wasted_bytes"`
	DuplicateGroups     []DuplicateGroup `json:"duplicate_groups"`
}

func (r *dashboardRepository) GetSummaryStats(ctx context.Context) (DashboardSummary, error) {
	var summary DashboardSummary

	query := `
		SELECT 
			COALESCE(SUM(size), 0) as total_data,
			COUNT(*) as total_files,
			COUNT(*) FILTER (WHERE COALESCE(flag, '1') != 'M') as pending_files,
			COUNT(*) FILTER (WHERE create_date < NOW() - INTERVAL '1 YEAR') as stale_files
		FROM binary_file
	`
	err := r.db.QueryRow(ctx, query).Scan(
		&summary.TotalDataMigratedBytes,
		&summary.TotalFilesMigrated,
		&summary.TotalPendingFiles,
		&summary.StaleFilesOver1Year,
	)
	if err != nil {
		return summary, fmt.Errorf("GetSummaryStats: %w", err)
	}

	summary.TotalFilesMigrated = summary.TotalFilesMigrated - summary.TotalPendingFiles

	if summary.TotalFilesMigrated+summary.TotalPendingFiles > 0 {
		summary.OverallProgressPercent = float64(summary.TotalFilesMigrated) / float64(summary.TotalFilesMigrated+summary.TotalPendingFiles) * 100
	} else {
		summary.OverallProgressPercent = 100
	}

	return summary, nil
}

func (r *dashboardRepository) GetExtensionStats(ctx context.Context) ([]ExtensionStat, error) {
	query := `
		SELECT 
			SUBSTRING(file_name FROM '\.([^\.]+)$') as ext,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size_bytes
		FROM binary_file
		WHERE file_name LIKE '%._%'
		GROUP BY ext
		ORDER BY size_bytes DESC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetExtensionStats: %w", err)
	}
	defer rows.Close()

	var stats []ExtensionStat
	for rows.Next() {
		var s ExtensionStat
		if err := rows.Scan(&s.Extension, &s.Count, &s.SizeBytes); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *dashboardRepository) GetModuleStats(ctx context.Context) ([]ModuleStat, error) {
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
	if err != nil {
		return nil, fmt.Errorf("GetModuleStats: %w", err)
	}
	defer rows.Close()

	var stats []ModuleStat
	for rows.Next() {
		var s ModuleStat
		if err := rows.Scan(&s.Module, &s.Count, &s.SizeBytes); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *dashboardRepository) GetTopLargeFiles(ctx context.Context) ([]LargeFile, error) {
	query := `
		SELECT 
			bin_id,
			file_name,
			path,
			size,
			COALESCE(SUBSTRING(file_name FROM '\.([^\.]+)$'), '') as ext
		FROM binary_file
		ORDER BY size DESC
		LIMIT 100
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetTopLargeFiles: %w", err)
	}
	defer rows.Close()

	var files []LargeFile
	for rows.Next() {
		var f LargeFile
		if err := rows.Scan(&f.BinID, &f.FileName, &f.Path, &f.SizeBytes, &f.Extension); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (r *dashboardRepository) GetDuplicateStats(ctx context.Context) (DataQualityStats, error) {
	query := `
		SELECT 
			checksum,
			COUNT(*) as count,
			MAX(size) as size_bytes,
			(COUNT(*) - 1) * MAX(size) as wasted_bytes,
			MAX(file_name) as example_name
		FROM binary_file
		WHERE checksum != '' AND size > 0
		GROUP BY checksum
		HAVING COUNT(*) > 1
		ORDER BY wasted_bytes DESC
		LIMIT 50
	`
	rows, err := r.db.Query(ctx, query)
	var stats DataQualityStats
	if err != nil {
		return stats, fmt.Errorf("GetDuplicateStats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var g DuplicateGroup
		if err := rows.Scan(&g.Checksum, &g.Count, &g.SizeBytes, &g.WastedBytes, &g.ExampleName); err != nil {
			return stats, err
		}
		stats.DuplicateGroups = append(stats.DuplicateGroups, g)
		stats.TotalDuplicateFiles += (g.Count - 1)
		stats.TotalWastedBytes += g.WastedBytes
	}
	return stats, nil
}

func (r *dashboardRepository) GetYearlyStats(ctx context.Context) ([]YearStat, error) {
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
	if err != nil {
		return nil, fmt.Errorf("GetYearlyStats: %w", err)
	}
	defer rows.Close()

	var stats []YearStat
	for rows.Next() {
		var s YearStat
		if err := rows.Scan(&s.Year, &s.Count, &s.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan yearly stat: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

type MonthlyStat struct {
	Year      int   `json:"year"`
	Month     int   `json:"month"`
	Count     int64 `json:"count"`
	SizeBytes int64 `json:"size_bytes"`
}

func (r *dashboardRepository) GetMonthlyStats(ctx context.Context) ([]MonthlyStat, error) {
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
	if err != nil {
		return nil, fmt.Errorf("GetMonthlyStats: %w", err)
	}
	defer rows.Close()

	var stats []MonthlyStat
	for rows.Next() {
		var s MonthlyStat
		if err := rows.Scan(&s.Year, &s.Month, &s.Count, &s.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan monthly stat: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}
func (r *dashboardRepository) GetJobHistory(ctx context.Context, clientID string) ([]JobHistory, int64, error) {
	var totalJobs int64
	var history []JobHistory

	// Get total count from log_file_rsync
	countQuery := "SELECT COUNT(*) FROM log_file_rsync"
	err := r.db.QueryRow(ctx, countQuery).Scan(&totalJobs)
	if err != nil {
		return []JobHistory{}, 0, nil
	}

	// Get recent jobs (limit 20)
	query := `
		SELECT 
			id, 
			'Background Migration' as name, 
			'Migrasi' as type, 
			0 as volume_bytes, 
			TO_CHAR(NOW(), 'YYYY-MM-DD HH24:MI') as date, 
			error_logs as status
		FROM log_file_rsync
		ORDER BY id DESC LIMIT 20
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return []JobHistory{}, 0, nil
	}
	defer rows.Close()

	for rows.Next() {
		var j JobHistory
		if err := rows.Scan(&j.ID, &j.Name, &j.Type, &j.VolumeBytes, &j.Date, &j.Status); err != nil {
			return nil, 0, err
		}
		
		j.Duration = "< 1m" // Batch duration is usually fast
		history = append(history, j)
	}
	
	if history == nil {
		history = []JobHistory{}
	}
	
	return history, totalJobs, nil
}
