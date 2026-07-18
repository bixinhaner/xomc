package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBridgeParameterSyncRoutingSchema(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "000030_bridge_parameter_sync_routing_schema.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read parameter sync routing bridge migration: %v", err)
	}

	sql := string(raw)
	required := []string{
		"ADD COLUMN IF NOT EXISTS source_event_id",
		"ADD COLUMN IF NOT EXISTS origin_event_type",
		"ADD COLUMN IF NOT EXISTS model_upload_intent_id",
		"CREATE TABLE IF NOT EXISTS public.model_upload_intents",
		"CREATE TABLE IF NOT EXISTS public.parameter_sync_admission_state",
		"CREATE TABLE IF NOT EXISTS public.parameter_sync_admission_reservations",
		"CREATE TABLE IF NOT EXISTS public.parameter_sync_event_failures",
		"CREATE TABLE IF NOT EXISTS public.parameter_sync_recovery_state",
		"CREATE INDEX IF NOT EXISTS idx_parameter_sync_requests_source_event",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_parameter_sync_requests_model_upload_intent",
		"-- +goose StatementBegin",
		"-- +goose StatementEnd",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("bridge migration missing %q", fragment)
		}
	}
}
