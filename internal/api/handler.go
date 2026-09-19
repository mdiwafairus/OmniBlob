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

	"pwni-file-sync/internal/auth"
	"pwni-file-sync/internal/entity"
	"pwni-file-sync/internal/repository"
	"pwni-file-sync/internal/storage"

	"github.com/rs/zerolog"
)

type Handler struct {
	storageRepo  *storage.StorageService
	binaryRepo   repository.BinaryFileRepository
	logger            zerolog.Logger
	maxUploadMB       int64
	allowedExtensions []string
	accessKey         string
	secretKey         string
}

func NewHandler(
	storageRepo *storage.StorageService,
	binaryRepo repository.BinaryFileRepository,
	logger zerolog.Logger,
	maxUploadMB int,
	allowedExtensions []string,
	accessKey string,
	secretKey string,
) *Handler {
	if maxUploadMB <= 0 {
		maxUploadMB = 100
	}

	exts := make([]string, 0, len(allowedExtensions))
	for _, ext := range allowedExtensions {
		exts = append(exts, strings.ToLower(strings.TrimSpace(ext)))
	}

	return &Handler{
		storageRepo:       storageRepo,
		binaryRepo:        binaryRepo,
		logger:            logger,
		maxUploadMB:       int64(maxUploadMB),
		allowedExtensions: exts,
		accessKey:         accessKey,
		secretKey:         secretKey,
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

	// Extension Validation
	if len(h.allowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowed := false
		for _, validExt := range h.allowedExtensions {
			if ext == validExt {
				allowed = true
				break
			}
		}
		if !allowed {
			h.writeJSON(w, http.StatusUnsupportedMediaType, APIResponse{
				Success: false,
				Error:   fmt.Sprintf("File extension %s is not allowed", ext),
			})
			return
		}

		// Magic Bytes / MIME Type Verification
		// Read first 512 bytes for sniffing
		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		
		// Rewind the file pointer so it can be saved fully later
		_, err = file.Seek(0, 0) // io.SeekStart is 0
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "Failed to read file stream for security verification",
			})
			return
		}

		contentType := http.DetectContentType(buffer[:n])

		// Ensure the detected content type matches the expected type for critical extensions
		expectedMimePrefix := map[string]string{
			".pdf":  "application/pdf",
			".png":  "image/png",
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".gif":  "image/gif",
			".mp4":  "video/mp4",
			".html": "text/html",
			".exe":  "application/x-executable",
		}

		if expected, ok := expectedMimePrefix[ext]; ok {
			if !strings.HasPrefix(contentType, expected) {
				h.logger.Warn().Str("filename", header.Filename).Str("detected_mime", contentType).Str("expected_mime", expected).Msg("MIME type spoofing detected")
				h.writeJSON(w, http.StatusUnsupportedMediaType, APIResponse{
					Success: false,
					Error:   fmt.Sprintf("File spoofing detected! Content (%s) does not match extension %s", contentType, ext),
				})
				return
			}
		}

		// Hard-block dangerous MIME types if they somehow bypass extension check
		// (e.g. uploading a .pdf that is actually an HTML or JS file)
		if ext != ".html" && strings.Contains(contentType, "text/html") {
			h.writeJSON(w, http.StatusUnsupportedMediaType, APIResponse{
				Success: false,
				Error:   "File spoofing detected: File contains HTML/Script payload",
			})
			return
		}
	}

	module := r.PathValue("bucket")
	if module == "" || module == "files" {
		// Fallback for legacy /api/v1/files/upload endpoint
		module = strings.TrimSpace(r.FormValue("module"))
		if module == "" {
			module = strings.TrimSpace(r.FormValue("modul"))
		}
		if module == "" {
			module = "general"
		}
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

	// Add metadata sidecar for data recovery (Orphaned Data Prevention)
	destAbsPath := filepath.Join(h.storageRepo.RootPath(), filepath.FromSlash(relPath))
	_ = h.storageRepo.SaveMetadataSidecar(destAbsPath, *binaryRecord)

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

	// --- Security & Preview Headers ---
	// Prevent browsers from guessing the MIME type (Padding evasion protection)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Strictly sandbox the content to disable JavaScript/ActiveX/Popups (Stored XSS protection)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")

	// Dynamic Content-Disposition (Preview vs Download)
	isDownload := r.URL.Query().Get("download")
	if isDownload == "true" || isDownload == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileMeta.FileName))
	} else {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, fileMeta.FileName))
	}

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

	module := r.PathValue("bucket")
	if module == "" || module == "files" {
		module = strings.TrimSpace(r.FormValue("module"))
		if module == "" {
			module = "general"
		}
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
		file, err := header.Open()
		if err != nil {
			results = append(results, FileResult{FileName: header.Filename, Error: err.Error()})
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

		// Add metadata sidecar
		destAbsPath := filepath.Join(h.storageRepo.RootPath(), filepath.FromSlash(relPath))
		_ = h.storageRepo.SaveMetadataSidecar(destAbsPath, *binaryRecord)

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

func (h *Handler) GeneratePresignedURL(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("method")
	if method == "" { method = "GET" } // Default to GET for viewing files

	path := r.URL.Query().Get("path")
	ref := r.URL.Query().Get("ref")
	binID := r.URL.Query().Get("bin_id")
	module := r.URL.Query().Get("module")

	// Priority 1: bin_id
	if path == "" && binID != "" {
		// Construct path for specific file ID
		path = fmt.Sprintf("/api/v1/files/%s", binID)
	}

	// Priority 2: ref_id (legacy/single file shortcut)
	if path == "" && ref != "" {
		if module == "" { module = "general" }
		path = fmt.Sprintf("/api/v1/%s/view?ref_id=%s", module, ref)
	}

	if path == "" { 
		http.Error(w, "path, bin_id, or ref is required", http.StatusBadRequest)
		return 
	}

	// Forward the download flag into the signed path if present
	isDownload := r.URL.Query().Get("download")
	if isDownload == "true" || isDownload == "1" {
		if strings.Contains(path, "?") {
			path += "&download=1"
		} else {
			path += "?download=1"
		}
	}
	
	// Default expiry to 1 hour if not specified
	expiry := 1 * time.Hour
	expiresInStr := r.URL.Query().Get("expires_in")
	if expiresInStr != "" {
		if secs, err := strconv.Atoi(expiresInStr); err == nil && secs > 0 {
			expiry = time.Duration(secs) * time.Second
		}
	}
	
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	baseURL := "http://" + r.Host
	presignedURL, err := auth.GeneratePresignedURL(method, baseURL, path, h.accessKey, h.secretKey, expiry)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"presigned_url": presignedURL})
}

