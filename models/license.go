package models

import "time"

// License 라이선스 정보
type License struct {
	ID            string  `json:"id" db:"id"`
	LicenseKey    string  `json:"license_key" db:"license_key"`
	ProductID     *string `json:"product_id" db:"product_id"`
	ProductName   string  `json:"product_name" db:"product_name"`
	CustomerName  string  `json:"customer_name" db:"customer_name"`
	CustomerEmail string  `json:"customer_email" db:"customer_email"`
	MaxDevices    int     `json:"max_devices" db:"max_devices"`
	ExpiresAt     string  `json:"expires_at" db:"expires_at"`
	Status        string  `json:"status" db:"status"` // active, revoked, expired
	Notes         string  `json:"notes" db:"notes"`
	CreatedAt     string  `json:"created_at" db:"created_at"`
	UpdatedAt     string  `json:"updated_at" db:"updated_at"`
}

// LicenseStatus 상태 상수
const (
	LicenseStatusActive  = "active"
	LicenseStatusRevoked = "revoked"
	LicenseStatusExpired = "expired"
)

// CreateLicenseRequest 라이선스 생성 요청
type CreateLicenseRequest struct {
	ProductID     string `json:"product_id"` // 제품 ID (선택사항)
	ProductName   string `json:"product_name" binding:"required"`
	CustomerName  string `json:"customer_name" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required,email"`
	MaxDevices    int    `json:"max_devices" binding:"required,min=1"`
	ExpiresAt     string `json:"expires_at" binding:"required"`
	Notes         string `json:"notes"`
}

// UpdateLicenseRequest 라이선스 수정 요청
type UpdateLicenseRequest struct {
	ProductName   string `json:"product_name"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	MaxDevices    int    `json:"max_devices"`
	ExpiresAt     string `json:"expires_at"`
	Notes         string `json:"notes"`
}

// DeactivateDeviceRequest 디바이스 비활성화 요청
type DeactivateDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// IsExpired 만료 여부 확인
func (l *License) IsExpired() bool {
	// 만료일이 현재 시간보다 이전이면 만료된 것
	return l.ExpiresAt < time.Now().Format("2006-01-02")
}

// IsActive 활성화 여부 확인
func (l *License) IsActive() bool {
	return l.Status == LicenseStatusActive && !l.IsExpired()
}
