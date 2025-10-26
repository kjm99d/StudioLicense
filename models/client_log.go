package models

// ClientLog 클라이언트 로그 모델
type ClientLog struct {
	ID         int64  `json:"id"`
	LicenseKey string `json:"license_key"`
	DeviceID   string `json:"device_id,omitempty"`
	Level      string `json:"level"`    // DEBUG, INFO, WARN, ERROR, FATAL
	Category   string `json:"category"` // APP, SYSTEM, LICENSE, NETWORK, etc.
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
	OSVersion  string `json:"os_version,omitempty"`
	ClientIP   string `json:"client_ip,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// ClientLogRequest 클라이언트 로그 전송 요청
type ClientLogRequest struct {
	LicenseKey string           `json:"license_key"`
	DeviceID   string           `json:"device_id,omitempty"`
	Logs       []ClientLogEntry `json:"logs"` // 배치 전송 지원
}

// ClientLogEntry 개별 로그 항목
type ClientLogEntry struct {
	Level      string `json:"level"`    // DEBUG, INFO, WARN, ERROR, FATAL
	Category   string `json:"category"` // APP, SYSTEM, LICENSE, NETWORK
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
	OSVersion  string `json:"os_version,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"` // 클라이언트 시간
}

// ClientLogFilter 로그 조회 필터
type ClientLogFilter struct {
	LicenseKey string `json:"license_key,omitempty"`
	DeviceID   string `json:"device_id,omitempty"`
	Level      string `json:"level,omitempty"`
	Category   string `json:"category,omitempty"`
	StartDate  string `json:"start_date,omitempty"`
	EndDate    string `json:"end_date,omitempty"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// 로그 레벨 상수
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
	LogLevelFatal = "FATAL"
)

// 로그 카테고리 상수
const (
	LogCategoryApp     = "APP"
	LogCategorySystem  = "SYSTEM"
	LogCategoryLicense = "LICENSE"
	LogCategoryNetwork = "NETWORK"
	LogCategoryPlugin  = "PLUGIN"
	LogCategoryOther   = "OTHER"
)
