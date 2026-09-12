package api

import (
	"encoding/json"
	"fmt"
	"io"
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
	allowedExts []string
}

func NewHandler(
	storageRepo *storage.StorageService,
	binaryRepo repository.BinaryFileRepository,
	logger zerolog.Logger,
	maxUploadMB int,
	allowedExts []string,
) *Handler {
	if maxUploadMB <= 0 {
		maxUploadMB = 100
	}
	exts := make([]string, 0, len(allowedExts))
	for _, ext := range allowedExts {
		exts = append(exts, strings.ToLower(strings.TrimSpace(ext)))
	}
	return &Handler{
		storageRepo: storageRepo,
		binaryRepo:  binaryRepo,
		logger:      logger,
		maxUploadMB: int64(maxUploadMB),
		allowedExts: exts,
	}
}

// isExtensionAllowed checks if a filename's extension is permitted
func (h *Handler) isExtensionAllowed(filename string) bool {
	if len(h.allowedExts) == 0 {
		return true // No restriction if list is empty
	}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, a := range h.allowedExts {
		if ext == a {
			return true
		}
	}
	return false
}

// isMimeMatchSecure validates if the physically detected MIME type matches the file extension claim
// and prevents malicious disguises (e.g. PHP/Exe renamed to PDF)
func (h *Handler) isMimeMatchSecure(filename string, detectedMime string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeBase := strings.Split(detectedMime, ";")[0]

	// Strictly block inherently dangerous MIME types regardless of extension (unless specifically intended)
	if mimeBase == "application/x-executable" || mimeBase == "application/x-mach-binary" || mimeBase == "application/x-elf" || mimeBase == "application/x-sh" || mimeBase == "text/x-php" {
		return false
	}

	// For specific extensions, ensure the detected MIME makes sense
	switch ext {
	case ".pdf":
		return mimeBase == "application/pdf"
	case ".jpg", ".jpeg":
		return mimeBase == "image/jpeg"
	case ".png":
		return mimeBase == "image/png"
	case ".gif":
		return mimeBase == "image/gif"
	case ".docx", ".xlsx", ".pptx", ".zip":
		return mimeBase == "application/zip"
	case ".html", ".htm":
		return mimeBase == "text/html"
	case ".txt", ".csv":
		return strings.HasPrefix(mimeBase, "text/")
	}

	// If extension isn't HTML, ensure detected MIME isn't HTML/JS (to prevent XSS payloads disguised as images/docs)
	if mimeBase == "text/html" || mimeBase == "text/javascript" {
		return false
	}

	// For extensions not explicitly mapped above, allow by default if it passes the danger checks
	return true
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
		Message: "omniBlob service is healthy and running",
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

	if !h.isExtensionAllowed(customFileName) {
		h.writeJSON(w, http.StatusUnsupportedMediaType, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("File extension not allowed for file: %s", customFileName),
		})
		return
	}

	// Read first 512 bytes for MIME detection (magic bytes)
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil && err != io.EOF {
		h.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to read file for inspection"})
		return
	}
	detectedMime := http.DetectContentType(buff)

	// Reset file pointer back to start
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			h.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to reset file pointer"})
			return
		}
	}

	if !h.isMimeMatchSecure(customFileName, detectedMime) {
		h.logger.Warn().Str("filename", customFileName).Str("detected_mime", detectedMime).Msg("Security check failed: File content does not match extension or is dangerous")
		h.writeJSON(w, http.StatusUnsupportedMediaType, APIResponse{
			Success: false,
			Error:   "Security check failed: File content does not match extension or contains dangerous data",
		})
		return
	}

	// 1. Stream file directly to storage & compute checksum
	relPath, checksum, size, err := h.storageRepo.Save(r.Context(), module, customPath, module, directory, referensiID, 0, customFileName, file)
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

	appName := "anonymous"
	if client := GetClientFromContext(r.Context()); client != nil {
		appName = client.Name
	}

	h.logger.Info().
		Str("app", appName).
		Int64("bin_id", binID).
		Str("module", module).
		Str("referensi_id", referensiID).
		Str("filename", customFileName).
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
	var fileMeta *entity.BinaryFile
	binID, err := strconv.ParseInt(idStr, 10, 64)
	if err == nil {
		// 1. Fetch metadata from PostgreSQL by bin_id
		fileMeta, err = h.binaryRepo.GetByID(r.Context(), binID)
	} else {
		// 1. Fetch metadata from PostgreSQL by file_name
		fileMeta, err = h.binaryRepo.GetByFileName(r.Context(), idStr)
	}

	if err != nil || fileMeta == nil {
		h.logger.Warn().Str("id_or_name", idStr).Err(err).Msg("File metadata not found in database")
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

// BulkUpload handles multiple file uploads in a single request.
func (h *Handler) BulkUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	maxBytes := h.maxUploadMB << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		h.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse multipart form: %v", err),
		})
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		files = r.MultipartForm.File["file"] // fallback
	}
	if len(files) == 0 {
		h.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Missing files"})
		return
	}

	module := strings.TrimSpace(r.FormValue("module"))
	if module == "" {
		module = "general"
	}
	referensiID := strings.TrimSpace(r.FormValue("referensi_id"))
	directory := strings.TrimSpace(r.FormValue("directory"))
	flagVal := strings.TrimSpace(r.FormValue("flag"))
	if flagVal == "" {
		flagVal = "1"
	}

	type FileResult struct {
		FileName string `json:"file_name"`
		BinID    int64  `json:"bin_id,omitempty"`
		Path     string `json:"path,omitempty"`
		URL      string `json:"url,omitempty"`
		Error    string `json:"error,omitempty"`
	}
	var results []FileResult

	for _, header := range files {
		if !h.isExtensionAllowed(header.Filename) {
			results = append(results, FileResult{FileName: header.Filename, Error: "File extension not allowed"})
			continue
		}

		file, err := header.Open()
		if err != nil {
			results = append(results, FileResult{FileName: header.Filename, Error: err.Error()})
			continue
		}

		buff := make([]byte, 512)
		if _, err := file.Read(buff); err != nil && err != io.EOF {
			results = append(results, FileResult{FileName: header.Filename, Error: "Failed to read file for inspection"})
			file.Close()
			continue
		}
		detectedMime := http.DetectContentType(buff)
		if seeker, ok := file.(io.Seeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				results = append(results, FileResult{FileName: header.Filename, Error: "Failed to reset file pointer"})
				file.Close()
				continue
			}
		}

		if !h.isMimeMatchSecure(header.Filename, detectedMime) {
			h.logger.Warn().Str("filename", header.Filename).Str("detected_mime", detectedMime).Msg("Security check failed in bulk upload")
			results = append(results, FileResult{FileName: header.Filename, Error: "Security check failed: File content does not match extension or is dangerous"})
			file.Close()
			continue
		}
		relPath, checksum, size, err := h.storageRepo.Save(r.Context(), module, "", module, directory, referensiID, 0, header.Filename, file)
		file.Close()

		if err != nil {
			results = append(results, FileResult{FileName: header.Filename, Error: err.Error()})
			continue
		}

		mimeType := header.Header.Get("Content-Type")
		if mimeType == "" || mimeType == "application/octet-stream" {
			mimeType = "application/octet-stream"
		}

		binaryRecord := &entity.BinaryFile{
			ReferensiID: referensiID,
			Module:      module,
			Directory:   directory,
			FileName:    header.Filename,
			Path:        relPath,
			Size:        size,
			MimeType:    mimeType,
			Checksum:    checksum,
			Flag:        flagVal,
			CreateDate:  time.Now(),
		}

		binID, err := h.binaryRepo.Insert(r.Context(), binaryRecord)
		if err != nil {
			results = append(results, FileResult{FileName: header.Filename, Error: err.Error()})
			continue
		}

		results = append(results, FileResult{
			FileName: header.Filename,
			BinID:    binID,
			Path:     relPath,
			URL:      fmt.Sprintf("/api/v1/%s/%d", module, binID),
		})
	}

	h.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Processed %d files", len(results)),
		Data:    results,
	})
}

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		h.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Missing file ID in URL path"})
		return
	}

	idStr := pathParts[3]
	var fileMeta *entity.BinaryFile
	binID, err := strconv.ParseInt(idStr, 10, 64)
	if err == nil {
		fileMeta, err = h.binaryRepo.GetByID(r.Context(), binID)
	} else {
		fileMeta, err = h.binaryRepo.GetByFileName(r.Context(), idStr)
	}

	if err != nil || fileMeta == nil {
		h.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "File not found in database index"})
		return
	}

	// Delete from storage
	err = h.storageRepo.Delete(r.Context(), fileMeta.Path)
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to delete physical file, might already be deleted or missing")
	}

	h.writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "File deleted successfully"})
}
