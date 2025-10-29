package services

import (
	"context"
	"database/sql"
)

// SQLExecutor는 서비스 계층이 데이터베이스 구현 세부사항으로부터 분리되도록 해주는 최소한의 인터페이스입니다.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type sqlDBExecutor struct {
	db *sql.DB
}

// NewSQLExecutor는 *sql.DB를 감싸는 SQLExecutor를 생성합니다.
func NewSQLExecutor(db *sql.DB) SQLExecutor {
	return &sqlDBExecutor{db: db}
}

func (s *sqlDBExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s *sqlDBExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s *sqlDBExecutor) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *sqlDBExecutor) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}
