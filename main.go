package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"studiolicense/database"
	_ "studiolicense/docs" // Swagger 문서
	"studiolicense/handlers"
	"studiolicense/logger"
	"studiolicense/middleware"
	"studiolicense/models"
	"studiolicense/scheduler"
	"studiolicense/services"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"
)

var productHTTPHandler *handlers.ProductHandler

// @title Studio License Server API
// @version 1.0
// @description 하드웨어 기반 라이선스 관리 서버
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT 토큰을 입력하세요. 형식: Bearer {token}

func main() {
	// 로거 초기화
	logConfig := logger.Config{
		Level:      logger.INFO, // 운영: INFO, 개발: DEBUG
		LogDir:     "./logs",
		MaxSize:    10 * 1024 * 1024, // 10MB
		MaxAge:     7,                // 7일
		UseColor:   true,
		ShowCaller: false,
		Prefix:     "",
	}

	if err := logger.Initialize(logConfig); err != nil {
		logger.Fatal("Failed to initialize logger: %v", err)
	}

	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	logger.Info("🚀 Studio License Server Starting")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 데이터베이스 초기화 (MySQL 전용)
	// DSN 형식: "user:password@tcp(host:port)/dbname"
	mysqlDSN := "root:root@tcp(localhost:3306)/studiolicense?parseTime=true&loc=Asia%2FSeoul"
	if err := database.Initialize(mysqlDSN); err != nil {
		logger.Fatal("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// 서비스 계층 초기화
	sqlExecutor := services.NewSQLExecutor(database.DB)
	adminResourceService := services.NewAdminResourcePermissionService(sqlExecutor)
	scopeResolver := services.NewResourceScopeResolver(adminResourceService)

	handlers.SetResourceScopeResolver(scopeResolver)
	handlers.SetAdminResourcePermissionService(adminResourceService)

	productService := services.NewProductService(sqlExecutor)
	productHTTPHandler = handlers.NewProductHandler(productService, scopeResolver)

	// 스케줄러 시작 (만료된 라이선스 자동 처리)
	scheduler.StartScheduler()

	// 라우터 설정
	mux := http.NewServeMux()

	// 정적 파일 서빙 (웹 프론트엔드)
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/web/", http.StripPrefix("/web/", fs))

	// Swagger 문서
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Public 엔드포인트
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/health", healthHandler)

	// 인증 API (관리자)
	mux.HandleFunc("/api/admin/login",
		middleware.ChainMiddleware(
			handlers.Login,
			middleware.LoggingMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 관리자 API (인증 필요)
	mux.HandleFunc("/api/admin/me",
		middleware.ChainMiddleware(
			handlers.GetMe,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 비밀번호 변경 API (인증 필요)
	mux.HandleFunc("/api/admin/change-password",
		middleware.ChainMiddleware(
			handlers.ChangePassword,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 라이선스 관리 API
	mux.HandleFunc("/api/admin/licenses",
		middleware.ChainMiddleware(
			licenseHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/admin/licenses/",
		middleware.ChainMiddleware(
			licenseDetailHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 라이선스 디바이스 조회 API (인증 필요)
	mux.HandleFunc("/api/admin/licenses/devices",
		middleware.ChainMiddleware(
			handlers.GetLicenseDevices,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionLicensesView),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 디바이스 비활성화 API (관리자, 인증 필요)
	mux.HandleFunc("/api/admin/devices/deactivate",
		middleware.ChainMiddleware(
			handlers.DeactivateDeviceByAdmin,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionDevicesManage),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 디바이스 재활성화 API (관리자, 인증 필요)
	mux.HandleFunc("/api/admin/devices/reactivate",
		middleware.ChainMiddleware(
			handlers.ReactivateDeviceByAdmin,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionDevicesManage),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 디바이스 활동 로그 조회 API (관리자, 인증 필요)
	mux.HandleFunc("/api/admin/devices/logs",
		middleware.ChainMiddleware(
			handlers.GetDeviceActivityLogs,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionDevicesView),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 비활성 디바이스 정리 API (관리자, 인증 필요)
	mux.HandleFunc("/api/admin/devices/cleanup",
		middleware.ChainMiddleware(
			handlers.CleanupInactiveDevices,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionDevicesManage),
			middleware.RequireRoles("super_admin"),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 디바이스 개별 삭제 API (관리자, 인증 필요)
	mux.HandleFunc("/api/admin/devices/delete",
		middleware.ChainMiddleware(
			handlers.DeleteDevice,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequireRoles("super_admin", "admin"),
			middleware.RequirePermissions(models.PermissionDevicesManage),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 제품 관리 API (인증 필요)
	mux.HandleFunc("/api/admin/products",
		middleware.ChainMiddleware(
			productHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/admin/products/",
		middleware.ChainMiddleware(
			productDetailHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 제품 파일 매핑 관리 API
	mux.HandleFunc("/api/admin/product-files",
		middleware.ChainMiddleware(
			productFileRouter,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 정책 관리 API (인증 필요)
	mux.HandleFunc("/api/admin/policies",
		middleware.ChainMiddleware(
			policyHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/admin/policies/",
		middleware.ChainMiddleware(
			policyDetailHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 대시보드 API
	mux.HandleFunc("/api/admin/dashboard/stats",
		middleware.ChainMiddleware(
			handlers.GetDashboardStats,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionDashboardView),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/admin/dashboard/activities",
		middleware.ChainMiddleware(
			handlers.GetRecentActivities,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionDashboardView),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 관리자 계정 관리 API (슈퍼 관리자 전용)
	mux.HandleFunc("/api/admin/admins",
		middleware.ChainMiddleware(
			handlers.ListAdmins,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequireRoles("super_admin"),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/admin/admins/create",
		middleware.ChainMiddleware(
			handlers.CreateAdmin,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequireRoles("super_admin"),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/admin/permissions/catalog",
		middleware.ChainMiddleware(
			handlers.GetAdminPermissionCatalog,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequireRoles("super_admin"),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 관리자 비밀번호 초기화 API (슈퍼 관리자 전용)
	mux.HandleFunc("/api/admin/admins/",
		middleware.ChainMiddleware(
			adminDetailHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequireRoles("super_admin"),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 클라이언트 API (인증 불필요)
	mux.HandleFunc("/api/license/activate",
		middleware.ChainMiddleware(
			handlers.ActivateLicense,
			middleware.LoggingMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/license/validate",
		middleware.ChainMiddleware(
			handlers.ValidateLicense,
			middleware.LoggingMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/license/files/",
		middleware.ChainMiddleware(
			handlers.DownloadProductFile,
			middleware.LoggingMiddleware,
			middleware.CORSMiddleware,
		))

	// 클라이언트 로그 API (인증 불필요)
	mux.HandleFunc("/api/client/logs",
		middleware.ChainMiddleware(
			handlers.SubmitClientLogs,
			middleware.LoggingMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 관리자 - 클라이언트 로그 조회 API (인증 필요)
	mux.HandleFunc("/api/admin/client-logs",
		middleware.ChainMiddleware(
			handlers.GetClientLogs,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionClientLogsView),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 관리자 - 클라이언트 로그 삭제 API (슈퍼 관리자 전용)
	mux.HandleFunc("/api/admin/client-logs/cleanup",
		middleware.ChainMiddleware(
			handlers.DeleteClientLogs,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.RequirePermissions(models.PermissionClientLogsManage),
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 파일 서버 API
	mux.HandleFunc("/api/admin/files",
		middleware.ChainMiddleware(
			fileHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	mux.HandleFunc("/api/admin/files/",
		middleware.ChainMiddleware(
			fileDetailHandler,
			middleware.LoggingMiddleware,
			middleware.AuthMiddleware,
			middleware.CORSMiddleware,
			middleware.SetJSONHeader,
		))

	// 서버 설정
	port := ":8080"
	server := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown 설정
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Warn("Received shutdown signal")
		database.Close()
		os.Exit(0)
	}()

	logger.Info("Server listening on http://localhost%s", port)
	logger.Info("Admin Panel: http://localhost%s/web/", port)
	logger.Info("Swagger UI: http://localhost%s/swagger/index.html", port)
	logger.Info("Log directory: ./logs")
	logger.Info("Database: MySQL - root:root@tcp(localhost:3306)/studiolicense")
	logger.Info("Default admin - username: admin, password: admin123")
	logger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Server failed to start: %v", err)
	}
}

// homeHandler 루트 핸들러
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Studio License Server",
		"version": "1.0.0",
	}
	w.Write([]byte(`{"status":"success","message":"Studio License Server","version":"1.0.0"}`))
	_ = response
}

// healthHandler 헬스체크 핸들러
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success","message":"Server is healthy"}`))
}

// licenseHandler 라이선스 목록/생성 핸들러
func licenseHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionLicensesView) {
			return
		}
		handlers.GetLicenses(w, r)
	case http.MethodPost:
		if !middleware.EnsurePermission(w, r, models.PermissionLicensesManage) {
			return
		}
		handlers.CreateLicense(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// licenseDetailHandler 라이선스 상세/수정/삭제 핸들러
func licenseDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/licenses/")
	path = strings.Trim(path, "/")
	if path != "" {
		if strings.Contains(path, "/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), "path_license_id", path)
		r = r.WithContext(ctx)
	}

	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionLicensesView) {
			return
		}
		handlers.GetLicense(w, r)
	case http.MethodPut:
		if !middleware.EnsurePermission(w, r, models.PermissionLicensesManage) {
			return
		}
		handlers.UpdateLicense(w, r)
	case http.MethodDelete:
		if !middleware.EnsurePermission(w, r, models.PermissionLicensesManage) {
			return
		}
		handlers.DeleteLicense(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// productHandler 제품 목록/생성 핸들러
func productHandler(w http.ResponseWriter, r *http.Request) {
	if productHTTPHandler == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("product handler not initialized", nil))
		return
	}

	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionProductsView) {
			return
		}
		productHTTPHandler.List(w, r)
	case http.MethodPost:
		if !middleware.EnsurePermission(w, r, models.PermissionProductsManage) {
			return
		}
		productHTTPHandler.Create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// productDetailHandler 제품 상세/수정/삭제 핸들러
func productDetailHandler(w http.ResponseWriter, r *http.Request) {
	if productHTTPHandler == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ErrorResponse("product handler not initialized", nil))
		return
	}

	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionProductsView) {
			return
		}
		productHTTPHandler.Get(w, r)
	case http.MethodPut:
		if !middleware.EnsurePermission(w, r, models.PermissionProductsManage) {
			return
		}
		productHTTPHandler.Update(w, r)
	case http.MethodDelete:
		if !middleware.EnsurePermission(w, r, models.PermissionProductsManage) {
			return
		}
		productHTTPHandler.Delete(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// productFileRouter 제품-파일 매핑 핸들러
func productFileRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionFilesView) {
			return
		}
		handlers.ListProductFiles(w, r)
	case http.MethodPost:
		if !middleware.EnsurePermission(w, r, models.PermissionFilesManage) {
			return
		}
		handlers.AttachProductFile(w, r)
	case http.MethodPut:
		if !middleware.EnsurePermission(w, r, models.PermissionFilesManage) {
			return
		}
		handlers.UpdateProductFile(w, r)
	case http.MethodDelete:
		if !middleware.EnsurePermission(w, r, models.PermissionFilesManage) {
			return
		}
		handlers.DeleteProductFile(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// policyHandler 정책 목록/생성 핸들러
func policyHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionPoliciesView) {
			return
		}
		handlers.GetAllPolicies(w, r)
	case http.MethodPost:
		if !middleware.EnsurePermission(w, r, models.PermissionPoliciesManage) {
			return
		}
		handlers.CreatePolicy(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// policyDetailHandler 정책 상세/수정/삭제 핸들러
func policyDetailHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionPoliciesView) {
			return
		}
		handlers.GetPolicy(w, r)
	case http.MethodPut:
		if !middleware.EnsurePermission(w, r, models.PermissionPoliciesManage) {
			return
		}
		handlers.UpdatePolicy(w, r)
	case http.MethodDelete:
		if !middleware.EnsurePermission(w, r, models.PermissionPoliciesManage) {
			return
		}
		handlers.DeletePolicy(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// fileHandler 파일 목록/업로드 핸들러
func fileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionFilesView) {
			return
		}
		handlers.ListFiles(w, r)
	case http.MethodPost:
		if !middleware.EnsurePermission(w, r, models.PermissionFilesManage) {
			return
		}
		handlers.UploadFile(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// fileDetailHandler 파일 상세/다운로드/삭제 핸들러
func fileDetailHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !middleware.EnsurePermission(w, r, models.PermissionFilesView) {
			return
		}
		handlers.GetFile(w, r)
	case http.MethodDelete:
		if !middleware.EnsurePermission(w, r, models.PermissionFilesManage) {
			return
		}
		handlers.DeleteFile(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// adminDetailHandler 관리자 상세/삭제 및 비밀번호 초기화 핸들러
func adminDetailHandler(w http.ResponseWriter, r *http.Request) {
	// 경로에서 admin_id 추출: /api/admin/admins/{admin_id} or /api/admin/admins/{admin_id}/reset-password
	pathParts := strings.Split(r.URL.Path, "/")
	// pathParts: ["", "api", "admin", "admins", "{admin_id}", ...] 또는 ["", "api", "admin", "admins", "{admin_id}", "reset-password"]

	if len(pathParts) < 5 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status":"error","message":"Invalid path"}`))
		return
	}

	adminID := pathParts[4]

	// URL에 컨텍스트로 adminID 저장 (PathValue 대신 사용)
	ctx := context.WithValue(r.Context(), "path_admin_id", adminID)
	r = r.WithContext(ctx)

	switch r.Method {
	case http.MethodPut:
		if len(pathParts) > 5 && pathParts[5] == "permissions" {
			handlers.UpdateAdminPermissions(w, r)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status":"error","message":"Invalid request"}`))
	case http.MethodPost:
		// POST로 비밀번호 초기화 (경로: /api/admin/admins/{admin_id}/reset-password)
		if len(pathParts) > 5 && pathParts[5] == "reset-password" {
			handlers.ResetAdminPassword(w, r)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status":"error","message":"Invalid request"}`))
	case http.MethodDelete:
		handlers.DeleteAdmin(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
