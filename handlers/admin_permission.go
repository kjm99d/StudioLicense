package handlers

import (
	"encoding/json"
	"net/http"
	"studiolicense/models"
)

// GetAdminPermissionCatalog 관리자 권한 목록 조회
func GetAdminPermissionCatalog(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(models.SuccessResponse("Permission catalog", models.AdminPermissionCatalog))
}
