package api

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pwni-file-sync/internal/entity"
	"pwni-file-sync/internal/repository"
	"pwni-file-sync/internal/storage"

	"github.com/rs/zerolog"
)

type Handler struct {
	storageRepo *storage.StorageService
	binaryRepo  repository.BinaryFileRepository
	logger      zerolog.Logger
	maxUploadMB int64
}

func NewHandler(
	storageRepo *storage.StorageService,
	binaryRepo repository.BinaryFileRepository,
	logger zerolog.Logger,
	maxUploadMB int,
) *Handler {
	if maxUploadMB <= 0 {
		maxUploadMB = 100
	}
	return &Handler{
		storageRepo: storageRepo,
		binaryRepo:  binaryRepo,
		logger:      logger,
		maxUploadMB: int64(maxUploadMB),
	}
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, resp APIResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, `{"success":false,"error":"JSON encode error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Connection", "close")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// HealthCheck returns service & database health status.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "pwni-file-sync service is healthy and running",
		Data: map[string]interface{}{
			"status":    "UP",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// Upload handles file upload from application servers.
// Form Data parameters:
// - file (multipart binary file)
// - module (e.g. "lapordiri", "paspor", "skck")
// - referensi_id (e.g. "LP-2026-001")
// - directory (optional)
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Error:   "Method not allowed, use POST",
		})
		return
	}

	maxBytes := h.maxUploadMB << 20 // Convert MB to Bytes
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		h.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse multipart form or file exceeds limit (%d MB): %v", h.maxUploadMB, err),
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Missing 'file' field in multipart form-data",
		})
		return
	}
	defer file.Close()

	module := strings.TrimSpace(r.FormValue("module"))
	if module == "" {
		module = strings.TrimSpace(r.FormValue("modul"))
	}
	if module == "" {
		module = "general"
	}

	referensiID := strings.TrimSpace(r.FormValue("referensi_id"))
	if referensiID == "" {
		referensiID = strings.TrimSpace(r.FormValue("uuid"))
	}

	directory := strings.TrimSpace(r.FormValue("directory"))
	if directory == "" {
		directory = strings.TrimSpace(r.FormValue("folder"))
	}
	if directory == "" {
		directory = strings.TrimSpace(r.FormValue("subfolder"))
	}

	flagVal := strings.TrimSpace(r.FormValue("flag"))
	if flagVal == "" {
		flagVal = "1"
	}

	customPath := strings.TrimSpace(r.FormValue("path"))
	customFileName := strings.TrimSpace(r.FormValue("file_name"))
	if customFileName == "" {
		customFileName = header.Filename
	}

	// 1. Stream file directly to storage & compute checksum
	relPath, checksum, size, err := h.storageRepo.Save(r.Context(), customPath, module, directory, referensiID, 0, customFileName, file)
	if err != nil {
		h.logger.Error().Err(err).Str("file", customFileName).Msg("Failed to save file to storage")
		h.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save file: %v", err),
		})
		return
	}

	// Determine MIME type
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		ext := filepath.Ext(customFileName)
		mimeType = mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	// 2. Insert metadata record into PostgreSQL
	binaryRecord := &entity.BinaryFile{
		ReferensiID: referensiID,
		Module:      module,
		Directory:   directory,
		FileName:    customFileName,
		Path:        relPath,
		Size:        size,
		MimeType:    mimeType,
		Checksum:    checksum,
		Flag:        flagVal,
		CreateDate:  time.Now(),
	}

	binID, err := h.binaryRepo.Insert(r.Context(), binaryRecord)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to insert binary_file record in PostgreSQL")
		h.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save metadata: %v", err),
		})
		return
	}

	h.logger.Info().
		Int64("bin_id", binID).
		Str("module", module).
		Str("referensi_id", referensiID).
		Str("filename", header.Filename).
		Int64("size_bytes", size).
		Str("checksum", checksum).
		Msg("File uploaded and indexed successfully")

	h.writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "File uploaded and indexed successfully",
		Data: map[string]interface{}{
			"bin_id":       binID,
			"referensi_id": referensiID,
			"module":       module,
			"file_name":    header.Filename,
			"path":         relPath,
			"url":          fmt.Sprintf("/api/v1/files/%d", binID),
			"size":         size,
			"mime_type":    mimeType,
			"checksum":     checksum,
		},
	})
}

// ServeFileByID serves a file by its database bin_id (e.g. GET /api/v1/files/123)
func (h *Handler) ServeFileByID(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		h.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Missing file ID in URL path"})
		return
	}

	idStr := pathParts[3]
	binID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid file ID"})
		return
	}

	// 1. Fetch metadata from PostgreSQL (~1 ms)
	fileMeta, err := h.binaryRepo.GetByID(r.Context(), binID)
	if err != nil {
		h.logger.Warn().Int64("bin_id", binID).Err(err).Msg("File metadata not found in database")
		h.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "File not found in database index"})
		return
	}

	// 2. Open file stream (with automatic fallback to legacy NFS directory)
	fileHandle, fileInfo, actualPath, err := h.storageRepo.Open(r.Context(), fileMeta.Path)
	if err != nil {
		// Fallback: try searching by fileMeta.FileName or Directory/FileName
		fallbackPath := filepath.Join(fileMeta.Directory, fileMeta.FileName)
		if fileMeta.Directory == "" {
			fallbackPath = filepath.Join(fileMeta.Module, fileMeta.FileName)
		}
		fileHandle, fileInfo, actualPath, err = h.storageRepo.Open(r.Context(), fallbackPath)
		if err != nil {
			h.logger.Error().Int64("bin_id", binID).Str("path", fileMeta.Path).Msg("Physical file not found on disk or legacy storage")
			h.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "Physical file not found on disk"})
			return
		}
	}
	defer fileHandle.Close()

	// 3. Set HTTP Headers & Stream using http.ServeContent (supports Range Requests & Caching)
	mimeType := fileMeta.MimeType
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(fileMeta.FileName))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	w.Header().Set("Content-Type", mimeType)
	if fileMeta.Checksum != "" {
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, fileMeta.Checksum))
	}
	w.Header().Set("Cache-Control", "public, max-age=86400") // 24h client cache
	w.Header().Set("X-Storage-Path", actualPath)

	// Stream file to client (zero memory overhead)
	http.ServeContent(w, r, fileMeta.FileName, fileInfo.ModTime(), fileHandle)
}

// ServeFileByRef serves a file by its referensi_id (e.g. GET /api/v1/files/view?ref_id=LP001&module=lapordiri)
func (h *Handler) ServeFileByRef(w http.ResponseWriter, r *http.Request) {
	refID := strings.TrimSpace(r.URL.Query().Get("ref_id"))
	module := strings.TrimSpace(r.URL.Query().Get("module"))

	if refID == "" {
		h.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Missing 'ref_id' query parameter"})
		return
	}

	files, err := h.binaryRepo.GetByReferensiID(r.Context(), refID, module)
	if err != nil || len(files) == 0 {
		h.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "No files found for the given reference ID"})
		return
	}

	// Serve the latest active file
	target := files[0]
	r.URL.Path = fmt.Sprintf("/api/v1/files/%d", target.BinID)
	h.ServeFileByID(w, r)
}
