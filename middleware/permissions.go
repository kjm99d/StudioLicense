package middleware

import (
	"encoding/json"
	"net/http"
	"studiolicense/models"
)

// RequirePermissions ensures the caller has all permissions
func RequirePermissions(perms ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !hasPermissions(r, perms...) {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: insufficient permission", nil))
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

// EnsurePermission short-circuits handler if permission is missing
func EnsurePermission(w http.ResponseWriter, r *http.Request, perm string) bool {
	if hasPermissions(r, perm) {
		return true
	}
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: insufficient permission", nil))
	return false
}

// HasPermission reports whether request context has permission
func HasPermission(r *http.Request, perm string) bool {
	return hasPermissions(r, perm)
}

func hasPermissions(r *http.Request, perms ...string) bool {
	if len(perms) == 0 {
		return true
	}

	if role, _ := r.Context().Value("role").(string); role == "super_admin" {
		return true
	}

	permSet, _ := r.Context().Value("permissions").(map[string]struct{})
	if len(permSet) == 0 {
		return false
	}
	for _, perm := range perms {
		if _, ok := permSet[perm]; !ok {
			return false
		}
	}
	return true
}
