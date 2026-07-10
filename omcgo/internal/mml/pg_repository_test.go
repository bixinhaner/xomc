package mml

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPgScriptRepository_TXTImportColumns locks the database contract for the
// TXT import workflow. newMMLTestPool skips this integration test when no
// PostgreSQL test database is configured.
func TestPgScriptRepository_TXTImportColumns(t *testing.T) {
	pool := newMMLTestPool(t)
	var count int
	err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		  FROM information_schema.columns
		 WHERE table_schema='public' AND table_name='mml_scripts'
		   AND column_name = ANY($1)`, []string{
		"import_session_id", "original_filename", "content_sha256", "validation_version",
		"validated_at", "plan_items", "validation_summary",
	}).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 7, count)

	var importSessionDefault string
	err = pool.QueryRow(context.Background(), `
		SELECT COALESCE(column_default, '')
		  FROM information_schema.columns
		 WHERE table_schema='public' AND table_name='mml_scripts'
		   AND column_name='import_session_id'`).Scan(&importSessionDefault)
	require.NoError(t, err)
	require.Contains(t, importSessionDefault, "gen_random_uuid()")

	var taskCount int
	err = pool.QueryRow(context.Background(), `
		SELECT count(*)
		  FROM information_schema.columns
		 WHERE table_schema='public' AND table_name='mml_tasks'
		   AND column_name = ANY($1)`, []string{
		"script_content_sha256", "script_validation_version",
	}).Scan(&taskCount)
	require.NoError(t, err)
	require.Equal(t, 2, taskCount)
}

// Legacy create callers have no TXT import session yet. The repository must
// assign one so the unique database contract does not reject the second create.
func TestPgScriptRepository_CreateAssignsImportSessionID(t *testing.T) {
	pool := newMMLTestPool(t)
	repo := NewPgScriptRepository(pool)
	ctx := context.Background()

	first := &MMLScript{ScriptName: "legacy import session first", Content: "LST DEVICE_INFO"}
	second := &MMLScript{ScriptName: "legacy import session second", Content: "LST DEVICE_INFO"}
	t.Cleanup(func() {
		if first.ID != uuid.Nil {
			_ = repo.Delete(ctx, first.ID)
		}
		if second.ID != uuid.Nil {
			_ = repo.Delete(ctx, second.ID)
		}
	})

	require.NoError(t, repo.Create(ctx, first))
	require.NoError(t, repo.Create(ctx, second))
	require.NotEqual(t, uuid.Nil, first.ImportSessionID)
	require.NotEqual(t, uuid.Nil, second.ImportSessionID)
	require.NotEqual(t, first.ImportSessionID, second.ImportSessionID)
}

// 回归：migration 000090 DROP `mml_command_params_rel`，migration 000095/000113
// 用 `mml_command_sub_fields` + `standard_params` 替代。`pg_repository.go` 内的
// CommandParamRepository SQL 一旦再次引用老表，运行时所有"从命令树选命令 →
// execute"链路都会因 `relation does not exist` 失败、`attachParamRefs` 静默
// 兜底导致 fanout 创建 0 device_tasks。该测试以源码字符串断言守门，避免回归。
func TestPgCommandParamRepository_SQLNoDeadTableReferences(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	dir := filepath.Dir(thisFile)

	body, err := os.ReadFile(filepath.Join(dir, "pg_repository.go"))
	require.NoError(t, err)

	src := string(body)
	// 只查"实际查询语句"形态（JOIN/FROM），避免误伤迁移历史的描述性注释。
	assert.NotContains(t, src, "JOIN mml_command_params_rel",
		"pg_repository.go 不允许 JOIN migration 000090 DROP 的 mml_command_params_rel；"+
			"如需读取命令-参数关系，走 mml_command_sub_fields JOIN standard_params。")
	assert.NotContains(t, src, "FROM mml_command_params_rel",
		"pg_repository.go 不允许 FROM migration 000090 DROP 的 mml_command_params_rel。")
	assert.Contains(t, src, "mml_command_sub_fields",
		"ListByCommandID/IDs 应当 JOIN mml_command_sub_fields")
	assert.Contains(t, src, "standard_params",
		"ListByCommandID/IDs 应当 JOIN standard_params 取 standardPath")
}

// TXT validation must remain set-based. This source-level contract complements
// validator unit tests (which count repository method calls) by preventing a
// future PG implementation from silently regressing to per-row lookups.
func TestPgScriptValidationRepository_UsesBatchQueries(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "pg_repository.go"))
	require.NoError(t, err)

	src := string(body)
	require.Contains(t, src, "WHERE command_code = ANY($1)")
	require.Contains(t, src, "WHERE csf.command_id = ANY($1)")
	require.Contains(t, src, "WHERE serial_number = ANY($1)")
	require.Contains(t, src, "mml_command_sub_fields csf")
}
