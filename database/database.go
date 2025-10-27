package database

import (
	"database/sql"
	"fmt"
	"studiolicense/logger"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

// Initialize 데이터베이스 초기화 (MySQL 전용)
// dsn: MySQL DSN 문자열 (예: "user:password@tcp(localhost:3306)/dbname")
func Initialize(dsn string) error {
	var err error

	// DSN이 없으면 에러
	if dsn == "" {
		return fmt.Errorf("MySQL DSN is required")
	}

	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 연결 테스트
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
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
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 제품 테이블
		`CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			description TEXT,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_products_status (status)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 정책 테이블
		`CREATE TABLE IF NOT EXISTS policies (
			id VARCHAR(50) PRIMARY KEY,
			policy_name VARCHAR(255) UNIQUE NOT NULL,
			policy_data LONGTEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
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
			expires_at DATETIME NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			notes TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL,
			FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE SET NULL,
			INDEX idx_licenses_key (license_key),
			INDEX idx_licenses_product (product_id),
			INDEX idx_licenses_status (status),
			INDEX idx_licenses_expires (expires_at)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 디바이스 활성화 테이블
		`CREATE TABLE IF NOT EXISTS device_activations (
			id VARCHAR(50) PRIMARY KEY,
			license_id VARCHAR(50) NOT NULL,
			device_fingerprint VARCHAR(255) NOT NULL,
			device_info LONGTEXT NOT NULL,
			device_name VARCHAR(255),
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			activated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_validated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deactivated_at DATETIME NULL,
			FOREIGN KEY (license_id) REFERENCES licenses(id) ON DELETE CASCADE,
			UNIQUE KEY unique_device (license_id, device_fingerprint),
			INDEX idx_devices_license (license_id),
			INDEX idx_devices_fingerprint (device_fingerprint),
			INDEX idx_devices_status (status)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 디바이스 활동 로그 테이블
		`CREATE TABLE IF NOT EXISTS device_activity_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			device_id VARCHAR(50) NOT NULL,
			license_id VARCHAR(50) NOT NULL,
			action VARCHAR(100) NOT NULL,
			details LONGTEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_admin (admin_id),
			INDEX idx_created (created_at)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 파일 자산 테이블
		`CREATE TABLE IF NOT EXISTS files (
			id VARCHAR(50) PRIMARY KEY,
			original_name VARCHAR(255) NOT NULL,
			stored_name VARCHAR(255) NOT NULL,
			description TEXT,
			mime_type VARCHAR(120) NOT NULL,
			file_size BIGINT NOT NULL,
			checksum VARCHAR(128),
			storage_path VARCHAR(500) NOT NULL,
			uploaded_by VARCHAR(50),
			uploaded_username VARCHAR(100),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_files_original_name (original_name),
			INDEX idx_files_uploaded_by (uploaded_by)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 제품-파일 매핑 테이블
		`CREATE TABLE IF NOT EXISTS product_files (
			id VARCHAR(50) PRIMARY KEY,
			product_id VARCHAR(50) NOT NULL,
			file_id VARCHAR(50) NOT NULL,
			label VARCHAR(255) NOT NULL,
			description TEXT,
			sort_order INT NOT NULL DEFAULT 0,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			delivery_url VARCHAR(1000),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
			FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
			UNIQUE KEY unique_product_file_label (product_id, label),
			INDEX idx_product_files_product (product_id),
			INDEX idx_product_files_active (is_active)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,

		// 클라이언트 로그 테이블
		`CREATE TABLE IF NOT EXISTS client_logs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			license_key VARCHAR(255) NOT NULL,
			device_id VARCHAR(50),
			level VARCHAR(20) NOT NULL,
			category VARCHAR(50) NOT NULL,
			message TEXT NOT NULL,
			details LONGTEXT,
			stack_trace LONGTEXT,
			app_version VARCHAR(50),
			os_version VARCHAR(100),
			client_ip VARCHAR(50),
			client_timestamp DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_license_key (license_key),
			INDEX idx_device_id (device_id),
			INDEX idx_level (level),
			INDEX idx_category (category),
			INDEX idx_created (created_at)
		) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`,
	}

	// MySQL 테이블 생성
	for _, sql := range baseTables {
		if _, err := DB.Exec(sql); err != nil {
			// 이미 존재하는 인덱스/테이블 오류, 존재하지 않는 컬럼 오류 무시
			if !contains(err.Error(), "already exists") &&
				!contains(err.Error(), "Duplicate key name") &&
				!contains(err.Error(), "Duplicate") &&
				!contains(err.Error(), "doesn't exist in table") {
				logger.Warn("SQL execution warning: %v", err)
			}
		}
	}

	return nil
}

// contains 문자열 포함 여부 확인
func contains(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
