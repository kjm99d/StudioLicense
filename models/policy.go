package models

// Policy 정책 정보
type Policy struct {
	ID         string `json:"id" db:"id"`
	PolicyName string `json:"policy_name" db:"policy_name"`
	PolicyData string `json:"policy_data" db:"policy_data"` // JSON 형식의 정책 데이터
	Status     string `json:"status" db:"status"`           // active, inactive
	CreatedAt  string `json:"created_at" db:"created_at"`
	UpdatedAt  string `json:"updated_at" db:"updated_at"`
}

// PolicyStatus 정책 상태 상수
const (
	PolicyStatusActive   = "active"
	PolicyStatusInactive = "inactive"
)

// CreatePolicyRequest 정책 생성 요청
type CreatePolicyRequest struct {
	PolicyName string `json:"policy_name" binding:"required"`
	PolicyData string `json:"policy_data" binding:"required"` // JSON 형식
}

// UpdatePolicyRequest 정책 수정 요청
type UpdatePolicyRequest struct {
	PolicyName string `json:"policy_name"`
	PolicyData string `json:"policy_data"`
	Status     string `json:"status"`
}

// PolicyResponse 클라이언트 응답용 정책 정보
type PolicyResponse struct {
	ID         string      `json:"id"`
	PolicyName string      `json:"policy_name"`
	PolicyData interface{} `json:"policy_data"` // JSON 파싱된 데이터
}
