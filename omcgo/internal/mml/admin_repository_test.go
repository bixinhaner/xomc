package mml

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// admin_repository_test.go — T-0176-PR-C ListByCommand / ListEnrichedByCommand 测试
//
// 覆盖：
//   - SQL 文本守门：ListByCommand 不再 WHERE csf.is_supported=true；
//     ListEnrichedByCommand 不再 AND csf.is_supported=true，且 paramModel
//     EXISTS 子查询追加 pm.is_supported=true
//   - 集成测试（需 TEST_PG_URL）：
//     · ListByCommand 同时返 is_supported=true 与 is_supported=false 的行
//     · ListEnrichedByCommand 只返 param_mappings.is_active+is_supported 的 path
// ============================================================

// Test_AdminRepository_SQLNoLongerFiltersBySubFieldIsSupported 源码守门：
// ListByCommand / ListEnrichedByCommand 删除了 csf.is_supported=true 过滤；
// ListEnrichedByCommand 的 paramModel EXISTS 子查询追加 pm.is_supported=true。
//
// 注：MarkUnsupportedByStandardPath（写路径）保留 csf.is_supported=true，
// 防止已 mark 行重复触发 updated_at；PR-E 把 auto-learn 迁到 param_mappings
// 后整个方法删除。这里用细粒度模式守门读路径。
func Test_AdminRepository_SQLNoLongerFiltersBySubFieldIsSupported(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "admin_repository.go"))
	require.NoError(t, err)
	src := string(body)

	// 1) ListByCommand 不再走 Squirrel Eq is_supported=true
	assert.NotContains(t, src, `sq.Eq{"is_supported": true}`,
		"PR-C: ListByCommand Squirrel WHERE must not include is_supported filter")

	// 2) ListEnrichedByCommand 的 WHERE 子句不再带 csf.is_supported=true
	//    （Match 的是 "WHERE csf.command_id = $1\n  AND csf.is_supported = true"
	//    形态；写路径 MarkUnsupportedByStandardPath 的 csf.is_supported=true 在
	//    "WHERE csf.standard_path_id ..." 之后出现，与读路径形态不同。）
	assert.NotContains(t, src, "WHERE csf.command_id = $1\n  AND csf.is_supported = true",
		"PR-C: ListEnrichedByCommand WHERE must not append csf.is_supported = true")

	// 3) param_mappings EXISTS 必须含 is_supported=true（PR-C 核心切换）
	assert.Contains(t, src, "pm.is_supported   = true",
		"PR-C: ListEnrichedByCommand EXISTS subquery must include pm.is_supported = true")
}

func TestBuildEnumOptions_UsesValuesWhenLabelsMissing(t *testing.T) {
	values := "PSK，SIM"
	got := buildEnumOptions(&values, nil)
	assert.Equal(t, []MMLParamEnumOption{
		{Value: "PSK", Label: "PSK"},
		{Value: "SIM", Label: "SIM"},
	}, got)
}

func Test_AdminRepository_SQLIncludesModelValueRules(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "admin_repository.go"))
	require.NoError(t, err)
	src := string(body)

	assert.Contains(t, src, "model_pm.default_value")
	assert.Contains(t, src, "model_pm.validation_pattern")
	assert.Contains(t, src, "COALESCE(model_pm.min_value, sp.min_value)")
	assert.Contains(t, src, "COALESCE(model_pm.max_value, sp.max_value)")
	assert.Contains(t, src, "COALESCE(model_pm.data_type, sp.data_type, 'string')")
	assert.Contains(t, src, "COALESCE(model_pm.access, sp.access, 'READ_ONLY')")
	assert.Contains(t, src, "model_pm.enum_values")
	assert.Contains(t, src, "model_pm.enum_labels")
	assert.Contains(t, src, "ORDER BY (pm.source = 'custom') DESC")
}

// Test_ListByCommand_NoLongerFiltersBySubFieldIsSupported 集成测试：
//
// 同一 command_id 下既有 is_supported=true 又有 is_supported=false 的 sub_field 行，
// 验证 ListByCommand 全部返回（不再被 is_supported=false 过滤掉）。
func Test_ListByCommand_NoLongerFiltersBySubFieldIsSupported(t *testing.T) {
	pool := newMMLTestPool(t)
	if pool == nil {
		return
	}
	ctx := context.Background()
	fx := newMMLFixture(t, pool)
	defer fx.cleanup()

	commandID := fx.insertCommand("LST_PRC_TEST_LISTBY", "chapter:PRC")
	fx.insertSubField(commandID, "Device.LIST.A", true, 1)
	fx.insertSubField(commandID, "Device.LIST.B", false, 2) // 老路径下会被过滤掉
	fx.insertSubField(commandID, "Device.LIST.C", true, 3)
	fx.insertSubField(commandID, "Device.LIST.D", false, 4) // 老路径下会被过滤掉

	repo := NewPgSubFieldRepository(pool)
	rows, err := repo.ListByCommand(ctx, commandID)
	require.NoError(t, err)

	require.Len(t, rows, 4, "PR-C: ListByCommand must return all sub_fields regardless of is_supported")

	// 校验排序仍按 sort_order ASC（默认行为不变）。
	for i := 0; i < len(rows)-1; i++ {
		assert.LessOrEqual(t, rows[i].SortOrder, rows[i+1].SortOrder,
			"ListByCommand must preserve sort_order ASC ordering")
	}
}

// Test_ListEnrichedByCommand_ParamMappingsIsSupportedFilter 集成测试：
//
// 2026-05-27 行为变更(撤销 T-0183，回到 T-0170):
//   - PR-C.A：active + supported=true → 返回,is_supported=true
//   - PR-C.B：active + supported=false → 不返回(物理过滤)
//   - PR-C.C：不在 param_mappings → 不返回(EXISTS 卡 "至少有一条 active+supported mapping")
//
// sub_field 自身 is_supported 取值不影响返回(csf.is_supported 列已不再读)。
func Test_ListEnrichedByCommand_ParamMappingsIsSupportedFilter(t *testing.T) {
	pool := newMMLTestPool(t)
	if pool == nil {
		return
	}
	ctx := context.Background()
	fx := newMMLFixture(t, pool)
	defer fx.cleanup()

	paramModelID := fx.insertParamModel()
	commandID := fx.insertCommand("LST_PRC_TEST_ENRICH", "chapter:PRC")

	fx.insertSubField(commandID, "Device.PRC-C.A", true, 1)
	fx.insertSubField(commandID, "Device.PRC-C.B", false, 2)
	fx.insertSubField(commandID, "Device.PRC-C.C", true, 3)

	fx.insertParamMapping(paramModelID, "Device.PRC-C.A", true, true)  // active + supported
	fx.insertParamMapping(paramModelID, "Device.PRC-C.B", true, false) // active + unsupported → 物理过滤
	// PRC-C.C 不建 mapping → EXISTS 假 → 不命中

	repo := NewPgSubFieldRepository(pool)
	rows, err := repo.ListEnrichedByCommand(ctx, commandID, &paramModelID)
	require.NoError(t, err)

	gotPaths := make(map[string]bool, len(rows))
	for _, r := range rows {
		gotPaths[r.Tr069Path] = r.IsSupported
	}
	assert.Equal(t, 1, len(gotPaths),
		"撤销 T-0183: 仅返 A(B 因 is_supported=false 物理过滤; C 无 mapping)")
	assert.True(t, gotPaths["Device.PRC-C.A"], "PRC-C.A is_supported should be true")
	_, gotB := gotPaths["Device.PRC-C.B"]
	assert.False(t, gotB, "PRC-C.B should be physically filtered (is_supported=false)")
	_, gotC := gotPaths["Device.PRC-C.C"]
	assert.False(t, gotC, "PRC-C.C still filtered (no active mapping at all)")
}

// Test_ListEnrichedByCommand_NilParamModelReturnsAllSubFields 集成测试（健壮性）：
//
// paramModelID=nil 时不叠加 param_mappings 过滤，应返完整 sub_field 集合
// （csf.is_supported 不再过滤亦覆盖此路径）。
func Test_ListEnrichedByCommand_NilParamModelReturnsAllSubFields(t *testing.T) {
	pool := newMMLTestPool(t)
	if pool == nil {
		return
	}
	ctx := context.Background()
	fx := newMMLFixture(t, pool)
	defer fx.cleanup()

	commandID := fx.insertCommand("LST_PRC_TEST_NILPM", "chapter:PRC")
	fx.insertSubField(commandID, "Device.NILPM.A", true, 1)
	fx.insertSubField(commandID, "Device.NILPM.B", false, 2)

	repo := NewPgSubFieldRepository(pool)
	rows, err := repo.ListEnrichedByCommand(ctx, commandID, nil)
	require.NoError(t, err)

	gotPaths := make([]string, 0, len(rows))
	for _, r := range rows {
		gotPaths = append(gotPaths, r.Tr069Path)
	}
	assert.ElementsMatch(t, []string{"Device.NILPM.A", "Device.NILPM.B"}, gotPaths,
		"paramModelID=nil should return all sub_fields regardless of csf.is_supported (PR-C)")
}

func Test_SubFieldReadPathsExcludeDeprecatedRows(t *testing.T) {
	pool := newMMLTestPool(t)
	if pool == nil {
		return
	}
	ctx := context.Background()
	fx := newMMLFixture(t, pool)
	defer fx.cleanup()

	commandID := fx.insertCommand("LST_DEPRECATED_SUBFIELDS", "chapter:DEPRECATED")
	fx.insertSubField(commandID, "Device.Deprecated.Active", true, 1)
	deprecatedID := fx.insertSubField(commandID, "Device.Deprecated.Hidden", true, 2)
	_, err := pool.Exec(ctx,
		`UPDATE mml_command_sub_fields SET deprecated_at = NOW() WHERE id = $1`,
		deprecatedID,
	)
	require.NoError(t, err)

	repo := NewPgSubFieldRepository(pool)

	basicRows, err := repo.ListByCommand(ctx, commandID)
	require.NoError(t, err)
	require.Len(t, basicRows, 1)
	assert.Contains(t, basicRows[0].MMLCode, "Device.Deprecated.Active")

	enrichedRows, err := repo.ListEnrichedByCommand(ctx, commandID, nil)
	require.NoError(t, err)
	require.Len(t, enrichedRows, 1)
	assert.Equal(t, "Device.Deprecated.Active", enrichedRows[0].Tr069Path)

	adminRows, err := repo.ListAdminByCommand(ctx, commandID)
	require.NoError(t, err)
	require.Len(t, adminRows, 1)
	assert.Equal(t, "Device.Deprecated.Active", adminRows[0].Tr069Path)
}

func Test_ListEnrichedByCommand_ReturnsStandardRange(t *testing.T) {
	pool := newMMLTestPool(t)
	if pool == nil {
		return
	}
	ctx := context.Background()
	fx := newMMLFixture(t, pool)
	defer fx.cleanup()

	commandID := fx.insertCommand("LST_STANDARD_RANGE", "chapter:RNG")
	standardPath := "Device.Range." + uuid.NewString()
	fx.insertSubField(commandID, standardPath, true, 1)
	_, err := pool.Exec(ctx, `
UPDATE standard_params
   SET data_type = 'unsignedInt', min_value = 2, max_value = 32
 WHERE standard_path = $1`, standardPath)
	require.NoError(t, err)

	repo := NewPgSubFieldRepository(pool)
	rows, err := repo.ListEnrichedByCommand(ctx, commandID, nil)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "unsignedInt", rows[0].ValueType)
	require.NotNil(t, rows[0].MinValue)
	require.NotNil(t, rows[0].MaxValue)
	assert.EqualValues(t, 2, *rows[0].MinValue)
	assert.EqualValues(t, 32, *rows[0].MaxValue)
}

func Test_ListEnrichedByCommand_ModelTypeOverridesStandardType(t *testing.T) {
	pool := newMMLTestPool(t)
	if pool == nil {
		return
	}
	ctx := context.Background()
	fx := newMMLFixture(t, pool)
	defer fx.cleanup()

	commandID := fx.insertCommand("LST_MODEL_TYPE", "chapter:TYPE")
	standardPath := "Device.ModelType." + uuid.NewString()
	fx.insertSubField(commandID, standardPath, true, 1)
	paramModelID := fx.insertParamModel()
	mappingID := fx.insertParamMapping(paramModelID, standardPath, true, true)
	_, err := pool.Exec(ctx,
		`UPDATE standard_params SET data_type = 'STRING' WHERE standard_path = $1`, standardPath)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE param_mappings
   SET data_type = 'U_INT', min_value = 22, max_value = 32
 WHERE id = $1`, mappingID)
	require.NoError(t, err)

	repo := NewPgSubFieldRepository(pool)
	rows, err := repo.ListEnrichedByCommand(ctx, commandID, &paramModelID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "U_INT", rows[0].ValueType)
	assert.EqualValues(t, 22, *rows[0].MinValue)
	assert.EqualValues(t, 32, *rows[0].MaxValue)
}

func Test_ListEnrichedByCommand_ModelAccessOverridesStandardAccess(t *testing.T) {
	pool := newMMLTestPool(t)
	if pool == nil {
		return
	}
	ctx := context.Background()
	fx := newMMLFixture(t, pool)
	defer fx.cleanup()

	commandID := fx.insertCommand("MOD_MODEL_ACCESS", "chapter:ACCESS")
	standardPath := "DeviceGSM.Bts.{i}." + uuid.NewString()
	fx.insertSubField(commandID, standardPath, true, 1)
	paramModelID := fx.insertParamModel()
	mappingID := fx.insertParamMapping(paramModelID, standardPath, true, true)

	_, err := pool.Exec(ctx,
		`UPDATE standard_params SET access = 'READ_ONLY' WHERE standard_path = $1`, standardPath)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`UPDATE param_mappings SET access = 'READ_WRITE' WHERE id = $1`, mappingID)
	require.NoError(t, err)

	repo := NewPgSubFieldRepository(pool)
	rows, err := repo.ListEnrichedByCommand(ctx, commandID, &paramModelID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, AccessTypeReadWrite, rows[0].AccessType,
		"product mapping access must drive MOD availability")
}

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
