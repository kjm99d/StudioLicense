package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// CreateProduct 제품 생성
// @Summary 제품 생성
// @Description 새로운 제품을 생성합니다
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateProductRequest true "제품 정보"
// @Success 201 {object} models.APIResponse{data=models.Product} "생성 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products [post]
func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	adminIDVal := r.Context().Value("admin_id")
	creatorID, _ := adminIDVal.(string)
	if strings.TrimSpace(creatorID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse("Missing creator context", nil))
		return
	}

	id, _ := utils.GenerateID("prod")
	now := time.Now().Format("2006-01-02 15:04:05")

	// 버전은 사용하지 않으므로 빈 문자열로 저장
	query := `INSERT INTO products (id, name, description, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := database.DB.Exec(query, id, req.Name, req.Description,
		models.ProductStatusActive, creatorID, now, now)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"name":  req.Name,
		}).Error("Failed to create product")

		// UNIQUE 제약 위반 확인 (SQLite와 MySQL 모두 지원)
		errMsg := err.Error()
		if strings.Contains(errMsg, "UNIQUE constraint failed") ||
			strings.Contains(errMsg, "Duplicate entry") ||
			strings.Contains(errMsg, "1062") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(models.ErrorResponse("이미 존재하는 제품명입니다", nil))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to create product", err))
		return
	}

	product := models.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      models.ProductStatusActive,
		CreatedBy:   creatorID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	logger.WithFields(map[string]interface{}{
		"product_id": id,
		"name":       req.Name,
	}).Info("Product created")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("Product created successfully", product))

	// 관리자 활동 로그
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionCreateProduct, "Product created: "+id)
	}
}

// GetProducts 제품 목록 조회
// @Summary 제품 목록 조회
// @Description 모든 제품 목록을 조회합니다
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "상태 필터 (active, inactive)"
// @Success 200 {object} models.APIResponse{data=[]models.Product} "조회 성공"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products [get]
func GetProducts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	scope, isSuper, adminID, err := resolveResourceScope(r, models.ResourceTypeProducts)
	if err != nil {
		logger.Error("Failed to evaluate product permissions: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate product permissions", err))
		return
	}

	query := "SELECT id, name, description, status, created_by, created_at, updated_at FROM products WHERE 1=1"
	args := make([]interface{}, 0)

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if !isSuper {
		filterSQL, filterArgs := utils.BuildResourceFilter(scope, "id", "created_by", adminID)
		query += filterSQL
		args = append(args, filterArgs...)
	}

	query += " ORDER BY created_at DESC"

	logger.Debug("GetProducts query: %s, args: %v", query, args)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		logger.Error("Failed to query products: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query products", err))
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var product models.Product
		var createdBy sql.NullString
		err := rows.Scan(&product.ID, &product.Name, &product.Description,
			&product.Status, &createdBy, &product.CreatedAt, &product.UpdatedAt)
		if err != nil {
			logger.Warn("Failed to scan product: %v", err)
			continue
		}
		if createdBy.Valid {
			product.CreatedBy = createdBy.String
		}
		products = append(products, product)
	}

	logger.Info("Retrieved %d products", len(products))
	json.NewEncoder(w).Encode(models.SuccessResponse("Products retrieved", products))
}

// GetProduct 제품 상세 조회
// @Summary 제품 상세 조회
// @Description 특정 제품의 상세 정보를 조회합니다
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "제품 ID"
// @Success 200 {object} models.APIResponse{data=models.Product} "조회 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 404 {object} models.APIResponse "제품 없음"
// @Router /api/admin/products/ [get]
func GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product ID is required", nil))
		return
	}

	scope, isSuper, adminID, err := resolveResourceScope(r, models.ResourceTypeProducts)
	if err != nil {
		logger.Error("Failed to evaluate product permissions: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate product permissions", err))
		return
	}
	if !isSuper && strings.EqualFold(scope.Mode, models.ResourceModeNone) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: product access denied", nil))
		return
	}
	if !isSuper && strings.EqualFold(scope.Mode, models.ResourceModeCustom) && !utils.CanAccessResource(scope, id, "", adminID) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: product access denied", nil))
		return
	}

	var product models.Product
	query := "SELECT id, name, description, status, created_by, created_at, updated_at FROM products WHERE id = ?"
	args := []interface{}{id}
	if !isSuper && strings.EqualFold(scope.Mode, models.ResourceModeOwn) {
		query += " AND created_by = ?"
		args = append(args, adminID)
	}

	var createdBy sql.NullString
	err = database.DB.QueryRow(query, args...).Scan(&product.ID, &product.Name,
		&product.Description, &product.Status, &createdBy, &product.CreatedAt, &product.UpdatedAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product not found", nil))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to retrieve product", err))
		return
	}

	if createdBy.Valid {
		product.CreatedBy = createdBy.String
	}

	if !isSuper && !utils.CanAccessResource(scope, product.ID, product.CreatedBy, adminID) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: product access denied", nil))
		return
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Product retrieved", product))
}

// UpdateProduct 제품 수정
// @Summary 제품 수정
// @Description 제품 정보를 수정합니다
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "제품 ID"
// @Param request body models.UpdateProductRequest true "수정할 정보"
// @Success 200 {object} models.APIResponse "수정 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products/ [put]
func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product ID is required", nil))
		return
	}

	var req models.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	query := `UPDATE products SET name = ?, description = ?, status = ?, updated_at = ?
		WHERE id = ?`

	_, err := database.DB.Exec(query, req.Name, req.Description, req.Status, time.Now().Format("2006-01-02 15:04:05"), id)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"product_id": id,
			"name":       req.Name,
		}).Error("Failed to update product")

		// UNIQUE 제약 위반 확인 (SQLite와 MySQL 모두 지원)
		errMsg := err.Error()
		if strings.Contains(errMsg, "UNIQUE constraint failed") ||
			strings.Contains(errMsg, "Duplicate entry") ||
			strings.Contains(errMsg, "1062") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(models.ErrorResponse("이미 존재하는 제품명입니다", nil))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update product", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"product_id": id,
	}).Info("Product updated")
	json.NewEncoder(w).Encode(models.SuccessResponse("Product updated successfully", nil))

	// 관리자 활동 로그
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionUpdateProduct, "Product updated: "+id)
	}
}

// DeleteProduct 제품 삭제
// @Summary 제품 삭제
// @Description 제품을 삭제합니다. 연결된 라이선스가 있으면 삭제가 제한됩니다(RESTRICT).
// @Tags 제품
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "제품 ID"
// @Success 200 {object} models.APIResponse "삭제 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 409 {object} models.APIResponse "연결된 라이선스로 인해 삭제 불가"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/products/ [delete]
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product ID is required", nil))
		return
	}

	// 라이선스 참조 존재 여부 확인 (RESTRICT 동작)
	var count int
	if err := database.DB.QueryRow("SELECT COUNT(*) FROM licenses WHERE product_id = ?", id).Scan(&count); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to check linked licenses", err))
		return
	}

	if count > 0 {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(models.ErrorResponse("Cannot delete product: linked licenses exist", nil))
		return
	}

	query := "DELETE FROM products WHERE id = ?"
	_, err := database.DB.Exec(query, id)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete product", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"product_id": id,
	}).Info("Product deleted")
	json.NewEncoder(w).Encode(models.SuccessResponse("Product deleted successfully", nil))

	// 관리자 활동 로그
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionDeleteProduct, "Product deleted: "+id)
	}
}
