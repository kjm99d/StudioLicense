package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// ReactivateDeviceByAdmin 관리자가 비활성 디바이스를 재활성화
// @Summary 디바이스 재활성화 (관리자)
// @Description 관리자가 비활성화된 디바이스를 다시 활성화합니다
// @Tags 관리자 - 디바이스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object{device_id=string} true "디바이스 ID"
// @Success 200 {object} models.APIResponse "재활성화 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 404 {object} models.APIResponse "디바이스 없음"
// @Failure 409 {object} models.APIResponse "이미 활성 상태 또는 슬롯 초과"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/devices/reactivate [post]
func ReactivateDeviceByAdmin(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	if req.DeviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device ID is required", nil))
		return
	}

	// 디바이스 정보 조회
	var device models.DeviceActivation
	var maxDevices int
	checkQuery := `SELECT d.id, d.license_id, d.device_name, d.status, l.max_devices
		FROM device_activations d
		JOIN licenses l ON d.license_id = l.id
		WHERE d.id = ?`
	err := database.DB.QueryRow(checkQuery, req.DeviceID).Scan(
		&device.ID, &device.LicenseID, &device.DeviceName, &device.Status, &maxDevices,
	)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"device_id":  req.DeviceID,
		}).Warn("Device not found")

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device not found", nil))
		return
	}

	// 이미 활성 상태인지 확인
	if device.Status == models.DeviceStatusActive {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device is already active", nil))
		return
	}

	// 현재 활성화된 디바이스 수 확인
	var activeCount int
	countQuery := "SELECT COUNT(*) FROM device_activations WHERE license_id = ? AND status = ?"
	database.DB.QueryRow(countQuery, device.LicenseID, models.DeviceStatusActive).Scan(&activeCount)

	if activeCount >= maxDevices {
		logger.WithFields(map[string]interface{}{
			"request_id":   requestID,
			"device_id":    req.DeviceID,
			"active_count": activeCount,
			"max_devices":  maxDevices,
		}).Warn("Cannot reactivate: maximum device limit reached")

		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(models.ErrorResponse("Maximum device limit reached. Deactivate another device first.", nil))
		return
	}

	// 디바이스 재활성화
	now := time.Now().Format("2006-01-02 15:04:05")
	updateQuery := `UPDATE device_activations 
		SET status = ?, last_validated_at = ?, deactivated_at = NULL 
		WHERE id = ?`

	_, err = database.DB.Exec(updateQuery, models.DeviceStatusActive, now, req.DeviceID)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"device_id":  req.DeviceID,
			"error":      err.Error(),
		}).Error("Failed to reactivate device")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to reactivate device", err))
		return
	}

	// 활동 로그 기록
	utils.LogDeviceActivity(device.ID, device.LicenseID, models.DeviceActionReactivated, "Reactivated by admin")

	// 관리자 활동 로그 기록
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionReactivateDev, "Device reactivated: "+device.ID)
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"device_id":   req.DeviceID,
		"device_name": device.DeviceName,
	}).Info("Device reactivated by admin")

	json.NewEncoder(w).Encode(models.SuccessResponse("Device reactivated successfully", nil))
}

// DeactivateDeviceByAdmin 관리자가 디바이스를 강제로 비활성화
// @Summary 디바이스 비활성화 (관리자)
// @Description 관리자가 특정 디바이스를 강제로 비활성화합니다
// @Tags 관리자 - 디바이스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object{device_id=string} true "디바이스 ID"
// @Success 200 {object} models.APIResponse "비활성화 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 404 {object} models.APIResponse "디바이스 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/devices/deactivate [post]
func DeactivateDeviceByAdmin(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("Invalid deactivate device request")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	if req.DeviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device ID is required", nil))
		return
	}

	// 디바이스 존재 확인
	var deviceName string
	checkQuery := "SELECT device_name FROM device_activations WHERE id = ?"
	err := database.DB.QueryRow(checkQuery, req.DeviceID).Scan(&deviceName)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"device_id":  req.DeviceID,
		}).Warn("Device not found")

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device not found", nil))
		return
	}

	// 디바이스 비활성화
	now := time.Now().Format("2006-01-02 15:04:05")
	updateQuery := `UPDATE device_activations 
		SET status = ?, deactivated_at = ? 
		WHERE id = ?`

	_, err = database.DB.Exec(updateQuery, models.DeviceStatusDeactivated, now, req.DeviceID)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"device_id":  req.DeviceID,
			"error":      err.Error(),
		}).Error("Failed to deactivate device")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to deactivate device", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"device_id":   req.DeviceID,
		"device_name": deviceName,
	}).Info("Device deactivated by admin")

	// 활동 로그 기록
	var licenseID string
	database.DB.QueryRow("SELECT license_id FROM device_activations WHERE id = ?", req.DeviceID).Scan(&licenseID)
	utils.LogDeviceActivity(req.DeviceID, licenseID, models.DeviceActionDeactivated, "Deactivated by admin")

	// 관리자 활동 로그 기록
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, models.AdminActionDeactivateDev, "Device deactivated: "+req.DeviceID)
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Device deactivated successfully", nil))
}

// GetDeviceActivityLogs 디바이스 활동 로그 조회
// @Summary 디바이스 활동 로그 조회
// @Description 특정 디바이스의 활동 로그를 조회합니다
// @Tags 관리자 - 디바이스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device_id query string true "디바이스 ID"
// @Success 200 {object} models.APIResponse "로그 조회 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/devices/logs [get]
func GetDeviceActivityLogs(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device ID is required", nil))
		return
	}

	query := `SELECT id, device_id, license_id, action, details, created_at 
		FROM device_activity_logs 
		WHERE device_id = ? 
		ORDER BY created_at DESC 
		LIMIT 50`

	rows, err := database.DB.Query(query, deviceID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query activity logs", err))
		return
	}
	defer rows.Close()

	logs := []models.DeviceActivityLog{}
	for rows.Next() {
		var log models.DeviceActivityLog
		err := rows.Scan(&log.ID, &log.DeviceID, &log.LicenseID, &log.Action, &log.Details, &log.CreatedAt)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Activity logs retrieved", logs))
}

// CleanupInactiveDevices 오래된 비활성 디바이스 자동 정리
// @Summary 비활성 디바이스 자동 정리
// @Description N일 이상 비활성 상태인 디바이스를 삭제합니다
// @Tags 관리자 - 디바이스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object{days=int} true "비활성 기간 (일)"
// @Success 200 {object} models.APIResponse "정리 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/devices/cleanup [post]
func CleanupInactiveDevices(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req struct {
		Days int `json:"days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	if req.Days < 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Days must be 0 or greater", nil))
		return
	}

	cutoffDate := time.Now().AddDate(0, 0, -req.Days).Format("2006-01-02 15:04:05")

	query := `DELETE FROM device_activations 
		WHERE status = ? AND deactivated_at IS NOT NULL AND deactivated_at <= ?`

	result, err := database.DB.Exec(query, models.DeviceStatusDeactivated, cutoffDate)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to cleanup inactive devices")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to cleanup inactive devices", err))
		return
	}

	rowsAffected, _ := result.RowsAffected()

	logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"days":          req.Days,
		"cutoff_date":   cutoffDate,
		"rows_affected": rowsAffected,
	}).Info("Inactive devices cleaned up")

	json.NewEncoder(w).Encode(models.SuccessResponse("Inactive devices cleaned up successfully", map[string]interface{}{
		"deleted_count": rowsAffected,
		"days":          req.Days,
	}))

	// 관리자 활동 로그 기록
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		details := "Cleanup inactive devices (days=" + strconv.Itoa(req.Days) + ") deleted=" + strconv.FormatInt(rowsAffected, 10)
		utils.LogAdminActivity(adminID, username, models.AdminActionCleanupDevices, details)
	}
}

// DeleteDevice 디바이스 개별 삭제 (관리자)
// @Summary 디바이스 개별 삭제
// @Description 특정 디바이스를 데이터베이스에서 완전히 삭제합니다
// @Tags 관리자 - 디바이스
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object{device_id=string} true "디바이스 ID"
// @Success 200 {object} models.APIResponse "삭제 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 404 {object} models.APIResponse "디바이스 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/devices/delete [post]
func DeleteDevice(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	if req.DeviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device ID is required", nil))
		return
	}

	// 디바이스 존재 확인
	var deviceName string
	checkQuery := "SELECT device_name FROM device_activations WHERE id = ?"
	err := database.DB.QueryRow(checkQuery, req.DeviceID).Scan(&deviceName)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"device_id":  req.DeviceID,
		}).Warn("Device not found")

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Device not found", nil))
		return
	}

	// 디바이스 삭제
	deleteQuery := "DELETE FROM device_activations WHERE id = ?"
	_, err = database.DB.Exec(deleteQuery, req.DeviceID)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"device_id":  req.DeviceID,
			"error":      err.Error(),
		}).Error("Failed to delete device")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete device", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"device_id":   req.DeviceID,
		"device_name": deviceName,
	}).Info("Device deleted by admin")

	// 관리자 활동 로그 기록
	if adminIDRaw := r.Context().Value("admin_id"); adminIDRaw != nil {
		adminID := adminIDRaw.(string)
		username, _ := r.Context().Value("username").(string)
		utils.LogAdminActivity(adminID, username, "delete_device", "Device deleted: "+deviceName+" ("+req.DeviceID+")")
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Device deleted successfully", nil))
}
