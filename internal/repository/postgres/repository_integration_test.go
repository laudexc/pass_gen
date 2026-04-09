package postgres

import (
	"context"
	"os"
	"testing"
)

func TestRepositoryIntegration_SaveHashAndAudit(t *testing.T) {
	dsn := os.Getenv("PASSGEN_TEST_DSN")
	if dsn == "" {
		t.Skip("PASSGEN_TEST_DSN is not set")
	}

	repo, cleanup := setupIntegrationRepo(t, dsn)
	defer cleanup()

	ctx := context.Background()

	if err := repo.SavePasswordHash(ctx, "$argon2id$v=19$m=65536,t=3,p=2$abc$def"); err != nil {
		t.Fatalf("SavePasswordHash returned error: %v", err)
	}
	if err := repo.SaveGenerationAudit(ctx, 12, 2); err != nil {
		t.Fatalf("SaveGenerationAudit returned error: %v", err)
	}

	var hashes int
	if err := repo.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM password_hashes").Scan(&hashes); err != nil {
		t.Fatalf("count password_hashes failed: %v", err)
	}
	if hashes != 1 {
		t.Fatalf("expected 1 row in password_hashes, got %d", hashes)
	}

	var audits int
	if err := repo.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM generation_audit").Scan(&audits); err != nil {
		t.Fatalf("count generation_audit failed: %v", err)
	}
	if audits != 1 {
		t.Fatalf("expected 1 row in generation_audit, got %d", audits)
	}
}

func setupIntegrationRepo(t *testing.T, dsn string) (*Repository, func()) {
	t.Helper()

	repo, err := NewRepository(dsn)
	if err != nil {
		t.Fatalf("NewRepository returned error: %v", err)
	}

	ctx := context.Background()
	if err := repo.Ping(ctx); err != nil {
		_ = repo.Close()
		t.Fatalf("Ping returned error: %v", err)
	}
	if err := repo.CreateSchema(ctx); err != nil {
		_ = repo.Close()
		t.Fatalf("CreateSchema returned error: %v", err)
	}

	if _, err := repo.db.ExecContext(ctx, "TRUNCATE TABLE password_hashes, generation_audit RESTART IDENTITY"); err != nil {
		_ = repo.Close()
		t.Fatalf("truncate failed: %v", err)
	}

	cleanup := func() {
		_, _ = repo.db.ExecContext(context.Background(), "TRUNCATE TABLE password_hashes, generation_audit RESTART IDENTITY")
		_ = repo.Close()
	}

	return repo, cleanup
}
