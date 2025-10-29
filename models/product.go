package models

// Product 제품 정보
type Product struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	Status      string `json:"status" db:"status"` // active, inactive
	CreatedBy   string `json:"created_by" db:"created_by"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

// ProductStatus 상태 상수
const (
	ProductStatusActive   = "active"
	ProductStatusInactive = "inactive"
)

// CreateProductRequest 제품 생성 요청
type CreateProductRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateProductRequest 제품 수정 요청
type UpdateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}
