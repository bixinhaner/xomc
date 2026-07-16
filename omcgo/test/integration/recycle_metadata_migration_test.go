package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecycleMetadataMigrationHandlesNullDeletedBy(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "000025_add_recycle_audit_metadata.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read recycle metadata migration: %v", err)
	}

	sql := string(raw)
	if !strings.Contains(sql, "recycle_executor = COALESCE(deleted_by, '')") {
		t.Fatal("migration must map nullable historical deleted_by to a non-null recycle_executor")
	}
}
