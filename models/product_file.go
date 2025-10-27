package models

// ProductFile describes the relationship between a product and a stored file asset.
type ProductFile struct {
	ID          string     `json:"id"`
	ProductID   string     `json:"product_id"`
	FileID      string     `json:"file_id"`
	Label       string     `json:"label"`
	Description string     `json:"description,omitempty"`
	SortOrder   int        `json:"sort_order"`
	IsActive    bool       `json:"is_active"`
	DeliveryURL string     `json:"delivery_url,omitempty"`
	File        *FileAsset `json:"file,omitempty"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

// AttachProductFileRequest represents the payload to link a file to a product.
type AttachProductFileRequest struct {
	ProductID   string `json:"product_id"`
	FileID      string `json:"file_id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	SortOrder   int    `json:"sort_order"`
	DeliveryURL string `json:"delivery_url,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// UpdateProductFileRequest represents the payload to update a product file mapping.
type UpdateProductFileRequest struct {
	ID          string  `json:"id"`
	Label       *string `json:"label,omitempty"`
	Description *string `json:"description,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	DeliveryURL *string `json:"delivery_url,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// ProductFileResponse is returned to clients during license validation.
type ProductFileResponse struct {
	ID          string `json:"id"`
	FileID      string `json:"file_id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	SortOrder   int    `json:"sort_order"`
	URL         string `json:"url"`
	DownloadURL string `json:"download_url"`
	DeliveryURL string `json:"delivery_url,omitempty"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	Checksum    string `json:"checksum,omitempty"`
	StoragePath string `json:"storage_path"`
	UpdatedAt   string `json:"updated_at"`
}
