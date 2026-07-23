package mml

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	taskpkg "github.com/omcgo/omcgo/internal/task"
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

func TestPgTaskRepository_IncrementStatsReconcilesFromDeviceTasks(t *testing.T) {
	pool := newMMLTestPool(t)
	ctx := context.Background()
	mmlRepo := NewPgTaskRepository(pool)
	deviceTaskRepo := taskpkg.NewPgTaskRepository(pool)

	mmlTask := &MMLTask{
		TaskName:     "stats reconcile " + uuid.NewString(),
		DeviceSNs:    []string{"MML-STATS-1", "MML-STATS-2", "MML-STATS-3", "MML-STATS-4", "MML-STATS-5"},
		Commands:     []map[string]interface{}{{"command_code": "LST DEVICE_INFO"}},
		ExecuteMode:  TaskExecuteModeDeviceBound,
		Status:       TaskRunning,
		Creator:      "repo-test",
		ExecuteType:  ExecuteImmediate,
		TotalDevices: 5,
		StartedAt:    func() *time.Time { now := time.Now(); return &now }(),
	}
	require.NoError(t, mmlRepo.Create(ctx, mmlTask))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM device_tasks WHERE source='mml' AND source_id=$1", mmlTask.ID.String())
		_ = mmlRepo.Delete(context.Background(), mmlTask.ID)
	})

	for idx, sn := range mmlTask.DeviceSNs {
		dt := &taskpkg.Task{
			ID:           uuid.NewString(),
			DeviceSN:     sn,
			Method:       "GetParameterValues",
			Params:       []byte("{}"),
			Priority:     10,
			CommandKey:   "",
			Status:       taskpkg.TaskStatusCompleted,
			MaxRetries:   3,
			CreatedAt:    time.Now(),
			Source:       taskpkg.TaskSourceMML,
			SourceID:     mmlTask.ID.String(),
			CommandIndex: idx,
			DeviceIndex:  idx,
		}
		dt.MarkSent("cwmp-" + sn)
		dt.MarkCompleted([]byte(`{"ok":true}`))
		require.NoError(t, deviceTaskRepo.Create(ctx, dt))
	}

	require.NoError(t, mmlRepo.IncrementStats(ctx, mmlTask.ID, 1, 0))
	require.NoError(t, mmlRepo.IncrementStats(ctx, mmlTask.ID, 1, 0))

	got, err := mmlRepo.GetByID(ctx, mmlTask.ID)
	require.NoError(t, err)
	require.Equal(t, 5, got.SuccessCount)
	require.Equal(t, 0, got.FailedCount)
}

func TestPgTaskRepository_DeleteRemovesDeviceTasks(t *testing.T) {
	pool := newMMLTestPool(t)
	ctx := context.Background()
	mmlRepo := NewPgTaskRepository(pool)
	deviceTaskRepo := taskpkg.NewPgTaskRepository(pool)

	mmlTask := &MMLTask{
		TaskName:     "delete cascade " + uuid.NewString(),
		DeviceSNs:    []string{"MML-DELETE-1", "MML-DELETE-2"},
		Commands:     []map[string]interface{}{{"command_code": "LST DEVICE_INFO"}},
		ExecuteMode:  TaskExecuteModeDeviceBound,
		Status:       TaskCompleted,
		Creator:      "repo-test",
		ExecuteType:  ExecuteImmediate,
		TotalDevices: 2,
	}
	require.NoError(t, mmlRepo.Create(ctx, mmlTask))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM device_tasks WHERE source='mml' AND source_id=$1", mmlTask.ID.String())
		_ = mmlRepo.Delete(context.Background(), mmlTask.ID)
	})

	for idx, sn := range mmlTask.DeviceSNs {
		dt := &taskpkg.Task{
			ID:           uuid.NewString(),
			DeviceSN:     sn,
			Method:       "GetParameterValues",
			Params:       []byte("{}"),
			Priority:     10,
			Status:       taskpkg.TaskStatusCompleted,
			MaxRetries:   3,
			CreatedAt:    time.Now(),
			Source:       taskpkg.TaskSourceMML,
			SourceID:     mmlTask.ID.String(),
			CommandIndex: idx,
			DeviceIndex:  idx,
		}
		dt.MarkSent("cwmp-" + sn)
		dt.MarkCompleted([]byte(`{"ok":true}`))
		require.NoError(t, deviceTaskRepo.Create(ctx, dt))
	}

	require.NoError(t, mmlRepo.Delete(ctx, mmlTask.ID))

	var remaining int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM device_tasks WHERE source='mml' AND source_id=$1", mmlTask.ID.String()).Scan(&remaining)
	require.NoError(t, err)
	require.Equal(t, 0, remaining)
}

func TestPgTaskRepository_ListFiltersTaskOrigin(t *testing.T) {
	pool := newMMLTestPool(t)
	ctx := context.Background()
	taskRepo := NewPgTaskRepository(pool)
	scriptRepo := NewPgScriptRepository(pool)

	script := &MMLScript{
		ScriptName:        "origin filter script " + uuid.NewString(),
		Content:           "LST DEVICE_INFO;SN-SCRIPT\n",
		OriginalFilename:  "origin-filter.txt",
		ContentSHA256:     "sha-origin-filter",
		ValidationVersion: ValidationVersion,
		Creator:           "repo-test",
		Status:            ScriptActive,
		Type:              ScriptTypeBatch,
	}
	require.NoError(t, scriptRepo.CreateImported(ctx, script))
	t.Cleanup(func() { _ = scriptRepo.Delete(context.Background(), script.ID) })

	consoleTask := &MMLTask{
		TaskName:     "origin console " + uuid.NewString(),
		DeviceSNs:    []string{"SN-CONSOLE"},
		Commands:     []map[string]interface{}{{"command_code": "LST DEVICE_INFO"}},
		ExecuteMode:  TaskExecuteModeCommon,
		Status:       TaskCompleted,
		Creator:      "repo-test",
		ExecuteType:  ExecuteImmediate,
		TotalDevices: 1,
	}
	scriptTask := &MMLTask{
		TaskName:     "origin script " + uuid.NewString(),
		ScriptID:     &script.ID,
		DeviceSNs:    []string{"SN-SCRIPT"},
		Commands:     []map[string]interface{}{{"command_code": "LST DEVICE_INFO"}},
		ExecuteMode:  TaskExecuteModeDeviceBound,
		Status:       TaskCompleted,
		Creator:      "repo-test",
		ExecuteType:  ExecuteImmediate,
		TotalDevices: 1,
	}
	require.NoError(t, taskRepo.Create(ctx, consoleTask))
	require.NoError(t, taskRepo.Create(ctx, scriptTask))
	t.Cleanup(func() {
		_ = taskRepo.Delete(context.Background(), consoleTask.ID)
		_ = taskRepo.Delete(context.Background(), scriptTask.ID)
	})

	consoleOrigin := TaskOriginConsole
	consoleList, err := taskRepo.List(ctx, TaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
		TaskName:    &consoleTask.TaskName,
		TaskOrigin:  &consoleOrigin,
	})
	require.NoError(t, err)
	require.Len(t, consoleList.Items, 1)
	require.Equal(t, consoleTask.ID, consoleList.Items[0].ID)
	require.Equal(t, TaskOriginConsole, consoleList.Items[0].TaskOrigin)
	require.Empty(t, consoleList.Items[0].ScriptName)

	scriptOrigin := TaskOriginScript
	scriptList, err := taskRepo.List(ctx, TaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
		TaskName:    &scriptTask.TaskName,
		TaskOrigin:  &scriptOrigin,
	})
	require.NoError(t, err)
	require.Len(t, scriptList.Items, 1)
	require.Equal(t, scriptTask.ID, scriptList.Items[0].ID)
	require.Equal(t, TaskOriginScript, scriptList.Items[0].TaskOrigin)
	require.Equal(t, script.ScriptName, scriptList.Items[0].ScriptName)
}

func TestPgTaskRepository_ListFiltersScriptName(t *testing.T) {
	pool := newMMLTestPool(t)
	ctx := context.Background()
	taskRepo := NewPgTaskRepository(pool)
	scriptRepo := NewPgScriptRepository(pool)

	matchingScript := &MMLScript{
		ScriptName:        "script name filter 巡检 " + uuid.NewString(),
		Content:           "LST DEVICE_INFO;SN-SCRIPT-A\n",
		OriginalFilename:  "script-name-filter-a.txt",
		ContentSHA256:     "sha-script-name-filter-a",
		ValidationVersion: ValidationVersion,
		Creator:           "repo-test",
		Status:            ScriptActive,
		Type:              ScriptTypeBatch,
	}
	otherScript := &MMLScript{
		ScriptName:        "script name filter 配置 " + uuid.NewString(),
		Content:           "LST DEVICE_INFO;SN-SCRIPT-B\n",
		OriginalFilename:  "script-name-filter-b.txt",
		ContentSHA256:     "sha-script-name-filter-b",
		ValidationVersion: ValidationVersion,
		Creator:           "repo-test",
		Status:            ScriptActive,
		Type:              ScriptTypeBatch,
	}
	require.NoError(t, scriptRepo.CreateImported(ctx, matchingScript))
	require.NoError(t, scriptRepo.CreateImported(ctx, otherScript))
	t.Cleanup(func() {
		_ = scriptRepo.Delete(context.Background(), matchingScript.ID)
		_ = scriptRepo.Delete(context.Background(), otherScript.ID)
	})

	matchingTask := &MMLTask{
		TaskName:     "script-name matched task " + uuid.NewString(),
		ScriptID:     &matchingScript.ID,
		DeviceSNs:    []string{"SN-SCRIPT-A"},
		Commands:     []map[string]interface{}{{"command_code": "LST DEVICE_INFO"}},
		ExecuteMode:  TaskExecuteModeDeviceBound,
		Status:       TaskCompleted,
		Creator:      "repo-test",
		ExecuteType:  ExecuteImmediate,
		TotalDevices: 1,
	}
	otherTask := &MMLTask{
		TaskName:     "script-name other task " + uuid.NewString(),
		ScriptID:     &otherScript.ID,
		DeviceSNs:    []string{"SN-SCRIPT-B"},
		Commands:     []map[string]interface{}{{"command_code": "LST DEVICE_INFO"}},
		ExecuteMode:  TaskExecuteModeDeviceBound,
		Status:       TaskCompleted,
		Creator:      "repo-test",
		ExecuteType:  ExecuteImmediate,
		TotalDevices: 1,
	}
	consoleTask := &MMLTask{
		TaskName:     "script-name console task " + uuid.NewString(),
		DeviceSNs:    []string{"SN-CONSOLE"},
		Commands:     []map[string]interface{}{{"command_code": "LST DEVICE_INFO"}},
		ExecuteMode:  TaskExecuteModeCommon,
		Status:       TaskCompleted,
		Creator:      "repo-test",
		ExecuteType:  ExecuteImmediate,
		TotalDevices: 1,
	}
	require.NoError(t, taskRepo.Create(ctx, matchingTask))
	require.NoError(t, taskRepo.Create(ctx, otherTask))
	require.NoError(t, taskRepo.Create(ctx, consoleTask))
	t.Cleanup(func() {
		_ = taskRepo.Delete(context.Background(), matchingTask.ID)
		_ = taskRepo.Delete(context.Background(), otherTask.ID)
		_ = taskRepo.Delete(context.Background(), consoleTask.ID)
	})

	scriptName := "巡检"
	list, err := taskRepo.List(ctx, TaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
		ScriptName:  &scriptName,
	})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, matchingTask.ID, list.Items[0].ID)
	require.Equal(t, matchingScript.ScriptName, list.Items[0].ScriptName)
	require.Equal(t, int64(1), list.Total)
}

func TestPgTaskRepository_ListIncludesLatestPeriodicRun(t *testing.T) {
	pool := newMMLTestPool(t)
	ctx := context.Background()
	taskRepo := NewPgTaskRepository(pool)

	taskName := "periodic latest run " + uuid.NewString()
	parent := &MMLTask{
		TaskName:     taskName,
		DeviceSNs:    []string{"SN-PERIODIC-1", "SN-PERIODIC-2"},
		Commands:     []map[string]interface{}{{"command_code": "LST DEVICE_INFO"}},
		ExecuteMode:  TaskExecuteModeDeviceBound,
		Status:       TaskCompleted,
		Creator:      "repo-test",
		ExecuteType:  ExecutePeriodic,
		TotalDevices: 2,
	}
	require.NoError(t, taskRepo.Create(ctx, parent))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM mml_tasks WHERE parent_task_id=$1", parent.ID)
		_ = taskRepo.Delete(context.Background(), parent.ID)
	})

	firstResult := ResultSuccess
	first := &MMLTask{
		TaskName:         taskName,
		DeviceSNs:        parent.DeviceSNs,
		Commands:         parent.Commands,
		ExecuteMode:      TaskExecuteModeDeviceBound,
		Status:           TaskCompleted,
		Creator:          "repo-test",
		ExecuteType:      ExecuteImmediate,
		PeriodicParentID: &parent.ID,
		TotalDevices:     2,
		SuccessCount:     2,
		FailedCount:      0,
		Result:           &firstResult,
		StartedAt:        func() *time.Time { ts := time.Now().Add(-10 * time.Minute); return &ts }(),
		FinishedAt:       func() *time.Time { ts := time.Now().Add(-9 * time.Minute); return &ts }(),
	}
	require.NoError(t, taskRepo.Create(ctx, first))

	latestResult := ResultPartial
	latest := &MMLTask{
		TaskName:         taskName,
		DeviceSNs:        parent.DeviceSNs,
		Commands:         parent.Commands,
		ExecuteMode:      TaskExecuteModeDeviceBound,
		Status:           TaskCompleted,
		Creator:          "repo-test",
		ExecuteType:      ExecuteImmediate,
		PeriodicParentID: &parent.ID,
		TotalDevices:     2,
		SuccessCount:     5,
		FailedCount:      1,
		Result:           &latestResult,
		StartedAt:        func() *time.Time { ts := time.Now().Add(-2 * time.Minute); return &ts }(),
		FinishedAt:       func() *time.Time { ts := time.Now().Add(-1 * time.Minute); return &ts }(),
	}
	require.NoError(t, taskRepo.Create(ctx, latest))

	executeType := ExecutePeriodic
	list, err := taskRepo.List(ctx, TaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
		TaskName:    &taskName,
		ExecuteType: &executeType,
	})

	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.NotNil(t, list.Items[0].LatestRun)
	require.Equal(t, latest.ID, list.Items[0].LatestRun.ID)
	require.Equal(t, latest.SuccessCount, list.Items[0].LatestRun.SuccessCount)
	require.Equal(t, latest.FailedCount, list.Items[0].LatestRun.FailedCount)
	require.Equal(t, latest.Result, list.Items[0].LatestRun.Result)
	require.Equal(t, 1, list.Items[0].LatestRun.CommandCount)
	require.Equal(t, 0, list.Items[0].LatestRun.PlanItemCount)
}

func TestPgScriptRepository_ImportedReplaceAndMetadataAreAtomic(t *testing.T) {
	pool := newMMLTestPool(t)
	repo := NewPgScriptRepository(pool)
	ctx := context.Background()
	script := &MMLScript{
		ScriptName: "imported repository test", Content: "LST DEVICE_INFO;SN1\n",
		OriginalFilename: "test.txt", ContentSHA256: "sha-repo", ValidationVersion: ValidationVersion,
		ValidatedAt:       func() *time.Time { now := time.Now(); return &now }(),
		PlanItems:         []MMLPlanItem{{LineNo: 1, DeviceSN: "SN1", Order: 1, CommandCode: "LST DEVICE_INFO"}},
		ValidationSummary: JSONMap{"summary": map[string]interface{}{"valid_lines": 1}}, Creator: "repo-test",
		Status: ScriptActive, Type: ScriptTypeBatch, Tags: []string{"old"},
	}
	repoErr := repo.CreateImported(ctx, script)
	require.NoError(t, repoErr)
	t.Cleanup(func() { _ = repo.Delete(ctx, script.ID) })
	loaded, err := repo.GetByImportSessionID(ctx, script.ImportSessionID)
	require.NoError(t, err)
	require.Equal(t, script.ContentSHA256, loaded.ContentSHA256)
	require.NoError(t, repo.UpdateMetadata(ctx, script.ID, "renamed", "description", []string{"new"}))
	loaded, err = repo.GetByID(ctx, script.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", loaded.ScriptName)
	stale := *loaded
	stale.ImportSessionID = uuid.New()
	stale.Content = "MOD DEVICE_INFO:USER_LABEL=x;SN1\n"
	stale.ContentSHA256 = "sha-repo-2"
	stale.PlanItems = []MMLPlanItem{{LineNo: 1, DeviceSN: "SN1", Order: 1, CommandCode: "MOD DEVICE_INFO"}}
	require.NoError(t, repo.ReplaceImported(ctx, &stale, loaded.UpdatedAt))
	reloaded, err := repo.GetByID(ctx, script.ID)
	require.NoError(t, err)
	require.Equal(t, "sha-repo-2", reloaded.ContentSHA256)
	require.ErrorIs(t, repo.ReplaceImported(ctx, &stale, loaded.UpdatedAt), ErrScriptVersionConflict)
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

func TestPgTaskRepository_ListUsesSummaryColumns(t *testing.T) {
	require.NotContains(t, taskListColumns, "commands")
	require.NotContains(t, taskListColumns, "plan_items")
	require.NotContains(t, taskListColumns, "results")
	require.Contains(t, taskListColumns, "mml_scripts.script_name")
	require.Contains(t, taskListColumns, "mml_tasks.total_devices")
	require.Contains(t, taskListColumns, "mml_tasks.success_count")
	require.Contains(t, taskListColumns, "mml_tasks.failed_count")
}

func TestPgTaskRepository_ProvidesLightweightResultStatsLookup(t *testing.T) {
	var repo interface{} = (*PgTaskRepository)(nil)
	require.Implements(t, (*TaskResultStatsRepository)(nil), repo)
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

func TestPgScriptValidationRepository_LoadStandardPathSupportMatchesTemplates(t *testing.T) {
	pool := newMMLTestPool(t)
	repo := NewPgScriptValidationRepository(pool)
	ctx := context.Background()

	const knownTemplate = "Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress"
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM standard_params WHERE standard_path=$1)`, knownTemplate).Scan(&exists)
	require.NoError(t, err)
	if !exists {
		t.Skipf("standard path fixture %s is not available", knownTemplate)
	}

	parameter := StandardPathLookup{Kind: StandardPathLookupParameter, Path: normalizeStandardPathTemplate("Device.IP.Interface.1.IPv4Address.3.IPAddress")}
	addObject := StandardPathLookup{Kind: StandardPathLookupObject, Path: normalizeStandardPathTemplate("Device.IP.Interface.1.IPv4Address.")}
	removeObject := StandardPathLookup{Kind: StandardPathLookupObject, Path: normalizeStandardPathTemplate("Device.IP.Interface.1.IPv4Address.3.")}
	missing := StandardPathLookup{Kind: StandardPathLookupParameter, Path: normalizeStandardPathTemplate("Device.NotInStandardParams.1.Enable")}

	support, err := repo.LoadStandardPathSupport(ctx, []StandardPathLookup{parameter, addObject, removeObject, missing})

	require.NoError(t, err)
	require.True(t, support[parameter.key()])
	require.True(t, support[addObject.key()])
	require.True(t, support[removeObject.key()])
	require.False(t, support[missing.key()])
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
