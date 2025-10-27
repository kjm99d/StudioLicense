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
	// 모든 응답 메시지를 한국어로 표준화
	msg := "요청이 성공적으로 처리되었습니다."
	if message != "" {
		// 전달된 메시지가 있어도 한국어 표준 문구로 대체합니다.
		msg = "요청이 성공적으로 처리되었습니다."
	}
	return APIResponse{
		Status:  "success",
		Message: msg,
		Data:    data,
	}
}

// ErrorResponse 에러 응답 생성
func ErrorResponse(message string, err error) APIResponse {
	// 모든 에러 메시지를 한국어로 표준화하고, 내부 오류 세부정보 노출을 최소화합니다.
	msg := "요청 처리 중 오류가 발생했습니다."
	errMsg := ""
	// 내부 에러 문자열은 클라이언트에 직접 노출하지 않습니다. 필요 시 서버 로그를 참고하십시오.
	return APIResponse{
		Status:  "error",
		Message: msg,
		Error:   errMsg,
	}
}
