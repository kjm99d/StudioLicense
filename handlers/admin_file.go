package handlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
)

const (
	maxFileUploadSize = 100 << 20 // 100 MB
)

var fileStorageBaseDir = filepath.Join("data", "files")

// ListFiles 파일 목록 조회
func ListFiles(w http.ResponseWriter, r *http.Request) {
	queryVals := r.URL.Query()
	search := strings.TrimSpace(queryVals.Get("q"))

	page := parsePositiveInt(queryVals.Get("page"), 1)
	pageSize := parsePositiveInt(queryVals.Get("limit"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	where := []string{}
	args := []interface{}{}
	if search != "" {
		like := "%" + search + "%"
		where = append(where, "(original_name LIKE ? OR uploaded_username LIKE ?)")
		args = append(args, like, like)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM files" + whereClause
	var total int
	if err := database.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to count files", err))
		return
	}

	dataQuery := `SELECT id, original_name, stored_name, description, mime_type, file_size, checksum,
		storage_path, uploaded_by, uploaded_username, created_at, updated_at
		FROM files` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	dataArgs := append(args, pageSize, offset)
	rows, err := database.DB.Query(dataQuery, dataArgs...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query files", err))
		return
	}
	defer rows.Close()

	result := []models.FileAsset{}
	for rows.Next() {
		var file models.FileAsset
		err := rows.Scan(
			&file.ID,
			&file.OriginalName,
			&file.StoredName,
			&file.Description,
			&file.MimeType,
			&file.FileSize,
			&file.Checksum,
			&file.StoragePath,
			&file.UploadedBy,
			&file.UploadedUsername,
			&file.CreatedAt,
			&file.UpdatedAt,
		)
		if err != nil {
			continue
		}
		file.DownloadURL = fmt.Sprintf("/api/admin/files/%s?download=1", file.ID)
		result = append(result, file)
	}

	resp := models.PaginatedResponse{
		Status:  "success",
		Message: "Files retrieved",
		Data:    result,
		Meta: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: total,
			TotalPages: calcTotalPages(total, pageSize),
		},
	}
	json.NewEncoder(w).Encode(resp)
}

// UploadFile 파일 업로드
func UploadFile(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	adminID, _ := r.Context().Value("admin_id").(string)
	username, _ := r.Context().Value("username").(string)

	if adminID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse("Admin context missing", nil))
		return
	}

	if err := os.MkdirAll(fileStorageBaseDir, 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to prepare storage directory", err))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileUploadSize+int64(1<<20))
	if err := r.ParseMultipartForm(maxFileUploadSize); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to parse upload request", err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("File field is required", err))
		return
	}
	defer file.Close()

	originalName := strings.TrimSpace(header.Filename)
	if originalName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Filename is required", nil))
		return
	}
	originalName = filepath.Base(originalName)

	description := strings.TrimSpace(r.FormValue("description"))

	fileID, err := utils.GenerateID("FILE")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to generate file ID", err))
		return
	}

	ext := strings.ToLower(filepath.Ext(originalName))
	storedName := strings.ToLower(fileID) + ext
	subDir := time.Now().Format("2006/01")
	relPath := filepath.Join(subDir, storedName)
	absPath := filepath.Join(fileStorageBaseDir, filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to create storage path", err))
		return
	}

	var reader io.Reader = file
	mimeType := header.Header.Get("Content-Type")

	if seeker, ok := file.(io.ReadSeeker); ok {
		buf := make([]byte, 512)
		n, _ := seeker.Read(buf)
		if mimeType == "" && n > 0 {
			mimeType = http.DetectContentType(buf[:n])
		}
		seeker.Seek(0, io.SeekStart)
	} else {
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		if mimeType == "" && n > 0 {
			mimeType = http.DetectContentType(buf[:n])
		}
		reader = io.MultiReader(bytes.NewReader(buf[:n]), file)
	}

	if mimeType == "" {
		if ext != "" {
			if detected := mime.TypeByExtension(ext); detected != "" {
				mimeType = detected
			}
		}
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	dst, err := os.Create(absPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to store file", err))
		return
	}
	defer dst.Close()

	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(dst, hash), reader)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to save file", err))
		return
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	now := utils.NowSeoul().Format("2006-01-02 15:04:05")
	relPath = filepath.ToSlash(relPath)

	insertQuery := `INSERT INTO files
		(id, original_name, stored_name, description, mime_type, file_size, checksum, storage_path, uploaded_by, uploaded_username, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = database.DB.Exec(insertQuery,
		fileID,
		originalName,
        storedName,
		description,
		mimeType,
		size,
		checksum,
		relPath,
		adminID,
		username,
		now,
		now,
	)
	if err != nil {
		os.Remove(absPath)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to persist file metadata", err))
		return
	}

	asset := models.FileAsset{
		ID:               fileID,
		OriginalName:     originalName,
		StoredName:       storedName,
		Description:      description,
		MimeType:         mimeType,
		FileSize:         size,
		Checksum:         checksum,
		StoragePath:      relPath,
		UploadedBy:       adminID,
		UploadedUsername: username,
		CreatedAt:        now,
		UpdatedAt:        now,
		DownloadURL:      fmt.Sprintf("/api/admin/files/%s?download=1", fileID),
	}

	utils.LogAdminActivity(adminID, username, models.AdminActionUploadFile, fmt.Sprintf("Uploaded file %s (%s)", fileID, originalName))

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"admin_id":   adminID,
		"file_id":    fileID,
		"size":       size,
	}).Info("File uploaded")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("File uploaded successfully", asset))
}

// GetFile 파일 메타 조회 또는 다운로드
func GetFile(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/api/admin/files/")
	if fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("File ID is required", nil))
		return
	}

	file, err := getFileRecord(fileID)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("File not found", nil))
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load file metadata", err))
		return
	}

	if _, download := r.URL.Query()["download"]; download {
		streamFileDownload(w, r, file)
		return
	}

	file.DownloadURL = fmt.Sprintf("/api/admin/files/%s?download=1", file.ID)
	json.NewEncoder(w).Encode(models.SuccessResponse("File retrieved", file))
}

// DeleteFile 파일 삭제
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/api/admin/files/")
	if fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("File ID is required", nil))
		return
	}

	adminID, _ := r.Context().Value("admin_id").(string)
	username, _ := r.Context().Value("username").(string)

	file, err := getFileRecord(fileID)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("File not found", nil))
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load file metadata", err))
		return
	}

	fullPath := filepath.Join(fileStorageBaseDir, filepath.FromSlash(file.StoragePath))
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete file from storage", err))
		return
	}

	if _, err := database.DB.Exec("DELETE FROM files WHERE id = ?", file.ID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete file metadata", err))
		return
	}

	if adminID != "" {
		utils.LogAdminActivity(adminID, username, models.AdminActionDeleteFile, fmt.Sprintf("Deleted file %s (%s)", file.ID, file.OriginalName))
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("File deleted", nil))
}

func getFileRecord(fileID string) (models.FileAsset, error) {
	query := `SELECT id, original_name, stored_name, description, mime_type, file_size, checksum,
		storage_path, uploaded_by, uploaded_username, created_at, updated_at
		FROM files WHERE id = ?`

	var file models.FileAsset
	err := database.DB.QueryRow(query, fileID).Scan(
		&file.ID,
		&file.OriginalName,
		&file.StoredName,
		&file.Description,
		&file.MimeType,
		&file.FileSize,
		&file.Checksum,
		&file.StoragePath,
		&file.UploadedBy,
		&file.UploadedUsername,
		&file.CreatedAt,
		&file.UpdatedAt,
	)

	return file, err
}

func streamFileDownload(w http.ResponseWriter, r *http.Request, file models.FileAsset) {
	fullPath := filepath.Join(fileStorageBaseDir, filepath.FromSlash(file.StoragePath))
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Stored file not found", nil))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to open file", err))
		}
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to stat file", err))
		return
	}

	encodedName := url.PathEscape(file.OriginalName)
	disposition := fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", sanitizeFilename(file.OriginalName), encodedName)
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.Header().Set("Content-Disposition", disposition)

	if adminID, ok := r.Context().Value("admin_id").(string); ok && adminID != "" {
		if username, ok := r.Context().Value("username").(string); ok {
			utils.LogAdminActivity(adminID, username, models.AdminActionDownloadFile, fmt.Sprintf("Downloaded file %s (%s)", file.ID, file.OriginalName))
		}
	}

	http.ServeContent(w, r, file.OriginalName, stat.ModTime(), f)
}

func parsePositiveInt(val string, fallback int) int {
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func calcTotalPages(total, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "\\", "")
	return name
}
