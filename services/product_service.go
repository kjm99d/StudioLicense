package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"studiolicense/models"
	"studiolicense/utils"
)

var (
	// ErrProductNameConflict는 동일한 이름의 제품이 이미 존재할 때 반환됩니다.
	ErrProductNameConflict = errors.New("product name already exists")
	// ErrProductNotFound는 제품이 존재하지 않을 때 반환됩니다.
	ErrProductNotFound = errors.New("product not found")
	// ErrProductLinkedLicenses는 연결된 라이선스로 인해 삭제가 제한될 때 반환됩니다.
	ErrProductLinkedLicenses = errors.New("product has linked licenses")
)

// ProductFilter는 제품 조회 시 필요한 필터 정보를 담습니다.
type ProductFilter struct {
	Status  string
	Scope   models.AdminResourcePermissionConfig
	IsSuper bool
	AdminID string
}

// ProductService는 제품 도메인에 대한 비즈니스 로직을 정의합니다.
type ProductService interface {
	Create(ctx context.Context, req models.CreateProductRequest, creatorID string) (models.Product, error)
	List(ctx context.Context, filter ProductFilter) ([]models.Product, error)
	Get(ctx context.Context, id string) (models.Product, error)
	Update(ctx context.Context, id string, req models.UpdateProductRequest) error
	Delete(ctx context.Context, id string) error
}

type productService struct {
	db SQLExecutor
}

// NewProductService는 ProductService 구현체를 생성합니다.
func NewProductService(db SQLExecutor) ProductService {
	return &productService{db: db}
}

func (s *productService) Create(ctx context.Context, req models.CreateProductRequest, creatorID string) (models.Product, error) {
	id, err := utils.GenerateID("prod")
	if err != nil {
		return models.Product{}, err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO products (id, name, description, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, req.Name, req.Description, models.ProductStatusActive, creatorID, now, now,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return models.Product{}, ErrProductNameConflict
		}
		return models.Product{}, err
	}

	return models.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Status:      models.ProductStatusActive,
		CreatedBy:   creatorID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *productService) List(ctx context.Context, filter ProductFilter) ([]models.Product, error) {
	query := `SELECT id, name, description, status, created_by, created_at, updated_at FROM products WHERE 1=1`
	args := make([]any, 0)

	if strings.TrimSpace(filter.Status) != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	if !filter.IsSuper {
		sqlFragment, fragmentArgs := utils.BuildResourceFilter(filter.Scope, "id", "created_by", filter.AdminID)
		query += sqlFragment
		args = append(args, fragmentArgs...)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		var (
			product   models.Product
			createdBy sql.NullString
		)
		if err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.Status, &createdBy, &product.CreatedAt, &product.UpdatedAt); err != nil {
			return nil, err
		}
		if createdBy.Valid {
			product.CreatedBy = createdBy.String
		}
		products = append(products, product)
	}

	return products, rows.Err()
}

func (s *productService) Get(ctx context.Context, id string) (models.Product, error) {
	var (
		product   models.Product
		createdBy sql.NullString
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, status, created_by, created_at, updated_at
		FROM products WHERE id = ?`,
		id,
	).Scan(&product.ID, &product.Name, &product.Description, &product.Status, &createdBy, &product.CreatedAt, &product.UpdatedAt)

	if err == sql.ErrNoRows {
		return models.Product{}, ErrProductNotFound
	}
	if err != nil {
		return models.Product{}, err
	}
	if createdBy.Valid {
		product.CreatedBy = createdBy.String
	}
	return product, nil
}

func (s *productService) Update(ctx context.Context, id string, req models.UpdateProductRequest) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE products
		SET name = ?, description = ?, status = ?, updated_at = ?
		WHERE id = ?`,
		req.Name, req.Description, req.Status, time.Now().Format("2006-01-02 15:04:05"), id,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrProductNameConflict
		}
		return err
	}

	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return ErrProductNotFound
	}
	return err
}

func (s *productService) Delete(ctx context.Context, id string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM licenses WHERE product_id = ?", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrProductLinkedLicenses
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM products WHERE id = ?", id)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return ErrProductNotFound
	}
	return err
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "1062")
}
