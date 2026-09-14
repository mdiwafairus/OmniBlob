package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/repository"
	"pwni-file-sync/internal/utils"

	"github.com/rs/zerolog"
)

type DashboardHandler struct {
	repo repository.DashboardRepository
	cfg  *config.ServerConfig
	mig  *config.MigrationConfig
	str  *config.StorageConfig
	log  zerolog.Logger

	// Simple in-memory cache to prevent DB overload
	cacheMu     sync.RWMutex
	cacheData   map[string]cacheEntry
}

type cacheEntry struct {
	data      interface{}
	timestamp time.Time
}

func NewDashboardHandler(repo repository.DashboardRepository, cfg *config.ServerConfig, mig *config.MigrationConfig, str *config.StorageConfig, log zerolog.Logger) *DashboardHandler {
	return &DashboardHandler{
		repo:      repo,
		cfg:       cfg,
		mig:       mig,
		str:       str,
		log:       log,
		cacheData: make(map[string]cacheEntry),
	}
}

func (h *DashboardHandler) getCached(key string) (interface{}, bool) {
	h.cacheMu.RLock()
	defer h.cacheMu.RUnlock()
	entry, exists := h.cacheData[key]
	if exists && time.Since(entry.timestamp) < 5*time.Minute {
		return entry.data, true
	}
	return nil, false
}

func (h *DashboardHandler) setCache(key string, data interface{}) {
	h.cacheMu.Lock()
	defer h.cacheMu.Unlock()
	h.cacheData[key] = cacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	if cached, ok := h.getCached("summary"); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	ctx := r.Context()
	
	summary, err := h.repo.GetSummaryStats(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get summary stats")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	diskSpace, err := utils.GetStorageDiskSpace(h.str.RootPath)
	if err != nil {
		h.log.Warn().Err(err).Msg("Failed to get disk space, ignoring")
	}

	var clientNames []string
	for _, c := range h.cfg.Clients {
		clientNames = append(clientNames, c.Name)
	}

	response := map[string]interface{}{
		"total_data_migrated_bytes": summary.TotalDataMigratedBytes,
		"total_files_migrated":      summary.TotalFilesMigrated,
		"total_pending_files":       summary.TotalPendingFiles,
		"overall_progress_percent":  summary.OverallProgressPercent,
		"stale_files_over_1_year":   summary.StaleFilesOver1Year,
		"migration_enabled":         h.mig.Enabled,
		"vm_total_bytes":            diskSpace.TotalBytes,
		"vm_used_bytes":             diskSpace.UsedBytes,
		"vm_free_bytes":             diskSpace.FreeBytes,
		"vm_used_percent":           diskSpace.UsedPercent,
		"root_path":                 h.str.RootPath,
		"legacy_path":               h.str.LegacyPath,
		"sharding_type":             h.str.ShardingType,
		"clients":                   clientNames,
	}

	h.setCache("summary", response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *DashboardHandler) Analytics(w http.ResponseWriter, r *http.Request) {
	if cached, ok := h.getCached("analytics"); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	ctx := r.Context()

	extStats, err := h.repo.GetExtensionStats(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get extension stats")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	modStats, err := h.repo.GetModuleStats(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get module stats")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	topFiles, err := h.repo.GetTopLargeFiles(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get top files")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	yearStats, err := h.repo.GetYearlyStats(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get yearly stats")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	monthStats, err := h.repo.GetMonthlyStats(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get monthly stats")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"extension_stats": extStats,
		"module_stats":    modStats,
		"top_large_files": topFiles,
		"yearly_stats":    yearStats,
		"monthly_stats":   monthStats,
	}

	h.setCache("analytics", response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *DashboardHandler) Quality(w http.ResponseWriter, r *http.Request) {
	if cached, ok := h.getCached("quality"); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	ctx := r.Context()

	qualityStats, err := h.repo.GetDuplicateStats(ctx)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get quality stats")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.setCache("quality", qualityStats)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(qualityStats)
}
