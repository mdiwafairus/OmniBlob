package api

import (
	"encoding/json"
	"net/http"

	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/service"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
	storageConfig    *config.StorageConfig
	clients          []string
}

func NewDashboardHandler(dashboardService *service.DashboardService, storageConfig *config.StorageConfig, clients []string) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		storageConfig:    storageConfig,
		clients:          clients,
	}
}

func (h *DashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	destPath := h.storageConfig.RootPath
	if destPath == "" {
		destPath = "C:\\"
	}

	summary, err := h.dashboardService.GetExecutiveSummary(r.Context(), destPath, h.clients)
	if err != nil {
		http.Error(w, "Failed to get dashboard summary: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (h *DashboardHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := h.dashboardService.GetStorageAnalytics(r.Context())
	if err != nil {
		http.Error(w, "Failed to get storage analytics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

func (h *DashboardHandler) GetQuality(w http.ResponseWriter, r *http.Request) {
	quality, err := h.dashboardService.GetDataQualityStats(r.Context())
	if err != nil {
		http.Error(w, "Failed to get data quality stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quality)
}
