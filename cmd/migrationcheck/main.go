package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"pass_gen/internal/repository/postgres"
)

func main() {
	var dsn string
	flag.StringVar(&dsn, "dsn", os.Getenv("PASSGEN_TEST_DSN"), "postgres dsn")
	flag.Parse()

	if dsn == "" {
		fmt.Fprintln(os.Stderr, "dsn is required (flag -dsn or PASSGEN_TEST_DSN env)")
		os.Exit(2)
	}

	repo, err := postgres.NewRepository(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repository init failed: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = repo.Close() }()

	ctx := context.Background()
	if err := repo.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "db ping failed: %v\n", err)
		os.Exit(1)
	}

	if err := repo.CreateSchema(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "schema apply failed: %v\n", err)
		os.Exit(1)
	}

	if err := checkTable(ctx, repo, "password_hashes"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := checkTable(ctx, repo, "generation_audit"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("migration check passed")
}

func checkTable(ctx context.Context, repo *postgres.Repository, table string) error {
	var exists bool
	query := `
SELECT EXISTS (
	SELECT 1
	FROM information_schema.tables
	WHERE table_schema = 'public' AND table_name = $1
)`
	if err := repo.DB().QueryRowContext(ctx, query, table).Scan(&exists); err != nil {
		return fmt.Errorf("table %s check failed: %w", table, err)
	}
	if !exists {
		return fmt.Errorf("table %s does not exist after migration", table)
	}
	return nil
}
