package models

// DeviceActivation 디바이스 활성화 정보
type DeviceActivation struct {
	ID                string  `json:"id" db:"id"`
	LicenseID         string  `json:"license_id" db:"license_id"`
	DeviceFingerprint string  `json:"device_fingerprint" db:"device_fingerprint"`
	DeviceInfo        string  `json:"device_info" db:"device_info"` // JSON 문자열
	DeviceName        string  `json:"device_name" db:"device_name"`
	Status            string  `json:"status" db:"status"` // active, deactivated
	ActivatedAt       string  `json:"activated_at" db:"activated_at"`
	LastValidatedAt   string  `json:"last_validated_at" db:"last_validated_at"`
	DeactivatedAt     *string `json:"deactivated_at,omitempty" db:"deactivated_at"`
}

// DeviceStatus 상태 상수
const (
	DeviceStatusActive      = "active"
	DeviceStatusDeactivated = "deactivated"
)

// DeviceInfo 디바이스 정보 구조체 (클라이언트에서 전달)
type DeviceInfo struct {
	CPUID         string `json:"cpu_id"`
	MotherboardSN string `json:"motherboard_sn"`
	MACAddress    string `json:"mac_address"`
	DiskSerial    string `json:"disk_serial"`
	MachineID     string `json:"machine_id"`
	OS            string `json:"os"`
	OSVersion     string `json:"os_version"`
	Hostname      string `json:"hostname"`
}

// ActivateRequest 라이선스 활성화 요청
type ActivateRequest struct {
	LicenseKey string     `json:"license_key" binding:"required"`
	DeviceInfo DeviceInfo `json:"device_info" binding:"required"`
}

// ValidateRequest 라이선스 검증 요청
type ValidateRequest struct {
	LicenseKey string     `json:"license_key" binding:"required"`
	DeviceInfo DeviceInfo `json:"device_info" binding:"required"`
}

// DeactivateRequest 라이선스 비활성화 요청
type DeactivateRequest struct {
	LicenseKey string     `json:"license_key" binding:"required"`
	DeviceInfo DeviceInfo `json:"device_info" binding:"required"`
}
