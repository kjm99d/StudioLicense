package scheduler

import (
	"fmt"
	"studiolicense/database"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// StartScheduler 스케줄러 시작
func StartScheduler() {
	logger.Info("Scheduler started")

	// 1시간마다 실행
	ticker := time.NewTicker(1 * time.Hour)

	// 서버 시작 시 즉시 한 번 실행
	UpdateExpiredLicenses()

	// 고루틴으로 주기적 실행
	go func() {
		for {
			<-ticker.C
			logger.Info("Scheduler tick: Running UpdateExpiredLicenses")
			UpdateExpiredLicenses()
		}
	}()
}

// UpdateExpiredLicenses 만료된 라이선스 상태 업데이트
func UpdateExpiredLicenses() {
	logger.Info("Running scheduled task: UpdateExpiredLicenses")

	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")

	// 먼저 만료 대상 확인
	checkQuery := `
		SELECT id, license_key, expires_at 
		FROM licenses 
		WHERE status = ? 
		AND expires_at < ? 
		AND expires_at IS NOT NULL
		LIMIT 10
	`

	rows, err := database.DB.Query(checkQuery, models.LicenseStatusActive, nowStr)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to check expired licenses")
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var (
			id        string
			key       string
			expiresAt string
		)
		if scanErr := rows.Scan(&id, &key, &expiresAt); scanErr != nil {
			logger.WithFields(map[string]interface{}{
				"error": scanErr.Error(),
			}).Warn("Failed to scan expired license row")
			continue
		}
		logger.WithFields(map[string]interface{}{
			"id":         id,
			"key":        key,
			"expires_at": expiresAt,
			"now":        nowStr,
		}).Info("Found expired license")
		count++
	}
	if err = rows.Err(); err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Row iteration error while scanning expired licenses")
	}
	logger.WithFields(map[string]interface{}{
		"count": count,
	}).Info("Expired licenses to update")

	// 업데이트 실행
	query := `
		UPDATE licenses 
		SET status = ?, updated_at = ? 
		WHERE status = ? 
		AND expires_at < ? 
		AND expires_at IS NOT NULL
	`

	result, err := database.DB.Exec(query,
		models.LicenseStatusExpired,
		nowStr,
		models.LicenseStatusActive,
		nowStr,
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
		"now":   nowStr,
	}).Info("Expired licenses updated")

	// 만료된 라이선스가 있으면 활동 로그 기록
	if rowsAffected > 0 {
		details := fmt.Sprintf("자동으로 %d개의 라이선스가 만료 처리되었습니다.", rowsAffected)
		utils.LogAdminActivity("system", "System", "라이선스 만료 처리", details)

		logger.WithFields(map[string]interface{}{
			"count": rowsAffected,
		}).Info("Admin activity logged for expired licenses")
	}
}
