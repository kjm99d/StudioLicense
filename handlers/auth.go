package handlers

import (
	"encoding/json"
	"net/http"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// Login 관리자 로그인
// @Summary 관리자 로그인
// @Description 관리자 계정으로 로그인하여 JWT 토큰을 발급받습니다
// @Tags 인증
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "로그인 정보"
// @Success 200 {object} models.APIResponse{data=models.LoginResponse} "로그인 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 실패"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/login [post]
func Login(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		}).Warn("Invalid login request body")

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"username":   req.Username,
	}).Info("Login attempt")

	// 관리자 조회
	var admin models.Admin
	query := "SELECT id, username, password, email, role, created_at, updated_at FROM admins WHERE username = ?"
	err := database.DB.QueryRow(query, req.Username).Scan(
		&admin.ID, &admin.Username, &admin.Password, &admin.Email,
		&admin.Role, &admin.CreatedAt, &admin.UpdatedAt,
	)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"username":   req.Username,
		}).Warn("Login failed - user not found")

		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid credentials", nil))
		return
	}

	// 비밀번호 검증 전 로깅 (디버깅용)
	logger.WithFields(map[string]interface{}{
		"request_id":   requestID,
		"username":     req.Username,
		"admin_id":     admin.ID,
		"stored_hash":  admin.Password,
		"hash_length":  len(admin.Password),
		"password_len": len(req.Password),
	}).Info("Attempting password check")

	// 비밀번호 검증
	if !utils.CheckPassword(admin.Password, req.Password) {
		logger.WithFields(map[string]interface{}{
			"request_id":   requestID,
			"username":     req.Username,
			"admin_id":     admin.ID,
			"stored_hash":  admin.Password,
			"hash_length":  len(admin.Password),
			"password_len": len(req.Password),
		}).Warn("Login failed - invalid password")

		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid credentials", nil))
		return
	}

	// JWT 토큰 생성
	token, expiresAt, err := utils.GenerateToken(admin.ID, admin.Username, admin.Role)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"admin_id":   admin.ID,
			"error":      err.Error(),
		}).Error("Failed to generate JWT token")

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to generate token", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"admin_id":   admin.ID,
		"username":   admin.Username,
	}).Info("Login successful")

	// 응답
	response := models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Admin:     &admin,
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Login successful", response))

	// 관리자 활동 로그
	utils.LogAdminActivity(admin.ID, admin.Username, models.AdminActionLogin, "Login successful")
}

// GetMe 현재 로그인된 관리자 정보
// @Summary 현재 사용자 정보 조회
// @Description 로그인된 관리자의 정보를 조회합니다
// @Tags 인증
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse{data=models.Admin} "조회 성공"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 404 {object} models.APIResponse "사용자 없음"
// @Router /api/admin/me [get]
func GetMe(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("admin_id").(string)

	var admin models.Admin
	query := "SELECT id, username, email, role, created_at, updated_at FROM admins WHERE id = ?"
	err := database.DB.QueryRow(query, adminID).Scan(
		&admin.ID, &admin.Username, &admin.Email,
		&admin.Role, &admin.CreatedAt, &admin.UpdatedAt,
	)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Admin not found", err))
		return
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Admin retrieved", admin))
}

// ChangePassword 관리자 비밀번호 변경
// @Summary 비밀번호 변경
// @Description 현재 비밀번호를 확인하고 새로운 비밀번호로 변경합니다
// @Tags 인증
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChangePasswordRequest true "비밀번호 변경 요청"
// @Success 200 {object} models.APIResponse "변경 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 필요/현재 비밀번호 불일치"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/change-password [post]
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	adminID := r.Context().Value("admin_id").(string)

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	if len(req.NewPassword) < 8 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("New password must be at least 8 characters", nil))
		return
	}

	// 현재 관리자 조회 (패스워드 포함)
	var hashed string
	err := database.DB.QueryRow("SELECT password FROM admins WHERE id = ?", adminID).Scan(&hashed)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse("Admin not found", err))
		return
	}

	// 기존 비밀번호 확인
	if !utils.CheckPassword(hashed, req.OldPassword) {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"admin_id":   adminID,
		}).Warn("Password change failed - wrong current password")

		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse("Current password is incorrect", nil))
		return
	}

	// 새 비밀번호 해싱 후 저장
	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"admin_id":   adminID,
			"error":      err.Error(),
		}).Error("Failed to hash password")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to hash password", err))
		return
	}

	// 해시 값 길이 확인 (디버깅용)
	logger.WithFields(map[string]interface{}{
		"request_id":  requestID,
		"admin_id":    adminID,
		"hash_length": len(newHash),
		"hash_value":  newHash,
	}).Info("New password hash generated")

	updateTime := time.Now().Format("2006-01-02 15:04:05")
	result, err := database.DB.Exec("UPDATE admins SET password = ?, updated_at = ? WHERE id = ?", newHash, updateTime, adminID)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"admin_id":   adminID,
			"error":      err.Error(),
		}).Error("Failed to update password in database")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update password", err))
		return
	}

	// 영향받은 행 수 확인
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		logger.WithFields(map[string]interface{}{
			"request_id":    requestID,
			"admin_id":      adminID,
			"rows_affected": rowsAffected,
			"error":         err,
		}).Error("Password update failed - admin not found or no rows affected")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update password", nil))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id":    requestID,
		"admin_id":      adminID,
		"rows_affected": rowsAffected,
	}).Info("Admin password changed successfully - verifying update")

	// 업데이트 검증 - 새 비밀번호가 실제로 저장되었는지 확인
	var verifyHash string
	verifyErr := database.DB.QueryRow("SELECT password FROM admins WHERE id = ?", adminID).Scan(&verifyHash)
	if verifyErr != nil {
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"admin_id":   adminID,
			"error":      verifyErr.Error(),
		}).Error("Failed to verify password update")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Password update verification failed", nil))
		return
	}

	// 저장된 해시가 예상한 해시와 일치하는지 확인
	if verifyHash != newHash {
		logger.WithFields(map[string]interface{}{
			"request_id":   requestID,
			"admin_id":     adminID,
			"new_hash":     newHash,
			"stored_hash":  verifyHash,
			"new_hash_len": len(newHash),
			"stored_len":   len(verifyHash),
		}).Error("Password hash mismatch after update - possible encoding issue")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Password update verification failed", nil))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"admin_id":   adminID,
	}).Info("Admin password changed successfully - verified")

	// 관리자 활동 로그
	username, _ := r.Context().Value("username").(string)
	utils.LogAdminActivity(adminID, username, models.AdminActionChangePassword, "Password changed successfully")

	// 응답 전송
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.SuccessResponse("Password changed successfully", nil))
}
