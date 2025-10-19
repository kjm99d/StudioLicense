package utils

import (
	"studiolicense/database"
	"studiolicense/logger"
	"time"
)

// LogDeviceActivity 디바이스 활동 로그 기록 헬퍼
func LogDeviceActivity(deviceID, licenseID, action, details string) {
	query := `INSERT INTO device_activity_logs (device_id, license_id, action, details, created_at) 
		VALUES (?, ?, ?, ?, ?)`
	_, err := database.DB.Exec(query, deviceID, licenseID, action, details, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		logger.Error("Failed to log device activity: %v", err)
	}
}
