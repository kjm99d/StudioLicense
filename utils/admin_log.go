package utils

import (
	"studiolicense/database"
	"studiolicense/logger"
	"time"
)

// LogAdminActivity 관리자 활동 로그 기록 헬퍼
func LogAdminActivity(adminID, username, action, details string) {
	_, err := database.DB.Exec(
		`INSERT INTO admin_activity_logs (admin_id, username, action, details, created_at) VALUES (?, ?, ?, ?, ?)`,
		adminID, username, action, details, time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		logger.Error("Failed to log admin activity: %v", err)
	}
}
