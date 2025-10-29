package models

// Admin 관리자 정보
type Admin struct {
	ID                  string                                   `json:"id" db:"id"`
	Username            string                                   `json:"username" db:"username"`
	Password            string                                   `json:"-" db:"password"` // bcrypt 해시
	Email               string                                   `json:"email" db:"email"`
	Role                string                                   `json:"role" db:"role"` // admin, superadmin
	Permissions         []string                                 `json:"permissions,omitempty" db:"-"`
	ResourcePermissions map[string]AdminResourcePermissionConfig `json:"resource_permissions,omitempty" db:"-"`
	CreatedAt           string                                   `json:"created_at" db:"created_at"`
	UpdatedAt           string                                   `json:"updated_at" db:"updated_at"`
}

// LoginRequest 로그인 요청
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 로그인 응답
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	Admin     *Admin `json:"admin"`
}

// ChangePasswordRequest 비밀번호 변경 요청
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
