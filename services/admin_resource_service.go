package services

import (
	"context"
	"database/sql"
	"strings"

	"studiolicense/models"
)

// AdminResourcePermissionService는 관리자 리소스 권한을 저장/조회하는 기능을 정의합니다.
type AdminResourcePermissionService interface {
	GetPermissions(ctx context.Context, adminID string) (map[string]models.AdminResourcePermissionConfig, error)
	SetPermissions(ctx context.Context, adminID string, payload map[string]models.AdminResourcePermissionConfig) (map[string]models.AdminResourcePermissionConfig, error)
	GetScope(ctx context.Context, adminID, resourceType string) (models.AdminResourcePermissionConfig, error)
}

type adminResourcePermissionService struct {
	db SQLExecutor
}

// NewAdminResourcePermissionService는 기본 구현체를 반환합니다.
func NewAdminResourcePermissionService(db SQLExecutor) AdminResourcePermissionService {
	return &adminResourcePermissionService{db: db}
}

func (s *adminResourcePermissionService) GetPermissions(ctx context.Context, adminID string) (map[string]models.AdminResourcePermissionConfig, error) {
	raw := make(map[string]models.AdminResourcePermissionConfig)

	scopeRows, err := s.db.QueryContext(ctx,
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

	selectionRows, err := s.db.QueryContext(ctx,
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

	return models.NormalizeAdminResourcePermissions(raw), nil
}

func (s *adminResourcePermissionService) SetPermissions(ctx context.Context, adminID string, payload map[string]models.AdminResourcePermissionConfig) (sanitized map[string]models.AdminResourcePermissionConfig, err error) {
	sanitized = models.NormalizeAdminResourcePermissions(payload)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx,
		"DELETE FROM admin_resource_scopes WHERE admin_id = ?",
		adminID,
	); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx,
		"DELETE FROM admin_resource_selections WHERE admin_id = ?",
		adminID,
	); err != nil {
		return nil, err
	}

	for resourceType, cfg := range sanitized {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO admin_resource_scopes (admin_id, resource_type, mode, created_at, updated_at)
			VALUES (?, ?, ?, NOW(), NOW())
			ON DUPLICATE KEY UPDATE mode = VALUES(mode), updated_at = NOW()
		`, adminID, resourceType, cfg.Mode); err != nil {
			return nil, err
		}

		if cfg.Mode != models.ResourceModeCustom {
			continue
		}
		for _, resourceID := range cfg.SelectedIDs {
			if _, err = tx.ExecContext(ctx, `
				INSERT INTO admin_resource_selections (admin_id, resource_type, resource_id, created_at)
				VALUES (?, ?, ?, NOW())
			`, adminID, resourceType, resourceID); err != nil {
				return nil, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return sanitized, nil
}

func (s *adminResourcePermissionService) GetScope(ctx context.Context, adminID, resourceType string) (models.AdminResourcePermissionConfig, error) {
	perms, err := s.GetPermissions(ctx, adminID)
	if err != nil {
		return models.AdminResourcePermissionConfig{}, err
	}
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	if cfg, ok := perms[resourceType]; ok {
		return cfg, nil
	}
	return models.AdminResourcePermissionConfig{Mode: models.ResourceModeAll}, nil
}
