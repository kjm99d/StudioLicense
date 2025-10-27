package utils

import (
	"studiolicense/database"
	"studiolicense/logger"
)

// LogAdminActivity 관리자 활동 로그 기록 헬퍼
func LogAdminActivity(adminID, username, action, details string) {
	_, err := database.DB.Exec(
		`INSERT INTO admin_activity_logs (admin_id, username, action, details, created_at) VALUES (?, ?, ?, ?, ?)`,
		adminID, username, action, details, NowSeoul(),
	)
	if err != nil {
		logger.Error("Failed to log admin activity: %v", err)
	}
}
