package models

// APIResponse 표준 API 응답 구조
type APIResponse struct {
	Status  string      `json:"status"` // success, error
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PaginatedResponse 페이징 응답
type PaginatedResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    Pagination  `json:"meta"`
}

// Pagination 페이징 정보
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
	TotalCount int `json:"total_count"`
}

// SuccessResponse 성공 응답 생성
func SuccessResponse(message string, data interface{}) APIResponse {
	return APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}
}

// ErrorResponse 에러 응답 생성
func ErrorResponse(message string, err error) APIResponse {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return APIResponse{
		Status:  "error",
		Message: message,
		Error:   errMsg,
	}
}
