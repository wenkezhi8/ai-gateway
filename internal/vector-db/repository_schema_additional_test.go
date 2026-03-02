package vectordb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLiteRepository_EnsureColumnAndImportJobColumn_ShouldAddMissingColumns(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := &SQLiteRepository{db: db}
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE vector_collections (id TEXT PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatalf("create vector_collections error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE vector_import_jobs (id TEXT PRIMARY KEY, collection_id TEXT NOT NULL)`); err != nil {
		t.Fatalf("create vector_import_jobs error = %v", err)
	}

	if err := repo.ensureColumn(ctx, "vector_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		t.Fatalf("ensureColumn(add) error = %v", err)
	}
	if err := repo.ensureColumn(ctx, "vector_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		t.Fatalf("ensureColumn(existing) error = %v", err)
	}

	if err := repo.ensureImportJobColumn(ctx, "retry_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		t.Fatalf("ensureImportJobColumn(add) error = %v", err)
	}
	if err := repo.ensureImportJobColumn(ctx, "retry_count", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		t.Fatalf("ensureImportJobColumn(existing) error = %v", err)
	}
}
