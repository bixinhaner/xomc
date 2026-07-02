package dashboard

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupLayoutTestPool 连接 OMCGO_TEST_DB_DSN 指向的真实 PG（dashboard_kpi_layouts 在主库）。
// 未设置 DSN 时 SKIP（不算 PASS）。这条集成测试专门覆盖真实 pgx 二进制协议下的
// timestamptz(updated_at) 扫描——上一回合用假 Row 的单测漏掉了 OID 1184 扫描缺陷。
func setupLayoutTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("OMCGO_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_TEST_DB_DSN not set, skipping kpi layout integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "connect test DB")
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test DB: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestKPILayoutRepo_RealPGRoundTrip 走真实 PG：Upsert 写一行再 GetByTech 读回，
// 验证 updated_at(timestamptz) 经 pgx 二进制协议能正确扫进 time.Time（回归上一回合
// "scan timestamptz OID 1184 into *string" 的读接口 500）。
func TestKPILayoutRepo_RealPGRoundTrip(t *testing.T) {
	pool := setupLayoutTestPool(t)
	repo := NewKPILayoutRepository(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uid := uuid.New()
	body := json.RawMessage(`{"panels":[{"title":"it-roundtrip","metrics":["K900010015"],"x":0,"y":0,"w":12,"h":8,"chartType":"line"}]}`)

	// Upsert：RETURNING 同样要扫 updated_at(timestamptz)。
	saved, err := repo.Upsert(ctx, techLTE, body, uid)
	require.NoError(t, err, "upsert must succeed (timestamptz RETURNING scan)")
	require.NotNil(t, saved)
	assert.Equal(t, techLTE, saved.Tech)
	assert.False(t, saved.UpdatedAt.IsZero(), "updated_at should be populated from now()")
	require.NotNil(t, saved.UpdatedBy)
	assert.Equal(t, uid, *saved.UpdatedBy)

	// GetByTech：核心回归——读真实 timestamptz 列不再 500。
	got, err := repo.GetByTech(ctx, techLTE)
	require.NoError(t, err, "GetByTech must scan timestamptz into time.Time without error")
	require.NotNil(t, got)
	assert.Equal(t, techLTE, got.Tech)
	assert.JSONEq(t, string(body), string(got.Layout))
	assert.False(t, got.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), got.UpdatedAt, time.Hour, "updated_at should be ~now")

	// 清理：把这行恢复成不带 updated_by 的占位，避免污染 seed 行语义。
	// （直接删会破坏 seed 三行的存在性死判；这里只复位被测制式。）
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer ccancel()
		_, _ = pool.Exec(cctx, `UPDATE public.dashboard_kpi_layouts SET updated_by = NULL WHERE tech = $1`, techLTE)
	})
}
