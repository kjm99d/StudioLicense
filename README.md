# Studio License Server

스튜디오 라이선스 서버는 소프트웨어 라이선스를 손쉽게 발급·관리하기 위한 Go 기반 백엔드와 현대적인 웹 대시보드를 제공하는 프로젝트입니다.  
Productos, 정책, 디바이스, 관리자에 대한 CRUD와 감사 로그, Swagger 기반의 REST API 문서를 포함하고 있습니다.

---

## 주요 기능

### 관리자 기능
- Super Admin / Admin 계층 구조와 RBAC(Role-Based Access Control)
- 서브 관리자 생성·비활성화·비밀번호 초기화
- 모든 관리자 활동에 대한 세부 로그 기록

### 라이선스 관리
- 라이선스 생성 / 조회 / 수정 / 삭제 / 폐기
- 제품·정책과 연계된 라이선스 발급
- 상태(활성, 만료, 폐기) 및 디바이스 할당량 실시간 추적
- 스케줄러를 통한 만료 라이선스 자동 처리

### 정책 관리
- JSON 기반 정책 정의 및 버전 관리
- 정책 CRUD 및 라이선스와의 매핑
- 정책 변경에 대한 감사 로그 자동 기록

### 디바이스 관리
- 하드웨어 지문을 이용한 디바이스 활성화
- 디바이스 비활성화/재활성화 API 제공
- 디바이스 활동 로그 및 청소(Cleanup) 작업

### 로깅 · 관측성
- 라이선스·디바이스 지표 요약 대시보드
- 관리자/디바이스 활동 로그
- 클라이언트 애플리케이션 로그 수집 엔드포인트
- 다중 출력 및 로테이션을 지원하는 구조화 로거

---

## 요구 사항

- Go 1.21 이상
- MySQL 8.0 이상
- ES6 모듈을 지원하는 최신 브라우저 (관리자 대시보드용)

---

## 디렉터리 구조

```
├── database/           # DB 초기화 및 헬퍼
├── handlers/           # REST API 핸들러
├── logger/             # 구조화 로거 및 로테이션
├── middleware/         # 인증/권한/로깅 미들웨어
├── models/             # DTO 및 상수 정의
├── scheduler/          # 백그라운드 작업 (만료 체크)
├── utils/              # 공용 유틸리티 (암호화, 시간 등)
├── web/                # 웹 대시보드 (index.html, JS 모듈)
└── docs/               # Swagger 문서
```

---

## 로컬 개발 환경 준비

### 1. 저장소 클론
```powershell
git clone https://github.com/your-org/StudioLicense.git
cd StudioLicense
```

### 2. PowerShell UTF-8 설정 (권장)
Windows PowerShell에서는 한글이 깨지는 것을 방지하기 위해 세션 시작 시 아래 명령을 실행하세요.

```powershell
chcp 65001 > $null
$PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'
$PSDefaultParameterValues['Set-Content:Encoding'] = 'utf8'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
```

### 3. 의존성 설치
```powershell
go mod tidy
```

### 4. 데이터베이스 설정
- `studiolicense` 데이터베이스를 MySQL에 생성합니다.
- 필요 시 `main.go`의 DSN을 환경에 맞게 수정하세요.
  ```
  root:root@tcp(localhost:3306)/studiolicense?parseTime=true&loc=Asia%2FSeoul
  ```
- 첫 실행 시 테이블과 기본 Super Admin(`admin` / `admin123`) 계정이 자동으로 생성됩니다.

### 5. 서버 실행
```powershell
go run main.go
```

관리자 대시보드: **http://localhost:8080/web/**  
API 문서(Swagger): **http://localhost:8080/swagger/index.html**

---

## 테스트 실행

```powershell
go test ./...
```

---

## 로그 · 타임존

- 모든 일시는 Asia/Seoul 타임존을 기준으로 `DATETIME` 컬럼에 저장됩니다.
- `utils/time.go`에서 시간 파싱/포맷을 일원화하고 있습니다.
- 로그는 콘솔과 `/logs/server-YYYY-MM-DD.log` 파일에 동시 기록되며, 로테이션 및 보관 기간은 `logger.Config`로 조정 가능합니다.

---

## API 추가 가이드

1. `models/`에 데이터 모델 정의
2. `handlers/`에 비즈니스 로직 및 핸들러 작성
3. `main.go`에 라우팅 등록
4. Swagger 주석(`@Summary`, `@Description` 등) 추가

## 웹 대시보드 확장

1. `web/js/pages/`에 새로운 페이지 모듈 추가
2. `web/index.html`에 필요한 구조물/모달 추가
3. `web/js/main.js`에서 모듈을 import 및 초기화
4. 필요에 따라 공통 유틸/컴포넌트 확장

---

## 기여 방법

1. 저장소를 포크하고 브랜치를 생성합니다.  
   `git checkout -b feature/awesome-feature`
2. 변경 사항을 커밋합니다.  
   `git commit -m "Add awesome feature"`
3. 테스트를 실행합니다.  
   `go test ./...`
4. 포크한 저장소에 푸시 후 Pull Request를 생성합니다.

문제나 개선 제안은 Issues 탭을 이용해 주세요.
