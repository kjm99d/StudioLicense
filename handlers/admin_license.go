package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

	// 제품 ID가 있으면 제품 정보 조회
	var productID *string
	if req.ProductID != "" {
		var productName string
		query := "SELECT name FROM products WHERE id = ? AND status = 'active'"
		err := database.DB.QueryRow(query, req.ProductID).Scan(&productName)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Product not found or inactive", nil))
			return
		}

		if err == nil {
			productID = &req.ProductID
			req.ProductName = productName
			// 제품 버전은 사용하지 않음
		}
	}

	// 정책 ID가 있으면 정책 정보 조회 및 검증
	var policyID *string
	if req.PolicyID != "" {
		var policyStatus string
		query := "SELECT status FROM policies WHERE id = ?"
		err := database.DB.QueryRow(query, req.PolicyID).Scan(&policyStatus)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Policy not found", nil))
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify policy", err))
			return
		}

		// 정책이 inactive 상태면 사용 불가
		if policyStatus != models.PolicyStatusActive {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Policy is not active", nil))
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
		INSERT INTO licenses (id, license_key, product_id, policy_id, product_name, customer_name, 
			customer_email, max_devices, expires_at, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = database.DB.Exec(query,
		id, licenseKey, productID, policyID, req.ProductName, req.CustomerName,
		req.CustomerEmail, req.MaxDevices, expiresAtStr, models.LicenseStatusActive,
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
		ProductID:     productID,
		PolicyID:      policyID,
		ProductName:   req.ProductName,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		MaxDevices:    req.MaxDevices,
		ExpiresAt:     expiresAtStr,
		Status:        models.LicenseStatusActive,
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

	// 전체 개수 조회
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM licenses WHERE 1=1"
	var args []interface{}

	if status != "" {
		countQuery += " AND status = ?"
		args = append(args, status)
	}
	if search != "" {
		countQuery += " AND (license_key LIKE ? OR customer_name LIKE ? OR customer_email LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	database.DB.QueryRow(countQuery, args...).Scan(&totalCount)

	// 데이터 조회
	offset := (page - 1) * pageSize
	query := `SELECT l.id, l.license_key, l.product_id, l.policy_id, l.product_name, 
		COALESCE(p.policy_name, '') as policy_name,
		l.customer_name, l.customer_email, l.max_devices,
		COALESCE((SELECT COUNT(*) FROM device_activations WHERE license_id = l.id AND status = 'active'), 0) as active_devices,
		l.expires_at, l.status, l.notes, l.created_at, l.updated_at 
		FROM licenses l
		LEFT JOIN policies p ON l.policy_id = p.id
		WHERE 1=1`

	if status != "" {
		query += " AND status = ?"
	}
	if search != "" {
		query += " AND (license_key LIKE ? OR customer_name LIKE ? OR customer_email LIKE ?)"
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := database.DB.Query(query, args...)
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
			&license.ExpiresAt, &license.Status, &license.Notes,
			&license.CreatedAt, &license.UpdatedAt,
		)
		if err != nil {
			continue
		}
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

// GetLicense 라이선스 상세 조회
// @Summary 라이선스 상세 조회
// @Description 특정 라이선스의 상세 정보를 조회합니다
// @Tags 관리자 - 라이선스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id query string true "라이선스 ID"
// @Success 200 {object} models.APIResponse{data=models.License} "조회 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 404 {object} models.APIResponse "라이선스 없음"
// @Router /api/admin/licenses/ [get]
func GetLicense(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("License ID is required", nil))
		return
	}

	var license models.License
	query := `SELECT l.id, l.license_key, l.product_id, l.policy_id, l.product_name, 
		COALESCE(p.policy_name, '') as policy_name,
		l.customer_name, l.customer_email, l.max_devices,
		COALESCE((SELECT COUNT(*) FROM device_activations WHERE license_id = l.id AND status = 'active'), 0) as active_devices,
		l.expires_at, l.status, l.notes, l.created_at, l.updated_at 
		FROM licenses l
		LEFT JOIN policies p ON l.policy_id = p.id
		WHERE l.id = ?`

	err := database.DB.QueryRow(query, id).Scan(
		&license.ID, &license.LicenseKey, &license.ProductID, &license.PolicyID, &license.ProductName,
		&license.PolicyName, &license.CustomerName, &license.CustomerEmail, &license.MaxDevices,
		&license.ActiveDevices,
		&license.ExpiresAt, &license.Status, &license.Notes,
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

	json.NewEncoder(w).Encode(models.SuccessResponse("License retrieved", license))
}

// UpdateLicense 라이선스 수정
func UpdateLicense(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
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
		var policyStatus string
		query := "SELECT status FROM policies WHERE id = ?"
		err := database.DB.QueryRow(query, req.PolicyID).Scan(&policyStatus)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Policy not found", nil))
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify policy", err))
			return
		}

		// 정책이 inactive 상태면 사용 불가
		if policyStatus != models.PolicyStatusActive {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse("Policy is not active", nil))
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
	query := `UPDATE licenses SET product_name = ?, customer_name = ?,
		customer_email = ?, max_devices = ?, expires_at = ?, notes = ?, policy_id = ?, updated_at = ?
		WHERE id = ?`

	_, err := database.DB.Exec(query,
		req.ProductName, req.CustomerName,
		req.CustomerEmail, req.MaxDevices, expiresAtStr, req.Notes,
		policyID,
		time.Now().Format("2006-01-02 15:04:05"), id,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update license", err))
		return
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("License updated successfully", nil))
}

// DeleteLicense 라이선스 삭제
func DeleteLicense(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
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
	id := r.URL.Query().Get("id")
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
