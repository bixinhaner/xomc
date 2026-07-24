package metrics

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveMetricDictionaryQueriesThenInsertsOnlyMissingPaths(t *testing.T) {
	tx := &recordingSparseMetadataTx{
		queryRows: [][][]any{
			{{"existing", int64(10), MetricTypeCounter, "sum", "number"}},
			{{"existing", int64(10), MetricTypeCounter, "sum", "number"}, {"new", int64(20), MetricTypeKPI, "pct", "%"}},
		},
	}

	ids, err := resolveMetricDictionary(context.Background(), tx, []metricDefinition{
		{path: "new", metricType: MetricTypeKPI, statisType: "pct", unit: "%"},
		{path: "existing", metricType: MetricTypeCounter, statisType: "sum", unit: "number"},
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"existing": 10, "new": 20}, ids)
	require.Len(t, tx.queries, 2)
	assert.Contains(t, tx.queries[0], "SELECT metric_path, metric_id")
	assert.Contains(t, tx.queries[0], "WHERE metric_path = ANY($1")
	require.Len(t, tx.execs, 1)
	assert.Contains(t, tx.execs[0], "ON CONFLICT (metric_path) DO NOTHING")
	assert.NotContains(t, strings.ToUpper(tx.execs[0]), "DO UPDATE")
	assert.Equal(t, []string{"new"}, tx.execArgs[0][0])
}

func TestResolveMetricDictionaryRejectsDuplicateMissingAndInconsistentMetadata(t *testing.T) {
	t.Run("duplicate definitions", func(t *testing.T) {
		_, err := resolveMetricDictionary(context.Background(), &recordingSparseMetadataTx{}, []metricDefinition{
			{path: "same", metricType: MetricTypeCounter},
			{path: "same", metricType: MetricTypeKPI},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate metric definition")
	})

	t.Run("missing after insert", func(t *testing.T) {
		tx := &recordingSparseMetadataTx{queryRows: [][][]any{{}, {}}}
		_, err := resolveMetricDictionary(context.Background(), tx, []metricDefinition{{path: "missing", metricType: MetricTypeCounter}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "was not resolved")
	})

	t.Run("existing immutable identity differs", func(t *testing.T) {
		tx := &recordingSparseMetadataTx{queryRows: [][][]any{{{"path", int64(10), MetricTypeKPI, "pct", "%"}}}}
		_, err := resolveMetricDictionary(context.Background(), tx, []metricDefinition{{path: "path", metricType: MetricTypeCounter}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incompatible immutable metadata")
	})
}

func TestResolveMetricDictionaryTreatsStatisTypeAndUnitAsImmutableMetadata(t *testing.T) {
	t.Run("queries every immutable metadata field", func(t *testing.T) {
		tx := &recordingSparseMetadataTx{queryRows: [][][]any{
			{{"path", int64(10), MetricTypeCounter, "sum", "number"}},
			{{"path", int64(10), MetricTypeCounter, "sum", "number"}},
		}}
		_, err := resolveMetricDictionary(context.Background(), tx, []metricDefinition{{
			path: "path", metricType: MetricTypeCounter, statisType: "sum", unit: "number",
		}})
		require.NoError(t, err)
		assert.Contains(t, tx.queries[0], "statis_type")
		assert.Contains(t, tx.queries[0], "unit")
	})

	t.Run("rejects incompatible stored fields", func(t *testing.T) {
		err := validateMetricDefinition(
			metricDefinition{path: "path", metricType: MetricTypeCounter, statisType: "sum", unit: "number"},
			resolvedMetricDefinition{metricDefinition: metricDefinition{path: "path", metricType: MetricTypeCounter, statisType: "pct", unit: "%"}, id: 10},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incompatible immutable metadata")
	})
}

func TestResolveMetricSetQueriesThenInsertsWithoutUpdatingMetricIDs(t *testing.T) {
	tx := &recordingSparseMetadataTx{queryRows: [][][]any{{}, {{int64(77), []int64{2, 5, 9}}}}}
	hash := MetricSetHash([]int64{9, 2, 5})

	setID, err := resolveMetricSet(context.Background(), tx, "product", "group", hex.EncodeToString(hash[:]), []int64{9, 2, 5})

	require.NoError(t, err)
	assert.Equal(t, int64(77), setID)
	require.Len(t, tx.queries, 2)
	assert.Contains(t, tx.queries[0], "WHERE product_key = $1")
	assert.Contains(t, tx.queries[0], "counter_group = $2")
	assert.Contains(t, tx.queries[0], "content_hash = decode($3, 'hex')")
	require.Len(t, tx.execs, 1)
	assert.Contains(t, tx.execs[0], "ON CONFLICT (product_key, counter_group, content_hash) DO NOTHING")
	assert.NotContains(t, strings.ToUpper(tx.execs[0]), "DO UPDATE")
	assert.Equal(t, []int64{2, 5, 9}, tx.execArgs[0][3])
}

func TestResolveMetricSetRejectsExistingMismatchedMetricIDs(t *testing.T) {
	tx := &recordingSparseMetadataTx{queryRows: [][][]any{{{int64(77), []int64{2, 5, 10}}}}}
	hash := MetricSetHash([]int64{2, 5, 9})

	_, err := resolveMetricSet(context.Background(), tx, "product", "group", hex.EncodeToString(hash[:]), []int64{2, 5, 9})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "inconsistent metric set")
	assert.Empty(t, tx.execs)
}

func TestResolveMetricSetRejectsNonCanonicalStoredMetricIDOrder(t *testing.T) {
	err := validateMetricSet([]int64{2, 5, 9}, resolvedMetricSet{id: 77, metricIDs: []int64{9, 2, 5}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "inconsistent metric set")
}

type recordingSparseMetadataTx struct {
	queries   []string
	queryArgs [][]any
	execs     []string
	execArgs  [][]any
	queryRows [][][]any
}

func (tx *recordingSparseMetadataTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	tx.queries = append(tx.queries, sql)
	tx.queryArgs = append(tx.queryArgs, args)
	if len(tx.queryRows) == 0 {
		return nil, errors.New("unexpected query")
	}
	rows := &sparseMetadataRows{rows: tx.queryRows[0]}
	tx.queryRows = tx.queryRows[1:]
	return rows, nil
}

func (tx *recordingSparseMetadataTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return sparseMetadataRow{err: err}
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return sparseMetadataRow{err: err}
		}
		return sparseMetadataRow{err: pgx.ErrNoRows}
	}
	return sparseMetadataRow{values: rows.(*sparseMetadataRows).current()}
}

func (tx *recordingSparseMetadataTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.execs = append(tx.execs, sql)
	tx.execArgs = append(tx.execArgs, args)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

type sparseMetadataRows struct {
	rows [][]any
	pos  int
	err  error
}

func (r *sparseMetadataRows) Close()                                       {}
func (r *sparseMetadataRows) Err() error                                   { return r.err }
func (r *sparseMetadataRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *sparseMetadataRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *sparseMetadataRows) RawValues() [][]byte                          { return nil }
func (r *sparseMetadataRows) Conn() *pgx.Conn                              { return nil }
func (r *sparseMetadataRows) Next() bool {
	if r.pos >= len(r.rows) {
		return false
	}
	r.pos++
	return true
}
func (r *sparseMetadataRows) Scan(dest ...any) error {
	if r.pos == 0 || r.pos > len(r.rows) {
		return errors.New("scan called without a row")
	}
	return scanSparseMetadataValues(r.rows[r.pos-1], dest...)
}
func (r *sparseMetadataRows) Values() ([]any, error) {
	if r.pos == 0 || r.pos > len(r.rows) {
		return nil, errors.New("values called without a row")
	}
	return r.rows[r.pos-1], nil
}
func (r *sparseMetadataRows) current() []any { return r.rows[r.pos-1] }

type sparseMetadataRow struct {
	values []any
	err    error
}

func (r sparseMetadataRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return scanSparseMetadataValues(r.values, dest...)
}

func scanSparseMetadataValues(values []any, dest ...any) error {
	if len(values) != len(dest) {
		return fmt.Errorf("scan values: got %d columns for %d destinations", len(values), len(dest))
	}
	for i, value := range values {
		out := reflect.ValueOf(dest[i])
		if out.Kind() != reflect.Ptr || out.IsNil() {
			return fmt.Errorf("scan destination %d is not a pointer", i)
		}
		out.Elem().Set(reflect.ValueOf(value))
	}
	return nil
}
