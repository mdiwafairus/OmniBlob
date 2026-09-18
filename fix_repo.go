package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	path := "internal/repository/binary_file_repository.go"
	data, err := ioutil.ReadFile(path)
	if err != nil { panic(err) }
	
	content := string(data)
	
	// Fix GetMigrationStats
	oldFunc := `func (r *binaryFileRepository) GetMigrationStats(ctx context.Context) (*entity.MigrationStats, error) {
	const query = 
		SELECT 
			COUNT(*) as total_files,
			SUM(CASE WHEN flag = 'M' THEN 1 ELSE 0 END) as migrated_files,
			COALESCE(SUM(CASE WHEN flag = 'M' THEN size ELSE 0 END), 0) as total_migrated_bytes
		FROM binary_file
	
	var stats entity.MigrationStats
	err := r.db.QueryRow(ctx, query).Scan(&stats.TotalFiles, &stats.MigratedFiles, &stats.TotalMigratedBytes)
	if err != nil {
		return nil, fmt.Errorf("GetMigrationStats: %w", err)
	}
	return &stats, nil
}`
	
	newFunc := `func (r *binaryFileRepository) GetMigrationStats(ctx context.Context) (*entity.MigrationStats, error) {
	const query = 
		SELECT 
			COUNT(*) as total_files,
			SUM(CASE WHEN flag = 'M' THEN 1 ELSE 0 END) as migrated_files,
			COALESCE(SUM(CASE WHEN flag = 'M' THEN size ELSE 0 END), 0) as total_migrated_bytes,
			SUM(CASE WHEN create_date < NOW() - INTERVAL '1 YEAR' THEN 1 ELSE 0 END) as stale_files
		FROM binary_file
	
	var stats entity.MigrationStats
	var staleFiles, migratedFiles, totalMigratedBytes *int64
	
	err := r.db.QueryRow(ctx, query).Scan(&stats.TotalFiles, &migratedFiles, &totalMigratedBytes, &staleFiles)
	
	if err != nil {
		return nil, fmt.Errorf("GetMigrationStats: %w", err)
	}
	
	if migratedFiles != nil { stats.MigratedFiles = *migratedFiles }
	if totalMigratedBytes != nil { stats.TotalMigratedBytes = *totalMigratedBytes }
	if staleFiles != nil { stats.StaleFiles = *staleFiles }
	
	return &stats, nil
}`

	content = strings.Replace(content, oldFunc, newFunc, 1)

	// Append missing methods
	appendStr := `

func (r *binaryFileRepository) GetModuleStats(ctx context.Context) ([]entity.ModuleStat, error) {
	query := 
		SELECT 
			COALESCE(NULLIF(module, ''), 'unknown') as mod,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size_bytes
		FROM binary_file
		GROUP BY mod
		ORDER BY size_bytes DESC
	
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
	query := 
		SELECT 
			EXTRACT(YEAR FROM create_date) as year,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size_bytes
		FROM binary_file
		WHERE create_date IS NOT NULL
		GROUP BY year
		ORDER BY year ASC
	
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
	query := 
		SELECT 
			EXTRACT(YEAR FROM create_date) as year,
			EXTRACT(MONTH FROM create_date) as month,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size_bytes
		FROM binary_file
		WHERE create_date IS NOT NULL
		GROUP BY year, month
		ORDER BY year ASC, month ASC
	
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

	query := 
		SELECT 
			id::VARCHAR, 
			'Background Migration' as name, 
			'Migrasi' as type, 
			0 as volume_bytes, 
			TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI') as date, 
			CASE WHEN error_logs IS NULL OR error_logs = '' THEN 'Success' ELSE 'Failed' END as status
		FROM log_file_rsync
		ORDER BY id DESC LIMIT 20
	
	
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
`
	ioutil.WriteFile(path, []byte(content + appendStr), 0644)
}
