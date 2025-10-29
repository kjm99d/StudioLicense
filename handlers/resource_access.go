package handlers

import (
	"errors"
	"net/http"
	"strings"
	"studiolicense/models"
	"studiolicense/utils"
)

func resolveResourceScope(r *http.Request, resourceType string) (models.AdminResourcePermissionConfig, bool, string, error) {
	role, _ := r.Context().Value("role").(string)
	if strings.EqualFold(role, "super_admin") {
		return models.AdminResourcePermissionConfig{Mode: models.ResourceModeAll}, true, "", nil
	}

	adminID, _ := r.Context().Value("admin_id").(string)
	if strings.TrimSpace(adminID) == "" {
		return models.AdminResourcePermissionConfig{}, false, "", errors.New("missing admin id in context")
	}

	scope, err := utils.GetAdminResourceScope(adminID, resourceType)
	if err != nil {
		return models.AdminResourcePermissionConfig{}, false, "", err
	}
	return scope, false, adminID, nil
}
