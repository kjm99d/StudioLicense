# Studio License Server

> 하드웨어 핑거프린트 기반 소프트웨어 라이선스를 발급·배포·감사하는 풀스택 솔루션  
> Go 1.21 + MySQL 8 + ES6 모듈 프런트엔드 + Swagger + JWT

---

## 목차

1. [프로젝트 개요](#프로젝트-개요)  
2. [기술 스택](#기술-스택)  
3. [주요 기능](#주요-기능)  
4. [아키텍처 한눈에 보기](#아키텍처-한눈에-보기)  
5. [시작하기](#시작하기)  
6. [환경 변수 및 설정](#환경-변수-및-설정)  
7. [실행 및 스크립트](#실행-및-스크립트)  
8. [디렉터리 구조](#디렉터리-구조)  
9. [적용 예시](#적용-예시)  
10. [SaaS 라이선스 온보딩 가이드](#saas-라이선스-온보딩-가이드)  
11. [개발 워크플로](#개발-워크플로)  
12. [기여 가이드](#기여-가이드)

---

## 프로젝트 개요

Studio License Server는 SaaS / 온프레미스 환경에서 **제품·정책·라이선스·디바이스**를 통합 관리하기 위한 관리 콘솔과 REST API를 제공합니다.  
RBAC 기반의 관리자 권한 모델과 세밀한 리소스 허용 범위(제품/정책/라이선스)를 지원하며, 모든 활동을 감사 로그로 남깁니다.

---

## 기술 스택

| 레이어 | 기술 | 비고 |
| ------ | ---- | ---- |
| Backend | **Go 1.21**, net/http, database/sql, bcrypt | REST API, RBAC, 활동 로그 |
| Database | **MySQL 8.0+**, SQL | 라이선스·디바이스·정책 저장 |
| Frontend | **ES6 Modules**, Materialize 기반 커스텀 UI | SPA 스타일 관리자 콘솔 |
| Auth & Security | **JWT**, bcrypt, RBAC/Resource Scope | 관리자 인증·권한 제어 |
| 문서화 | **Swaggo (`swag`)**, Swagger UI | API 스펙 자동화 |
| 운영 편의 | Scheduler, 구조화 로거, PowerShell 스크립트 | 만료 처리, 로그 로테이션, 로컬 개발 |

> 외부 프레임워크 의존을 최소화하고 Go 표준 라이브러리와 슬림한 패키지 조합으로 구성했습니다.

---

## 주요 기능

### 👥 관리자 / RBAC
- Super Admin / Admin 역할 분리
- 기능 권한 + 리소스(제품·정책·라이선스) 범위 지정
- 관리자 활동(생성·수정·삭제·로그인 등) 감사 로그

### 🎫 라이선스 & 정책
- 라이선스 CRUD, 폐기, 만료 스케줄링
- 제품/정책과 연동된 라이선스 발급
- 정책 JSON 편집기(폼/JSON 양식 전환)

### 📦 제품 & 파일 배포
- 제품 CRUD
- 제품 전용 파일 관리 (모달 UI + 외부 링크 지원)
- 제품-파일 매핑 및 정렬, 사용자 노출명 관리
- 파일 다운로드 시 JWT 기반 단기 서명 URL 발급으로 안전한 배포

### 💻 디바이스 & 로그
- 하드웨어 지문 기반 활성화/비활성화 API
- 디바이스 활동 로그 조회 및 청소 스케줄러
- 클라이언트 측 로그 업로드 엔드포인트
- 디바이스 슬롯 자동 복구, 만료 라이선스 일괄 처리 스케줄러

### 📊 대시보드 & 모니터링
- 라이선스/디바이스 요약 지표
- 최근 활동 / 경고 카드
- 모든 API에 Swagger 자동 문서화 제공

---

## 아키텍처 한눈에 보기

```
┌─────────────┐
│  web/       │  ES6 모듈 기반 SPA (Materialize 커스텀)
│  ├─ index   │  모달, 탭, 검색 필터 UI
│  └─ js/     │  페이지/컴포넌트/서비스 모듈
└──────┬──────┘
       │ REST API
┌──────▼──────┐
│  handlers/  │  HTTP 핸들러 (입력검증, 응답 포맷)
├──────┬──────┤
│ services/   │  비즈니스 로직 (DB 인터페이스, 리소스 권한)
├──────┬──────┤
│ database/   │  MySQL 초기화, 마이그레이션, 샘플 데이터
└──────┴──────┘
       │
┌──────▼──────┐
│  MySQL 8+   │
└─────────────┘
```

- **services/**: 최근 도입된 레이어로, 제품 관리·리소스 권한·스코프 계산을 담당합니다. 핸들러는 인터페이스에만 의존하므로 테스트/교체가 쉬워졌습니다.
- **handlers/resource_access.go**: 주입형 `ResourceScopeResolver`를 사용해 super admin과 커스텀 스코프를 구분합니다.
- **web/js/pages/products.js**: 제품 파일 관리가 테이블 확장 패널 → 모달 구조로 리팩터링되어, UI 가시성과 상태 초기화가 개선되었습니다.

---

## 시작하기

### 0. 요구 사항
- Go 1.21 이상
- MySQL 8.0 이상 (로컬, Docker, 또는 클라우드 매니지드 서비스)
- PowerShell 7 이상 또는 Bash 셸
- (옵션) Swagger 문서 생성을 위한 `swag` CLI

### 1. 저장소 클론
```bash
git clone https://github.com/your-org/StudioLicense.git
cd StudioLicense
```

### 2. PowerShell UTF-8 환경 (옵션이지만 권장)
```powershell
chcp 65001 > $null
$PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'
$PSDefaultParameterValues['Set-Content:Encoding'] = 'utf8'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
```

### 3. 의존성 설치
```bash
go mod tidy
```

### 4. 데이터베이스 준비
1. MySQL 8.0 이상 실행
2. 관리자 권한 계정 준비 (`root` 또는 별도 계정)
3. 빈 `studiolicense` 데이터베이스 생성
4. `main.go`의 DSN이 환경과 일치하는지 확인  
   기본값: `root:root@tcp(localhost:3306)/studiolicense?parseTime=true&loc=Asia%2FSeoul`
5. 서버 첫 실행 시 테이블/인덱스/샘플 데이터, 
   기본 Super Admin 계정(`admin` / `admin123`)이 자동 생성됩니다.

### 5. 서버 실행
```bash
go run main.go
```

- 관리자 대시보드: <http://localhost:8080/web/>
- Swagger UI: <http://localhost:8080/swagger/index.html>

---

## 환경 변수 및 설정

| 변수 | 기본값 | 설명 |
| ---- | ------ | ---- |
| `MYSQL_DSN` | `root:root@tcp(localhost:3306)/studiolicense?parseTime=true&loc=Asia%2FSeoul` | 데이터베이스 연결 정보 |
| `LOG_LEVEL` | `INFO` | `DEBUG/INFO/WARN/ERROR` |
| `LOG_DIR` | `./logs` | 로그 파일 저장 경로 |
| `JWT_SECRET` | (필수) | 관리자 인증 토큰 서명 키 |
| `SCHEDULER_ENABLED` | `true` | 만료 라이선스/디바이스 정리 스케줄러 ON/OFF |

> **TIP**: `.env` 파일을 사용하지 않고 Go 환경변수나 Docker Compose를 통해 주입하는 방식을 추천합니다.

---

## 실행 및 스크립트

```bash
# 서버 실행
go run main.go

# 전체 테스트
go test ./...

# Swagger 문서 재생성 (swaggo 설치 필요)
swag init -g main.go
```

> 프런트엔드는 ES6 네이티브 모듈을 사용하므로 별도의 번들링 단계가 필요 없습니다.

---

## 디렉터리 구조

```
├── database/                   # DB 초기화 및 샘플 데이터
├── handlers/                   # HTTP 핸들러 (입력 검증 · 시리얼라이즈)
├── logger/                     # 구조화 로거, 파일 로테이션
├── middleware/                 # Auth, RBAC, 로깅, CORS
├── models/                     # DTO, 상수, Enum
├── scheduler/                  # 만료 라이선스/디바이스 정리 작업
├── services/                   # 비즈니스 로직 (제품/권한/스코프)
├── utils/                      # 공용 함수 (암호화, 시간, ID 등)
├── web/                        # 관리자 웹 대시보드
│   ├── index.html              # SPA 엔트리
│   ├── styles.css              # 커스텀 스타일
│   └── js/                     # ES6 페이지/컴포넌트/서비스 모듈
└── docs/                       # Swagger JSON/YAML
```

---

## 적용 예시

- **데스크톱 상용 소프트웨어**  
  Windows/macOS에서 실행되는 그래픽·CAD·음원 편집 툴이 로컬 PC 하드웨어 지문을 기준으로 라이선스를 활성화/비활성화하도록 구성할 수 있습니다. 관리자는 만료/갱신을 콘솔에서 처리하고, 클라이언트는 `/api/license/activate`와 `/api/license/validate` 엔드포인트를 통해 상태를 확인합니다.

- **온프레미스 설치형 SaaS 모듈**  
  고객사별 설치 버전의 백엔드 서비스나 에이전트가 라이선스 유효성을 주기적으로 검증하도록 연동할 수 있습니다. 제품·정책 체계를 활용해 고객사마다 다른 기능 플래그를 부여하고, 장비 수 제한을 RBAC로 제어합니다.

- **플러그인 / 애드온 마켓**  
  메인 애플리케이션과 별도로 판매되는 플러그인(예: 영상 편집용 이펙트, IDE 확장팩)이 자체 라이선스를 요구하는 경우, 이 서버를 중앙 허브로 사용해 키 발급·폐기를 자동화할 수 있습니다. 플러그인 관리 포털에서 REST API를 호출하여 키를 생성하고 사용자에게 전달합니다.

- **산업용 장비/펌웨어 배포**  
  공장 자동화 장비나 IoT 게이트웨이의 펌웨어를 제품별 리소스로 등록하고, 현장 엔지니어가 장비 교체 시 라이선스를 재할당하도록 운영할 수 있습니다. 디바이스 로그 API를 통해 현장 이벤트를 수집하여 감사 로그로 남깁니다.

> 예시 프로젝트들은 `docs/swagger.yaml`에 정의된 REST API를 클라이언트(데스크톱 앱, CLI, 서비스)에서 호출하는 방식으로 통합하면 됩니다. JWT 토큰을 발급받아 관리자 기능을 호출하거나, 최종 상품에서는 `/api/license/*` 클라이언트 엔드포인트만 노출하는 식으로 구성할 수 있습니다.

---

## SaaS 라이선스 온보딩 가이드

다중 고객에게 서비스형(Managed SaaS)으로 라이선스를 제공하려면 아래의 패턴으로 온보딩을 진행하면 됩니다.

### 1. 고객사 테넌트 모델 정의
- **제품(Product)**: 고객사가 사용하는 대표 서비스 라인. 예) `studio-pro`, `studio-basic`
- **정책(Policy)**: 서비스 플래그, 요금제별 제한(디바이스 수, 기능 단위)을 JSON으로 정의
- **라이선스(License)**: 고객사 테넌트당 1개 이상 발급, 고객 메타데이터와 만료일 관리

> 정책 JSON 예시는 `docs/swagger.yaml` → `models.PolicyData` 참고. 플래그형, 제한형 JSON 구조를 자유롭게 설계할 수 있습니다.

### 2. 관리자 콘솔 또는 API로 리소스 생성
1. **제품 생성**  
   - `/api/admin/products` POST (또는 웹 콘솔 사용)  
   - `status=active`로 등록 후, 고객사가 접근 가능한 파일·정책과 연결
2. **정책 템플릿 작성**  
   - `/api/admin/policies` POST  
   - Policy JSON에 기능 플래그, 동시 디바이스 수 제한 등 SaaS 플랜별 제약을 정의
3. **라이선스 발급**  
   - `/api/admin/licenses` POST  
   - `product_id`, `policy_id`, `customer_name`, `customer_email`, `max_devices`, `expires_at` 등 입력  
   - 발급 후 응답에서 `license_key` 확보 (클라이언트 앱 or 고객 관리자 포털로 전달)

### 3. 고객 환경에서 라이선스 활성화
1. 고객이 설치한 데스크톱/서버 에이전트가 라이선스 키와 디바이스 정보를 제출  
   ```http
   POST /api/license/activate
   Authorization: Bearer {클라이언트용 JWT 또는 공용 키}
   {
     "license_key": "LIC-XXXX-XXXX-XXXX",
     "device_info": {
       "hostname": "...",
       "mac_address": "...",
       ...
     }
   }
   ```
2. 서버는 허용된 디바이스 수, 만료 정보, 정책 여부를 검증 후 활성화 ID 반환  
3. 클라이언트는 주기적으로 `/api/license/validate`를 호출해 상태가 유효한지 확인

### 4. 고객 해지/만료 처리
- 관리자가 `/api/admin/licenses?id=...` DELETE 또는 만료일 수정으로 접근 차단
- 만료된 라이선스는 스케줄러가 자동으로 `status=expired` 처리
- 고객 디바이스가 validate 요청 시 `403`을 받게 되어 서비스 접근이 차단

### 5. 다중 환경 운영 팁
- 고객사별 별도 라이선스 정책이 필요하면 **정책 복제**로 맞춤 JSON을 제작
- RBAC 리소스 모드를 활용해 내부 직원에게 고객별 License 접근 범위를 제한
- 고객 포털이 있다면, 관리자 API를 호출하여 라이선스 발급/취소를 자동화
- 로그 API(`/api/admin/devices/logs`, `/api/admin/licenses/devices`)로 SLA 준수 및 감사 데이터를 확인

> SaaS 기반 주문/구독 시스템과 연동할 때는 주문 완료 이벤트에서 라이선스를 자동 생성하고, 결제 실패/해지 이벤트에서 `RevokeLicense` API를 호출하는 패턴을 추천합니다.

---

## 개발 워크플로

1. **feature 브랜치 생성**  
   `git checkout -b feature/add-x`
2. **변경 & 자동화**  
   - 핸들러 추가 → 서비스에 비즈니스 로직 구현  
   - Swagger 주석(`@Summary`, `@Tags`, `@Router`) 유지
3. **테스트**  
   `go test ./...`  
   서비스 레이어는 인터페이스 기반이라 단위 테스트를 작성하기 쉽습니다.
4. **빌드 검증**  
   `go build ./...`
5. **PR 생성 시 체크리스트**  
   - [ ] 테스트 통과
   - [ ] Swagger 문서 최신 상태
   - [ ] README 또는 docs 업데이트 (필요 시)
   - [ ] UI 변경은 스크린샷 첨부

---

## 기여 가이드

1. Issue 또는 Discussion으로 개선 아이디어/버그를 공유합니다.  
2. Fork → Branch → Commit → Pull Request 순으로 진행합니다.
3. 커밋 메시지는 **한국어/영어 혼용 가능**, 단 의미가 명확하도록 작성합니다.
4. 대규모 변경(PR)에는 테스트 전략 및 회귀 검증 방법을 기술해주세요.

감사합니다! 문제가 있거나 새로운 기능 아이디어가 있다면 언제든지 Issue를 열어 주세요.
