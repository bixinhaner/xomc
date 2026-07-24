package compare

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestExplainSQLGuardsBeforeAnalyzeOrDelete(t *testing.T) {
	t.Parallel()

	sql := readExplainSQL(t)
	firstDanger := firstPositiveIndex(
		strings.Index(sql, "\nEXPLAIN (ANALYZE"),
		strings.Index(sql, "DELETE FROM"),
	)
	if firstDanger < 0 {
		t.Fatal("SQL has no EXPLAIN ANALYZE or DELETE to guard")
	}
	guard := sql[:firstDanger]

	for _, required := range []string{
		"task8_isolated_safe_environment",
		"task8_expected_database",
		"current_database()",
		"task8_validation.pm_explain_clone_sentinel",
		"SET search_path = pg_catalog, public;",
	} {
		if !strings.Contains(guard, required) {
			t.Errorf("guard before first dangerous statement does not contain %q", required)
		}
	}
	if count := strings.Count(guard, "SELECT 1 / 0 AS task8_guard_refusal;"); count < 5 {
		t.Errorf("guard has %d ON_ERROR_STOP refusal errors, want at least 5", count)
	}
	for _, message := range []string{
		"isolation flag must be on",
		"expected database is required",
		"database identity mismatch",
		"clone sentinel missing",
		"clone sentinel does not match current database",
	} {
		if !strings.Contains(guard, message) {
			t.Errorf("guard does not expose refusal reason %q", message)
		}
	}
}

func TestExplainSQLUsesQualifiedCoreRelations(t *testing.T) {
	t.Parallel()

	sql := readExplainSQL(t)
	unqualified := regexp.MustCompile(`(?m)\b(?:FROM|JOIN|USING|DELETE FROM)\s+(pm_[a-z][a-z0-9_]*)\b`)
	if matches := unqualified.FindAllString(sql, -1); len(matches) > 0 {
		t.Fatalf("unqualified PM relations: %v", matches)
	}
	for _, relation := range []string{
		"public.pm_ingest_batches",
		"public.pm_measurement_anchors",
		"public.pm_metric_values",
		"public.pm_hourly_bucket_versions",
		"public.pm_hourly_anchors",
		"public.pm_hourly_values",
	} {
		if !strings.Contains(sql, relation) {
			t.Errorf("SQL does not reference %s", relation)
		}
	}
}

func TestExplainSQLRequiresCorrelatedRepresentativeRows(t *testing.T) {
	t.Parallel()

	sql := readExplainSQL(t)
	for _, required := range []string{
		"representative_raw_sample_available",
		"a.ingest_batch_id = b.ingest_batch_id",
		`v."time" = a."time" AND v.anchor_id = a.anchor_id`,
		"representative cleanup source with values is required",
		"representative_cleanup_sample_available",
		"sample_cleanup_bucket_version",
		"representative active hourly rows are required",
		"representative_active_hourly_sample_available",
		"QUERY active hourly latest metric",
		"QUERY active hourly dashboard range",
		`ver.status = 'active'`,
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("SQL does not contain representative-data requirement %q", required)
		}
	}
	if count := strings.Count(sql, "SELECT 1 / 0 AS task8_guard_refusal;"); count < 8 {
		t.Errorf("SQL has %d refusal errors, want precondition failures to exit nonzero", count)
	}
}

func readExplainSQL(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "scripts", "pm_explain_core_queries.sql")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read EXPLAIN SQL: %v", err)
	}
	return string(contents)
}

func firstPositiveIndex(values ...int) int {
	first := -1
	for _, value := range values {
		if value >= 0 && (first < 0 || value < first) {
			first = value
		}
	}
	return first
}
