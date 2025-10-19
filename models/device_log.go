package models

// DeviceActivityLog 디바이스 활동 로그
type DeviceActivityLog struct {
	ID        int    `json:"id" db:"id"`
	DeviceID  string `json:"device_id" db:"device_id"`
	LicenseID string `json:"license_id" db:"license_id"`
	Action    string `json:"action" db:"action"` // activated, validated, deactivated, reactivated
	Details   string `json:"details" db:"details"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

// 활동 액션 타입 상수
const (
	DeviceActionActivated   = "activated"
	DeviceActionValidated   = "validated"
	DeviceActionDeactivated = "deactivated"
	DeviceActionReactivated = "reactivated"
)
