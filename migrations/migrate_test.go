package migrations

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyUpgradesSourceOnlySchema(t *testing.T) {
	dsn := os.Getenv("CULLY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CULLY_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "cully_migrate_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := `"` + name + `"`
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE") })
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = name + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err = pool.Exec(ctx, "CREATE TABLE cully_schema_versions(version integer PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, schema); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "INSERT INTO cully_schema_versions(version) VALUES(2)"); err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	if _, err = pool.Exec(ctx, "INSERT INTO cully_entries(id,owner_subject,section,entry_type,summary,assistant,occurred_at) VALUES($1::uuid,'owner-a','company','work','old note','codex',now())", id); err != nil {
		t.Fatal(err)
	}
	if err = Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = Apply(ctx, pool); err != nil {
		t.Fatal("migration is not idempotent:", err)
	}
	var summary string
	var ref *string
	if err = pool.QueryRow(ctx, "SELECT summary,session_ref FROM cully_entries WHERE id=$1::uuid", id).Scan(&summary, &ref); err != nil || summary != "old note" || ref != nil {
		t.Fatalf("old record lost: summary=%q ref=%v err=%v", summary, ref, err)
	}
	var version5, validated bool
	if err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM cully_schema_versions WHERE version=5)").Scan(&version5); err != nil || !version5 {
		t.Fatalf("version 5 missing: %v", err)
	}
	if err = pool.QueryRow(ctx, "SELECT convalidated FROM pg_constraint WHERE conrelid='cully_entries'::regclass AND conname='cully_entries_entry_type_check'").Scan(&validated); err != nil || !validated {
		t.Fatalf("task entry constraint not validated: %v", err)
	}
}
