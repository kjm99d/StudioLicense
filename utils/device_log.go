package utils

import (
	"studiolicense/database"
	"studiolicense/logger"
)

// LogDeviceActivity 디바이스 활동 로그 기록 헬퍼
func LogDeviceActivity(deviceID, licenseID, action, details string) {
	query := `INSERT INTO device_activity_logs (device_id, license_id, action, details, created_at) 
		VALUES (?, ?, ?, ?, ?)`
	_, err := database.DB.Exec(query, deviceID, licenseID, action, details, NowSeoul())
	if err != nil {
		logger.Error("Failed to log device activity: %v", err)
	}
}
