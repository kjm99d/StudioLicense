package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"studiolicense/database"
	"studiolicense/models"
)

// GetDashboardStats 대시보드 통계
func GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})

	// 총 라이선스 수
	var totalLicenses, activeLicenses, expiredLicenses, revokedLicenses int
	database.DB.QueryRow("SELECT COUNT(*) FROM licenses").Scan(&totalLicenses)
	database.DB.QueryRow("SELECT COUNT(*) FROM licenses WHERE status = ?", models.LicenseStatusActive).Scan(&activeLicenses)
	database.DB.QueryRow("SELECT COUNT(*) FROM licenses WHERE status = ?", models.LicenseStatusExpired).Scan(&expiredLicenses)
	database.DB.QueryRow("SELECT COUNT(*) FROM licenses WHERE status = ?", models.LicenseStatusRevoked).Scan(&revokedLicenses)

	// 총 활성화된 디바이스 수
	var totalDevices int
	database.DB.QueryRow("SELECT COUNT(*) FROM device_activations WHERE status = ?", models.DeviceStatusActive).Scan(&totalDevices)

	stats["total_licenses"] = totalLicenses
	stats["active_licenses"] = activeLicenses
	stats["expired_licenses"] = expiredLicenses
	stats["revoked_licenses"] = revokedLicenses
	stats["total_active_devices"] = totalDevices

	json.NewEncoder(w).Encode(models.SuccessResponse("Dashboard stats retrieved", stats))
}

// GetRecentActivities 최근 활동 내역
func GetRecentActivities(w http.ResponseWriter, r *http.Request) {
	// 최근 활동: 디바이스 활동과 관리자 활동을 통합하여 최신순으로 제공
	qType := r.URL.Query().Get("type")     // device | admin | ""
	qAction := r.URL.Query().Get("action") // 특정 액션
	qLimit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			qLimit = n
		}
	}

	// 기본 쿼리 구성 (MySQL/SQLite 모두 호환)
	baseDevice := `SELECT 'device' AS type, al.action, al.details, al.created_at AS created_at,
			   al.id AS sort_id,
			   l.license_key, l.customer_name, l.product_name,
			   d.device_name, d.device_fingerprint,
			   NULL AS admin_username
		FROM device_activity_logs al
		JOIN device_activations d ON al.device_id = d.id
		JOIN licenses l ON al.license_id = l.id`
	baseAdmin := `SELECT 'admin' AS type, a.action, a.details, a.created_at AS created_at,
			   a.id AS sort_id,
			   NULL AS license_key, NULL AS customer_name, NULL AS product_name,
			   NULL AS device_name, NULL AS device_fingerprint,
			   a.username AS admin_username
		FROM admin_activity_logs a`

	var (
		query string
		args  []interface{}
	)
	switch qType {
	case "device":
		if qAction != "" {
			baseDevice += " WHERE al.action = ?"
			args = append(args, qAction)
		}
		query = baseDevice + " ORDER BY al.created_at DESC, al.id DESC LIMIT ?"
		args = append(args, qLimit)
	case "admin":
		if qAction != "" {
			baseAdmin += " WHERE a.action = ?"
			args = append(args, qAction)
		}
		query = baseAdmin + " ORDER BY a.created_at DESC, a.id DESC LIMIT ?"
		args = append(args, qLimit)
	default:
		// 전체: 두 쿼리를 UNION ALL
		if qAction != "" {
			baseDevice += " WHERE al.action = ?"
			baseAdmin += " WHERE a.action = ?"
			args = append(args, qAction, qAction)
		}
		// UNION ALL 후 created_at과 id로 정렬
		query = "SELECT * FROM (" + baseDevice + " UNION ALL " + baseAdmin + ") AS combined ORDER BY created_at DESC, sort_id DESC LIMIT ?"
		args = append(args, qLimit)
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("Failed to query activities", err))
		return
	}
	defer rows.Close()

	activities := []map[string]interface{}{}
	for rows.Next() {
		var typ, action, details, createdAt string
		var sortId int64
		var licenseKey, customerName, productName sql.NullString
		var deviceName, deviceFingerprint sql.NullString
		var adminUsername sql.NullString

		if err := rows.Scan(&typ, &action, &details, &createdAt, &sortId, &licenseKey, &customerName, &productName, &deviceName, &deviceFingerprint, &adminUsername); err != nil {
			continue
		}

		item := map[string]interface{}{
			"type":       typ,
			"action":     action,
			"details":    details,
			"created_at": createdAt,
		}
		if licenseKey.Valid {
			item["license_key"] = licenseKey.String
		}
		if customerName.Valid {
			item["customer_name"] = customerName.String
		}
		if productName.Valid {
			item["product_name"] = productName.String
		}
		if deviceName.Valid {
			item["device_name"] = deviceName.String
		}
		if deviceFingerprint.Valid {
			item["fingerprint"] = deviceFingerprint.String
		}
		if adminUsername.Valid {
			item["admin_username"] = adminUsername.String
		}
		activities = append(activities, item)
	}

	json.NewEncoder(w).Encode(models.SuccessResponse("Recent activities retrieved", activities))
}
