package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParameterSyncOutboxReadyIndexMigration(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "000031_add_parameter_sync_outbox_ready_index.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read parameter sync outbox index migration: %v", err)
	}

	sql := string(raw)
	required := []string{
		"-- +goose NO TRANSACTION",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_parameter_sync_outbox_ready_created",
		"ON public.parameter_sync_outbox (created_at, id)",
		"WHERE status IN ('pending', 'failed')",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("outbox index migration missing %q", fragment)
		}
	}
}
