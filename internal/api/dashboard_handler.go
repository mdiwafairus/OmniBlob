package api

import (
	"encoding/json"
	"net/http"

	"pwni-file-sync/internal/service"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

func (h *DashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	// Panggil service untuk mendapatkan summary
	// Hardcode "C:\\" for now as destination path to check free space, 
	// ideally we take this from config
	summary, err := h.dashboardService.GetExecutiveSummary(r.Context(), "C:\\")
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
