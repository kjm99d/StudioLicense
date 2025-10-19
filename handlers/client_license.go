package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// ActivateLicense 라이선스 활성화 (최초 등록)
// @Summary 라이선스 활성화
// @Description 라이선스 키를 사용하여 디바이스를 활성화합니다
// @Tags 클라이언트 - 라이선스
// @Accept json
// @Produce json
// @Param request body models.ActivateRequest true "활성화 정보"
// @Success 201 {object} models.APIResponse "활성화 성공"
// @Success 200 {object} models.APIResponse "이미 활성화됨"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 403 {object} models.APIResponse "라이선스 비활성/만료/기기수 초과"
// @Failure 404 {object} models.APIResponse "라이선스 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
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

	// 라이선스 조회
	var license models.License
	query := `SELECT id, license_key, product_name, customer_name, max_devices, 
		expires_at, status FROM licenses WHERE license_key = ?`

	err := database.DB.QueryRow(query, req.LicenseKey).Scan(
		&license.ID, &license.LicenseKey, &license.ProductName,
		&license.CustomerName, &license.MaxDevices, &license.ExpiresAt, &license.Status,
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

	// 라이선스 유효성 검사
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

	// 디바이스 핑거프린트 생성
	fingerprint := utils.GenerateDeviceFingerprint(
		req.DeviceInfo.ClientID,
		req.DeviceInfo.CPUID,
		req.DeviceInfo.MotherboardSN,
		req.DeviceInfo.MACAddress,
		req.DeviceInfo.DiskSerial,
		req.DeviceInfo.MachineID,
	)

	// 이미 활성화된 디바이스인지 확인
	var existingID string
	checkQuery := "SELECT id FROM device_activations WHERE license_id = ? AND device_fingerprint = ? AND status = ?"
	err = database.DB.QueryRow(checkQuery, license.ID, fingerprint, models.DeviceStatusActive).Scan(&existingID)

	if err == nil {
		// 이미 활성화됨
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"license_key": req.LicenseKey,
			"device_id":   existingID,
		}).Info("Device already activated")

		json.NewEncoder(w).Encode(models.SuccessResponse("Device already activated", map[string]interface{}{
			"license_key": license.LicenseKey,
			"device_id":   existingID,
			"expires_at":  license.ExpiresAt,
		}))
		return
	}

	// 현재 활성화된 디바이스 수 확인
	var activeCount int
	countQuery := "SELECT COUNT(*) FROM device_activations WHERE license_id = ? AND status = ?"
	database.DB.QueryRow(countQuery, license.ID, models.DeviceStatusActive).Scan(&activeCount)

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

	// 디바이스 정보 JSON 직렬화
	deviceInfoJSON, _ := json.Marshal(req.DeviceInfo)

	// 디바이스 활성화
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

	// 활동 로그 기록
	utils.LogDeviceActivity(deviceID, license.ID, models.DeviceActionActivated, "Device activated")

	// 제품 ID 조회 및 정책 정보 가져오기
	var productID string
	prodQuery := "SELECT product_id FROM licenses WHERE id = ?"
	database.DB.QueryRow(prodQuery, license.ID).Scan(&productID)

	var policies []models.PolicyResponse
	if productID != "" {
		policyQuery := `SELECT id, policy_name, policy_data FROM policies WHERE product_id = ? AND status = ?`
		rows, err := database.DB.Query(policyQuery, productID, models.PolicyStatusActive)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p models.PolicyResponse
				var policyDataStr string
				err := rows.Scan(&p.ID, &p.PolicyName, &policyDataStr)
				if err == nil {
					// JSON 문자열을 interface{}로 파싱
					json.Unmarshal([]byte(policyDataStr), &p.PolicyData)
					policies = append(policies, p)
				}
			}
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("License activated successfully", map[string]interface{}{
		"license_key":  license.LicenseKey,
		"device_id":    deviceID,
		"product_name": license.ProductName,
		"expires_at":   license.ExpiresAt,
		"policies":     policies,
	}))
}

// ValidateLicense 라이선스 검증 (앱 실행 시)
// @Summary 라이선스 검증
// @Description 앱 실행 시 라이선스를 검증합니다
// @Tags 클라이언트 - 라이선스
// @Accept json
// @Produce json
// @Param request body models.ValidateRequest true "검증 정보"
// @Success 200 {object} models.APIResponse "검증 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 403 {object} models.APIResponse "라이선스 비활성/만료 또는 디바이스 미등록"
// @Failure 404 {object} models.APIResponse "라이선스 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
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

	// 라이선스 조회
	var license models.License
	query := `SELECT id, license_key, product_name, expires_at, status FROM licenses WHERE license_key = ?`

	err := database.DB.QueryRow(query, req.LicenseKey).Scan(
		&license.ID, &license.LicenseKey, &license.ProductName,
		&license.ExpiresAt, &license.Status,
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

	// 라이선스 유효성 검사
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

	// 디바이스 핑거프린트 생성
	fingerprint := utils.GenerateDeviceFingerprint(
		req.DeviceInfo.ClientID,
		req.DeviceInfo.CPUID,
		req.DeviceInfo.MotherboardSN,
		req.DeviceInfo.MACAddress,
		req.DeviceInfo.DiskSerial,
		req.DeviceInfo.MachineID,
	)

	// 디바이스 활성화 여부 확인
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

	// 마지막 검증 시간 업데이트
	updateQuery := "UPDATE device_activations SET last_validated_at = ? WHERE id = ?"
	database.DB.Exec(updateQuery, time.Now().Format("2006-01-02 15:04:05"), deviceID)

	// 제품 ID 조회 및 정책 정보 가져오기
	var productID string
	prodQuery := "SELECT product_id FROM licenses WHERE id = ?"
	database.DB.QueryRow(prodQuery, license.ID).Scan(&productID)

	var policies []models.PolicyResponse
	if productID != "" {
		policyQuery := `SELECT id, policy_name, policy_data FROM policies WHERE product_id = ? AND status = ?`
		rows, err := database.DB.Query(policyQuery, productID, models.PolicyStatusActive)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p models.PolicyResponse
				var policyDataStr string
				err := rows.Scan(&p.ID, &p.PolicyName, &policyDataStr)
				if err == nil {
					// JSON 문자열을 interface{}로 파싱
					json.Unmarshal([]byte(policyDataStr), &p.PolicyData)
					policies = append(policies, p)
				}
			}
		}
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("License is valid", map[string]interface{}{
		"license_key":  license.LicenseKey,
		"product_name": license.ProductName,
		"expires_at":   license.ExpiresAt,
		"valid":        true,
		"policies":     policies,
	}))
}
