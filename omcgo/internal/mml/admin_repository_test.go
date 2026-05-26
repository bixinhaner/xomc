package mml

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
// 三组 standardPath：
//   - PR-C.A：在 param_mappings is_active=true AND is_supported=true → 应返回
//   - PR-C.B：is_active=true AND is_supported=false → 不应返回（PR-C 收敛真值源）
//   - PR-C.C：不在 param_mappings → 不应返回（EXISTS 必然 false）
//
// sub_field 自身不论 is_supported 取值（PR-C 删了 csf 过滤）。
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

	// 三个 sub_field，sub_field.is_supported 全部混合，证明 PR-C 不读 csf.is_supported。
	fx.insertSubField(commandID, "Device.PRC-C.A", true, 1)
	fx.insertSubField(commandID, "Device.PRC-C.B", false, 2) // 故意 csf.is_supported=false
	fx.insertSubField(commandID, "Device.PRC-C.C", true, 3)

	// param_mappings 配置：
	fx.insertParamMapping(paramModelID, "Device.PRC-C.A", true, true)  // active + supported → 命中
	fx.insertParamMapping(paramModelID, "Device.PRC-C.B", true, false) // active + unsupported → 不命中
	// PRC-C.C 不建 mapping → EXISTS 假 → 不命中

	repo := NewPgSubFieldRepository(pool)
	rows, err := repo.ListEnrichedByCommand(ctx, commandID, &paramModelID)
	require.NoError(t, err)

	gotPaths := make([]string, 0, len(rows))
	for _, r := range rows {
		gotPaths = append(gotPaths, r.Tr069Path)
	}
	assert.ElementsMatch(t, []string{"Device.PRC-C.A"}, gotPaths,
		"PR-C: ListEnrichedByCommand must only return paths where param_mappings is_active=true AND is_supported=true")
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

