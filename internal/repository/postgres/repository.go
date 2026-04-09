package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repository struct {
	db *sql.DB
}

func (r *Repository) DB() *sql.DB {
	return r.db
}

func NewRepository(dsn string) (*Repository, error) {
	if dsn == "" {
		return nil, errors.New("postgres dsn is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *Repository) CreateSchema(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS password_hashes (
	id BIGSERIAL PRIMARY KEY,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS generation_audit (
	id BIGSERIAL PRIMARY KEY,
	password_length INT NOT NULL,
	password_count INT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
	_, err := r.db.ExecContext(ctx, schema)
	return err
}

func (r *Repository) SavePasswordHash(ctx context.Context, hash string) error {
	if hash == "" {
		return errors.New("hash is required")
	}

	_, err := r.db.ExecContext(ctx, `INSERT INTO password_hashes (password_hash) VALUES ($1)`, hash)
	return err
}

func (r *Repository) SaveGenerationAudit(ctx context.Context, length int, count int) error {
	if length <= 0 || count <= 0 {
		return errors.New("length and count must be positive")
	}

	_, err := r.db.ExecContext(ctx, `INSERT INTO generation_audit (password_length, password_count) VALUES ($1, $2)`, length, count)
	return err
}
