package dictsource

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeReader 实现 SyncRowReader,记录最近一次 Query 的 SQL,返预设 rows。
type fakeReader struct {
	mu         sync.Mutex
	lastSQL    string
	rows       [][2]string // [(label, value), ...]
	rowsErr    error
	scanErr    error
	queryCount int

	countSQL string
	countVal int
	countErr error
}

func (f *fakeReader) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastSQL = sql
	f.queryCount++
	if f.rowsErr != nil {
		return nil, f.rowsErr
	}
	return &fakeRows{rows: f.rows, scanErr: f.scanErr}, nil
}

func (f *fakeReader) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.countSQL = sql
	return &fakeRow{val: f.countVal, err: f.countErr}
}

type fakeRows struct {
	rows    [][2]string
	idx     int
	scanErr error
	closed  bool
}

func (r *fakeRows) Next() bool {
	if r.idx >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	if r.idx == 0 || r.idx > len(r.rows) {
		return errors.New("scan out of range")
	}
	row := r.rows[r.idx-1]
	*(dest[0].(*string)) = row[0]
	*(dest[1].(*string)) = row[1]
	return nil
}
func (r *fakeRows) Close()                                       { r.closed = true }
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }

type fakeRow struct {
	val int
	err error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*int)) = r.val
	return nil
}

// fakeWriter 实现 SyncWriter,记录调用参数 + 可注入失败。
type fakeWriter struct {
	mu              sync.Mutex
	upsertCalls     int
	upsertRows      []PairLV
	upsertInserted  int
	upsertUpdated   int
	upsertErr       error
	deleteCalls     int
	deleteKeep      []string
	deleteCount     int
	deleteErr       error
	countCalls      int
	countVal        int
	countErr        error
	metadataCalls   int
	metadataStatus  string
	metadataErrMsg  string
	metadataCount   int
	metadataErr     error
}

func (w *fakeWriter) UpsertAuto(ctx context.Context, dictID int64, rows []PairLV) (int, int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.upsertCalls++
	w.upsertRows = rows
	if w.upsertErr != nil {
		return 0, 0, w.upsertErr
	}
	return w.upsertInserted, w.upsertUpdated, nil
}
func (w *fakeWriter) DeleteAutoNotIn(ctx context.Context, dictID int64, keepValues []string) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.deleteCalls++
	w.deleteKeep = keepValues
	if w.deleteErr != nil {
		return 0, w.deleteErr
	}
	return w.deleteCount, nil
}
func (w *fakeWriter) CountAuto(ctx context.Context, dictID int64) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.countCalls++
	if w.countErr != nil {
		return 0, w.countErr
	}
	return w.countVal, nil
}
func (w *fakeWriter) UpdateMetadata(ctx context.Context, dictID int64, status, errMsg string, count int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.metadataCalls++
	w.metadataStatus = status
	w.metadataErrMsg = errMsg
	w.metadataCount = count
	return w.metadataErr
}

func newEngineWithMocks(t *testing.T, reader *fakeReader, writer *fakeWriter) *SyncEngine {
	t.Helper()
	reg, err := LoadDefault()
	require.NoError(t, err)
	return NewSyncEngine(reader, writer, reg, nil)
}

func TestSyncOne_HappyPath_InsertsUpdatesDeletes(t *testing.T) {
	reader := &fakeReader{
		rows: [][2]string{
			{"BaiBLQ", "BaiBLQ"},
			{"BaiBNQ", "BaiBNQ"},
			{"BM", "BM"},
		},
	}
	writer := &fakeWriter{
		upsertInserted: 2, upsertUpdated: 1, deleteCount: 1, countVal: 3,
	}
	eng := newEngineWithMocks(t, reader, writer)

	res, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 42, Name: "设备型号", SourceTable: "devices",
		LabelField: "product_class", ValueField: "product_class",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, res.Inserted)
	assert.Equal(t, 1, res.Updated)
	assert.Equal(t, 1, res.Deleted)
	assert.Equal(t, 3, res.TotalAuto)
	// SQL 写入了表名 / 字段名
	assert.Contains(t, reader.lastSQL, "FROM devices")
	assert.Contains(t, reader.lastSQL, "product_class")
	// keepValues 透传给 DeleteAutoNotIn
	assert.Equal(t, []string{"BaiBLQ", "BaiBNQ", "BM"}, writer.deleteKeep)
}

func TestSyncOne_RejectsTableNotInWhitelist(t *testing.T) {
	eng := newEngineWithMocks(t, &fakeReader{}, &fakeWriter{})
	_, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 1, SourceTable: "evil_table", LabelField: "x", ValueField: "x",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSourceTable)
}

func TestSyncOne_RejectsFieldNotInWhitelist(t *testing.T) {
	eng := newEngineWithMocks(t, &fakeReader{}, &fakeWriter{})
	// table 在白名单,字段不在
	_, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 1, SourceTable: "devices", LabelField: "password_hash", ValueField: "product_class",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidLabelField)
}

func TestSyncOne_LimitExceededIsRejected(t *testing.T) {
	// 构造 MaxSourceRows 行
	rows := make([][2]string, MaxSourceRows)
	for i := range rows {
		rows[i] = [2]string{"L", "V"}
	}
	reader := &fakeReader{rows: rows}
	eng := newEngineWithMocks(t, reader, &fakeWriter{})
	_, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 1, SourceTable: "devices", LabelField: "product_class", ValueField: "product_class",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLimitExceeded)
}

func TestSyncOne_QueryErrorPropagates(t *testing.T) {
	reader := &fakeReader{rowsErr: errors.New("pg down")}
	eng := newEngineWithMocks(t, reader, &fakeWriter{})
	_, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 1, SourceTable: "devices", LabelField: "product_class", ValueField: "product_class",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pg down")
}

func TestSyncOne_WriterFailureRollsBackResult(t *testing.T) {
	reader := &fakeReader{rows: [][2]string{{"a", "a"}}}
	writer := &fakeWriter{upsertErr: errors.New("upsert blew up")}
	eng := newEngineWithMocks(t, reader, writer)
	_, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 1, SourceTable: "devices", LabelField: "product_class", ValueField: "product_class",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert auto details")
}

func TestSyncOne_MissingSourceBindingFails(t *testing.T) {
	eng := newEngineWithMocks(t, &fakeReader{}, &fakeWriter{})
	_, err := eng.SyncOne(context.Background(), SyncDict{
		ID: 1, SourceTable: "", LabelField: "x", ValueField: "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing source binding")
}

func TestPreview_HappyPath(t *testing.T) {
	reader := &fakeReader{
		rows: [][2]string{
			{"L1", "V1"}, {"L2", "V2"}, {"L3", "V3"},
		},
		countVal: 25,
	}
	eng := newEngineWithMocks(t, reader, &fakeWriter{})
	pairs, total, err := eng.Preview(context.Background(), "devices", "product_class", "product_class", 10)
	require.NoError(t, err)
	assert.Len(t, pairs, 3)
	assert.Equal(t, 25, total)
}

func TestPreview_RejectsUnknownTable(t *testing.T) {
	eng := newEngineWithMocks(t, &fakeReader{}, &fakeWriter{})
	_, _, err := eng.Preview(context.Background(), "evil", "a", "b", 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSourceTable)
}

func TestPreview_TrimsExtraRowsToLimit(t *testing.T) {
	// fetchSource 返 5 行,limit=2 应只返前 2 行
	rows := [][2]string{{"a", "1"}, {"b", "2"}, {"c", "3"}, {"d", "4"}, {"e", "5"}}
	reader := &fakeReader{rows: rows, countVal: 100}
	eng := newEngineWithMocks(t, reader, &fakeWriter{})
	pairs, total, err := eng.Preview(context.Background(), "devices", "product_class", "product_class", 2)
	require.NoError(t, err)
	assert.Len(t, pairs, 2)
	assert.Equal(t, 100, total)
}

func TestSyncAll_SkipsFailedDictsAndUpdatesMetadata(t *testing.T) {
	// 3 个字典:第 1 和第 3 ok,第 2 因表非法 fail
	dicts := []SyncDict{
		{ID: 1, Name: "A", SourceTable: "devices", LabelField: "product_class", ValueField: "product_class"},
		{ID: 2, Name: "B", SourceTable: "evil", LabelField: "x", ValueField: "x"},
		{ID: 3, Name: "C", SourceTable: "products", LabelField: "code", ValueField: "code"},
	}
	reader := &fakeReader{rows: [][2]string{{"l", "v"}}}
	writer := &fakeWriter{countVal: 1}
	eng := newEngineWithMocks(t, reader, writer)

	ok, failed := eng.SyncAll(context.Background(), func(ctx context.Context) ([]SyncDict, error) {
		return dicts, nil
	})
	assert.Equal(t, 2, ok)
	assert.Equal(t, 1, failed)
	// 3 张都应有 UpdateMetadata 调用
	assert.GreaterOrEqual(t, writer.metadataCalls, 3)
}

func TestTruncateErr_ClampsToMaxLen(t *testing.T) {
	short := "boom"
	assert.Equal(t, short, truncateErr(short))

	long := strings.Repeat("x", 1000)
	got := truncateErr(long)
	assert.Equal(t, 500, len(got))
}
