package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// CreateLicense 라이선스 생성
// @Summary 라이선스 생성
// @Description 새로운 라이선스를 생성합니다
// @Tags 관리자 - 라이선스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateLicenseRequest true "라이선스 정보"
// @Success 201 {object} models.APIResponse{data=models.License} "생성 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/licenses [post]
func CreateLicense(w http.ResponseWriter, r *http.Request) {
	var req models.CreateLicenseRequest
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

	// 만료일 검증: 과거 날짜 차단
	now := time.Now().Format("2006-01-02 15:04:05")

	// ISO 8601 형식의 ExpiresAt을 파싱하여 YYYY-MM-DD로 변환
	expiresTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		// 다른 형식일 수도 있으니 파싱 시도
		expiresTime, err = time.Parse("2006-01-02", req.ExpiresAt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Invalid expiration date format", err))
			return
		}
	}

	expiresAtStr := expiresTime.Format("2006-01-02")
	if expiresAtStr < time.Now().Format("2006-01-02") {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Expiration date cannot be in the past", nil))
		return
	}

	// 제품 ID는 필수이며 활성 제품이어야 합니다.
	req.ProductID = strings.TrimSpace(req.ProductID)
	if req.ProductID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Product ID is required", nil))
		return
	}

	var productName string
	productQuery := "SELECT name FROM products WHERE id = ? AND status = 'active'"
	if err := database.DB.QueryRow(productQuery, req.ProductID).Scan(&productName); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Product not found or inactive", nil))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify product", err))
		return
	}
	productID := req.ProductID
	productIDPtr := &productID

	// 정책 ID가 있으면 정책 존재 여부만 확인
	var policyID *string
	if req.PolicyID != "" {
		var policyExists int
		query := "SELECT COUNT(*) FROM policies WHERE id = ?"
		err := database.DB.QueryRow(query, req.PolicyID).Scan(&policyExists)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify policy", err))
			return
		}

		if policyExists == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Policy not found", nil))
			return
		}

		policyID = &req.PolicyID
	}

	// 라이선스 키 생성
	licenseKey, err := utils.GenerateLicenseKey()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to generate license key", err))
		return
	}

	// ID 생성
	id, err := utils.GenerateID("lic")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to generate ID", err))
		return
	}

	// DB에 저장
	query := `
		INSERT INTO licenses (id, license_key, product_id, policy_id, customer_name, 
			customer_email, max_devices, expires_at, status, created_by, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = database.DB.Exec(query,
		id, licenseKey, productID, policyID, req.CustomerName,
		req.CustomerEmail, req.MaxDevices, expiresAtStr, models.LicenseStatusActive, creatorID,
		req.Notes, now, now,
	)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"license_id": id,
			"customer":   req.CustomerName,
			"expires_at": req.ExpiresAt,
			"query":      query,
		}).Error("Failed to create license")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to create license", err))
		return
	}

	// 생성된 라이선스 조회
	license := models.License{
		ID:            id,
		LicenseKey:    licenseKey,
		ProductID:     productIDPtr,
		PolicyID:      policyID,
		ProductName:   productName,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		MaxDevices:    req.MaxDevices,
		ExpiresAt:     expiresAtStr,
		Status:        models.LicenseStatusActive,
		CreatedBy:     creatorID,
		Notes:         req.Notes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("License created successfully", license))

	// 관리자 활동 로그
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionCreateLicense, "License created: "+license.ID)
	}
}

// GetLicenses 라이선스 목록 조회
// @Summary 라이선스 목록 조회
// @Description 라이선스 목록을 페이징하여 조회합니다
// @Tags 관리자 - 라이선스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "페이지 번호" default(1)
// @Param page_size query int false "페이지 크기" default(20)
// @Param status query string false "상태 필터 (active, expired, revoked)"
// @Param search query string false "검색어 (라이선스 키, 고객명, 이메일)"
// @Success 200 {object} models.PaginatedResponse{data=[]models.License} "조회 성공"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/licenses [get]
func GetLicenses(w http.ResponseWriter, r *http.Request) {
	// 쿼리 파라미터
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	scope, isSuper, adminID, err := resolveResourceScope(r, models.ResourceTypeLicenses)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate license permissions", err))
		return
	}

	// 전체 개수 조회
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM licenses WHERE 1=1"
	countArgs := make([]interface{}, 0)

	if status != "" {
		countQuery += " AND status = ?"
		countArgs = append(countArgs, status)
	}
	if search != "" {
		countQuery += " AND (license_key LIKE ? OR customer_name LIKE ? OR customer_email LIKE ?)"
		searchPattern := "%" + search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
	}
	if !isSuper {
		filterSQL, filterArgs := utils.BuildResourceFilter(scope, "id", "created_by", adminID)
		countQuery += filterSQL
		countArgs = append(countArgs, filterArgs...)
	}

	database.DB.QueryRow(countQuery, countArgs...).Scan(&totalCount)

	// 데이터 조회
	offset := (page - 1) * pageSize
	query := `SELECT l.id, l.license_key, l.product_id, l.policy_id,
		COALESCE(prod.name, '') as product_name,
		COALESCE(pol.policy_name, '') as policy_name,
		l.customer_name, l.customer_email, l.max_devices,
	COALESCE((SELECT COUNT(*) FROM device_activations WHERE license_id = l.id AND status = 'active'), 0) as active_devices,
	l.expires_at, l.status, l.created_by, l.notes, l.created_at, l.updated_at 
		FROM licenses l
		LEFT JOIN products prod ON l.product_id = prod.id
		LEFT JOIN policies pol ON l.policy_id = pol.id
		WHERE 1=1`

	dataArgs := make([]interface{}, 0)
	if status != "" {
		query += " AND l.status = ?"
		dataArgs = append(dataArgs, status)
	}
	if search != "" {
		query += " AND (l.license_key LIKE ? OR l.customer_name LIKE ? OR l.customer_email LIKE ?)"
		searchPattern := "%" + search + "%"
		dataArgs = append(dataArgs, searchPattern, searchPattern, searchPattern)
	}
	if !isSuper {
		filterSQL, filterArgs := utils.BuildResourceFilter(scope, "l.id", "l.created_by", adminID)
		query += filterSQL
		dataArgs = append(dataArgs, filterArgs...)
	}

	query += " ORDER BY l.created_at DESC LIMIT ? OFFSET ?"
	dataArgs = append(dataArgs, pageSize, offset)

	rows, err := database.DB.Query(query, dataArgs...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query licenses", err))
		return
	}
	defer rows.Close()

	licenses := []models.License{}
	for rows.Next() {
		var license models.License
		err := rows.Scan(
			&license.ID, &license.LicenseKey, &license.ProductID, &license.PolicyID, &license.ProductName,
			&license.PolicyName, &license.CustomerName, &license.CustomerEmail, &license.MaxDevices,
			&license.ActiveDevices,
			&license.ExpiresAt, &license.Status, &license.CreatedBy, &license.Notes,
			&license.CreatedAt, &license.UpdatedAt,
		)
		if err != nil {
			continue
		}
		license.ExpiresAt = normalizeDateOnly(license.ExpiresAt)
		licenses = append(licenses, license)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	response := models.PaginatedResponse{
		Status:  "success",
		Message: "Licenses retrieved",
		Data:    licenses,
		Meta: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
			TotalCount: totalCount,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// licenseIDFromRequest extracts the license ID from context, path, or query parameters.
func licenseIDFromRequest(r *http.Request) string {
	if id, _ := r.Context().Value("path_license_id").(string); strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	if id := r.PathValue("license_id"); strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return strings.TrimSpace(r.URL.Query().Get("id"))
}

// GetLicense 라이선스 상세 조회
// @Summary 라이선스 상세 조회
// @Description 특정 라이선스의 상세 정보를 조회합니다
// @Tags 관리자 - 라이선스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "라이선스 ID"
// @Success 200 {object} models.APIResponse{data=models.License} "조회 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 404 {object} models.APIResponse "라이선스 없음"
// @Router /api/admin/licenses/{id} [get]
func GetLicense(w http.ResponseWriter, r *http.Request) {
	id := licenseIDFromRequest(r)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("License ID is required", nil))
		return
	}

	scope, isSuper, adminID, err := resolveResourceScope(r, models.ResourceTypeLicenses)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate license permissions", err))
		return
	}

	if !isSuper && strings.EqualFold(scope.Mode, models.ResourceModeNone) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: license access denied", nil))
		return
	}
	if !isSuper && strings.EqualFold(scope.Mode, models.ResourceModeCustom) && !utils.CanAccessResource(scope, id, "", adminID) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: license access denied", nil))
		return
	}

	var license models.License
	query := `SELECT l.id, l.license_key, l.product_id, l.policy_id,
		COALESCE(prod.name, '') as product_name,
		COALESCE(pol.policy_name, '') as policy_name,
		l.customer_name, l.customer_email, l.max_devices,
		COALESCE((SELECT COUNT(*) FROM device_activations WHERE license_id = l.id AND status = 'active'), 0) as active_devices,
		l.expires_at, l.status, l.created_by, l.notes, l.created_at, l.updated_at 
		FROM licenses l
		LEFT JOIN products prod ON l.product_id = prod.id
		LEFT JOIN policies pol ON l.policy_id = pol.id
		WHERE l.id = ?`
	args := []interface{}{id}
	if !isSuper && strings.EqualFold(scope.Mode, models.ResourceModeOwn) {
		query += " AND l.created_by = ?"
		args = append(args, adminID)
	}

	err = database.DB.QueryRow(query, args...).Scan(
		&license.ID, &license.LicenseKey, &license.ProductID, &license.PolicyID, &license.ProductName,
		&license.PolicyName, &license.CustomerName, &license.CustomerEmail, &license.MaxDevices,
		&license.ActiveDevices,
		&license.ExpiresAt, &license.Status, &license.CreatedBy, &license.Notes,
		&license.CreatedAt, &license.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("License not found", nil))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to retrieve license", err))
		return
	}

	if !isSuper && !utils.CanAccessResource(scope, license.ID, license.CreatedBy, adminID) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: license access denied", nil))
		return
	}

	license.ExpiresAt = normalizeDateOnly(license.ExpiresAt)

	json.NewEncoder(w).Encode(models.SuccessResponse("License retrieved", license))
}

// UpdateLicense 라이선스 수정
func UpdateLicense(w http.ResponseWriter, r *http.Request) {
	id := licenseIDFromRequest(r)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("License ID is required", nil))
		return
	}

	var req models.UpdateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	// ExpiresAt 형식 변환 (필요한 경우)
	var expiresAtStr string = req.ExpiresAt
	if req.ExpiresAt != "" {
		expiresTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			expiresTime, err = time.Parse("2006-01-02", req.ExpiresAt)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(models.ErrorResponse("Invalid expiration date format", err))
				return
			}
		}
		expiresAtStr = expiresTime.Format("2006-01-02")
	}

	// 정책 ID 검증 (빈 문자열이 아닌 경우만)
	var policyID *string
	if req.PolicyID != "" {
		var policyExists int
		query := "SELECT COUNT(*) FROM policies WHERE id = ?"
		err := database.DB.QueryRow(query, req.PolicyID).Scan(&policyExists)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify policy", err))
			return
		}

		if policyExists == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Policy not found", nil))
			return
		}

		policyID = &req.PolicyID
	}

	// 최대 디바이스 수 검증: 현재 활성 디바이스 수보다 작게 설정 불가
	if req.MaxDevices > 0 {
		var activeCount int
		countQuery := "SELECT COUNT(*) FROM device_activations WHERE license_id = ? AND status = ?"
		err := database.DB.QueryRow(countQuery, id, models.DeviceStatusActive).Scan(&activeCount)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to check active devices", err))
			return
		}

		if req.MaxDevices < activeCount {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse(
				fmt.Sprintf("Cannot reduce max devices to %d. Currently %d devices are active. Please deactivate devices first.",
					req.MaxDevices, activeCount),
				nil))
			return
		}
	}

	// policy_id 포함한 업데이트 쿼리
	query := `UPDATE licenses SET customer_name = ?,
		customer_email = ?, max_devices = ?, expires_at = ?, notes = ?, policy_id = ?, updated_at = ?
		WHERE id = ?`

	_, err := database.DB.Exec(query,
		req.CustomerName,
		req.CustomerEmail, req.MaxDevices, expiresAtStr, req.Notes,
		policyID,
		time.Now().Format("2006-01-02 15:04:05"), id,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update license", err))
		return
	}

	// 만료일 변경 시 상태 자동 조정
	if req.ExpiresAt != "" {
		now := time.Now().Format("2006-01-02 15:04:05")

		// 현재 상태 확인
		var currentStatus, licenseKey string
		statusQuery := "SELECT status, license_key FROM licenses WHERE id = ?"
		database.DB.QueryRow(statusQuery, id).Scan(&currentStatus, &licenseKey)

		// 관리자 정보 가져오기
		var adminID, username string
		if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
			adminID = adminIDRaw.(string)
		}
		if usernameRaw := r.Context().Value("username"); usernameRaw != nil {
			username = usernameRaw.(string)
		}

		// 만료일이 미래이면서 만료 상태인 경우 -> 활성화
		if expiresAtStr > now[:10] && currentStatus == models.LicenseStatusExpired {
			updateStatusQuery := "UPDATE licenses SET status = ?, updated_at = ? WHERE id = ?"
			database.DB.Exec(updateStatusQuery, models.LicenseStatusActive, now, id)

			logger.WithFields(map[string]interface{}{
				"license_id": id,
				"old_status": currentStatus,
				"new_status": models.LicenseStatusActive,
				"expires_at": expiresAtStr,
			}).Info("License auto-reactivated due to expiration date extension")

			// 활동 로그 기록
			details := fmt.Sprintf("라이선스 키: %s, 만료일 연장으로 자동 재활성화 (만료일: %s)", licenseKey, expiresAtStr)
			utils.LogAdminActivity(adminID, username, "라이선스 재활성화", details)
		}

		// 만료일이 과거이면서 활성 상태인 경우 -> 만료 처리
		if expiresAtStr < now[:10] && currentStatus == models.LicenseStatusActive {
			updateStatusQuery := "UPDATE licenses SET status = ?, updated_at = ? WHERE id = ?"
			database.DB.Exec(updateStatusQuery, models.LicenseStatusExpired, now, id)

			logger.WithFields(map[string]interface{}{
				"license_id": id,
				"old_status": currentStatus,
				"new_status": models.LicenseStatusExpired,
				"expires_at": expiresAtStr,
			}).Info("License auto-expired due to past expiration date")

			// 활동 로그 기록
			details := fmt.Sprintf("라이선스 키: %s, 과거 만료일 설정으로 자동 만료 처리 (만료일: %s)", licenseKey, expiresAtStr)
			utils.LogAdminActivity(adminID, username, "라이선스 만료 처리", details)
		}
	}

	// 라이선스 수정 활동 로그 기록
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)

		// 변경 내역을 상세히 기록
		var changes []string
		if req.CustomerName != "" {
			changes = append(changes, fmt.Sprintf("고객명: %s", req.CustomerName))
		}
		if req.MaxDevices > 0 {
			changes = append(changes, fmt.Sprintf("최대 디바이스: %d", req.MaxDevices))
		}
		if req.ExpiresAt != "" {
			changes = append(changes, fmt.Sprintf("만료일: %s", expiresAtStr))
		}

		details := fmt.Sprintf("라이선스 ID: %s | 변경사항: %s", id, strings.Join(changes, " | "))
		utils.LogAdminActivity(adminID, username, "라이선스 수정", details)
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("License updated successfully", nil))
}

// DeleteLicense 라이선스 삭제
func DeleteLicense(w http.ResponseWriter, r *http.Request) {
	id := licenseIDFromRequest(r)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("License ID is required", nil))
		return
	}

	query := "DELETE FROM licenses WHERE id = ?"
	_, err := database.DB.Exec(query, id)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete license", err))
		return
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("License deleted successfully", nil))

	// 관리자 활동 로그
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionDeleteLicense, "License deleted: "+id)
	}
}

// RevokeLicense 라이선스 폐기
func RevokeLicense(w http.ResponseWriter, r *http.Request) {
	id := licenseIDFromRequest(r)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("License ID is required", nil))
		return
	}

	query := "UPDATE licenses SET status = ?, updated_at = ? WHERE id = ?"
	_, err := database.DB.Exec(query, models.LicenseStatusRevoked, time.Now().Format("2006-01-02 15:04:05"), id)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to revoke license", err))
		return
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("License revoked successfully", nil))
}

// GetLicenseDevices 라이선스의 활성화된 디바이스 목록
func GetLicenseDevices(w http.ResponseWriter, r *http.Request) {
	licenseID := r.URL.Query().Get("id")
	if licenseID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("License ID is required", nil))
		return
	}

	scope, isSuper, adminID, err := resolveResourceScope(r, models.ResourceTypeLicenses)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate license permissions", err))
		return
	}

	var owner string
	if err := database.DB.QueryRow("SELECT created_by FROM licenses WHERE id = ?", licenseID).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("License not found", nil))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load license", err))
		return
	}

	if !isSuper && !utils.CanAccessResource(scope, licenseID, owner, adminID) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: license access denied", nil))
		return
	}

	query := `SELECT id, license_id, device_fingerprint, device_info, device_name, 
		status, activated_at, last_validated_at, deactivated_at 
		FROM device_activations WHERE license_id = ? ORDER BY activated_at DESC`

	rows, err := database.DB.Query(query, licenseID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query devices", err))
		return
	}
	defer rows.Close()

	devices := []models.DeviceActivation{}
	for rows.Next() {
		var device models.DeviceActivation
		err := rows.Scan(
			&device.ID, &device.LicenseID, &device.DeviceFingerprint,
			&device.DeviceInfo, &device.DeviceName, &device.Status,
			&device.ActivatedAt, &device.LastValidatedAt, &device.DeactivatedAt,
		)
		if err != nil {
			continue
		}
		devices = append(devices, device)
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Devices retrieved", devices))
}
func normalizeDateOnly(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Format("2006-01-02")
		}
	}
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}
