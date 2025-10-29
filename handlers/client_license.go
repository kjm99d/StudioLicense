package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// ActivateLicense는 주어진 라이선스 키에 대해 디바이스 활성화를 생성하거나 재사용합니다.
// @Summary 라이선스 활성화
// @Description 디바이스를 라이선스에 등록하고 정책 및 다운로드 가능한 제품 파일 정보를 반환합니다.
// @Tags 라이선스-클라이언트
// @Accept json
// @Produce json
// @Param request body models.ActivateRequest true "활성화 요청 본문"
// @Success 201 {object} models.APIResponse "활성화 완료"
// @Success 200 {object} models.APIResponse "이미 활성화된 디바이스"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 403 {object} models.APIResponse "라이선스 비활성/만료 또는 디바이스 제한 초과"
// @Failure 404 {object} models.APIResponse "라이선스를 찾을 수 없음"
// @Failure 500 {object} models.APIResponse "서버 내부 오류"
// @Router /api/license/activate [post]
func ActivateLicense(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req models.ActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("Invalid activate request")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"license_key": req.LicenseKey,
		"device_name": req.DeviceInfo.Hostname,
	}).Info("License activation attempt")

	// 라이선스 메타데이터를 조회합니다.
	var license models.License
	var productID sql.NullString
	var policyID sql.NullString
	query := `SELECT l.id, l.license_key, l.product_id, l.policy_id,
		COALESCE(prod.name, '') as product_name,
		l.customer_name, l.max_devices, 
		l.expires_at, l.status
		FROM licenses l
		LEFT JOIN products prod ON l.product_id = prod.id
		WHERE l.license_key = ?`

	err := database.DB.QueryRow(query, req.LicenseKey).Scan(
		&license.ID,
		&license.LicenseKey,
		&productID,
		&policyID,
		&license.ProductName,
		&license.CustomerName,
		&license.MaxDevices,
		&license.ExpiresAt,
		&license.Status,
	)

	if err == sql.ErrNoRows {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
		}).Warn("License not found")

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("License not found", nil))
		return
	}

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"error":       err.Error(),
		}).Error("Failed to query license")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query license", err))
		return
	}

	if productID.Valid {
		license.ProductID = new(string)
		*license.ProductID = productID.String
	}

	if policyID.Valid {
		license.PolicyID = new(string)
		*license.PolicyID = policyID.String
	}

	// 라이선스가 활성 상태이며 만료되지 않았는지 확인합니다.
	if license.Status != models.LicenseStatusActive {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"status":      license.Status,
		}).Warn("License is not active")

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("License is not active", nil))
		return
	}

	if license.IsExpired() {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"expires_at":  license.ExpiresAt,
		}).Warn("License has expired")

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("License has expired", nil))
		return
	}

	// 디바이스 정보를 이용해 핑거프린트를 생성합니다.
	fingerprint := utils.GenerateDeviceFingerprint(
		req.DeviceInfo.ClientID,
		req.DeviceInfo.CPUID,
		req.DeviceInfo.MotherboardSN,
		req.DeviceInfo.MACAddress,
		req.DeviceInfo.DiskSerial,
		req.DeviceInfo.MachineID,
	)

	// 해당 디바이스가 이미 활성화되어 있는지 확인합니다.
	var existingID string
	checkQuery := "SELECT id FROM device_activations WHERE license_id = ? AND device_fingerprint = ? AND status = ?"
	err = database.DB.QueryRow(checkQuery, license.ID, fingerprint, models.DeviceStatusActive).Scan(&existingID)

	if err == nil {
		// 이미 활성화된 디바이스이므로 기존 정보를 그대로 반환합니다.
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"device_id":   existingID,
		}).Info("Device already activated")

		json.NewEncoder(w).Encode(models.SuccessResponse("Device already activated", map[string]interface{}{
			"license_key":   license.LicenseKey,
			"device_id":     existingID,
			"expires_at":    license.ExpiresAt,
			"product_id":    stringValue(license.ProductID),
			"policies":      loadPoliciesForLicense(license.PolicyID),
			"product_files": loadProductFilesForProduct(license.ProductID),
		}))
		return
	}

	if err != nil && err != sql.ErrNoRows {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"error":       err.Error(),
		}).Error("Failed to verify device activation")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify device activation", err))
		return
	}

	// 라이선스에 허용된 활성 디바이스 수를 초과했는지 검사합니다.
	var activeCount int
	countQuery := "SELECT COUNT(*) FROM device_activations WHERE license_id = ? AND status = ?"
	err = database.DB.QueryRow(countQuery, license.ID, models.DeviceStatusActive).Scan(&activeCount)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"error":       err.Error(),
		}).Error("Failed to count active devices")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to count active devices", err))
		return
	}

	if activeCount >= license.MaxDevices {
		logger.WithFields(map[string]interface{}{
			"request_id":   requestID,
			"license_key":  req.LicenseKey,
			"active_count": activeCount,
			"max_devices":  license.MaxDevices,
		}).Warn("Maximum device limit reached")

		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Maximum device limit reached", nil))
		return
	}

	// 디바이스 정보를 JSON 문자열로 직렬화합니다.
	deviceInfoJSON, _ := json.Marshal(req.DeviceInfo)

	// 새로운 디바이스 활성화 데이터를 저장합니다.
	deviceID, _ := utils.GenerateID("dev")
	insertQuery := `INSERT INTO device_activations 
		(id, license_id, device_fingerprint, device_info, device_name, status, activated_at, last_validated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = database.DB.Exec(insertQuery,
		deviceID, license.ID, fingerprint, string(deviceInfoJSON),
		req.DeviceInfo.Hostname, models.DeviceStatusActive, now, now,
	)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"error":       err.Error(),
		}).Error("Failed to activate device")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to activate device", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"license_key": req.LicenseKey,
		"device_id":   deviceID,
		"device_name": req.DeviceInfo.Hostname,
		"customer":    license.CustomerName,
	}).Info("License activated successfully")

	// 디바이스 활동 로그를 남깁니다.
	utils.LogDeviceActivity(deviceID, license.ID, models.DeviceActionActivated, "Device activated")

	policies := loadPoliciesForLicense(license.PolicyID)
	productFiles := loadProductFilesForProduct(license.ProductID)
	productIDValue := stringValue(license.ProductID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("License activated successfully", map[string]interface{}{
		"license_key":   license.LicenseKey,
		"device_id":     deviceID,
		"product_id":    productIDValue,
		"product_name":  license.ProductName,
		"expires_at":    license.ExpiresAt,
		"policies":      policies,
		"product_files": productFiles,
	}))
}

// ValidateLicense는 등록된 디바이스가 라이선스를 사용할 수 있는지 검증합니다.
// @Summary 라이선스 검증
// @Description 라이선스에 등록된 디바이스인지 확인하고 정책 및 제품 파일 정보를 반환합니다.
// @Tags 라이선스-클라이언트
// @Accept json
// @Produce json
// @Param request body models.ValidateRequest true "검증 요청 본문"
// @Success 200 {object} models.APIResponse "검증 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 403 {object} models.APIResponse "라이선스 비활성, 미등록 디바이스 또는 만료"
// @Failure 404 {object} models.APIResponse "라이선스를 찾을 수 없음"
// @Failure 500 {object} models.APIResponse "서버 내부 오류"
// @Router /api/license/validate [post]
func ValidateLicense(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req models.ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"license_key": req.LicenseKey,
	}).Debug("License validation request")

	// 검증을 위해 라이선스 메타데이터를 다시 조회합니다.
	var license models.License
	var productID sql.NullString
	var policyID sql.NullString
	query := `SELECT l.id, l.license_key, l.product_id, l.policy_id,
		COALESCE(prod.name, '') as product_name,
		l.expires_at, l.status
		FROM licenses l
		LEFT JOIN products prod ON l.product_id = prod.id
		WHERE l.license_key = ?`

	err := database.DB.QueryRow(query, req.LicenseKey).Scan(
		&license.ID,
		&license.LicenseKey,
		&productID,
		&policyID,
		&license.ProductName,
		&license.ExpiresAt,
		&license.Status,
	)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("License not found", nil))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query license", err))
		return
	}

	if productID.Valid {
		license.ProductID = new(string)
		*license.ProductID = productID.String
	}

	if policyID.Valid {
		license.PolicyID = new(string)
		*license.PolicyID = policyID.String
	}

	// 라이선스가 활성 상태이며 만료되지 않았는지 확인합니다.
	if license.Status != models.LicenseStatusActive {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("License is not active", nil))
		return
	}

	if license.IsExpired() {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("License has expired", nil))
		return
	}

	// 디바이스 핑거프린트를 생성합니다.
	fingerprint := utils.GenerateDeviceFingerprint(
		req.DeviceInfo.ClientID,
		req.DeviceInfo.CPUID,
		req.DeviceInfo.MotherboardSN,
		req.DeviceInfo.MACAddress,
		req.DeviceInfo.DiskSerial,
		req.DeviceInfo.MachineID,
	)

	// 활성화된 디바이스인지 확인합니다.
	var deviceID string
	deviceQuery := "SELECT id FROM device_activations WHERE license_id = ? AND device_fingerprint = ? AND status = ?"
	err = database.DB.QueryRow(deviceQuery, license.ID, fingerprint, models.DeviceStatusActive).Scan(&deviceID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device not activated", nil))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to verify device", err))
		return
	}

	// 마지막 검증 시각을 갱신합니다.
	updateQuery := "UPDATE device_activations SET last_validated_at = ? WHERE id = ?"
	database.DB.Exec(updateQuery, time.Now().Format("2006-01-02 15:04:05"), deviceID)

	policies := loadPoliciesForLicense(license.PolicyID)
	productFiles := loadProductFilesForProduct(license.ProductID)
	productIDValue := stringValue(license.ProductID)

	json.NewEncoder(w).Encode(models.SuccessResponse("License is valid", map[string]interface{}{
		"license_key":   license.LicenseKey,
		"product_id":    productIDValue,
		"product_name":  license.ProductName,
		"expires_at":    license.ExpiresAt,
		"valid":         true,
		"policies":      policies,
		"product_files": productFiles,
	}))
}

func loadPoliciesForLicense(policyID *string) []models.PolicyResponse {
	if policyID == nil || *policyID == "" {
		return nil
	}

	query := `SELECT id, policy_name, policy_data FROM policies WHERE id = ?`
	rows, err := database.DB.Query(query, *policyID)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"policy_id": *policyID,
			"error":     err.Error(),
		}).Error("Failed to load policies for license")
		return nil
	}
	defer rows.Close()

	policies := []models.PolicyResponse{}
	for rows.Next() {
		var (
			p             models.PolicyResponse
			policyDataStr string
		)
		if err := rows.Scan(&p.ID, &p.PolicyName, &policyDataStr); err != nil {
			logger.Warn("Failed to scan policy row: %v", err)
			continue
		}

		if err := json.Unmarshal([]byte(policyDataStr), &p.PolicyData); err != nil {
			logger.Warn("Failed to parse policy JSON: %v", err)
			continue
		}

		policies = append(policies, p)
	}

	return policies
}

func loadProductFilesForProduct(productID *string) []models.ProductFileResponse {
	if productID == nil || *productID == "" {
		return nil
	}

	query := `SELECT pf.id, pf.file_id, pf.label, pf.description, pf.sort_order, pf.delivery_url, pf.updated_at,
		f.mime_type, f.file_size, f.checksum, f.storage_path
		FROM product_files pf
		JOIN files f ON pf.file_id = f.id
		WHERE pf.product_id = ? AND pf.is_active = 1
		ORDER BY pf.sort_order ASC, pf.created_at DESC`

	rows, err := database.DB.Query(query, *productID)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"product_id": *productID,
			"error":      err.Error(),
		}).Error("Failed to load product files for product")
		return nil
	}
	defer rows.Close()

	files := []models.ProductFileResponse{}
	for rows.Next() {
		var (
			item        models.ProductFileResponse
			description sql.NullString
			deliveryURL sql.NullString
			checksum    sql.NullString
		)

		if err := rows.Scan(
			&item.ID,
			&item.FileID,
			&item.Label,
			&description,
			&item.SortOrder,
			&deliveryURL,
			&item.UpdatedAt,
			&item.MimeType,
			&item.FileSize,
			&checksum,
			&item.StoragePath,
		); err != nil {
			logger.Warn("Failed to scan product file mapping: %v", err)
			continue
		}

		if description.Valid {
			item.Description = description.String
		}

		if deliveryURL.Valid {
			item.DeliveryURL = deliveryURL.String
		}

		if checksum.Valid {
			item.Checksum = checksum.String
		}

		if item.DeliveryURL != "" {
			item.URL = item.DeliveryURL
		}

		if signedQuery, err := utils.GenerateSignedDownloadQuery(item.FileID, 5*time.Minute); err != nil {
			logger.WithFields(map[string]interface{}{
				"product_id": *productID,
				"file_id":    item.FileID,
				"error":      err.Error(),
			}).Error("Failed to generate signed download URL for product file")
			item.DownloadURL = ""
			if item.URL == "" {
				item.URL = ""
			}
		} else {
			signedURL := fmt.Sprintf("/api/license/files/%s?%s", item.FileID, signedQuery)
			item.DownloadURL = signedURL
			if item.URL == "" {
				item.URL = signedURL
			}
		}

		files = append(files, item)
	}

	return files
}

func stringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
