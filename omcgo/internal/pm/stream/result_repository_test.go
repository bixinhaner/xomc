package stream

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestReplaceWindowResultsFirstRevisionUsesOneBulkUpsertWithoutDelete(t *testing.T) {
	tx := &recordingResultTx{captureRows: true}
	key := resultRepositoryTestKey()
	metrics := []finalizedMetric{
		resultRepositoryTestMetric("C000010070", 11),
		resultRepositoryTestMetric("C000010080", 22),
	}

	count, err := ReplaceWindowResults(
		context.Background(), tx, key, 1, metrics, resultCompleteness{
			coverage: finalizationCoverage{
				MissingSlots: 0, DataComplete: true, PeriodComplete: true,
				VersionExpectedSlots: 12, NaturalSlots: 12,
			},
			receivedSlots: 12,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("result count = %d, want 2", count)
	}
	if tx.copyCalls != 1 {
		t.Fatalf("CopyFrom calls = %d, want 1", tx.copyCalls)
	}
	if len(tx.copiedRows) != 2 {
		t.Fatalf("copied rows = %d, want 2", len(tx.copiedRows))
	}
	if got := countSQLContaining(tx.sqls, "INSERT INTO pm_aggregation_results "); got != 1 {
		t.Fatalf("result upsert statements = %d, want 1; SQL=%v", got, tx.sqls)
	}
	for _, query := range tx.sqls {
		if strings.Contains(strings.ToUpper(query), "DELETE FROM PM_AGGREGATION_RESULTS") {
			t.Fatalf("revision=1 must not delete result rows: %s", query)
		}
	}
}

func TestReplaceWindowResultsRevisionDeletesOnlyMetricsAbsentFromStaging(t *testing.T) {
	tx := &recordingResultTx{captureRows: true}

	_, err := ReplaceWindowResults(
		context.Background(), tx, resultRepositoryTestKey(), 2,
		[]finalizedMetric{resultRepositoryTestMetric("C000010070", 33)},
		resultCompleteness{coverage: finalizationCoverage{
			DataComplete: true, PeriodComplete: true,
			VersionExpectedSlots: 12, NaturalSlots: 12,
		}, receivedSlots: 12},
	)
	if err != nil {
		t.Fatal(err)
	}

	var deleteSQL string
	for _, query := range tx.sqls {
		if strings.Contains(strings.ToUpper(query), "DELETE FROM PM_AGGREGATION_RESULTS") {
			deleteSQL = query
		}
	}
	if deleteSQL == "" {
		t.Fatal("revision>1 must delete stale result metrics")
	}
	upper := strings.ToUpper(deleteSQL)
	for _, fragment := range []string{
		"NOT EXISTS",
		"METRIC_ID",
		"DIMENSION_KEY",
		"OBJECT_LDN",
		"TECHNOLOGY",
		"PM_AGGREGATION_RESULTS_STAGING",
	} {
		if !strings.Contains(upper, fragment) {
			t.Fatalf("stale-delete lacks %q identity predicate: %s", fragment, deleteSQL)
		}
	}
}

func TestReplaceWindowResultsUpsertRefreshesRevisionValueAndCompleteness(t *testing.T) {
	tx := &recordingResultTx{captureRows: true}
	metric := resultRepositoryTestMetric("K900010076", 91.25)

	_, err := ReplaceWindowResults(
		context.Background(), tx, resultRepositoryTestKey(), 3,
		[]finalizedMetric{metric},
		resultCompleteness{
			coverage: finalizationCoverage{
				MissingSlots: 1, DataComplete: false, PeriodComplete: false,
				VersionExpectedSlots: 12, NaturalSlots: 12,
			},
			receivedSlots: 11,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.copiedRows) != 1 {
		t.Fatalf("copied rows = %d, want 1", len(tx.copiedRows))
	}
	row := tx.copiedRows[0]
	if got := row[tx.columnIndex(t, "metric_value")]; got != 91.25 {
		t.Fatalf("staged metric_value = %#v, want 91.25", got)
	}
	if got := row[tx.columnIndex(t, "revision")]; got != 3 {
		t.Fatalf("staged revision = %#v, want 3", got)
	}
	if got := row[tx.columnIndex(t, "complete")]; got != false {
		t.Fatalf("staged complete = %#v, want false", got)
	}
	if got := row[tx.columnIndex(t, "period_complete")]; got != false {
		t.Fatalf("staged period_complete = %#v, want false", got)
	}

	upsertSQL := findSQLContaining(t, tx.sqls, "ON CONFLICT")
	for _, assignment := range []string{
		"metric_value = EXCLUDED.metric_value",
		"revision = EXCLUDED.revision",
		"complete = EXCLUDED.complete",
		"period_complete = EXCLUDED.period_complete",
	} {
		if !strings.Contains(upsertSQL, assignment) {
			t.Fatalf("upsert does not refresh %q: %s", assignment, upsertSQL)
		}
	}
}

func BenchmarkReplaceWindowResults20000Metrics(b *testing.B) {
	metrics := make([]finalizedMetric, 20_000)
	for index := range metrics {
		metrics[index] = resultRepositoryTestMetric(
			fmt.Sprintf("C%08d", index),
			float64(index),
		)
	}
	completeness := resultCompleteness{
		coverage: finalizationCoverage{
			DataComplete: true, PeriodComplete: true,
			VersionExpectedSlots: 12, NaturalSlots: 12,
		},
		receivedSlots: 12,
	}
	key := resultRepositoryTestKey()
	b.ResetTimer()
	for range b.N {
		tx := &recordingResultTx{}
		if _, err := ReplaceWindowResults(
			context.Background(), tx, key, 2, metrics, completeness,
		); err != nil {
			b.Fatal(err)
		}
		if tx.copyCalls != 1 ||
			countSQLContaining(tx.sqls, "INSERT INTO pm_aggregation_results ") != 1 {
			b.Fatalf("database calls scale with result count: copy=%d SQL=%v", tx.copyCalls, tx.sqls)
		}
	}
}

func resultRepositoryTestKey() WindowKey {
	start := time.Date(2026, 7, 30, 14, 0, 0, 0, time.UTC)
	return WindowKey{
		TaskID:        uuid.MustParse("10000000-0000-4000-8000-000000000001"),
		TaskVersionID: uuid.MustParse("20000000-0000-4000-8000-000000000001"),
		EntityKey:     "Network",
		Granularity:   GranularityHourly,
		Start:         start,
		End:           start.Add(time.Hour),
	}
}

func resultRepositoryTestMetric(metricID string, value float64) finalizedMetric {
	return finalizedMetric{
		Definition: ContributionValue{
			Dimension:     DimensionNetwork,
			DimensionKey:  "Network",
			DimensionName: "Network",
			Technology:    "LTE",
			MetricPath:    metricID,
		},
		MetricID: metricID, MetricType: "counter",
		Operation: AggregationSum, Value: value,
		SampleCount: 12, FormulaComplete: true,
	}
}

type recordingResultTx struct {
	sqls        []string
	copyCalls   int
	copyColumns []string
	copiedRows  [][]any
	copiedCount int64
	captureRows bool
}

func (tx *recordingResultTx) Exec(
	_ context.Context,
	sql string,
	_ ...any,
) (pgconn.CommandTag, error) {
	tx.sqls = append(tx.sqls, sql)
	if strings.Contains(sql, "INSERT INTO pm_aggregation_results ") {
		return pgconn.NewCommandTag(fmt.Sprintf("INSERT 0 %d", tx.copiedCount)), nil
	}
	return pgconn.NewCommandTag("UPDATE 0"), nil
}

func (tx *recordingResultTx) CopyFrom(
	_ context.Context,
	_ pgx.Identifier,
	columns []string,
	source pgx.CopyFromSource,
) (int64, error) {
	tx.copyCalls++
	tx.copyColumns = append([]string(nil), columns...)
	for source.Next() {
		values, err := source.Values()
		if err != nil {
			return 0, err
		}
		tx.copiedCount++
		if tx.captureRows {
			tx.copiedRows = append(tx.copiedRows, append([]any(nil), values...))
		}
	}
	if err := source.Err(); err != nil {
		return 0, err
	}
	return tx.copiedCount, nil
}

func (tx *recordingResultTx) columnIndex(t *testing.T, name string) int {
	t.Helper()
	for index, column := range tx.copyColumns {
		if column == name {
			return index
		}
	}
	t.Fatalf("CopyFrom columns lack %q: %v", name, tx.copyColumns)
	return -1
}

func countSQLContaining(queries []string, fragment string) int {
	count := 0
	for _, query := range queries {
		if strings.Contains(query, fragment) {
			count++
		}
	}
	return count
}

func findSQLContaining(t *testing.T, queries []string, fragment string) string {
	t.Helper()
	for _, query := range queries {
		if strings.Contains(query, fragment) {
			return query
		}
	}
	t.Fatalf("SQL does not contain %q: %v", fragment, queries)
	return ""
}
