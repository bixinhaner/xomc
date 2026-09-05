package main

import (
	"os"
	"strings"
	"testing"
)

func TestAssistantUpgradeReconciliationIncludesMenuAndGrant(t *testing.T) {
	for _, tc := range []struct {
		path     string
		required []string
	}{
		{"../../migrations/seed/000001_init_seed.sql", []string{"aaaa0008-1000-0000-0000-000000000011", "system:active-intelligence", "INSERT INTO public.role_menus", "10000000-0000-0000-0000-000000000002", "10000000-0000-0000-0000-000000000003"}},
		{"../../migrations/000001_init_schema.sql", []string{"CREATE TABLE IF NOT EXISTS agent_assistants", "CREATE TABLE IF NOT EXISTS agent_assistant_versions", "CREATE TABLE IF NOT EXISTS agent_assistant_runs", "ADD COLUMN IF NOT EXISTS scope_digest"}},
	} {
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := extractMainReconcileSQL(string(data))
		if err != nil {
			t.Fatal(err)
		}
		for _, required := range tc.required {
			if !strings.Contains(actual, required) {
				t.Errorf("existing database upgrade skips %q", required)
			}
		}
	}
}
