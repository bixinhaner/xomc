package mml

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-0176-PR-E：MarkUnsupportedByStandardPath 写入真值源迁到 param_mappings；
// 本文件用 PG 集成测试守门：
//   - 入参 paramModelID 必须生效（不跨 paramModel 污染）
//   - is_supported=true 守护防重复触动 updated_at
//   - 0 行受影响是合法情况（path 不在 mappings 或已 false）
//   - 空字符串 path 短路返 0
//
// 测试模式：连不上 PG（TEST_PG_URL 或本地 docker-compose）→ t.Skip。
// 数据隔离：每个测试用独立的 param_models / param_mappings 行，前缀
// "TEST-PR-E-" 让 cleanup 一发命中。

const testParamModelNamePrefix = "TEST-PR-E-"

// pgPoolForAdminRepoTest 返回测试 pool；连不上则 Skip。
func pgPoolForAdminRepoTest(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		dsn = "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no PG available: %v", err)
		return nil
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("no PG available (ping fail): %v", err)
		return nil
	}
	t.Cleanup(func() { pool.Close() })

	// 校验 param_models / param_mappings 表存在（避免未跑迁移时跑测试）
	var ok bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (
		    SELECT 1 FROM information_schema.tables WHERE table_name='param_mappings'
		 ) AND EXISTS (
		    SELECT 1 FROM information_schema.tables WHERE table_name='param_models'
		 )`,
	).Scan(&ok); err != nil || !ok {
		t.Skipf("param_models / param_mappings tables not present: err=%v ok=%v", err, ok)
		return nil
	}
	return pool
}

// insertParamModel 插入测试用 param_model，name 带 TEST-PR-E- 前缀便于清理。
func insertParamModel(t *testing.T, pool *pgxpool.Pool, suffix string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO param_models (id, name, is_active) VALUES ($1, $2, true)`,
		id, testParamModelNamePrefix+suffix,
	)
	require.NoError(t, err, "insert test param_model")
	return id
}

// insertParamMapping 插入一条 mapping；is_supported 默认 true。
func insertParamMapping(
	t *testing.T, pool *pgxpool.Pool,
	paramModelID uuid.UUID, standardPath, privatePath string, isSupported bool,
) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO param_mappings (
		    id, param_model_id, standard_path, private_path,
		    entry_type, is_storable, is_active, is_supported
		 ) VALUES ($1, $2, $3, $4, 'parameter', true, true, $5)`,
		id, paramModelID, standardPath, privatePath, isSupported,
	)
	require.NoError(t, err, "insert test param_mapping")
	return id
}

// fetchIsSupported / fetchUpdatedAt 取行的字段，用于断言。
func fetchIsSupported(t *testing.T, pool *pgxpool.Pool, mappingID uuid.UUID) bool {
	t.Helper()
	var v bool
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT is_supported FROM param_mappings WHERE id=$1`, mappingID,
	).Scan(&v))
	return v
}

// fetchUpdatedAtNanos 用 epoch nanos 比较，避免 time.Time 精度抖动。
func fetchUpdatedAtNanos(t *testing.T, pool *pgxpool.Pool, mappingID uuid.UUID) int64 {
	t.Helper()
	var v int64
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT EXTRACT(EPOCH FROM updated_at)::bigint * 1000000000 +
		        EXTRACT(MICROSECONDS FROM updated_at)::bigint % 1000000 * 1000
		   FROM param_mappings WHERE id=$1`, mappingID,
	).Scan(&v))
	return v
}

// cleanupTestParamModels 清掉 TEST-PR-E- 前缀的 param_models（CASCADE 删 mappings）。
func cleanupTestParamModels(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, _ = pool.Exec(context.Background(),
		`DELETE FROM param_models WHERE name LIKE $1`,
		testParamModelNamePrefix+"%",
	)
}

// ----------------------------------------------------------------------------
// PR-E 守门测试
// ----------------------------------------------------------------------------

// 写入路径正确：MarkUnsupportedByStandardPath 把目标 (pm, path) 标 false，
// 其它 path 不动。
func Test_MarkUnsupportedByStandardPath_WritesParamMappings(t *testing.T) {
	pool := pgPoolForAdminRepoTest(t)
	defer cleanupTestParamModels(t, pool)

	pmID := insertParamModel(t, pool, "writes")
	pathA := "Device.DeviceInfo.UserLabel"
	pathB := "Device.DeviceInfo.SoftwareVersion"
	idA := insertParamMapping(t, pool, pmID, pathA, "X_BAI.User.Label", true)
	idB := insertParamMapping(t, pool, pmID, pathB, "X_BAI.SW.Version", true)

	repo := NewPgSubFieldRepository(pool)
	n, err := repo.MarkUnsupportedByStandardPath(context.Background(), pmID, pathA)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "应仅影响 pathA 一行")

	assert.False(t, fetchIsSupported(t, pool, idA), "pathA 被 mark unsupported")
	assert.True(t, fetchIsSupported(t, pool, idB), "pathB 不受影响")
}

// paramModelID 隔离：同 standardPath 在两个 paramModel 下都 true → 只标目标 pm 那一行
func Test_MarkUnsupportedByStandardPath_OnlyTargetParamModel(t *testing.T) {
	pool := pgPoolForAdminRepoTest(t)
	defer cleanupTestParamModels(t, pool)

	pmTarget := insertParamModel(t, pool, "target")
	pmOther := insertParamModel(t, pool, "other")
	path := "Device.X.Same"
	idTarget := insertParamMapping(t, pool, pmTarget, path, "Priv.X.Same", true)
	idOther := insertParamMapping(t, pool, pmOther, path, "Priv.X.Same", true)

	repo := NewPgSubFieldRepository(pool)
	n, err := repo.MarkUnsupportedByStandardPath(context.Background(), pmTarget, path)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	assert.False(t, fetchIsSupported(t, pool, idTarget), "target pm 行 → false")
	assert.True(t, fetchIsSupported(t, pool, idOther), "other pm 同 path → 不污染")
}

// 0 行受影响：path 不在 param_mappings → 返 0 不报错
func Test_MarkUnsupportedByStandardPath_NoMatchingRow_Returns0(t *testing.T) {
	pool := pgPoolForAdminRepoTest(t)
	defer cleanupTestParamModels(t, pool)

	pmID := insertParamModel(t, pool, "nomatch")

	repo := NewPgSubFieldRepository(pool)
	n, err := repo.MarkUnsupportedByStandardPath(
		context.Background(), pmID, "Device.NotExist.Foo",
	)
	require.NoError(t, err, "缺映射不应报错（auto-learn 经常碰到 path 没字典化）")
	assert.Equal(t, int64(0), n)
}

// 空字符串 path：短路返 0
func Test_MarkUnsupportedByStandardPath_EmptyPathReturns0(t *testing.T) {
	pool := pgPoolForAdminRepoTest(t)
	defer cleanupTestParamModels(t, pool)

	pmID := insertParamModel(t, pool, "empty")

	repo := NewPgSubFieldRepository(pool)
	n, err := repo.MarkUnsupportedByStandardPath(context.Background(), pmID, "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "空 path 必须 silent return 0")
}

// 已 is_supported=false 的行守护：MarkUnsupported 不再触动 updated_at
func Test_MarkUnsupportedByStandardPath_AlreadyFalse_NotTouched(t *testing.T) {
	pool := pgPoolForAdminRepoTest(t)
	defer cleanupTestParamModels(t, pool)

	pmID := insertParamModel(t, pool, "alreadyfalse")
	path := "Device.Already.False"
	id := insertParamMapping(t, pool, pmID, path, "Priv.Already.False", false /* is_supported=false */)

	before := fetchUpdatedAtNanos(t, pool, id)

	repo := NewPgSubFieldRepository(pool)
	n, err := repo.MarkUnsupportedByStandardPath(context.Background(), pmID, path)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "已 false 的行不重复 UPDATE，0 rows affected")

	after := fetchUpdatedAtNanos(t, pool, id)
	assert.Equal(t, before, after, "updated_at 必须保持不变（is_supported=true 守护生效）")
	assert.False(t, fetchIsSupported(t, pool, id), "is_supported 仍是 false")
}

// 辅助：确保前缀清理 SQL 命中（防回归，删错前缀也能发现）
func Test_TestParamModelNamePrefix_NotEmpty(t *testing.T) {
	require.NotEmpty(t, testParamModelNamePrefix)
	require.Equal(t, "TEST-PR-E-", testParamModelNamePrefix,
		"前缀变更必须同步更新 cleanupTestParamModels")
}
