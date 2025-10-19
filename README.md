# Studio License Server

디바이스 하드웨어 기반 라이선스 관리 서버입니다.

## 주요 기능

- Super Admin & Sub Admin 계층
- Sub Admin 관리 (생성, 비밀번호 초기화, 삭제)
- 라이선스 관리 (생성, 조회, 수정, 삭제)
- 제품 관리
- 디바이스 관리
- 대시보드 및 활동 로깅

## 요구사항

- Go 1.21 이상
- MySQL 8.0 이상

## 데이터베이스 셋업

### MySQL 데이터베이스 생성

```bash
# MySQL에서 데이터베이스 생성 (문자열 인코딩 설정)
mysql -u root -p -e "CREATE DATABASE studiolicense CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

### 자동 테이블 생성

Go 서버를 실행하면 다음 테이블들이 **자동으로 생성**됩니다:

- **admins** - 관리자 계정 (ID, 비밀번호, 역할)
- **products** - 제품 정보
- **licenses** - 라이선스 키 및 활성화 정보
- **device_activations** - 디바이스별 활성화 기록
- **admin_activity_logs** - 관리자 작업 로그
- **device_activity_logs** - 디바이스 접근 로그

> **별도의 마이그레이션 작업은 필요 없습니다.** 데이터베이스만 생성하면 나머지는 모두 자동으로 처리됩니다.

### MySQL 연결 설정

**main.go의 database.Initialize 호출 부분 (기본값):**

```go
// MySQL 연결 (기본값)
database.Initialize("mysql", "root:root@tcp(localhost:3306)/studiolicense")
```

**MySQL 접속 정보를 변경하려면:**

```go
// 형식: database.Initialize("mysql", "username:password@tcp(host:port)/dbname")
database.Initialize("mysql", "myuser:mypassword@tcp(192.168.1.100:3306)/studiolicense")
```

## 설치 및 실행

### 1단계: MySQL 데이터베이스 생성
```bash
mysql -u root -p -e "CREATE DATABASE studiolicense CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

### 2단계: 의존성 설치
```bash
go mod tidy
```

### 3단계: 서버 실행
```bash
go run main.go
```

> 서버 시작 시 다음이 **자동으로 실행됩니다:**
> - ✅ 테이블 생성 (CREATE TABLE IF NOT EXISTS)
> - ✅ 기본 관리자 계정 생성 (admin / admin123) - 첫 실행 시에만
> - ✅ 샘플 제품 생성 - 첫 실행 시에만

### 4단계: 웹 접속
```
http://localhost:8080/web/
- Username: admin
- Password: admin123
```

### 재시작 시 동작
- 데이터베이스 종료 → 재시작 시 **데이터는 모두 보존됨** ✅
- 테이블/기본 관리자는 **다시 생성되지 않음** ✅
- 테이블이 이미 존재하면 CREATE TABLE IF NOT EXISTS는 아무 작업도 하지 않음

## 관리자 역할

### Super Admin
- 모든 관리 기능
- Sub Admin 관리

### Admin
- 라이선스 관리
- 제품 관리
- 디바이스 관리
- 대시보드 조회

## API 엔드포인트

> 상세한 API 문서는 `/docs/swagger.json` 또는 `/docs/swagger.yaml` 참고

### 관리자 API
- POST /api/admin/login - 로그인
- GET /api/admin/me - 현재 관리자 정보
- GET /api/admin/admins - Sub Admin 목록
- POST /api/admin/admins/create - Sub Admin 생성
- POST /api/admin/admins/{id}/reset-password - 비밀번호 초기화
- DELETE /api/admin/admins/{id} - Sub Admin 삭제
- GET/POST/PUT/DELETE /api/admin/licenses - 라이선스 관리
- GET/POST/PUT/DELETE /api/admin/products - 제품 관리
- GET /api/admin/dashboard/* - 대시보드 및 활동 로그

### 클라이언트 API (인증 불필요)
- POST /api/license/activate - 라이선스 활성화
- POST /api/license/validate - 라이선스 검증
- POST /api/license/deactivate - 라이선스 비활성화

## 프로젝트 구조

```
StudioLicense/
├── main.go                    # 서버 진입점
├── go.mod                     # Go 모듈 정의
├── database/
│   └── database.go            # DB 초기화 및 스키마
├── models/                    # 데이터 모델
│   ├── admin.go
│   ├── admin_activity.go
│   ├── device.go
│   ├── device_log.go
│   ├── license.go
│   ├── product.go
│   └── response.go
├── handlers/                  # API 핸들러
│   ├── admin_device.go
│   ├── admin_license.go
│   ├── admin_user.go
│   ├── auth.go
│   ├── client_license.go
│   ├── dashboard.go
│   └── product.go
├── middleware/                # 미들웨어
│   ├── logging.go
│   └── roles.go
├── utils/                     # 유틸리티
│   ├── admin_log.go
│   ├── crypto.go
│   ├── device_log.go
│   └── jwt.go
├── scheduler/                 # 스케줄러
│   └── scheduler.go
├── logger/                    # 로거
│   └── logger.go
├── web/                       # 프론트엔드
│   ├── index.html
│   ├── styles.css
│   ├── app.js
│   ├── products.js
│   └── js/
│       ├── api.js
│       ├── main.js
│       ├── modals.js
│       ├── state.js
│       ├── ui.js
│       ├── utils.js
│       └── pages/
│           ├── admins.js
│           ├── dashboard.js
│           └── licenses.js
├── logs/                      # 로그 디렉토리 (자동 생성)
└── README.md
```

## 보안

- JWT 토큰 기반 인증
- bcrypt 비밀번호 해싱
- 디바이스 핑거프린트 해싱
- 역할 기반 접근 제어
- 활동 로깅

## 개선사항

- 모달 스타일링 개선
- Sub Admin 관리 기능
- 상태 뱃지 일관성
- API 응답 캐싱
- 활동 로그 한글 표시

## 라이선스

MIT
