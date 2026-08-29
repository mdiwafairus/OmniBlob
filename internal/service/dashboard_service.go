package service

import (
	"context"
	"math"

	"pwni-file-sync/internal/entity"
	"pwni-file-sync/internal/repository"

	"golang.org/x/sys/windows"
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

func (s *DashboardService) GetExecutiveSummary(ctx context.Context, destPath string) (*entity.DashboardSummary, error) {
	stats, err := s.binaryRepo.GetMigrationStats(ctx)
	if err != nil {
		return nil, err
	}

	var progress float64
	if stats.TotalFiles > 0 {
		progress = float64(stats.MigratedFiles) / float64(stats.TotalFiles) * 100.0
		// Round to 2 decimal places
		progress = math.Round(progress*100) / 100
	}

	freeSpace := s.getFreeDiskSpace(destPath)

	// In a real scenario, we would calculate this based on recent log_file_rsync entries
	// For MVP/Placeholder, we can set 0 until the worker starts pushing latency metrics
	var liveTransferRate float64 = 0
	var currentLatencyMs float64 = 0

	summary := &entity.DashboardSummary{
		TotalDataMigratedBytes:    stats.TotalMigratedBytes,
		TotalFilesMigrated:        stats.MigratedFiles,
		TotalPendingFiles:         stats.TotalFiles - stats.MigratedFiles,
		OverallProgressPercent:    progress,
		LiveTransferRateMBps:      liveTransferRate,
		CurrentLatencyMs:          currentLatencyMs,
		DestinationFreeSpaceBytes: freeSpace,
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

	return &entity.StorageAnalytics{
		ExtensionStats: extStats,
		TopLargeFiles:  topFiles,
	}, nil
}

func (s *DashboardService) GetDataQualityStats(ctx context.Context) (*entity.DataQualityStats, error) {
	return s.binaryRepo.GetDataQualityStats(ctx)
}

// getFreeDiskSpace gets the free disk space for a given path on Windows.
// For production, you might want to switch between syscalls depending on the OS (using build tags).
func (s *DashboardService) getFreeDiskSpace(path string) uint64 {
	if path == "" {
		path = "C:\\"
	}
	
	// Convert path to UTF16 pointer for Windows API
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	
	var freeBytesAvailableToCaller, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	
	err = windows.GetDiskFreeSpaceEx(pathPtr, &freeBytesAvailableToCaller, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return 0
	}
	
	return freeBytesAvailableToCaller
}
