package service

import (
	"context"
	"math"

	"pwni-file-sync/internal/entity"
	"pwni-file-sync/internal/repository"

	"github.com/shirou/gopsutil/v3/disk"
)

type DashboardService struct {
	binaryRepo repository.BinaryFileRepository
	logRepo    repository.LogRepository
}

func NewDashboardService(binaryRepo repository.BinaryFileRepository, logRepo repository.LogRepository) *DashboardService {
	return &DashboardService{
		binaryRepo: binaryRepo,
		logRepo:    logRepo,
	}
}

func (s *DashboardService) GetExecutiveSummary(ctx context.Context, destPath string, clients []string) (*entity.DashboardSummary, error) {
	stats, err := s.binaryRepo.GetMigrationStats(ctx)
	if err != nil {
		return nil, err
	}

	var progress float64
	if stats.TotalFiles > 0 {
		progress = float64(stats.MigratedFiles) / float64(stats.TotalFiles) * 100.0
		progress = math.Round(progress*100) / 100
	}

	freeSpace, totalSpace, usedSpace, usedPercent := s.getDiskSpaceStats(destPath)

	var liveTransferRate float64 = 0
	var currentLatencyMs float64 = 0

	jobHistory, totalJobs, err := s.binaryRepo.GetJobHistory(ctx, "")
	if err != nil {
		jobHistory = []entity.JobHistory{}
	}

	summary := &entity.DashboardSummary{
		TotalDataMigratedBytes:    stats.TotalMigratedBytes,
		TotalFilesMigrated:        stats.MigratedFiles,
		TotalPendingFiles:         stats.TotalFiles - stats.MigratedFiles,
		OverallProgressPercent:    progress,
		LiveTransferRateMBps:      liveTransferRate,
		CurrentLatencyMs:          currentLatencyMs,
		DestinationFreeSpaceBytes: freeSpace,
		
		StaleFilesOver1Year: stats.StaleFiles,
		MigrationEnabled:    true,
		VmTotalBytes:        totalSpace,
		VmUsedBytes:         usedSpace,
		VmFreeBytes:         freeSpace,
		VmUsedPercent:       usedPercent,
		RootPath:            destPath,
		LegacyPath:          "/legacy",
		ShardingType:        "date-based",
		Clients:             clients,
		TotalLifetimeMigrations: totalJobs,
		MigrationHistory:    jobHistory,
	}

	return summary, nil
}

func (s *DashboardService) GetStorageAnalytics(ctx context.Context) (*entity.StorageAnalytics, error) {
	extStats, err := s.binaryRepo.GetExtensionStats(ctx)
	if err != nil {
		return nil, err
	}

	topFiles, err := s.binaryRepo.GetTopLargeFiles(ctx, 100)
	if err != nil {
		return nil, err
	}
	
	modStats, err := s.binaryRepo.GetModuleStats(ctx)
	if err != nil {
		return nil, err
	}
	
	yStats, err := s.binaryRepo.GetYearlyStats(ctx)
	if err != nil {
		return nil, err
	}
	
	mStats, err := s.binaryRepo.GetMonthlyStats(ctx)
	if err != nil {
		return nil, err
	}

	return &entity.StorageAnalytics{
		ExtensionStats: extStats,
		TopLargeFiles:  topFiles,
		ModuleStats:    modStats,
		YearlyStats:    yStats,
		MonthlyStats:   mStats,
	}, nil
}

func (s *DashboardService) GetDataQualityStats(ctx context.Context) (*entity.DataQualityStats, error) {
	return s.binaryRepo.GetDataQualityStats(ctx)
}

// getDiskSpaceStats gets the disk space stats for a given path across platforms.
func (s *DashboardService) getDiskSpaceStats(path string) (free uint64, total uint64, used uint64, percent float64) {
	if path == "" {
		path = "/"
	}
	
	usageStat, err := disk.Usage(path)
	if err != nil {
		// Fallback safely if path error occurs
		return 0, 0, 0, 0
	}
	
	return usageStat.Free, usageStat.Total, usageStat.Used, usageStat.UsedPercent
}
