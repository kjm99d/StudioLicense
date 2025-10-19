package middleware

import (
	"encoding/json"
	"net/http"
	"studiolicense/database"
	"studiolicense/models"
)

// RequireRoles wraps a handler and allows access only if the admin role is one of allowedRoles.
func RequireRoles(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	set := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		set[r] = struct{}{}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Always verify latest role from DB to avoid stale JWT claims
			adminID, _ := r.Context().Value("admin_id").(string)
			if adminID == "" {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(models.ErrorResponse("Unauthorized", nil))
				return
			}
			var role string
			if err := database.DB.QueryRow("SELECT role FROM admins WHERE id = ?", adminID).Scan(&role); err != nil {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: insufficient role", err))
				return
			}
			if _, ok := set[role]; !ok {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(models.ErrorResponse("Forbidden: insufficient role", nil))
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}
