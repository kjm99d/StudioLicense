package utils

import (
	"sort"
	"strings"
	"studiolicense/database"
	"studiolicense/models"
)

// InvalidPermissionError indicates assignment contains unknown permission
type InvalidPermissionError struct {
	Permission string
}

func (e *InvalidPermissionError) Error() string {
	return "invalid permission: " + e.Permission
}

// GetAdminPermissions returns ordered permissions for admin
func GetAdminPermissions(adminID string) ([]string, error) {
	rows, err := database.DB.Query("SELECT permission FROM admin_permissions WHERE admin_id = ? ORDER BY permission ASC", adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := make([]string, 0)
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, err
		}
		perms = append(perms, perm)
	}
	return perms, nil
}

// SetAdminPermissions replaces permissions assigned to admin
func SetAdminPermissions(adminID string, permissions []string) error {
	normalized, err := normalizePermissions(permissions)
	if err != nil {
		return err
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec("DELETE FROM admin_permissions WHERE admin_id = ?", adminID); err != nil {
		return err
	}

	if len(normalized) > 0 {
		stmt, err := tx.Prepare("INSERT INTO admin_permissions (admin_id, permission) VALUES (?, ?)")
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, perm := range normalized {
			if _, err = stmt.Exec(adminID, perm); err != nil {
				return err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func normalizePermissions(perms []string) ([]string, error) {
	set := make(map[string]struct{})
	for _, perm := range perms {
		perm = strings.TrimSpace(perm)
		if perm == "" {
			continue
		}
		if !models.IsValidAdminPermission(perm) {
			return nil, &InvalidPermissionError{Permission: perm}
		}
		set[perm] = struct{}{}
	}

	// 관리 권한을 부여할 때 조회 권한을 자동으로 포함
	if _, ok := set[models.PermissionDevicesManage]; ok {
		set[models.PermissionDevicesView] = struct{}{}
	}

	if len(set) == 0 {
		return []string{}, nil
	}

	result := make([]string, 0, len(set))
	for perm := range set {
		result = append(result, perm)
	}
	sort.Strings(result)
	return result, nil
}
