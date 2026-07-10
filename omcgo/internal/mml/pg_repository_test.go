package mml

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	require.Contains(t, src, "standardValidationRules")
}

// This is deliberately PG-backed when a local test database is available: it
// proves the import repository applies runtime rules to the actual lowercase
// data_type values emitted by the standard catalog, rather than only to fakes.
func TestPgScriptValidationRepository_LoadsStandardRuntimeRules(t *testing.T) {
	pool := newMMLTestPool(t)
	repo := NewPgScriptValidationRepository(pool)
	ctx := context.Background()

	for _, valueType := range []string{"unsignedint", "boolean"} {
		t.Run(valueType, func(t *testing.T) {
			var code string
			err := pool.QueryRow(ctx, `
SELECT c.command_code
  FROM mml_commands c
  JOIN mml_command_sub_fields csf ON csf.command_id = c.id
  JOIN standard_params sp ON sp.id = csf.standard_path_id
 WHERE c.source = 'standard'
   AND c.deprecated_at IS NULL
   AND lower(sp.data_type) = $1
 ORDER BY c.command_code
 LIMIT 1`, valueType).Scan(&code)
			if err != nil {
				t.Skipf("no standard %s command available: %v", valueType, err)
			}

			commands, err := repo.LoadCommandsByCodes(ctx, []string{code}, ValidationActor{})
			require.NoError(t, err)
			command, ok := commands[code]
			require.True(t, ok)
			var matched *MMLParamRef
			for i := range command.ParamRefs {
				if command.ParamRefs[i].ValueType == valueType {
					matched = &command.ParamRefs[i]
					break
				}
			}
			require.NotNil(t, matched)
			require.NotEmpty(t, matched.JsRegex)
			if valueType == "boolean" {
				require.Equal(t, []interface{}{"true", "false", "0", "1"}, matched.ValueConstraint["enum"])
			}
		})
	}
}

func TestPgScriptValidationRepository_MarksDuplicateVisibleCustomCodesAmbiguous(t *testing.T) {
	pool := newMMLTestPool(t)
	ctx := context.Background()
	code := "LST TASK3_" + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM mml_custom_command WHERE command_code=$1", code)
	})

	for _, suffix := range []string{"A", "B"} {
		_, err := pool.Exec(ctx, `
INSERT INTO mml_custom_command
    (command_name, command_code, operation_type, command_scope, parameters, param_paths, creator)
VALUES ($1, $2, 'LST', 'public', '{}'::jsonb, '[]'::jsonb, 'task3-validator-test')`,
			"task3 duplicate "+suffix+" "+uuid.NewString(), code)
		require.NoError(t, err)
	}

	commands, err := NewPgScriptValidationRepository(pool).LoadCommandsByCodes(ctx, []string{code}, ValidationActor{})
	require.NoError(t, err)
	require.True(t, commands[code].Ambiguous)
}
