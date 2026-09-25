// Package dbtest gives store contract tests a freshly migrated Postgres
// schema (phase-persistence.md verification). Tests call Pool; without
// TEST_DATABASE_URL they are skipped, so plain `go test ./...` never
// needs a database.
package dbtest

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db"
)

// Pool returns a pool whose search_path is a schema of its own
// ("test_<name>"), dropped and re-migrated first -- go test runs packages
// in parallel, so each package gets a separate schema.
func Pool(t *testing.T, name string) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping the Postgres store run")
	}
	ctx := context.Background()
	schema := "test_" + name
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE; CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

// User inserts a bare users row, for stores whose tables reference users.
func User(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, display_name, password_hash) VALUES ($1, $1 || '@test', $1, '\x00')`, id); err != nil {
		t.Fatal(err)
	}
}
