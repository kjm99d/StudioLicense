package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
)

// ListProductFiles returns product-file mappings for a given product or a specific mapping id.
func ListProductFiles(w http.ResponseWriter, r *http.Request) {
	productFileID := strings.TrimSpace(r.URL.Query().Get("id"))
	productID := strings.TrimSpace(r.URL.Query().Get("product_id"))

	switch {
	case productFileID != "":
		item, err := loadProductFileByID(productFileID)
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Product file not found", nil))
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load product file", err))
			return
		}

		json.NewEncoder(w).Encode(models.SuccessResponse("Product file retrieved", item))
		return

	case productID != "":
		query := `SELECT pf.id, pf.product_id, pf.file_id, pf.label, pf.description, pf.sort_order, pf.is_active, pf.delivery_url, pf.created_at, pf.updated_at,
            f.original_name, f.stored_name, f.description, f.mime_type, f.file_size, f.checksum, f.storage_path, f.uploaded_by, f.uploaded_username, f.created_at, f.updated_at
            FROM product_files pf
            JOIN files f ON pf.file_id = f.id
            WHERE pf.product_id = ?
            ORDER BY pf.sort_order ASC, pf.created_at DESC`

		rows, err := database.DB.Query(query, productID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query product files", err))
			return
		}
		defer rows.Close()

		items := []models.ProductFile{}
		for rows.Next() {
			item, err := scanProductFile(rows)
			if err != nil {
				logger.Warn("Failed to scan product file: %v", err)
				continue
			}
			items = append(items, item)
		}

		json.NewEncoder(w).Encode(models.SuccessResponse("Product files retrieved", items))
		return
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("product_id or id is required", nil))
		return
	}
}

// AttachProductFile links an existing file asset to a product.
func AttachProductFile(w http.ResponseWriter, r *http.Request) {
	var req models.AttachProductFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	req.ProductID = strings.TrimSpace(req.ProductID)
	req.FileID = strings.TrimSpace(req.FileID)
	req.Label = strings.TrimSpace(req.Label)
	req.Description = strings.TrimSpace(req.Description)
	req.DeliveryURL = strings.TrimSpace(req.DeliveryURL)

	if req.ProductID == "" || req.FileID == "" || req.Label == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("product_id, file_id, and label are required", nil))
		return
	}

	// Ensure product exists
	var productCount int
	if err := database.DB.QueryRow("SELECT COUNT(*) FROM products WHERE id = ?", req.ProductID).Scan(&productCount); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify product", err))
		return
	}
	if productCount == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product not found", nil))
		return
	}

	// Ensure file exists
	var fileCount int
	if err := database.DB.QueryRow("SELECT COUNT(*) FROM files WHERE id = ?", req.FileID).Scan(&fileCount); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify file", err))
		return
	}
	if fileCount == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("File not found", nil))
		return
	}

	isActive := 1
	if req.IsActive != nil && !*req.IsActive {
		isActive = 0
	}

	createdAt := utils.FormatDateTimeForDB(utils.NowSeoul())
	mappingID, err := utils.GenerateID("pfile")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to generate mapping ID", err))
		return
	}

	_, err = database.DB.Exec(
		`INSERT INTO product_files (id, product_id, file_id, label, description, sort_order, is_active, delivery_url, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mappingID,
		req.ProductID,
		req.FileID,
		req.Label,
		nullIfEmpty(req.Description),
		req.SortOrder,
		isActive,
		nullIfEmpty(req.DeliveryURL),
		createdAt,
		createdAt,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to attach file to product", err))
		return
	}

	item, err := loadProductFileByID(mappingID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load created product file", err))
		return
	}

	if adminID, ok := r.Context().Value("admin_id").(string); ok && adminID != "" {
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionAttachProductFile, fmt.Sprintf("Attached file %s to product %s", req.FileID, req.ProductID))
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("Product file attached", item))
}

// UpdateProductFile updates metadata for a product-file mapping.
func UpdateProductFile(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateProductFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("id is required", nil))
		return
	}

	current, err := loadProductFileByID(req.ID)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product file not found", nil))
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load product file", err))
		return
	}

	setClauses := []string{}
	args := []interface{}{}

	if req.Label != nil {
		label := strings.TrimSpace(*req.Label)
		if label == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("label cannot be empty", nil))
			return
		}
		setClauses = append(setClauses, "label = ?")
		args = append(args, label)
	}

	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		setClauses = append(setClauses, "description = ?")
		args = append(args, nullIfEmpty(description))
	}

	if req.SortOrder != nil {
		setClauses = append(setClauses, "sort_order = ?")
		args = append(args, *req.SortOrder)
	}

	if req.DeliveryURL != nil {
		deliveryURL := strings.TrimSpace(*req.DeliveryURL)
		setClauses = append(setClauses, "delivery_url = ?")
		args = append(args, nullIfEmpty(deliveryURL))
	}

	if req.IsActive != nil {
		if *req.IsActive {
			setClauses = append(setClauses, "is_active = 1")
		} else {
			setClauses = append(setClauses, "is_active = 0")
		}
	}

	if len(setClauses) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("No fields to update", nil))
		return
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, utils.FormatDateTimeForDB(utils.NowSeoul()))
	args = append(args, req.ID)

	query := fmt.Sprintf("UPDATE product_files SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	result, err := database.DB.Exec(query, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update product file", err))
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify update", err))
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product file not found", nil))
		return
	}

	updated, err := loadProductFileByID(req.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load updated product file", err))
		return
	}

	if adminID, ok := r.Context().Value("admin_id").(string); ok && adminID != "" {
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionUpdateProductFile, fmt.Sprintf("Updated product file %s for product %s", req.ID, current.ProductID))
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Product file updated", updated))
}

// DeleteProductFile removes a product-file mapping.
func DeleteProductFile(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("id is required", nil))
		return
	}

	item, err := loadProductFileByID(id)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product file not found", nil))
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load product file", err))
		return
	}

	result, err := database.DB.Exec("DELETE FROM product_files WHERE id = ?", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete product file", err))
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify deletion", err))
		return
	}
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product file not found", nil))
		return
	}

	if adminID, ok := r.Context().Value("admin_id").(string); ok && adminID != "" {
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionDeleteProductFile, fmt.Sprintf("Detached file %s from product %s", item.FileID, item.ProductID))
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Product file detached", nil))
}

func loadProductFileByID(id string) (models.ProductFile, error) {
	query := `SELECT pf.id, pf.product_id, pf.file_id, pf.label, pf.description, pf.sort_order, pf.is_active, pf.delivery_url, pf.created_at, pf.updated_at,
        f.original_name, f.stored_name, f.description, f.mime_type, f.file_size, f.checksum, f.storage_path, f.uploaded_by, f.uploaded_username, f.created_at, f.updated_at
        FROM product_files pf
        JOIN files f ON pf.file_id = f.id
        WHERE pf.id = ?`

	row := database.DB.QueryRow(query, id)
	return scanProductFile(row)
}

func scanProductFile(scanner interface {
	Scan(dest ...interface{}) error
}) (models.ProductFile, error) {
	var (
		pf               models.ProductFile
		description      sql.NullString
		deliveryURL      sql.NullString
		fileDescription  sql.NullString
		fileChecksum     sql.NullString
		uploadedBy       sql.NullString
		uploadedUsername sql.NullString
		isActive         int
		originalName     string
		storedName       string
		mimeType         string
		fileSize         int64
		storagePath      string
		fileCreatedAt    string
		fileUpdatedAt    string
	)

	if err := scanner.Scan(
		&pf.ID,
		&pf.ProductID,
		&pf.FileID,
		&pf.Label,
		&description,
		&pf.SortOrder,
		&isActive,
		&deliveryURL,
		&pf.CreatedAt,
		&pf.UpdatedAt,
		&originalName,
		&storedName,
		&fileDescription,
		&mimeType,
		&fileSize,
		&fileChecksum,
		&storagePath,
		&uploadedBy,
		&uploadedUsername,
		&fileCreatedAt,
		&fileUpdatedAt,
	); err != nil {
		return models.ProductFile{}, err
	}

	pf.Description = stringIfValid(description)
	pf.IsActive = isActive != 0
	pf.DeliveryURL = stringIfValid(deliveryURL)

	asset := &models.FileAsset{
		ID:               pf.FileID,
		OriginalName:     originalName,
		StoredName:       storedName,
		MimeType:         mimeType,
		FileSize:         fileSize,
		StoragePath:      storagePath,
		CreatedAt:        fileCreatedAt,
		UpdatedAt:        fileUpdatedAt,
		DownloadURL:      fmt.Sprintf("/api/admin/files/%s?download=1", pf.FileID),
		Description:      stringIfValid(fileDescription),
		Checksum:         stringIfValid(fileChecksum),
		UploadedBy:       stringIfValid(uploadedBy),
		UploadedUsername: stringIfValid(uploadedUsername),
	}

	pf.File = asset
	return pf, nil
}

func nullIfEmpty(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func stringIfValid(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
