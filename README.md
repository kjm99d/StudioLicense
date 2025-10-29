# Studio License Server

> 하드웨어 지문 기반 소프트웨어 라이선스를 발급·배포·감사하는 풀스택 솔루션  
> Go 1.21 + MySQL 8 + ES6 모듈 프런트엔드

---

## 목차

1. [프로젝트 개요](#프로젝트-개요)  
2. [주요 기능](#주요-기능)  
3. [아키텍처 한눈에 보기](#아키텍처-한눈에-보기)  
4. [시작하기](#시작하기)  
5. [환경 변수 및 설정](#환경-변수-및-설정)  
6. [실행 및 스크립트](#실행-및-스크립트)  
7. [디렉터리 구조](#디렉터리-구조)  
8. [개발 워크플로](#개발-워크플로)  
9. [기여 가이드](#기여-가이드)

---

## 프로젝트 개요

Studio License Server는 SaaS / 온프레미스 환경에서 **제품·정책·라이선스·디바이스**를 통합 관리하기 위한 관리 콘솔과 REST API를 제공합니다.  
RBAC 기반의 관리자 권한 모델과 세밀한 리소스 허용 범위(제품/정책/라이선스)를 지원하며, 모든 활동을 감사 로그로 남깁니다.

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

### 💻 디바이스 & 로그
- 하드웨어 지문 기반 활성화/비활성화 API
- 디바이스 활동 로그 조회 및 청소 스케줄러
- 클라이언트 측 로그 업로드 엔드포인트

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
2. `studiolicense` 데이터베이스 생성
3. `main.go`의 DSN이 환경과 일치하는지 확인  
   기본값: `root:root@tcp(localhost:3306)/studiolicense?parseTime=true&loc=Asia%2FSeoul`
4. 서버 첫 실행 시 테이블/인덱스/샘플 데이터, 
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
