package services

import (
	"context"
	"errors"
	"strings"

	"studiolicense/models"
)

// ResourceScopeResolver는 리소스 접근 스코프를 계산합니다.
type ResourceScopeResolver interface {
	Resolve(ctx context.Context, role, adminID, resourceType string) (models.AdminResourcePermissionConfig, bool, error)
}

var ErrMissingAdminID = errors.New("missing admin id in context")

type resourceScopeResolver struct {
	permissions AdminResourcePermissionService
}

// NewResourceScopeResolver는 기본 구현체를 생성합니다.
func NewResourceScopeResolver(permissions AdminResourcePermissionService) ResourceScopeResolver {
	return &resourceScopeResolver{permissions: permissions}
}

func (r *resourceScopeResolver) Resolve(ctx context.Context, role, adminID, resourceType string) (models.AdminResourcePermissionConfig, bool, error) {
	if strings.EqualFold(role, "super_admin") {
		return models.AdminResourcePermissionConfig{Mode: models.ResourceModeAll}, true, nil
	}
	if strings.TrimSpace(adminID) == "" {
		return models.AdminResourcePermissionConfig{}, false, ErrMissingAdminID
	}
	scope, err := r.permissions.GetScope(ctx, adminID, resourceType)
	return scope, false, err
}

// NoopResourceScopeResolver는 항상 모든 리소스 접근을 허용하는 기본 구현입니다.
type NoopResourceScopeResolver struct{}

func (NoopResourceScopeResolver) Resolve(context.Context, string, string, string) (models.AdminResourcePermissionConfig, bool, error) {
	return models.AdminResourcePermissionConfig{Mode: models.ResourceModeAll}, true, nil
}
