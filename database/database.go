package database

import (
	"database/sql"
	"fmt"
	"studiolicense/logger"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

var DB *sql.DB
var dbType string // 데이터베이스 타입 저장

// Initialize 데이터베이스 초기화
// dbType: "sqlite" 또는 "mysql"
// dbPath: SQLite 파일 경로 또는 MySQL DSN
func Initialize(t, dbPath string) error {
	var err error

	// 기본값 설정
	if t == "" {
		t = "sqlite"
	}
	if dbPath == "" {
		if t == "sqlite" {
			dbPath = "./license.db"
		}
	}

	// 전역 dbType 저장
	dbType = t

	DB, err = sql.Open(dbType, dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 연결 테스트
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// SQLite 전용: 외래키 강제 활성화 (기본값 off)
	if dbType == "sqlite" {
		if _, err := DB.Exec("PRAGMA foreign_keys = ON"); err != nil {
			return fmt.Errorf("failed to enable foreign keys: %w", err)
		}
	}

	// 테이블 생성
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// 기본 관리자 계정 생성
	if err := createDefaultAdmin(); err != nil {
		return fmt.Errorf("failed to create default admin: %w", err)
	}

	// 슈퍼 관리자 보장: 기존 DB에서 super_admin이 하나도 없으면 가장 오래된 관리자 1명을 승격
	if err := ensureSuperAdminExists(); err != nil {
		return fmt.Errorf("failed to ensure super_admin exists: %w", err)
	}

	// 샘플 제품 생성
	if err := createSampleProducts(); err != nil {
		return fmt.Errorf("failed to create sample products: %w", err)
	}

	logger.Info("Database initialized successfully")
	return nil
}

// createTables 테이블 생성
func createTables() error {
	// SQLite와 MySQL 모두 지원하는 스키마
	baseTables := []string{
		// 관리자 테이블
		`CREATE TABLE IF NOT EXISTS admins (
			id VARCHAR(50) PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			email VARCHAR(100) NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'admin',
			created_at VARCHAR(50) NOT NULL DEFAULT '',
			updated_at VARCHAR(50) NOT NULL DEFAULT ''
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 제품 테이블
		`CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			created_at VARCHAR(50) NOT NULL DEFAULT '',
			updated_at VARCHAR(50) NOT NULL DEFAULT ''
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 라이선스 테이블
		`CREATE TABLE IF NOT EXISTS licenses (
			id VARCHAR(50) PRIMARY KEY,
			license_key VARCHAR(255) UNIQUE NOT NULL,
			product_id VARCHAR(50),
			policy_id VARCHAR(50),
			product_name VARCHAR(255) NOT NULL,
			customer_name VARCHAR(255) NOT NULL,
			customer_email VARCHAR(100) NOT NULL,
			max_devices INT NOT NULL DEFAULT 1,
			expires_at VARCHAR(50) NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			notes TEXT,
			created_at VARCHAR(50) NOT NULL DEFAULT '',
			updated_at VARCHAR(50) NOT NULL DEFAULT '',
			FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL,
			FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE SET NULL
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 정책 테이블
		`CREATE TABLE IF NOT EXISTS policies (
			id VARCHAR(50) PRIMARY KEY,
			policy_name VARCHAR(255) UNIQUE NOT NULL,
			policy_data LONGTEXT NOT NULL,
			created_at VARCHAR(50) NOT NULL DEFAULT '',
			updated_at VARCHAR(50) NOT NULL DEFAULT ''
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 디바이스 활성화 테이블
		`CREATE TABLE IF NOT EXISTS device_activations (
			id VARCHAR(50) PRIMARY KEY,
			license_id VARCHAR(50) NOT NULL,
			device_fingerprint VARCHAR(255) NOT NULL,
			device_info LONGTEXT NOT NULL,
			device_name VARCHAR(255),
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			activated_at VARCHAR(50) NOT NULL DEFAULT '',
			last_validated_at VARCHAR(50) NOT NULL DEFAULT '',
			deactivated_at VARCHAR(50),
			FOREIGN KEY (license_id) REFERENCES licenses(id) ON DELETE CASCADE,
			UNIQUE KEY unique_device (license_id, device_fingerprint)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 디바이스 활동 로그 테이블
		`CREATE TABLE IF NOT EXISTS device_activity_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			device_id VARCHAR(50) NOT NULL,
			license_id VARCHAR(50) NOT NULL,
			action VARCHAR(100) NOT NULL,
			details LONGTEXT,
			created_at VARCHAR(50) NOT NULL DEFAULT '',
			FOREIGN KEY (device_id) REFERENCES device_activations(id) ON DELETE CASCADE,
			FOREIGN KEY (license_id) REFERENCES licenses(id) ON DELETE CASCADE,
			INDEX idx_device (device_id),
			INDEX idx_license (license_id),
			INDEX idx_created (created_at)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 관리자 활동 로그 테이블
		`CREATE TABLE IF NOT EXISTS admin_activity_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			admin_id VARCHAR(50) NOT NULL,
			username VARCHAR(100) NOT NULL,
			action VARCHAR(100) NOT NULL,
			details LONGTEXT,
			created_at VARCHAR(50) NOT NULL DEFAULT '',
			INDEX idx_admin (admin_id),
			INDEX idx_created (created_at)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 인덱스 생성
		`CREATE INDEX IF NOT EXISTS idx_products_status ON products(status)`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_key ON licenses(license_key)`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_product ON licenses(product_id)`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_status ON licenses(status)`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_expires ON licenses(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_product ON policies(product_id)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_status ON policies(status)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_license ON device_activations(license_id)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_fingerprint ON device_activations(device_fingerprint)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_status ON device_activations(status)`,
	}

	// 데이터베이스별 처리
	if dbType == "sqlite" {
		// SQLite: PRAGMA 추가
		sqliteTables := []string{
			`PRAGMA foreign_keys = OFF`,
		}
		sqliteTables = append(sqliteTables, baseTables...)
		sqliteTables = append(sqliteTables, `PRAGMA foreign_keys = ON`)

		for _, sql := range sqliteTables {
			if _, err := DB.Exec(sql); err != nil {
				// SQLite에서 지원하지 않는 문법 무시
				if !contains(err.Error(), "syntax error") {
					return fmt.Errorf("failed to execute SQL: %w", err)
				}
			}
		}
	} else {
		// MySQL: 일반 실행
		for _, sql := range baseTables {
			if _, err := DB.Exec(sql); err != nil {
				// 이미 존재하는 인덱스/테이블 오류 무시
				if !contains(err.Error(), "already exists") {
					return fmt.Errorf("failed to execute SQL: %w", err)
				}
			}
		}
	}

	return nil
}

// contains 문자열 포함 여부 확인
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && s[0:len(substr)] == substr || len(s) > len(substr)))
}

// createDefaultAdmin 기본 관리자 계정 생성
func createDefaultAdmin() error {
	// 기존 관리자가 있는지 확인
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if err != nil {
		return err
	}

	// 이미 관리자가 있으면 스킵
	if count > 0 {
		return nil
	}

	// bcrypt 해시 생성 (비밀번호: admin123)
	hashedPassword := "$2a$10$qSCYloReyQ4gid/Gxf4gquDv3LaMmzC/2lnxvnfAAKnRkkaqXoOha" // admin123

	query := `
		INSERT INTO admins (id, username, password, email, role)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = DB.Exec(query, "admin-001", "admin", hashedPassword, "admin@example.com", "super_admin")
	if err != nil {
		return err
	}

	logger.Info("Default admin created (username: admin, password: admin123)")
	return nil
}

// ensureSuperAdminExists super_admin이 하나도 없으면 가장 먼저 생성된 관리자를 super_admin으로 승격
func ensureSuperAdminExists() error {
	var cnt int
	if err := DB.QueryRow("SELECT COUNT(1) FROM admins WHERE role = 'super_admin'").Scan(&cnt); err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}

	var id, username string
	err := DB.QueryRow("SELECT id, username FROM admins ORDER BY created_at ASC LIMIT 1").Scan(&id, &username)
	if err != nil {
		return err
	}

	if _, err := DB.Exec("UPDATE admins SET role = 'super_admin' WHERE id = ?", id); err != nil {
		return err
	}
	logger.Info("No super_admin found. Promoted admin '%s' to super_admin.", username)
	return nil
}

// createSampleProducts 샘플 제품 생성
func createSampleProducts() error {
	// 기존 제품이 있는지 확인
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil {
		logger.Warn("Failed to check existing products: %v", err)
		return err
	}

	logger.Info("Existing products count: %d", count)

	// 이미 제품이 있으면 스킵
	if count > 0 {
		logger.Info("Products already exist, skipping sample product creation")
		return nil
	}

	// 샘플 제품 생성
	sampleProducts := []map[string]string{
		{"id": "prod-001", "name": "Studio Pro", "description": "Professional edition"},
		{"id": "prod-002", "name": "Studio Standard", "description": "Standard edition"},
		{"id": "prod-003", "name": "Studio Basic", "description": "Basic edition"},
	}

	query := `INSERT INTO products (id, name, description, status, created_at, updated_at) 
		VALUES (?, ?, ?, ?, NOW(), NOW())`

	successCount := 0
	for _, product := range sampleProducts {
		result, err := DB.Exec(query, product["id"], product["name"], product["description"], "active")
		if err != nil {
			logger.Error("Failed to create sample product %s: %v", product["id"], err)
		} else {
			rowsAffected, _ := result.RowsAffected()
			logger.Info("Sample product created: %s (rows affected: %d)", product["id"], rowsAffected)
			successCount++
		}
	}

	logger.Info("Sample products creation complete: %d/%d created", successCount, len(sampleProducts))
	return nil
}

// Close 데이터베이스 연결 종료
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
