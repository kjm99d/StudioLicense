package scheduler

import (
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"time"
)

// StartScheduler 스케줄러 시작
func StartScheduler() {
	logger.Info("Scheduler started")

	// 매일 자정에 실행
	ticker := time.NewTicker(24 * time.Hour)

	// 서버 시작 시 즉시 한 번 실행
	go func() {
		UpdateExpiredLicenses()

		for range ticker.C {
			UpdateExpiredLicenses()
		}
	}()
}

// UpdateExpiredLicenses 만료된 라이선스 상태 업데이트
func UpdateExpiredLicenses() {
	logger.Info("Running scheduled task: UpdateExpiredLicenses")

	query := `
		UPDATE licenses 
		SET status = ?, updated_at = ? 
		WHERE status = ? 
		AND expires_at < ? 
		AND expires_at IS NOT NULL
	`

	now := time.Now()
	result, err := database.DB.Exec(query,
		models.LicenseStatusExpired,
		now,
		models.LicenseStatusActive,
		now,
	)

	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to update expired licenses")
		return
	}

	rowsAffected, _ := result.RowsAffected()

	logger.WithFields(map[string]interface{}{
		"count": rowsAffected,
	}).Info("Expired licenses updated")
}
