# Studio License Server

디바이스 하드웨어 기반 라이선스 관리 서버입니다.

## 주요 기능

### 👥 관리자 시스템
- **Super Admin & Sub Admin 계층 구조**
- Sub Admin 관리 (생성, 비밀번호 초기화, 삭제)
- 역할 기반 접근 제어 (RBAC)
- 비밀번호 변경 기능

### 🎫 라이선스 관리
- 라이선스 생성, 조회, 수정, 삭제
- 제품별 라이선스 발급
- **정책 기반 라이선스 제어**
- 라이선스 상태 관리 (활성/만료/폐기)
- 최대 디바이스 수 제한
- 만료일 관리

### 🛡️ 정책 관리 (Policy System)
- **제품과 독립적인 정책 시스템**
- 정책 생성, 수정, 삭제
- JSON 형식의 유연한 정책 데이터
- 정책 활성/비활성 상태 관리
- 라이선스에 정책 할당 및 변경
- 정책 삭제 시 라이선스 정책 변경 가능

### 📦 제품 관리
- 제품 CRUD 기능
- 제품별 상태 관리
- 제품별 라이선스 통계

### 🖥️ 디바이스 관리
- 하드웨어 핑거프린트 기반 디바이스 인증
- 디바이스 활성화/비활성화/재활성화
- **실시간 디바이스 슬롯 추적** (남은 슬롯/최대 슬롯)
- 디바이스별 활동 로그
- **비활성 디바이스 정리** (0일부터 설정 가능)
- 디바이스 상세 정보 표시 (Client ID, Hostname, OS 등)

### 📊 대시보드 & 로깅
- 실시간 통계 (라이선스, 디바이스)
- 관리자 활동 로그
- 디바이스 활동 로그
- **클라이언트 로그 시스템** (신규)
  - 클라이언트 애플리케이션 로그 수집
  - 로그 레벨별 필터링 (DEBUG, INFO, WARN, ERROR, FATAL)
  - 카테고리별 분류
  - 스택 트레이스 기록
  - 날짜 기반 로그 정리 기능
- 필터링 및 검색 기능

## 요구사항

- Go 1.21 이상
- MySQL 8.0 이상

## 데이터베이스 셋업

### MySQL 설정

```bash
# MySQL에서 데이터베이스 생성
mysql -u root -p -e "CREATE DATABASE studiolicense CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
```

```go
// main.go (기본 설정)
database.Initialize("root:root@tcp(localhost:3306)/studiolicense")
```

### 자동 테이블 생성

Go 서버를 실행하면 다음 테이블들이 **자동으로 생성**됩니다:

- **admins** - 관리자 계정 (ID, 비밀번호, 역할)
- **products** - 제품 정보
- **policies** - 정책 정보
- **licenses** - 라이선스 키 및 활성화 정보 (정책 ID 포함)
- **device_activations** - 디바이스별 활성화 기록
- **admin_activity_logs** - 관리자 작업 로그
- **device_activity_logs** - 디바이스 접근 로그
- **client_logs** - 클라이언트 애플리케이션 로그 (신규)

> **별도의 마이그레이션 작업은 필요 없습니다.** 데이터베이스만 생성하면 나머지는 모두 자동으로 처리됩니다.

## 설치 및 실행

### 1단계: 프로젝트 클론
```bash
git clone https://github.com/kjm99d/StudioLicense.git
cd StudioLicense
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
> - ✅ 만료된 라이선스 자동 체크 (매일 00:00)

### 4단계: 웹 접속
```
http://localhost:8080/web/
```

**기본 관리자 계정:**
- Username: `admin`
- Password: `admin123`

> ⚠️ **첫 로그인 후 반드시 비밀번호를 변경하세요!**

## 관리자 역할

### Super Admin (최고 관리자)
- ✅ 모든 관리 기능 접근
- ✅ Sub Admin 생성, 수정, 삭제
- ✅ 디바이스 정리 기능
- ✅ 모든 활동 로그 조회

### Admin (일반 관리자)
- ✅ 라이선스 관리
- ✅ 제품 관리
- ✅ 정책 관리
- ✅ 디바이스 조회 및 관리
- ✅ 대시보드 조회
- ❌ Sub Admin 관리 불가
- ❌ 디바이스 정리 불가

## 정책 시스템 (Policy System)

### 정책이란?

정책은 라이선스에 적용할 수 있는 규칙이나 설정을 JSON 형식으로 정의한 것입니다.

### 정책 예시

```json
{
  "feature": "premium",
  "max_projects": 100,
  "export_formats": ["mp4", "avi", "mov"],
  "cloud_storage": true
}
```

### 정책 관리

1. **정책 생성**: 정책명과 JSON 데이터로 정책 생성
2. **정책 수정**: 정책 데이터 및 상태(활성/비활성) 수정
3. **정책 삭제**: 정책 삭제 (연결된 라이선스는 정책 없음으로 변경)
4. **라이선스에 적용**: 라이선스 생성 시 또는 수정 시 정책 선택

### 라이선스-정책 관계

- 라이선스는 **선택적으로** 하나의 정책을 가질 수 있음
- 정책 없이도 라이선스 발급 가능
- 라이선스 수정 시 정책 변경 가능
- 정책 삭제 시 해당 정책을 사용하는 라이선스의 정책을 변경 필요

## 디바이스 핑거프린트 시스템

### 디바이스 식별 방법

디바이스는 하드웨어 정보를 조합한 **핑거프린트**로 식별됩니다.

### 핑거프린트 구성 요소

```
SHA256(client_id|cpu_id|motherboard_sn|mac_address|disk_serial|machine_id)
```

- **Client ID**: 클라이언트 애플리케이션에서 생성한 고유 ID
- **CPU ID**: CPU 제조사 및 모델 정보
- **Motherboard SN**: 메인보드 시리얼 번호
- **MAC Address**: 네트워크 인터페이스 MAC 주소
- **Disk Serial**: 디스크 시리얼 번호
- **Machine ID**: 운영체제 머신 ID

> ⚠️ **주의**: Hostname은 핑거프린트에 포함되지 않습니다. (변경 가능하므로)

### 디바이스 활성화 프로세스

1. 클라이언트가 하드웨어 정보를 수집
2. 서버로 라이선스 키와 함께 전송
3. 서버가 핑거프린트 생성 및 저장
4. 최대 디바이스 수 확인
5. 슬롯이 남아있으면 활성화 성공

### 디바이스 검증 프로세스

1. 클라이언트가 라이선스 키와 핑거프린트 전송
2. 서버가 핑거프린트 일치 여부 확인
3. 라이선스 만료 여부 확인
4. 디바이스 활성 상태 확인
5. 모두 통과하면 검증 성공

### 디바이스 슬롯 관리

**예시 시나리오:**
- 라이선스 최대 디바이스: 30개
- 현재 활성 디바이스: 27개
- 표시: `3/30` (남은 슬롯 3개)

**디바이스 비활성화 시:**
- 활성 디바이스: 27 → 26개
- 표시: `4/30` (남은 슬롯 4개)
- 즉시 새로운 디바이스 등록 가능

**디바이스 재활성화 시:**
- 활성 디바이스: 26 → 27개
- 표시: `3/30` (남은 슬롯 3개)
- 슬롯이 남아있어야 재활성화 가능

**최대 디바이스 수 변경 제한:**
- 현재 27개 활성 → 최대 30개는 가능
- 현재 27개 활성 → 최대 25개는 **불가능**
- 오류: "Cannot reduce max devices to 25. Currently 27 devices are active."

## API 엔드포인트

### 관리자 인증 API
- `POST /api/admin/login` - 로그인
- `POST /api/admin/change-password` - 비밀번호 변경
- `GET /api/admin/me` - 현재 관리자 정보

### 관리자 관리 API (Super Admin 전용)
- `GET /api/admin/admins` - Sub Admin 목록
- `POST /api/admin/admins/create` - Sub Admin 생성
- `POST /api/admin/admins/:id/reset-password` - 비밀번호 초기화
- `DELETE /api/admin/admins/:id` - Sub Admin 삭제

### 라이선스 관리 API
- `GET /api/admin/licenses` - 라이선스 목록 (페이징, 필터링)
- `GET /api/admin/licenses?id={id}` - 라이선스 상세
- `POST /api/admin/licenses` - 라이선스 생성
- `PUT /api/admin/licenses?id={id}` - 라이선스 수정
- `DELETE /api/admin/licenses?id={id}` - 라이선스 삭제
- `GET /api/admin/licenses/devices?id={id}` - 라이선스 디바이스 목록

### 정책 관리 API
- `GET /api/admin/policies` - 정책 목록
- `GET /api/admin/policies/:id` - 정책 상세
- `POST /api/admin/policies` - 정책 생성
- `PUT /api/admin/policies/:id` - 정책 수정
- `DELETE /api/admin/policies/:id` - 정책 삭제

### 제품 관리 API
- `GET /api/admin/products` - 제품 목록
- `POST /api/admin/products` - 제품 생성
- `PUT /api/admin/products/:id` - 제품 수정
- `DELETE /api/admin/products/:id` - 제품 삭제

### 디바이스 관리 API
- `POST /api/admin/devices/deactivate` - 디바이스 비활성화
- `POST /api/admin/devices/reactivate` - 디바이스 재활성화
- `POST /api/admin/devices/cleanup` - 비활성 디바이스 정리 (0일 이상)
- `GET /api/admin/devices/logs?device_id={id}` - 디바이스 활동 로그

### 대시보드 API
- `GET /api/admin/dashboard/stats` - 통계 조회
- `GET /api/admin/dashboard/activities` - 활동 로그 조회

### 클라이언트 로그 API
- `POST /api/client/logs` - 클라이언트 로그 전송 (인증 불필요)
- `GET /api/admin/client-logs` - 로그 조회 (페이징, 필터링)
- `DELETE /api/admin/client-logs/cleanup` - 날짜 기반 로그 정리

### 클라이언트 API (인증 불필요)
- `POST /api/license/activate` - 라이선스 활성화
- `POST /api/license/validate` - 라이선스 검증

> 📚 상세한 API 문서는 `http://localhost:8080/swagger/` 에서 확인 가능

## 프로젝트 구조

```
StudioLicense/
├── main.go                    # 서버 진입점 및 라우팅
├── go.mod                     # Go 모듈 정의
├── go.sum                     # 의존성 체크섬
│
├── database/
│   └── database.go            # DB 초기화 및 스키마 생성
│
├── models/                    # 데이터 모델
│   ├── admin.go               # 관리자 모델
│   ├── admin_activity.go      # 관리자 활동 로그
│   ├── device.go              # 디바이스 모델
│   ├── device_log.go          # 디바이스 활동 로그
│   ├── license.go             # 라이선스 모델
│   ├── product.go             # 제품 모델
│   └── response.go            # API 응답 구조
│
├── handlers/                  # API 핸들러
│   ├── admin_device.go        # 디바이스 관리
│   ├── admin_license.go       # 라이선스 관리
│   ├── admin_user.go          # Sub Admin 관리
│   ├── auth.go                # 인증
│   ├── client_license.go      # 클라이언트 라이선스 API
│   ├── dashboard.go           # 대시보드
│   └── product.go             # 제품 관리
│
├── middleware/                # 미들웨어
│   ├── logging.go             # 요청/응답 로깅
│   └── roles.go               # 역할 기반 접근 제어
│
├── utils/                     # 유틸리티
│   ├── admin_log.go           # 관리자 활동 로깅
│   ├── crypto.go              # 암호화 (bcrypt, hash)
│   ├── device_log.go          # 디바이스 활동 로깅
│   └── jwt.go                 # JWT 토큰 생성/검증
│
├── scheduler/                 # 스케줄러
│   └── scheduler.go           # 크론 작업 (만료 체크 등)
│
├── logger/                    # 로거
│   └── logger.go              # 구조화된 로깅
│
├── web/                       # 프론트엔드
│   ├── index.html             # 메인 HTML
│   ├── styles.css             # 전역 스타일
│   ├── app.js                 # 레거시 JS (디바이스 렌더링)
│   ├── products.js            # 제품 관리 JS
│   └── js/
│       ├── api.js             # API 호출 유틸
│       ├── main.js            # 메인 앱 로직
│       ├── modals.js          # 모달 시스템
│       ├── state.js           # 전역 상태 관리
│       ├── ui.js              # UI 헬퍼
│       ├── utils.js           # 유틸리티 함수
│       ├── legacy-bridge.js   # 레거시 코드 브릿지
│       └── pages/
│           ├── admins.js      # 관리자 페이지
│           ├── client-logs.js # 클라이언트 로그 페이지 (신규)
│           ├── dashboard.js   # 대시보드 페이지
│           ├── licenses.js    # 라이선스 페이지
│           ├── policies.js    # 정책 페이지
│           └── products.js    # 제품 페이지
│
├── docs/                      # Swagger API 문서
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
│
├── logs/                      # 로그 파일 (자동 생성)
├── API_TESTS.md               # API 테스트 가이드
└── README.md                  # 이 파일
```

## 보안 기능

### 인증 & 권한
- ✅ JWT 토큰 기반 인증 (Bearer Token)
- ✅ bcrypt 비밀번호 해싱 (cost 10)
- ✅ 역할 기반 접근 제어 (Super Admin / Admin)
- ✅ 토큰 만료 시간 관리

### 데이터 보호
- ✅ 디바이스 핑거프린트 해싱 (SHA-256)
- ✅ SQL Injection 방지 (Prepared Statement)
- ✅ XSS 방지 (HTML Escape)

### 감사 추적
- ✅ 모든 관리자 활동 로깅
- ✅ 디바이스 활동 로깅
- ✅ 로그인 시도 기록

## 최근 업데이트

### v2.3.0 - 데이터베이스 단순화 & UI 개선 (2025-10-26)

#### 주요 변경사항
- 🗄️ **MySQL 전용으로 전환**
  - SQLite 지원 제거로 코드베이스 단순화
  - MySQL 최적화된 쿼리 및 인덱스 구조
  - 안정적인 프로덕션 환경 지원

- 📝 **클라이언트 로그 시스템 추가**
  - 클라이언트 애플리케이션에서 로그 수집
  - 5가지 로그 레벨 (DEBUG, INFO, WARN, ERROR, FATAL)
  - 라이선스 키 및 디바이스별 필터링
  - 스택 트레이스 및 상세 정보 기록
  - 날짜 기반 로그 정리 기능

- 🎨 **UI/UX 대폭 개선**
  - 로그인 시 alert 팝업 제거 (인라인 에러 표시)
  - 모달 z-index 충돌 문제 전면 해결
  - 모든 생성/수정/삭제 작업에서 일관된 모달 처리
  - 관리자 생성, 정책 관리, 라이선스 관리, 제품 관리, 비밀번호 변경, 로그 정리 등 모든 모달에서 alert가 최상위에 표시
  - 에러 발생 시에도 모달을 먼저 닫고 alert 표시

- 🐛 **버그 수정**
  - 대시보드 활동 로그 500 에러 수정 (UNION 쿼리 타입 불일치)
  - MySQL INDEX 생성 구문 수정 (`IF NOT EXISTS` 제거)
  - NULL 값 처리 개선 (빈 문자열로 통일)
  - 에러 메시지 로깅 강화

#### 기술적 개선
- `database.Initialize()` 함수 단순화 (단일 DSN 파라미터)
- 테이블 생성 시 인덱스를 함께 정의하여 안정성 향상
- 모달 닫기 → 300ms 대기 → alert 표시 패턴 전역 적용
- `showConfirm` → 작업 수행 → 모달 닫기 → `showAlert` 순서 표준화

### v2.2.0 - 프로젝트 전반 검토 & 동기화 (2025-10-26)

#### 개선사항
- 📚 **Swagger 문서 완전 동기화**
  - 모든 API 엔드포인트 문서화 완료
  - 클라이언트 로그 API 3개 추가
  - 요청/응답 스키마 정확성 향상
  - 95% 이상 문서화 달성

- 🔍 **코드 품질 개선**
  - 미사용 함수 및 중복 코드 제거
  - 일관된 에러 처리 패턴 적용
  - 코드 주석 및 문서 개선

### v2.1.0 - 디바이스 관리 강화 (2025-10-20)

#### 신규 기능
- 🖥️ **디바이스 관리 개선**
  - 실시간 디바이스 슬롯 추적 (X/Y 형식으로 표시)
  - 비활성화 시 즉시 슬롯 해제
  - 재활성화 시 즉시 슬롯 사용
  - 상세 창 및 목록에서 실시간 업데이트

- 🧹 **비활성 디바이스 정리 강화**
  - 0일부터 설정 가능 (기존 1일 이상에서 변경)
  - 0일 설정 시 모든 비활성 디바이스 즉시 삭제
  - MySQL 날짜 형식 호환성 개선
  - 정리 로그에 cutoff_date 표시

- 🔒 **라이선스 수정 검증 강화**
  - 최대 디바이스 수를 현재 활성 디바이스보다 작게 설정 불가
  - 명확한 오류 메시지 제공
  - "현재 X개 디바이스가 활성화되어 있습니다" 안내

- 🎨 **UI/UX 개선**
  - 디바이스 상세 정보 표시 (Client ID, Hostname, OS, OS Version 등)
  - 모달 z-index 우선순위 조정 (Alert > 일반 모달)
  - 디바이스 카드에서 불필요한 삭제 버튼 제거
  - 디바이스 슬롯 레이블 변경 ("최대 디바이스" → "디바이스 슬롯")

#### 개선사항
- 디바이스 비활성화/재활성화 후 상세 창 실시간 갱신
- 라이선스 목록에서 남은 슬롯 표시 개선
- cleanup API에서 날짜 형식 문자열로 변환하여 MySQL 호환성 향상
- 모달 우선순위 시스템 개선 (15000+ for alerts)

#### 버그 수정
- 비활성 디바이스 정리 쿼리 수정 (`<` → `<=`)
- 디바이스 정보 JSON 파싱 오류 수정 (snake_case 지원)
- Alert 모달이 일반 모달 뒤에 표시되는 문제 해결

### v2.0.0 - 정책 시스템 도입 (2025-10-20)

#### 신규 기능
- 🛡️ **정책 관리 시스템 추가**
  - 제품과 독립적인 정책 시스템
  - JSON 기반 유연한 정책 데이터
  - 정책 CRUD 기능
  - 정책 활성/비활성 상태 관리

- 🎫 **라이선스-정책 연동**
  - 라이선스 생성 시 정책 선택
  - 라이선스 수정 시 정책 변경
  - 정책 없이도 라이선스 발급 가능

- 🎨 **UI/UX 개선**
  - 정책 관리 페이지 추가
  - 라이선스 수정 모달 추가
  - 테이블 버튼 순서 개선 (상세-수정-삭제)
  - 일관된 디자인 시스템 적용

- 📊 **활동 로그 강화**
  - 정책 생성/수정/삭제 로그
  - 라이선스 수정 로그
  - 이모지 아이콘으로 가독성 향상

#### 개선사항
- 라이선스 테이블에 정책 컬럼 추가
- 라이선스 상세 페이지에 정책 정보 표시
- 정책 삭제 시 라이선스 영향도 처리
- 모달 스타일 및 레이아웃 개선

## 개발 가이드

### 새로운 API 추가

1. `models/` 에 데이터 모델 추가
2. `handlers/` 에 핸들러 함수 작성
3. `main.go` 에 라우트 등록
4. Swagger 주석 추가 (`@Summary`, `@Description` 등)

### 프론트엔드 페이지 추가

1. `web/js/pages/` 에 페이지 JS 파일 생성
2. `web/index.html` 에 HTML 섹션 추가
3. `web/js/main.js` 에서 import 및 라우팅 연결
4. 필요한 모달을 `index.html` 에 추가

### 데이터베이스 스키마 변경

1. `database/database.go` 에서 `CREATE TABLE` 문 수정
2. 기존 DB 백업 후 삭제하고 재실행하여 테이블 재생성
3. 또는 `ALTER TABLE` 문을 직접 실행

## 문제 해결

### 포트 충돌 (8080 사용 중)
```bash
# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F

# Linux/Mac
lsof -i :8080
kill -9 <PID>
```

### 데이터베이스 연결 실패
- MySQL 서비스가 실행 중인지 확인
- 접속 정보(username, password, host) 확인
- 데이터베이스가 생성되었는지 확인

### 로그인 실패
- 기본 계정: `admin / admin123`
- 비밀번호를 잊었다면 DB에서 직접 수정 또는 DB 재생성

## 라이선스

MIT License

## 기여

이슈와 PR은 언제나 환영합니다!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 문의

프로젝트 관련 문의사항은 Issues 탭을 이용해주세요.
