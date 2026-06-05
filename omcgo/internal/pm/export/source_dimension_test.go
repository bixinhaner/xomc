package export

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── 轻量 pgx.Rows / PgQuerier stub（adhocSource 按维度映射行的取证用）──────────

// adhocFakeRows 把固定的 [][]any 行喂给 Scan（按指针逐列赋值），模拟 adhocSelectCols 15 列结果。
type adhocFakeRows struct {
	rows [][]any
	i    int
}

func (f *adhocFakeRows) Next() bool { f.i++; return f.i <= len(f.rows) }
func (f *adhocFakeRows) Scan(dest ...any) error {
	row := f.rows[f.i-1]
	for j, d := range dest {
		switch p := d.(type) {
		case *uuid.UUID:
			*p = row[j].(uuid.UUID)
		case *string:
			*p = row[j].(string)
		case **string:
			if row[j] == nil {
				*p = nil
			} else {
				v := row[j].(string)
				*p = &v
			}
		case *float64:
			*p = row[j].(float64)
		case *time.Time:
			*p = row[j].(time.Time)
		}
	}
	return nil
}
func (f *adhocFakeRows) Err() error                        { return nil }
func (f *adhocFakeRows) Close()                            {}
func (f *adhocFakeRows) CommandTag() pgconn.CommandTag     { return pgconn.CommandTag{} }
func (f *adhocFakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *adhocFakeRows) Values() ([]any, error)            { return nil, nil }
func (f *adhocFakeRows) RawValues() [][]byte               { return nil }
func (f *adhocFakeRows) Conn() *pgx.Conn                   { return nil }

// adhocStubQuerier 把一批固定行作为 Query 结果返回（Scan 端按 adhocSelectCols 列序解析）。
type adhocStubQuerier struct {
	rows [][]any
}

func (s *adhocStubQuerier) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (s *adhocStubQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &adhocFakeRows{rows: s.rows}, nil
}
func (s *adhocStubQuerier) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

// strptr 是测试用 *string 字面量助手（nil 表示 SQL NULL）。
func strptr(s string) *string { return &s }

// adhocRow 按 adhocSelectCols 列序构造一行（15 列）：
// id, oui, sn, metric_path, metric_type, metric_value, statis_type, granularity,
// time, start_time, end_time, object_ldn, product_id, product_name, device_group_name。
func adhocRow(oui, sn, metricPath string, value float64, ldn, productID, productName, groupName *string) []any {
	tm := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	derefOr := func(p *string) any {
		if p == nil {
			return nil
		}
		return *p
	}
	return []any{
		uuid.New(), oui, sn, metricPath, "counter", value, "sum", "15min",
		tm, tm, tm, derefOr(ldn), derefOr(productID), derefOr(productName), derefOr(groupName),
	}
}

// device_group 维度：首列对象名取设备组名，小区列不填（聚合维度）。
func TestAdhocSource_DeviceGroupLabel(t *testing.T) {
	stub := &adhocStubQuerier{rows: [][]any{
		adhocRow("", "", "C1", 1, strptr("DeviceGroup=abcdef1234-0000"), nil, nil, strptr("华东A组")),
	}}
	src := newAdhocSource(stub, uuid.New(), time.Time{}, time.Time{}, "device_group", 0)
	rows, _, err := src.Next(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "华东A组", rows[0].Device)
	assert.Equal(t, "", rows[0].CellPLMN, "聚合维度不填小区")
}

// device_group 维度组名缺失：回退 uuid 前 8（剥 DeviceGroup= 前缀）。
func TestAdhocSource_DeviceGroupFallback(t *testing.T) {
	stub := &adhocStubQuerier{rows: [][]any{
		adhocRow("", "", "C1", 1, strptr("DeviceGroup=abcdef1234-0000"), nil, nil, nil),
	}}
	src := newAdhocSource(stub, uuid.New(), time.Time{}, time.Time{}, "device_group", 0)
	rows, _, err := src.Next(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "abcdef12", rows[0].Device)
}

// product 维度：首列取产品名。
func TestAdhocSource_ProductLabel(t *testing.T) {
	stub := &adhocStubQuerier{rows: [][]any{
		adhocRow("", "", "C1", 1, nil, strptr("11112222-3333-4444"), strptr("BLX 产品"), nil),
	}}
	src := newAdhocSource(stub, uuid.New(), time.Time{}, time.Time{}, "product", 0)
	rows, _, err := src.Next(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "BLX 产品", rows[0].Device)
	assert.Equal(t, "", rows[0].CellPLMN)
}

// device 维度：首列只用 SN（与页面一致），小区列填 object_ldn 原文（输出时拆 Cell ID/PLMN）。
func TestAdhocSource_DeviceLabelKeepsCell(t *testing.T) {
	stub := &adhocStubQuerier{rows: [][]any{
		adhocRow("OUI1", "SN1", "C1", 1, strptr("Cellid=111"), nil, nil, nil),
	}}
	src := newAdhocSource(stub, uuid.New(), time.Time{}, time.Time{}, "device", 0)
	rows, _, err := src.Next(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "SN1", rows[0].Device)
	assert.Equal(t, "Cellid=111", rows[0].CellPLMN, "device 维度保留真实小区")
}

// aggregate_group 维度：首列「聚合组(N个设备)」，小区列空。
func TestAdhocSource_AggregateGroupLabel(t *testing.T) {
	stub := &adhocStubQuerier{rows: [][]any{
		adhocRow("", "AGGREGATED", "C1", 1, nil, nil, nil, nil),
	}}
	src := newAdhocSource(stub, uuid.New(), time.Time{}, time.Time{}, "aggregate_group", 2)
	rows, _, err := src.Next(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "聚合组(2个设备)", rows[0].Device)
	assert.Equal(t, "", rows[0].CellPLMN)
}
