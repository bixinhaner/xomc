//go:build integration

package device

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// issue #203：FindStaleDevicesByClass 的 NULL product_class 边界——真 PG 求值版。
//
// DB-free 版（device_repository_test.go）从 SQL/CASE 形状证明 NULL product_class
// 落 ELSE（eNB）分支；本文件在真实 PostgreSQL 上跑一遍 CASE 三值逻辑，直接验证
// COALESCE(NULL,'') LIKE 'UPS%' → false，NULL ILIKE '%cpe%' → NULL
// → 走 ELSE（基站阈值），不会被误用 UPS/CPE 阈值。
//
// 无 PG 时（OMCGO_DB_DSN 未设）Skip，不阻塞 `go test ./internal/device/...`。
//
// 运行：OMCGO_DB_DSN=postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable \
//        go test -tags integration -count=1 ./internal/device/...

func openPoolOrSkipDevice(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set; skipping device integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// TestFindStaleByClass_NullProductClass_UsesENBThreshold_RealPG 在真实 PG 上断言：
// product_class 为 NULL 的设备，CASE WHEN ups/cpe 谓词都不为 TRUE，因此选用
// ELSE（基站 eNB）阈值。用一条不依赖具体表数据的查询，直接对 PG 求值 CASE
// 表达式：给 NULL 当作 product_class 时返回 ELSE 的哨兵值。
func TestFindStaleByClass_NullProductClass_UsesENBThreshold_RealPG(t *testing.T) {
	pool := openPoolOrSkipDevice(t)
	ctx := context.Background()

	// 直接在 PG 上求值与生产同形的 CASE：NULL product_class → 走 ELSE（ENB 分支）。
	// 测试使用 100/300/600 作为哨兵值区分分支，验证的是 CASE 选择逻辑，不是默认阈值常量。
	// d 是单行子查询，product_class 列取 NULL，复刻 UPS/CPE 谓词的判定。
	const q = `
SELECT CASE
WHEN COALESCE(d.product_class, '') LIKE 'UPS%' THEN 300
WHEN (
    d.product_class ILIKE '%cpe%' OR
    d.product_class ILIKE '%home%' OR
    d.product_class ILIKE '%residential%' OR
    d.product_class ILIKE '%indoor%'
) THEN 600 ELSE 100 END AS chosen_threshold
FROM (SELECT NULL::text AS product_class) d`

	var chosen int
	require.NoError(t, pool.QueryRow(ctx, q).Scan(&chosen))
	assert.Equal(t, 100, chosen,
		"NULL product_class 应落 ELSE（ENB 分支）返回 100，而非 THEN（CPE 分支）的 600")

	// 对照：显式 CPE product_class 应命中 CPE 分支（600）。
	const qCPE = `
SELECT CASE
WHEN COALESCE(d.product_class, '') LIKE 'UPS%' THEN 300
WHEN (
    d.product_class ILIKE '%cpe%' OR
    d.product_class ILIKE '%home%' OR
    d.product_class ILIKE '%residential%' OR
    d.product_class ILIKE '%indoor%'
) THEN 600 ELSE 100 END AS chosen_threshold
FROM (SELECT 'XYZ-CPE-100'::text AS product_class) d`

	var chosenCPE int
	require.NoError(t, pool.QueryRow(ctx, qCPE).Scan(&chosenCPE))
	assert.Equal(t, 600, chosenCPE, "含 'cpe' 关键字的 product_class 应命中 CPE 阈值 600")

	const qUPS = `
SELECT CASE
WHEN COALESCE(d.product_class, '') LIKE 'UPS%' THEN 300
WHEN (
    d.product_class ILIKE '%cpe%' OR
    d.product_class ILIKE '%home%' OR
    d.product_class ILIKE '%residential%' OR
    d.product_class ILIKE '%indoor%'
) THEN 600 ELSE 100 END AS chosen_threshold
FROM (SELECT 'UPS_M3_BMU'::text AS product_class) d`

	var chosenUPS int
	require.NoError(t, pool.QueryRow(ctx, qUPS).Scan(&chosenUPS))
	assert.Equal(t, 300, chosenUPS, "UPS product_class 应优先命中 UPS 阈值 300")
}
