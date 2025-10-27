package models

// FileAsset 파일 서버에 저장된 자산 메타데이터
type FileAsset struct {
	ID               string `json:"id"`
	OriginalName     string `json:"original_name"`
	StoredName       string `json:"stored_name"`
	Description      string `json:"description,omitempty"`
	MimeType         string `json:"mime_type"`
	FileSize         int64  `json:"file_size"`
	Checksum         string `json:"checksum,omitempty"`
	StoragePath      string `json:"storage_path"`
	UploadedBy       string `json:"uploaded_by,omitempty"`
	UploadedUsername string `json:"uploaded_username,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	DownloadURL      string `json:"download_url,omitempty"`
}

// FileUploadResult 업로드 결과 응답
type FileUploadResult struct {
	FileAsset
}
