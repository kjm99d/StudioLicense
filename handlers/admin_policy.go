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

// CreatePolicy 정책 생성
// @Summary 정책 생성
// @Description 새로운 정책을 생성합니다
// @Tags 관리자 - 정책
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreatePolicyRequest true "정책 정보"
// @Success 201 {object} models.APIResponse{data=models.Policy} "생성 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 409 {object} models.APIResponse "정책명 중복"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/policies [post]
func CreatePolicy(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	adminIDVal := r.Context().Value("admin_id")
	creatorID, _ := adminIDVal.(string)
	if strings.TrimSpace(creatorID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.ErrorResponse("Missing creator context", nil))
		return
	}

	var req models.CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	// 정책명 중복 확인
	var existingPolicy string
	dupQuery := "SELECT id FROM policies WHERE policy_name = ?"
	err := database.DB.QueryRow(dupQuery, req.PolicyName).Scan(&existingPolicy)
	if err == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(models.ErrorResponse("Policy name already exists", nil))
		return
	}

	if err != sql.ErrNoRows {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to check policy existence", err))
		return
	}

	// 새로운 정책 생성
	policyID, err := utils.GenerateID("POLICY")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to generate policy ID", err))
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")

	query := `INSERT INTO policies (id, policy_name, policy_data, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err = database.DB.Exec(query, policyID, req.PolicyName, req.PolicyData, creatorID, now, now)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to create policy", err))
		return
	}

	// 정책 정보 반환
	policy := models.Policy{
		ID:         policyID,
		PolicyName: req.PolicyName,
		PolicyData: req.PolicyData,
		CreatedBy:  creatorID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// 활동 로그 기록
	utils.LogAdminActivity(creatorID, "admin", models.AdminActionCreatePolicy, "Policy created: "+policyID)

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"admin_id":   creatorID,
		"policy_id":  policyID,
	}).Info("Policy created")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SuccessResponse("Policy created successfully", policy))
}

// GetAllPolicies 모든 정책 조회
// @Summary 모든 정책 조회
// @Description 모든 정책을 조회합니다
// @Tags 관리자 - 정책
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse "조회 성공"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/policies [get]
func GetAllPolicies(w http.ResponseWriter, r *http.Request) {
	scope, isSuper, adminID, err := resolveResourceScope(r, models.ResourceTypePolicies)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate policy permissions", err))
		return
	}

	query := `SELECT id, policy_name, policy_data, created_by, created_at, updated_at
		FROM policies WHERE 1=1`
	args := make([]interface{}, 0)
	if !isSuper {
		filterSQL, filterArgs := utils.BuildResourceFilter(scope, "id", "created_by", adminID)
		query += filterSQL
		args = append(args, filterArgs...)
	}
	query += " ORDER BY created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query policies", err))
		return
	}
	defer rows.Close()

	policies := []models.Policy{}
	for rows.Next() {
		var policy models.Policy
		err := rows.Scan(
			&policy.ID, &policy.PolicyName, &policy.PolicyData, &policy.CreatedBy,
			&policy.CreatedAt, &policy.UpdatedAt,
		)
		if err != nil {
			continue
		}
		policies = append(policies, policy)
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Policies retrieved", policies))
}

// GetPolicy 정책 상세 조회
// @Summary 정책 상세 조회
// @Description 특정 정책의 상세 정보를 조회합니다
// @Tags 관리자 - 정책
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param policy_id path string true "정책 ID"
// @Success 200 {object} models.APIResponse{data=models.Policy} "조회 성공"
// @Failure 404 {object} models.APIResponse "정책 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/policies/{policy_id} [get]
func GetPolicy(w http.ResponseWriter, r *http.Request) {
	policyID := r.URL.Path[len("/api/admin/policies/"):]
	if policyID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Policy ID is required", nil))
		return
	}

	scope, isSuper, adminID, err := resolveResourceScope(r, models.ResourceTypePolicies)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to evaluate policy permissions", err))
		return
	}

	var policy models.Policy
	query := `SELECT id, policy_name, policy_data, created_by, created_at, updated_at
		FROM policies WHERE id = ?`
	args := []interface{}{policyID}
	if !isSuper && strings.EqualFold(scope.Mode, models.ResourceModeOwn) {
		query += " AND created_by = ?"
		args = append(args, adminID)
	}

	err = database.DB.QueryRow(query, args...).Scan(
		&policy.ID, &policy.PolicyName, &policy.PolicyData, &policy.CreatedBy,
		&policy.CreatedAt, &policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Policy not found", nil))
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query policy", err))
		return
	}

	if !isSuper && !utils.CanAccessResource(scope, policy.ID, policy.CreatedBy, adminID) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: policy access denied", nil))
		return
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Policy retrieved", policy))
}

// UpdatePolicy 정책 수정
// @Summary 정책 수정
// @Description 기존 정책을 수정합니다
// @Tags 관리자 - 정책
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param policy_id path string true "정책 ID"
// @Param request body models.UpdatePolicyRequest true "수정 정보"
// @Success 200 {object} models.APIResponse{data=models.Policy} "수정 성공"
// @Failure 400 {object} models.APIResponse "잘못된 요청"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 404 {object} models.APIResponse "정책 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/policies/{policy_id} [put]
func UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	adminID := r.Context().Value("admin_id")

	policyID := r.URL.Path[len("/api/admin/policies/"):]
	if policyID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Policy ID is required", nil))
		return
	}

	var req models.UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	// 정책 존재 여부 확인
	var policy models.Policy
	query := `SELECT id, policy_name, policy_data, created_at, updated_at
		FROM policies WHERE id = ?`
	err := database.DB.QueryRow(query, policyID).Scan(
		&policy.ID, &policy.PolicyName, &policy.PolicyData,
		&policy.CreatedAt, &policy.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Policy not found", nil))
		return
	}

	// 필드별 업데이트 (제공된 필드만)
	if req.PolicyName != "" {
		policy.PolicyName = req.PolicyName
	}
	if req.PolicyData != "" {
		policy.PolicyData = req.PolicyData
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	policy.UpdatedAt = now

	updateQuery := `UPDATE policies SET policy_name = ?, policy_data = ?, updated_at = ? WHERE id = ?`
	_, err = database.DB.Exec(updateQuery, policy.PolicyName, policy.PolicyData, now, policyID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update policy", err))
		return
	}

	// 활동 로그 기록
	utils.LogAdminActivity(adminID.(string), "admin", models.AdminActionUpdatePolicy, "Policy updated: "+policyID)

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"admin_id":   adminID,
		"policy_id":  policyID,
	}).Info("Policy updated")

	json.NewEncoder(w).Encode(models.SuccessResponse("Policy updated successfully", policy))
}

// DeletePolicy 정책 삭제
// @Summary 정책 삭제
// @Description 정책을 삭제합니다
// @Tags 관리자 - 정책
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param policy_id path string true "정책 ID"
// @Success 200 {object} models.APIResponse "삭제 성공"
// @Failure 401 {object} models.APIResponse "인증 필요"
// @Failure 404 {object} models.APIResponse "정책 없음"
// @Failure 500 {object} models.APIResponse "서버 에러"
// @Router /api/admin/policies/{policy_id} [delete]
func DeletePolicy(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	adminID := r.Context().Value("admin_id")

	policyID := r.URL.Path[len("/api/admin/policies/"):]
	if policyID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Policy ID is required", nil))
		return
	}

	// 정책 존재 여부 확인
	var id string
	checkQuery := "SELECT id FROM policies WHERE id = ?"
	err := database.DB.QueryRow(checkQuery, policyID).Scan(&id)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Policy not found", nil))
		return
	}

	// 정책 삭제
	deleteQuery := "DELETE FROM policies WHERE id = ?"
	result, err := database.DB.Exec(deleteQuery, policyID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete policy", err))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.ErrorResponse("Policy not found", nil))
		return
	}

	// 활동 로그 기록
	utils.LogAdminActivity(adminID.(string), "admin", models.AdminActionDeletePolicy, "Policy deleted: "+policyID)

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"admin_id":   adminID,
		"policy_id":  policyID,
	}).Info("Policy deleted")

	json.NewEncoder(w).Encode(models.SuccessResponse("Policy deleted successfully", nil))
}
