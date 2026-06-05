package export

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// errQuerier 是 Query 恒返错的 PgQuerier 桩：让 loadNames 走"查库失败 → 全部回退编号"分支，
// 从而在不接真库的前提下覆盖 resolveColumns 的回退路径。
type errQuerier struct{}

func (errQuerier) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (errQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("no db")
}
func (errQuerier) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

// resolveColumns：保持入参顺序、并入类型；缓存命中取本地化名，未命中回退编号本身。
func TestResolveColumns_OrderTypeAndFallback(t *testing.T) {
	r := newNameResolver(errQuerier{}, "")
	r.cache["K001"] = "上行吞吐" // 预置缓存命中，避免查库

	cols := r.resolveColumns(context.Background(), []colKey{
		{code: "K001", mtype: "kpi"},
		{code: "C999", mtype: "counter"}, // 缓存未命中 + 查库失败 → 回退编号
	})

	assert.Len(t, cols, 2)
	assert.Equal(t, WideColumn{Code: "K001", Type: "kpi", Name: "上行吞吐"}, cols[0])
	assert.Equal(t, WideColumn{Code: "C999", Type: "counter", Name: "C999"}, cols[1])
	// 列名 = 指标友好名（与页面表格一致）；名缺失回退编号。
	assert.Equal(t, "上行吞吐", cols[0].header())
	assert.Equal(t, "C999", cols[1].header())
}

// 空列集：返回空切片，不查库。
func TestResolveColumns_Empty(t *testing.T) {
	r := newNameResolver(errQuerier{}, "")
	cols := r.resolveColumns(context.Background(), nil)
	assert.Empty(t, cols)
}
