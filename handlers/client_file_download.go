package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"studiolicense/models"
	"studiolicense/utils"
)

// DownloadProductFile serves a client-facing, signed download URL for product file assets.
// @Summary 제품 파일 다운로드 (클라이언트용)
// @Description 서명된 다운로드 URL을 사용하여 제품 파일을 전송합니다.
// @Tags 라이선스-파일
// @Produce octet-stream
// @Param file_id path string true "파일 ID"
// @Param exp query int true "만료 타임스탬프(Unix)"
// @Param nonce query string true "임의 난수"
// @Param sig query string true "서명 값"
// @Success 200 "파일 스트림"
// @Failure 400 {object} models.APIResponse "요청 파라미터 오류"
// @Failure 403 {object} models.APIResponse "서명 검증 실패 또는 만료"
// @Failure 404 {object} models.APIResponse "파일 없음"
// @Failure 500 {object} models.APIResponse "내부 오류"
// @Router /api/license/files/{file_id} [get]
func DownloadProductFile(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/api/license/files/")
	if fileID == "" || fileID == r.URL.Path {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse("File ID is required", nil))
		return
	}

	exp := r.URL.Query().Get("exp")
	nonce := r.URL.Query().Get("nonce")
	sig := r.URL.Query().Get("sig")

	if err := utils.ValidateSignedDownloadRequest(fileID, exp, nonce, sig); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(models.ErrorResponse("Invalid or expired download link", err))
		return
	}

	file, err := getFileRecord(fileID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.ErrorResponse("File not found", nil))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse("Failed to load file metadata", err))
		}
		return
	}

	// streamFileDownload handles errors and response headers internally.
	streamFileDownload(w, r, file)
}
