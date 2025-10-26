package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"time"
)

// SubmitClientLogs 클라이언트 로그 전송
// @Summary 클라이언트 로그 전송
// @Description 클라이언트 애플리케이션에서 발생한 로그를 서버로 전송합니다 (배치 전송 지원)
// @Tags 클라이언트 - 로그
// @Accept json
// @Produce json
// @Param request body models.ClientLogRequest true "로그 데이터"
// @Success 201 {object} models.APIResponse "로그 전송 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 404 {object} models.APIResponse "라이선스 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/client/logs [post]
func SubmitClientLogs(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req models.ClientLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("Invalid client log request")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	// 라이선스 키 검증 (선택사항: 유효한 라이선스인지 확인)
	if req.LicenseKey != "" {
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM licenses WHERE license_key = ?)"
		err := database.DB.QueryRow(checkQuery, req.LicenseKey).Scan(&exists)
		if err != nil || !exists {
			logger.WithFields(map[string]interface{}{
				"request_id":  requestID,
				"license_key": req.LicenseKey,
			}).Warn("Invalid license key in client log")

			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Invalid license key", nil))
			return
		}
	}

	// 클라이언트 IP 추출
	clientIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = xff
	}

	// 로그 배치 삽입
	insertQuery := `INSERT INTO client_logs 
		(license_key, device_id, level, category, message, details, stack_trace, 
		 app_version, os_version, client_ip, client_timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now().Format("2006-01-02 15:04:05")
	insertedCount := 0

	for _, log := range req.Logs {
		// 로그 레벨 검증 (기본값: INFO)
		level := log.Level
		if level == "" {
			level = models.LogLevelInfo
		}

		// 카테고리 검증 (기본값: OTHER)
		category := log.Category
		if category == "" {
			category = models.LogCategoryOther
		}

		// 타임스탬프 (클라이언트 시간 또는 서버 시간)
		timestamp := log.Timestamp
		if timestamp == "" {
			timestamp = now
		}

		_, err := database.DB.Exec(insertQuery,
			req.LicenseKey, req.DeviceID, level, category, log.Message,
			log.Details, log.StackTrace, log.AppVersion, log.OSVersion,
			clientIP, timestamp, now,
		)

		if err != nil {
			logger.WithFields(map[string]interface{}{
				"request_id":  requestID,
				"license_key": req.LicenseKey,
				"error":       err.Error(),
			}).Error("Failed to insert client log")
			continue
		}

		insertedCount++
	}

	if insertedCount == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to save logs", nil))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"license_key": req.LicenseKey,
		"device_id":   req.DeviceID,
		"log_count":   insertedCount,
	}).Info("Client logs received")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("Logs submitted successfully", map[string]interface{}{
		"received": len(req.Logs),
		"saved":    insertedCount,
	}))
}

// GetClientLogs 클라이언트 로그 조회 (관리자용)
// @Summary 클라이언트 로그 조회
// @Description 클라이언트가 전송한 로그를 조회합니다
// @Tags 관리자 - 로그
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param license_key query string false "라이선스 키"
// @Param device_id query string false "디바이스 ID"
// @Param level query string false "로그 레벨 (DEBUG, INFO, WARN, ERROR, FATAL)"
// @Param category query string false "카테고리 (APP, SYSTEM, LICENSE, NETWORK, etc.)"
// @Param start_date query string false "시작 날짜 (YYYY-MM-DD)"
// @Param end_date query string false "종료 날짜 (YYYY-MM-DD)"
// @Param page query int false "페이지 번호 (기본값: 1)"
// @Param page_size query int false "페이지 크기 (기본값: 50)"
// @Success 200 {object} models.PaginatedResponse "로그 목록"
// @Failure 401 {object} models.APIResponse "인증 실패"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/client-logs [get]
func GetClientLogs(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	// 쿼리 파라미터 파싱
	query := r.URL.Query()
	licenseKey := query.Get("license_key")
	deviceID := query.Get("device_id")
	level := query.Get("level")
	category := query.Get("category")
	startDate := query.Get("start_date")
	endDate := query.Get("end_date")

	page := 1
	pageSize := 50
	if p := query.Get("page"); p != "" {
		if _, err := fmt.Sscanf(p, "%d", &page); err != nil || page < 1 {
			page = 1
		}
	}
	if ps := query.Get("page_size"); ps != "" {
		if _, err := fmt.Sscanf(ps, "%d", &pageSize); err != nil || pageSize < 1 || pageSize > 1000 {
			pageSize = 50
		}
	}

	// 동적 쿼리 생성
	baseQuery := "FROM client_logs WHERE 1=1"
	args := []interface{}{}

	if licenseKey != "" {
		baseQuery += " AND license_key = ?"
		args = append(args, licenseKey)
	}
	if deviceID != "" {
		baseQuery += " AND device_id = ?"
		args = append(args, deviceID)
	}
	if level != "" {
		baseQuery += " AND level = ?"
		args = append(args, level)
	}
	if category != "" {
		baseQuery += " AND category = ?"
		args = append(args, category)
	}
	if startDate != "" {
		baseQuery += " AND created_at >= ?"
		args = append(args, startDate+" 00:00:00")
	}
	if endDate != "" {
		baseQuery += " AND created_at <= ?"
		args = append(args, endDate+" 23:59:59")
	}

	// 총 개수 조회
	var totalCount int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := database.DB.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to count client logs")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to count logs", err))
		return
	}

	// 로그 목록 조회
	offset := (page - 1) * pageSize
	selectQuery := `SELECT id, license_key, device_id, level, category, message, 
		details, stack_trace, app_version, os_version, client_ip, client_timestamp, created_at 
		` + baseQuery + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := database.DB.Query(selectQuery, args...)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to query client logs")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query logs", err))
		return
	}
	defer rows.Close()

	logs := []models.ClientLog{}
	for rows.Next() {
		var log models.ClientLog
		var deviceID, details, stackTrace, appVersion, osVersion, clientIP, clientTimestamp sql.NullString

		err := rows.Scan(
			&log.ID, &log.LicenseKey, &deviceID, &log.Level, &log.Category,
			&log.Message, &details, &stackTrace, &appVersion, &osVersion,
			&clientIP, &clientTimestamp, &log.CreatedAt,
		)

		if err != nil {
			logger.WithFields(map[string]interface{}{
				"request_id": requestID,
				"error":      err.Error(),
			}).Error("Failed to scan client log")
			continue
		}

		if deviceID.Valid {
			log.DeviceID = deviceID.String
		}
		if details.Valid {
			log.Details = details.String
		}
		if stackTrace.Valid {
			log.StackTrace = stackTrace.String
		}
		if appVersion.Valid {
			log.AppVersion = appVersion.String
		}
		if osVersion.Valid {
			log.OSVersion = osVersion.String
		}
		if clientIP.Valid {
			log.ClientIP = clientIP.String
		}

		logs = append(logs, log)
	}

	// 페이징 정보
	totalPages := (totalCount + pageSize - 1) / pageSize
	pagination := models.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		TotalCount: totalCount,
	}

	json.NewEncoder(w).Encode(models.PaginatedResponse{
		Status:  "success",
		Message: "Client logs retrieved successfully",
		Data:    logs,
		Meta:    pagination,
	})
}

// DeleteClientLogs 클라이언트 로그 삭제 (관리자용)
// @Summary 클라이언트 로그 삭제
// @Description 특정 기간 이전의 클라이언트 로그를 삭제합니다
// @Tags 관리자 - 로그
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param before_date query string true "이전 날짜 (YYYY-MM-DD) - 이 날짜 이전의 로그 삭제"
// @Success 200 {object} models.APIResponse "삭제 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 실패"
// @Failure 403 {object} models.APIResponse "권한 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/client-logs/cleanup [delete]
func DeleteClientLogs(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	beforeDate := r.URL.Query().Get("before_date")
	if beforeDate == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("before_date parameter is required", nil))
		return
	}

	// 삭제 실행
	deleteQuery := "DELETE FROM client_logs WHERE created_at < ?"
	result, err := database.DB.Exec(deleteQuery, beforeDate+" 00:00:00")
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to delete client logs")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete logs", err))
		return
	}

	deletedCount, _ := result.RowsAffected()

	logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"before_date":   beforeDate,
		"deleted_count": deletedCount,
	}).Info("Client logs deleted")

	json.NewEncoder(w).Encode(models.SuccessResponse("Logs deleted successfully", map[string]interface{}{
		"deleted_count": deletedCount,
		"before_date":   beforeDate,
	}))
}
