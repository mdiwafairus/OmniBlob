package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pwni-file-sync/internal/config"

	"github.com/rs/zerolog"
)

type ExplorerHandler struct {
	cfg *config.StorageConfig
	mig *config.MigrationConfig
	log zerolog.Logger
}

func NewExplorerHandler(cfg *config.StorageConfig, mig *config.MigrationConfig, log zerolog.Logger) *ExplorerHandler {
	return &ExplorerHandler{
		cfg: cfg,
		mig: mig,
		log: log,
	}
}

type FileNode struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDirectory  bool      `json:"is_directory"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
}

func (h *ExplorerHandler) ListDirectory(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target") // "legacy" or "omniblob"
	reqPath := r.URL.Query().Get("path")

	if reqPath == "" {
		reqPath = "/"
	}

	var basePath string
	if target == "legacy" {
		if h.cfg.LegacyPath == "" {
			http.Error(w, "Legacy path not configured", http.StatusBadRequest)
			return
		}
		basePath = h.cfg.LegacyPath
	} else if target == "omniblob" {
		if h.cfg.RootPath == "" {
			http.Error(w, "Root path not configured", http.StatusBadRequest)
			return
		}
		basePath = h.cfg.RootPath
	} else {
		http.Error(w, "Invalid target", http.StatusBadRequest)
		return
	}

	// Clean paths to prevent directory traversal
	cleanReqPath := filepath.Clean("/" + reqPath)
	fullPath := filepath.Join(basePath, cleanReqPath)

	// Ensure the resolved fullPath starts with the basePath
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to resolve absolute base path")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to resolve absolute full path")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Case-insensitive check for windows, but simple prefix check is usually sufficient if cleaned properly
	if !strings.HasPrefix(strings.ToLower(absFullPath), strings.ToLower(absBasePath)) {
		h.log.Warn().Str("req_path", reqPath).Msg("Directory traversal attempt blocked")
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	entries, err := os.ReadDir(absFullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty list if folder doesn't exist yet
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]FileNode{})
			return
		}
		h.log.Error().Err(err).Str("path", absFullPath).Msg("Failed to read directory")
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	var nodes []FileNode
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}
		
		nodePath := filepath.ToSlash(filepath.Join(cleanReqPath, entry.Name()))
		if cleanReqPath == "/" {
			nodePath = "/" + entry.Name()
		}

		size := info.Size()
		if entry.IsDir() {
			// Calculate recursive directory size
			dirAbsPath := filepath.Join(absFullPath, entry.Name())
			var dirSize int64
			filepath.WalkDir(dirAbsPath, func(_ string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					if dInfo, err := d.Info(); err == nil {
						dirSize += dInfo.Size()
					}
				}
				return nil
			})
			size = dirSize
		}

		nodes = append(nodes, FileNode{
			Name:         entry.Name(),
			Path:         nodePath,
			IsDirectory:  entry.IsDir(),
			Size:         size,
			ModifiedTime: info.ModTime(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}
