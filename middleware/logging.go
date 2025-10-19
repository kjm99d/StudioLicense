package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"studiolicense/logger"
	"studiolicense/models"
	"studiolicense/utils"
	"time"
)

// responseWriter HTTP 응답을 캡처하기 위한 래퍼
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += n
	return n, err
}

// LoggingMiddleware HTTP 요청/응답 로깅 미들웨어
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 응답 래퍼 생성
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 요청 ID 생성 (추적용)
		requestID := generateRequestID()
		ctx := context.WithValue(r.Context(), "request_id", requestID)

		// 요청 로깅
		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"method":     r.Method,
			"path":       r.URL.Path,
			"query":      r.URL.RawQuery,
			"ip":         getClientIP(r),
			"user_agent": r.UserAgent(),
		}).Info("HTTP Request")

		// 다음 핸들러 실행
		next.ServeHTTP(rw, r.WithContext(ctx))

		// 응답 로깅
		duration := time.Since(start)

		logLevel := getLogLevelForStatus(rw.statusCode)
		logger.WithFields(map[string]interface{}{
			"request_id":  requestID,
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      rw.statusCode,
			"duration_ms": duration.Milliseconds(),
			"size":        rw.written,
		}).Log(logLevel, "HTTP Response")
	}
}

// getLogLevelForStatus 상태 코드에 따른 로그 레벨 결정
func getLogLevelForStatus(statusCode int) logger.LogLevel {
	switch {
	case statusCode >= 500:
		return logger.ERROR
	case statusCode >= 400:
		return logger.WARN
	default:
		return logger.INFO
	}
}

// getClientIP 클라이언트 IP 추출
func getClientIP(r *http.Request) string {
	// X-Forwarded-For 헤더 확인 (프록시 환경)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// X-Real-IP 헤더 확인
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// RemoteAddr 사용
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// generateRequestID 요청 ID 생성
func generateRequestID() string {
	id, _ := utils.GenerateID("")
	return id
}

// AuthMiddleware JWT 인증 미들웨어 (로깅 추가)
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value("request_id")

		// Authorization 헤더 확인
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			logger.WithFields(map[string]interface{}{
				"request_id": requestID,
				"ip":         getClientIP(r),
			}).Warn("Missing authorization header")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrorResponse("Authorization header required", nil))
			return
		}

		// Bearer 토큰 추출
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.WithFields(map[string]interface{}{
				"request_id": requestID,
				"ip":         getClientIP(r),
			}).Warn("Invalid authorization header format")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrorResponse("Invalid authorization header format", nil))
			return
		}

		token := parts[1]

		// 토큰 검증
		claims, err := utils.ValidateToken(token)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"request_id": requestID,
				"ip":         getClientIP(r),
				"error":      err.Error(),
			}).Warn("Invalid or expired token")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(models.ErrorResponse("Invalid or expired token", err))
			return
		}

		logger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"admin_id":   claims.AdminID,
			"username":   claims.Username,
		}).Debug("Admin authenticated")

		// Context에 관리자 정보 저장
		ctx := context.WithValue(r.Context(), "admin_id", claims.AdminID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "role", claims.Role)

		// 다음 핸들러 실행
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// SetJSONHeader JSON 헤더 설정 미들웨어
func SetJSONHeader(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	}
}

// CORSMiddleware CORS 설정 미들웨어
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// ChainMiddleware 미들웨어 체인
func ChainMiddleware(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
