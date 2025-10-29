package utils

import (
	"strings"
	"studiolicense/database"
	"studiolicense/models"
)

// GetAdminResourcePermissions loads the resource-based access configuration for an admin.
func GetAdminResourcePermissions(adminID string) (map[string]models.AdminResourcePermissionConfig, error) {
	raw := make(map[string]models.AdminResourcePermissionConfig)

	scopeRows, err := database.DB.Query(
		"SELECT resource_type, mode FROM admin_resource_scopes WHERE admin_id = ?",
		adminID,
	)
	if err != nil {
		return nil, err
	}
	defer scopeRows.Close()

	for scopeRows.Next() {
		var resourceType, mode string
		if err := scopeRows.Scan(&resourceType, &mode); err != nil {
			return nil, err
		}
		resourceType = strings.ToLower(strings.TrimSpace(resourceType))
		if !models.IsValidResourceType(resourceType) {
			continue
		}
		cfg := raw[resourceType]
		cfg.Mode = mode
		raw[resourceType] = cfg
	}
	if err := scopeRows.Err(); err != nil {
		return nil, err
	}

	selectionRows, err := database.DB.Query(
		"SELECT resource_type, resource_id FROM admin_resource_selections WHERE admin_id = ?",
		adminID,
	)
	if err != nil {
		return nil, err
	}
	defer selectionRows.Close()

	for selectionRows.Next() {
		var resourceType, resourceID string
		if err := selectionRows.Scan(&resourceType, &resourceID); err != nil {
			return nil, err
		}
		resourceType = strings.ToLower(strings.TrimSpace(resourceType))
		if !models.IsValidResourceType(resourceType) {
			continue
		}
		if strings.TrimSpace(resourceID) == "" {
			continue
		}
		cfg := raw[resourceType]
		cfg.SelectedIDs = append(cfg.SelectedIDs, resourceID)
		raw[resourceType] = cfg
	}
	if err := selectionRows.Err(); err != nil {
		return nil, err
	}

	// Normalize ensures defaults are present for every resource type.
	return models.NormalizeAdminResourcePermissions(raw), nil
}

// SetAdminResourcePermissions replaces the resource access configuration for an admin.
func SetAdminResourcePermissions(adminID string, payload map[string]models.AdminResourcePermissionConfig) (map[string]models.AdminResourcePermissionConfig, error) {
	sanitized := models.NormalizeAdminResourcePermissions(payload)

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for resourceType, cfg := range sanitized {
		if _, err = tx.Exec(`
			INSERT INTO admin_resource_scopes (admin_id, resource_type, mode, created_at, updated_at)
			VALUES (?, ?, ?, NOW(), NOW())
			ON DUPLICATE KEY UPDATE mode = VALUES(mode), updated_at = NOW()
		`, adminID, resourceType, cfg.Mode); err != nil {
			return nil, err
		}

		if _, err = tx.Exec(
			"DELETE FROM admin_resource_selections WHERE admin_id = ? AND resource_type = ?",
			adminID, resourceType,
		); err != nil {
			return nil, err
		}

		if cfg.Mode == models.ResourceModeCustom {
			for _, resourceID := range cfg.SelectedIDs {
				if _, err = tx.Exec(`
					INSERT INTO admin_resource_selections (admin_id, resource_type, resource_id, created_at)
					VALUES (?, ?, ?, NOW())
				`, adminID, resourceType, resourceID); err != nil {
					return nil, err
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return sanitized, nil
}

// GetAdminResourceScope returns the resource permission configuration for a single type.
func GetAdminResourceScope(adminID, resourceType string) (models.AdminResourcePermissionConfig, error) {
	perms, err := GetAdminResourcePermissions(adminID)
	if err != nil {
		return models.AdminResourcePermissionConfig{}, err
	}
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	if cfg, ok := perms[resourceType]; ok {
		return cfg, nil
	}
	return models.AdminResourcePermissionConfig{Mode: models.ResourceModeAll}, nil
}

// BuildResourceFilter generates an SQL fragment for filtering by resource permissions.
func BuildResourceFilter(scope models.AdminResourcePermissionConfig, idColumn string, ownerColumn string, adminID string) (string, []interface{}) {
	mode := strings.ToLower(strings.TrimSpace(scope.Mode))
	switch mode {
	case "", models.ResourceModeAll:
		return "", nil
	case models.ResourceModeNone:
		return " AND 1=0", nil
	case models.ResourceModeOwn:
		if ownerColumn == "" || strings.TrimSpace(adminID) == "" {
			return " AND 1=0", nil
		}
		return " AND " + ownerColumn + " = ?", []interface{}{adminID}
	case models.ResourceModeCustom:
		if len(scope.SelectedIDs) == 0 {
			return " AND 1=0", nil
		}
		placeholders := make([]string, len(scope.SelectedIDs))
		args := make([]interface{}, 0, len(scope.SelectedIDs))
		for i, id := range scope.SelectedIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		return " AND " + idColumn + " IN (" + strings.Join(placeholders, ",") + ")", args
	default:
		return "", nil
	}
}

// CanAccessResource reports whether the scope allows access to a specific resource ID/owner.
func CanAccessResource(scope models.AdminResourcePermissionConfig, resourceID, ownerID, adminID string) bool {
	mode := strings.ToLower(strings.TrimSpace(scope.Mode))
	switch mode {
	case "", models.ResourceModeAll:
		return true
	case models.ResourceModeNone:
		return false
	case models.ResourceModeOwn:
		return strings.EqualFold(strings.TrimSpace(ownerID), strings.TrimSpace(adminID))
	case models.ResourceModeCustom:
		target := strings.TrimSpace(resourceID)
		for _, id := range scope.SelectedIDs {
			if strings.EqualFold(strings.TrimSpace(id), target) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
