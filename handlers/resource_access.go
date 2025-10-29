package handlers

import (
	"net/http"

	"studiolicense/models"
	"studiolicense/services"
)

var scopeResolver services.ResourceScopeResolver = services.NoopResourceScopeResolver{}

// SetResourceScopeResolver는 핸들러에서 사용할 리소스 스코프 해석기를 주입합니다.
func SetResourceScopeResolver(resolver services.ResourceScopeResolver) {
	if resolver == nil {
		scopeResolver = services.NoopResourceScopeResolver{}
		return
	}
	scopeResolver = resolver
}

func resolveResourceScope(r *http.Request, resourceType string) (models.AdminResourcePermissionConfig, bool, string, error) {
	role, _ := r.Context().Value("role").(string)
	adminID, _ := r.Context().Value("admin_id").(string)
	scope, isSuper, err := scopeResolver.Resolve(r.Context(), role, adminID, resourceType)
	return scope, isSuper, adminID, err
}
