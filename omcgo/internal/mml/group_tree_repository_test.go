package mml

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// group_tree_repository_test.go — T-0176-PR-C 单元 + 集成测试
//
// 覆盖：
//   - SQL 文本守门：确认子查询已删除，c.target_paths 直接被 SELECT
//   - 集成测试（需 TEST_PG_URL）：target_paths 来自 c.target_paths 列，
//     与 sub_field.is_supported 取值无关
// ============================================================

// Test_BuildTree_SQLNoSubFieldIsSupportedSubquery 源码守门：T-0176-PR-C 删了
// "SELECT jsonb_agg ... WHERE csf.is_supported = true" 子查询，命令树 SQL 不再
// 二次读 sub_field.is_supported 决定 target_paths。trigger refresh_mml_command_target_paths
// (migrations/000113) 已维护 c.target_paths 列为完整聚合，子查询冗余。
func Test_BuildTree_SQLNoSubFieldIsSupportedSubquery(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "group_tree_repository.go"))
	require.NoError(t, err)
	src := string(body)

	assert.NotContains(t, src, "csf.is_supported = true",
		"PR-C: command tree SQL must not filter mml_command_sub_fields.is_supported")
	assert.NotContains(t, src, "csf.is_supported=true",
		"PR-C: command tree SQL must not filter mml_command_sub_fields.is_supported (compact form)")
	// 守住"直接读列"形态。
	assert.Contains(t, src, "COALESCE(c.target_paths, '[]'::jsonb) AS target_paths",
		"PR-C: c.target_paths column should be SELECTed directly")
}

// Test_BuildTree_TargetPathsFromColumn_NoSubFieldFilter 集成测试：
//
// 准备一个 command 行带 c.target_paths JSON 列；sub_field 表里同时存在
// is_supported=true 与 is_supported=false 的行；验证 BuildTree 返回的
// target_paths 来自 c.target_paths 列（即"删了 sub_field 子查询后 c.target_paths
// 直接生效"）。
//
// 注意：trigger refresh_mml_command_target_paths 在 sub_field INSERT 时会
// **覆盖** c.target_paths（聚合所有 sub_field，不过滤 is_supported）。为隔离
// 测试目标（验证 SQL 不再做 sub_field 子查询），先 INSERT sub_field 行让
// trigger 回填，再独立 UPDATE c.target_paths 为期望值即可——SELECT 应直接
// 读这个 UPDATE 后的值。
func Test_BuildTree_TargetPathsFromColumn_NoSubFieldFilter(t *testing.T) {
	pool := newMMLTestPool(t)
	if pool == nil {
		return
	}
	ctx := context.Background()
	fx := newMMLFixture(t, pool)
	defer fx.cleanup()

	commandID := fx.insertCommand("LST_PRC_TEST_COL_TGT", "chapter:PRC")
	// 注入 4 个 sub_field：2 supported / 2 unsupported。trigger 触发后会把
	// c.target_paths 回填为 4 path 全集（基于 sub_field 全集，不读 is_supported）。
	fx.insertSubField(commandID, "Device.X.A", true, 1)
	fx.insertSubField(commandID, "Device.X.B", false, 2)
	fx.insertSubField(commandID, "Device.X.C", true, 3)
	fx.insertSubField(commandID, "Device.X.D", false, 4)

	// 让 c.target_paths 直接取一个显式值，验证 SELECT 走的是 column 而非子查询。
	const sentinel = `["Device.X.SENTINEL", "Device.X.ONLY"]`
	_, err := pool.Exec(ctx,
		`UPDATE mml_commands SET target_paths = $1::jsonb WHERE id = $2`,
		sentinel, commandID)
	require.NoError(t, err)

	repo := NewPgGroupTreeRepository(pool)
	nodes, err := repo.BuildTree(ctx, fx.chapterCode, "zh-CN")
	require.NoError(t, err)

	got := findCommandTargetPaths(t, nodes, commandID)
	require.NotNil(t, got, "command not found in built tree")

	var paths []string
	require.NoError(t, json.Unmarshal(got, &paths))
	assert.Equal(t, []string{"Device.X.SENTINEL", "Device.X.ONLY"}, paths,
		"target_paths should come from c.target_paths column verbatim, not from sub_field subquery")
}

// findCommandTargetPaths 递归搜索 tree，返回指定 commandID 的 rawTargetPaths。
func findCommandTargetPaths(t *testing.T, nodes []GroupTreeNode, commandID uuid.UUID) []byte {
	t.Helper()
	for _, n := range nodes {
		for i := range n.Commands {
			if n.Commands[i].ID == commandID {
				return n.Commands[i].TargetPathsRaw()
			}
		}
		if raw := findCommandTargetPaths(t, n.Children, commandID); raw != nil {
			return raw
		}
	}
	return nil
}

// ============================================================
// PG 测试夹具
// ============================================================

// newMMLTestPool 与 task/pg_repository_test.go pattern 对齐：连不上 PG 自动 t.Skip。
func newMMLTestPool(t *testing.T) *pgxpool.Pool {
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
	// 校验关键表存在
	for _, tbl := range []string{
		"mml_command_groups", "mml_commands", "mml_command_sub_fields",
		"standard_params", "param_mappings", "param_models",
	} {
		var exists bool
		if err := pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name=$1)",
			tbl).Scan(&exists); err != nil || !exists {
			t.Skipf("table %s not present: err=%v exists=%v", tbl, err, exists)
			return nil
		}
	}
	// 校验 mml_command_sub_fields.is_supported 列存在（PR-F 之前必须有）
	var hasCol bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.columns
		   WHERE table_name='mml_command_sub_fields' AND column_name='is_supported')`,
	).Scan(&hasCol); err != nil || !hasCol {
		t.Skipf("mml_command_sub_fields.is_supported column not present: err=%v exists=%v", err, hasCol)
		return nil
	}
	return pool
}

// mmlFixture 维护一个测试 chapter 下的 group + commands + standard_params + sub_fields，
// 退出时 best-effort 清理。
type mmlFixture struct {
	t           *testing.T
	pool        *pgxpool.Pool
	suffix      string
	chapterCode string
	groupID     uuid.UUID
	cmdIDs      []uuid.UUID
	stdParamIDs []uuid.UUID
	pmIDs       []uuid.UUID
}

func newMMLFixture(t *testing.T, pool *pgxpool.Pool) *mmlFixture {
	t.Helper()
	// suffix 隔离不同测试用例之间，避免 chapter_code / group_code unique 冲突。
	// chapter_code 列宽 VARCHAR(8)，所以总长度限制 8 chars；用 "T" 前缀 + 6 hex。
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:7]
	fx := &mmlFixture{
		t:           t,
		pool:        pool,
		suffix:      suffix,
		chapterCode: "T" + suffix, // 8 chars 上限
	}
	fx.insertChapterGroup()
	return fx
}

// insertChapterGroup 创建顶层 chapter 行；BuildTree 需此为 root 才会扫到该子树。
// param_version 取 DB 现有的 v2.3 anchor（FK NOT NULL），不影响测试断言。
// 若部署里 anchor 不同，testQueryVersionCode 兜底捞一条。
func (fx *mmlFixture) insertChapterGroup() {
	fx.t.Helper()
	id := uuid.New()
	fx.groupID = id
	groupCode := "chapter:" + fx.chapterCode
	versionCode := fx.firstParamVersionCode()
	_, err := fx.pool.Exec(context.Background(), `
INSERT INTO mml_command_groups
    (id, group_code, group_name_zh, group_name_en, param_version,
     display_order, path, chapter_code, name_i18n, source, catalog_protected)
VALUES
    ($1, $2, 'PR-C test', 'PR-C test', $3, 999, $4::ltree, $5, '{}'::jsonb, 'admin', false)`,
		id, groupCode, versionCode, fx.chapterCode, fx.chapterCode)
	require.NoError(fx.t, err)
}

// firstParamVersionCode 拿到第一个可用的 mml_param_versions.version_code 作 FK 兜底；
// 部署里若空则 t.Skip。
func (fx *mmlFixture) firstParamVersionCode() string {
	fx.t.Helper()
	var v string
	err := fx.pool.QueryRow(context.Background(),
		`SELECT version_code FROM mml_param_versions ORDER BY version_code LIMIT 1`).Scan(&v)
	if err != nil {
		fx.t.Skipf("no mml_param_versions row present: %v", err)
	}
	return v
}

// insertCommand 在 fixture group 下挂一条命令，初始 target_paths = '[]'。
func (fx *mmlFixture) insertCommand(logicalCode, _ string) uuid.UUID {
	fx.t.Helper()
	id := uuid.New()
	_, err := fx.pool.Exec(context.Background(), `
INSERT INTO mml_commands
    (id, command_name, command_code, category, description, rpc_method,
     operation_type, target_paths, group_id,
     command_name_i18n, require_confirm, confirm_msg_i18n,
     logical_name_i18n, source, catalog_protected)
VALUES
    ($1, $2, $3, 'test', '', 'GetParameterValues',
     'LST', '[]'::jsonb, $4,
     '{}'::jsonb, false, '{}'::jsonb,
     '{}'::jsonb, 'admin', false)`,
		id, logicalCode, logicalCode+"_CC_"+fx.suffix, fx.groupID)
	require.NoError(fx.t, err)
	fx.cmdIDs = append(fx.cmdIDs, id)
	return id
}

// insertSubField 插入一个 sub_field（standard_path 自动建 standard_params 行）。
func (fx *mmlFixture) insertSubField(commandID uuid.UUID, standardPath string, supported bool, sortOrder int) uuid.UUID {
	fx.t.Helper()
	spID := fx.getOrInsertStandardParam(standardPath)
	id := uuid.New()
	_, err := fx.pool.Exec(context.Background(), `
INSERT INTO mml_command_sub_fields
    (id, command_id, standard_path_id, mml_code, label_i18n,
     default_selected, is_required, sort_order, is_supported)
VALUES
    ($1, $2, $3, $4, '{}'::jsonb, false, false, $5, $6)`,
		id, commandID, spID,
		"MML_"+standardPath+"_"+fx.suffix,
		sortOrder, supported)
	require.NoError(fx.t, err)
	return id
}

// getOrInsertStandardParam 拿到 standard_path 对应的 standard_params.id；不存在则创建。
func (fx *mmlFixture) getOrInsertStandardParam(standardPath string) uuid.UUID {
	fx.t.Helper()
	var id uuid.UUID
	err := fx.pool.QueryRow(context.Background(),
		`SELECT id FROM standard_params WHERE standard_path = $1`, standardPath).Scan(&id)
	if err == nil {
		return id
	}
	id = uuid.New()
	_, err = fx.pool.Exec(context.Background(), `
INSERT INTO standard_params
    (id, standard_path, data_type, access, entry_type)
VALUES
    ($1, $2, 'string', 'READ_ONLY', 'parameter')
ON CONFLICT (standard_path) DO NOTHING`,
		id, standardPath)
	require.NoError(fx.t, err)
	// 处理 ON CONFLICT race：再查一次。
	_ = fx.pool.QueryRow(context.Background(),
		`SELECT id FROM standard_params WHERE standard_path = $1`, standardPath).Scan(&id)
	fx.stdParamIDs = append(fx.stdParamIDs, id)
	return id
}

// insertParamMapping 在指定 paramModel 下登记 standardPath 的 mapping，
// 控制 is_active / is_supported 两位真值；返回 mapping id。
func (fx *mmlFixture) insertParamMapping(paramModelID uuid.UUID, standardPath string, isActive, isSupported bool) uuid.UUID {
	fx.t.Helper()
	// 同时建 standard_params 行（如果还没有），与 EXISTS subquery 的 JOIN 对齐。
	_ = fx.getOrInsertStandardParam(standardPath)
	id := uuid.New()
	// 2026-05-29 migration 000219:唯一约束改 (param_model_id, private_path),
	// ON CONFLICT 同步切到新键(本 fixture 里 standard_path 与 private_path 同值,
	// 行为不变)。
	_, err := fx.pool.Exec(context.Background(), `
INSERT INTO param_mappings
    (id, param_model_id, standard_path, private_path,
     entry_type, is_active, is_supported)
VALUES
    ($1, $2, $3, $4, 'parameter', $5, $6)
ON CONFLICT (param_model_id, private_path) DO UPDATE
   SET is_active = EXCLUDED.is_active,
       is_supported = EXCLUDED.is_supported`,
		id, paramModelID, standardPath, standardPath, isActive, isSupported)
	require.NoError(fx.t, err)
	fx.pmIDs = append(fx.pmIDs, id)
	return id
}

// insertParamModel 建一个测试 param_model 并返回 ID。
func (fx *mmlFixture) insertParamModel() uuid.UUID {
	fx.t.Helper()
	id := uuid.New()
	_, err := fx.pool.Exec(context.Background(), `
INSERT INTO param_models
    (id, name, description, is_active)
VALUES
    ($1, $2, 'PR-C test param_model', true)`,
		id, "PRC_PM_"+fx.suffix)
	require.NoError(fx.t, err)
	return id
}

// cleanup 反向删除 fixture 创建的所有 row。
func (fx *mmlFixture) cleanup() {
	ctx := context.Background()
	for _, id := range fx.pmIDs {
		_, _ = fx.pool.Exec(ctx, `DELETE FROM param_mappings WHERE id = $1`, id)
	}
	for _, id := range fx.cmdIDs {
		_, _ = fx.pool.Exec(ctx, `DELETE FROM mml_command_sub_fields WHERE command_id = $1`, id)
		_, _ = fx.pool.Exec(ctx, `DELETE FROM mml_commands WHERE id = $1`, id)
	}
	if fx.groupID != uuid.Nil {
		_, _ = fx.pool.Exec(ctx, `DELETE FROM mml_command_groups WHERE id = $1`, fx.groupID)
	}
	// standard_params 跨用例共享，sub_fields 删了之后可以保留也可以清；
	// 这里 best-effort 清掉 fixture 自己创建的，未引用的 standard_params 不动以避免误伤其他测试。
	for _, id := range fx.stdParamIDs {
		_, _ = fx.pool.Exec(ctx,
			`DELETE FROM standard_params WHERE id = $1
			   AND NOT EXISTS (SELECT 1 FROM mml_command_sub_fields WHERE standard_path_id = $1)
			   AND NOT EXISTS (SELECT 1 FROM param_mappings WHERE standard_path = (
			     SELECT standard_path FROM standard_params WHERE id = $1
			   ))`, id)
	}
}
