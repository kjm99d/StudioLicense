package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
)

// AdminCreateRequest 서브 관리자 생성 요청
type AdminCreateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AdminPermissionsUpdateRequest 관리자 권한 갱신 요청
type AdminPermissionsUpdateRequest struct {
	Permissions         []string                                        `json:"permissions"`
	ResourcePermissions map[string]models.AdminResourcePermissionConfig `json:"resource_permissions"`
}

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ListAdmins 관리자 목록 조회 (super_admin 전용)
func ListAdmins(w http.ResponseWriter, r *http.Request) {
	rows, err := database.DB.Query("SELECT id, username, email, role, created_at, updated_at FROM admins ORDER BY created_at DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query admins", err))
		return
	}
	defer rows.Close()

	list := make([]models.Admin, 0)
	for rows.Next() {
		var a models.Admin
		if err := rows.Scan(&a.ID, &a.Username, &a.Email, &a.Role, &a.CreatedAt, &a.UpdatedAt); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to scan admin", err))
			return
		}
		if a.Role == "super_admin" {
			a.Permissions = models.AllAdminPermissionKeys()
		} else {
			perms, err := utils.GetAdminPermissions(a.ID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load permissions", err))
				return
			}
			a.Permissions = perms
		}

		resourcePerms, err := utils.GetAdminResourcePermissions(a.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load resource permissions", err))
			return
		}
		a.ResourcePermissions = resourcePerms

		// Password는 비워둠
		list = append(list, a)
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Admins retrieved", list))
}

// CreateAdmin 서브 관리자 생성 (super_admin 전용)
func CreateAdmin(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	creatorID, _ := r.Context().Value("admin_id").(string)
	creatorName, _ := r.Context().Value("username").(string)

	var req AdminCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	// 기본 검증
	if len(req.Username) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Username must be at least 3 characters", nil))
		return
	}
	if !emailRegex.MatchString(req.Email) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid email format", nil))
		return
	}
	if len(req.Password) < 8 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Password must be at least 8 characters", nil))
		return
	}

	// 사용자명 중복 확인
	var exists int
	if err := database.DB.QueryRow("SELECT COUNT(1) FROM admins WHERE username = ?", req.Username).Scan(&exists); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to check username", err))
		return
	}
	if exists > 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Username already exists", nil))
		return
	}

	// ID 생성 및 비밀번호 해싱
	id, err := utils.GenerateID("adm")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to generate id", err))
		return
	}
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to hash password", err))
		return
	}

	// 기본 역할은 'admin' (서브 관리자)
	role := "admin"

	// 저장
	_, err = database.DB.Exec("INSERT INTO admins (id, username, password, email, role) VALUES (?, ?, ?, ?, ?)", id, req.Username, hash, req.Email, role)
	if err != nil {
		// SQLite unique 위반 등 처리
		if err == sql.ErrNoRows {
			// unlikely here
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to create admin", err))
		return
	}

	if err := utils.SetAdminPermissions(id, []string{}); err != nil {
		switch err.(type) {
		case *utils.InvalidPermissionError:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse(err.Error(), nil))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to assign permissions", err))
		}
		_, _ = database.DB.Exec("DELETE FROM admins WHERE id = ?", id)
		return
	}

	perms, err := utils.GetAdminPermissions(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load permissions", err))
		return
	}

	resourcePerms, err := utils.GetAdminResourcePermissions(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load resource permissions", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"creator_id": creatorID,
		"username":   req.Username,
	}).Info("Sub-admin created")

	// 활동 로그
	utils.LogAdminActivity(creatorID, creatorName, models.AdminActionCreateAdmin, "Created admin: "+req.Username)

	// 응답 (비밀번호는 제외)
	created := models.Admin{
		ID:                  id,
		Username:            req.Username,
		Email:               req.Email,
		Role:                role,
		Permissions:         perms,
		ResourcePermissions: resourcePerms,
	}
	json.NewEncoder(w).Encode(models.SuccessResponse("Admin created", created))
}

// UpdateAdminPermissions 서브 관리자 권한 업데이트 (super_admin 전용)
func UpdateAdminPermissions(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	actorID, _ := r.Context().Value("admin_id").(string)
	actorName, _ := r.Context().Value("username").(string)

	adminID, _ := r.Context().Value("path_admin_id").(string)
	if adminID == "" {
		adminID = r.PathValue("admin_id")
	}
	if adminID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Admin ID is required", nil))
		return
	}

	var req AdminPermissionsUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid request body", err))
		return
	}

	var username, role string
	if err := database.DB.QueryRow("SELECT username, role FROM admins WHERE id = ?", adminID).Scan(&username, &role); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Admin not found", nil))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load admin", err))
		return
	}

	if role == "super_admin" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Cannot modify super admin permissions", nil))
		return
	}

	if err := utils.SetAdminPermissions(adminID, req.Permissions); err != nil {
		switch err.(type) {
		case *utils.InvalidPermissionError:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.ErrorResponse(err.Error(), nil))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to assign permissions", err))
		}
		return
	}

	resourcePerms, err := utils.SetAdminResourcePermissions(adminID, req.ResourcePermissions)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to assign resource permissions", err))
		return
	}

	perms, err := utils.GetAdminPermissions(adminID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load permissions", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"actor_id":   actorID,
		"admin_id":   adminID,
	}).Info("Admin permissions updated")

	utils.LogAdminActivity(actorID, actorName, models.AdminActionUpdateAdminPerms, "Updated permissions for admin: "+username)

	json.NewEncoder(w).Encode(models.SuccessResponse("Permissions updated", map[string]interface{}{
		"permissions":          perms,
		"resource_permissions": resourcePerms,
	}))
}

// ResetAdminPassword 관리자 비밀번호 초기화 (super_admin 전용)
func ResetAdminPassword(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	creatorID, _ := r.Context().Value("admin_id").(string)
	creatorName, _ := r.Context().Value("username").(string)

	adminID, _ := r.Context().Value("path_admin_id").(string)
	if adminID == "" {
		adminID = r.PathValue("admin_id")
	}
	if adminID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Admin ID is required", nil))
		return
	}

	// Super admin은 비밀번호 초기화 불가
	var role string
	if err := database.DB.QueryRow("SELECT role FROM admins WHERE id = ?", adminID).Scan(&role); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Admin not found", nil))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to check admin", err))
		return
	}

	if role == "super_admin" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Cannot reset super admin password", nil))
		return
	}

	// 임시 비밀번호 생성 (8-10 자리 랜덤)
	tempPassword := utils.GenerateTempPassword(10)
	hash, err := utils.HashPassword(tempPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to hash password", err))
		return
	}

	// 비밀번호 업데이트
	_, err = database.DB.Exec("UPDATE admins SET password = ? WHERE id = ?", hash, adminID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to update password", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"creator_id": creatorID,
		"admin_id":   adminID,
	}).Info("Admin password reset")

	// 활동 로그
	utils.LogAdminActivity(creatorID, creatorName, models.AdminActionResetPassword, "Reset password for admin: "+adminID)

	json.NewEncoder(w).Encode(models.SuccessResponse("Password reset", map[string]string{
		"temp_password": tempPassword,
	}))
}

// DeleteAdmin 관리자 계정 삭제 (super_admin 전용)
func DeleteAdmin(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	creatorID, _ := r.Context().Value("admin_id").(string)
	creatorName, _ := r.Context().Value("username").(string)

	adminID, _ := r.Context().Value("path_admin_id").(string)
	if adminID == "" {
		adminID = r.PathValue("admin_id")
	}
	if adminID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("Admin ID is required", nil))
		return
	}

	// 본인 삭제 방지
	if adminID == creatorID {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Cannot delete your own account", nil))
		return
	}

	// Super admin 삭제 불가
	var role, username string
	if err := database.DB.QueryRow("SELECT role, username FROM admins WHERE id = ?", adminID).Scan(&role, &username); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("Admin not found", nil))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to check admin", err))
		return
	}

	if role == "super_admin" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Cannot delete super admin", nil))
		return
	}

	// 삭제 처리
	_, err := database.DB.Exec("DELETE FROM admins WHERE id = ?", adminID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to delete admin", err))
		return
	}

	logger.WithFields(map[string]interface{}{
		"request_id": requestID,
		"creator_id": creatorID,
		"admin_id":   adminID,
	}).Info("Admin account deleted")

	// 활동 로그
	utils.LogAdminActivity(creatorID, creatorName, models.AdminActionDeleteAdmin, "Deleted admin: "+username)

	json.NewEncoder(w).Encode(models.SuccessResponse("Admin deleted", nil))
}
