# API 테스트 예제

## 1. 관리자 로그인

```bash
curl -X POST http://localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"admin\",\"password\":\"admin123\"}"
```

**응답에서 token을 복사하세요!**

---

## 2. 라이선스 생성

```bash
# Windows PowerShell
$token = "YOUR_TOKEN_HERE"

Invoke-RestMethod -Uri "http://localhost:8080/api/admin/licenses" `
  -Method POST `
  -Headers @{
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
  } `
  -Body (@{
    product_name = "MyApp Pro"
    product_version = "1.0.0"
    customer_name = "홍길동"
    customer_email = "hong@example.com"
    max_devices = 2
    expires_at = "2026-10-19T00:00:00Z"
    notes = "테스트 라이선스"
  } | ConvertTo-Json)
```

**응답에서 license_key를 복사하세요!**

---

## 3. 라이선스 목록 조회

```bash
# Windows PowerShell
Invoke-RestMethod -Uri "http://localhost:8080/api/admin/licenses" `
  -Method GET `
  -Headers @{
    "Authorization" = "Bearer $token"
  }
```

---

## 4. 라이선스 활성화 (클라이언트)

```bash
# Windows PowerShell
$licenseKey = "YOUR_LICENSE_KEY_HERE"

Invoke-RestMethod -Uri "http://localhost:8080/api/license/activate" `
  -Method POST `
  -Headers @{
    "Content-Type" = "application/json"
  } `
  -Body (@{
    license_key = $licenseKey
    device_info = @{
      cpu_id = "BFEBFBFF000906E9"
      motherboard_sn = "ABC123456789"
      mac_address = "00:1A:2B:3C:4D:5E"
      disk_serial = "S3Y1NY0M123456"
      machine_id = "12345678-1234-1234-1234-123456789012"
      os = "Windows 11"
      os_version = "22H2"
      hostname = "TEST-PC"
    }
  } | ConvertTo-Json)
```

---

## 5. 라이선스 검증 (앱 실행 시)

```bash
# Windows PowerShell
Invoke-RestMethod -Uri "http://localhost:8080/api/license/validate" `
  -Method POST `
  -Headers @{
    "Content-Type" = "application/json"
  } `
  -Body (@{
    license_key = $licenseKey
    device_info = @{
      cpu_id = "BFEBFBFF000906E9"
      motherboard_sn = "ABC123456789"
      mac_address = "00:1A:2B:3C:4D:5E"
      disk_serial = "S3Y1NY0M123456"
      machine_id = "12345678-1234-1234-1234-123456789012"
      os = "Windows 11"
      os_version = "22H2"
      hostname = "TEST-PC"
    }
  } | ConvertTo-Json)
```

---

## 6. 대시보드 통계

```bash
# Windows PowerShell
Invoke-RestMethod -Uri "http://localhost:8080/api/admin/dashboard/stats" `
  -Method GET `
  -Headers @{
    "Authorization" = "Bearer $token"
  }
```

---

## cURL 버전 (Linux/Mac/Git Bash)

### 로그인
```bash
curl -X POST http://localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### 라이선스 생성
```bash
TOKEN="YOUR_TOKEN_HERE"

curl -X POST http://localhost:8080/api/admin/licenses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "MyApp Pro",
    "product_version": "1.0.0",
    "customer_name": "홍길동",
    "customer_email": "hong@example.com",
    "max_devices": 2,
    "expires_at": "2026-10-19T00:00:00Z",
    "notes": "테스트 라이선스"
  }'
```

### 라이선스 활성화
```bash
LICENSE_KEY="YOUR_LICENSE_KEY_HERE"

curl -X POST http://localhost:8080/api/license/activate \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "'$LICENSE_KEY'",
    "device_info": {
      "cpu_id": "BFEBFBFF000906E9",
      "motherboard_sn": "ABC123456789",
      "mac_address": "00:1A:2B:3C:4D:5E",
      "disk_serial": "S3Y1NY0M123456",
      "machine_id": "12345678-1234-1234-1234-123456789012",
      "os": "Windows 11",
      "os_version": "22H2",
      "hostname": "TEST-PC"
    }
  }'
```
