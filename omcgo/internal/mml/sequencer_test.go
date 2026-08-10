package mml

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ============================================================
// sequencer_test.go — R-4.3 ADD 复合流程纯函数测试
//
// 不覆盖完整 Sequencer.OnTaskCompleted 流程（需要 mock taskRepo / enqueuer /
// fanouter，留给集成测试）；仅覆盖新增的纯函数：
//   - isAddObjectMethod / isSpvCmdEntry — 条件判定
//   - extractInstanceNumber — JSON 解析容错
//   - substituteNewInstance — 双形态（in-memory + JSON-roundtrip）path 替换
// ============================================================

func TestIsAddObjectMethod(t *testing.T) {
	assert.True(t, isAddObjectMethod("AddObject"))
	assert.True(t, isAddObjectMethod("addobject"))     // case-insensitive
	assert.True(t, isAddObjectMethod("  AddObject  ")) // trim space
	assert.False(t, isAddObjectMethod(""))
	assert.False(t, isAddObjectMethod("SetParameterValues"))
	assert.False(t, isAddObjectMethod("GetParameterValues"))
}

func TestIsSpvCmdEntry(t *testing.T) {
	assert.True(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": "SetParameterValues"}))
	assert.True(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": "setparametervalues"}))
	assert.False(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": "AddObject"}))
	assert.False(t, isSpvCmdEntry(map[string]interface{}{}))                  // 缺字段
	assert.False(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": 42}))  // 非 string
	assert.False(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": nil})) // nil
}

func TestExtractInstanceNumber(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		wantN  int
		wantOK bool
	}{
		{
			name:   "标准 ACS handler 写入形态",
			input:  `{"method":"AddObjectResponse","raw_response":"...","instance_number":5}`,
			wantN:  5,
			wantOK: true,
		},
		{
			name:   "纯 instance_number",
			input:  `{"instance_number":42}`,
			wantN:  42,
			wantOK: true,
		},
		{
			name:   "instance_number = 0（有效，0 也是合法实例号）",
			input:  `{"instance_number":0}`,
			wantN:  0,
			wantOK: true,
		},
		{
			name:   "字符串形态 fallback",
			input:  `{"instance_number":"7"}`,
			wantN:  7,
			wantOK: true,
		},
		{
			name:   "缺 instance_number 字段",
			input:  `{"method":"AddObjectResponse"}`,
			wantN:  0,
			wantOK: false,
		},
		{
			name:   "非 JSON",
			input:  `not json`,
			wantN:  0,
			wantOK: false,
		},
		{
			name:   "空字符串",
			input:  ``,
			wantN:  0,
			wantOK: false,
		},
		{
			name:   "字符串数字非法",
			input:  `{"instance_number":"abc"}`,
			wantN:  0,
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotN, gotOK := extractInstanceNumber(json.RawMessage(tc.input))
			assert.Equal(t, tc.wantOK, gotOK)
			if tc.wantOK {
				assert.Equal(t, tc.wantN, gotN)
			}
		})
	}
}

// TestSubstituteNewInstance_InMemoryShape — param_refs 是 []MMLParamRef
// （直接 in-memory，单测场景）。
func TestSubstituteNewInstance_InMemoryShape(t *testing.T) {
	cmd := map[string]interface{}{
		"rpc_method": "SetParameterValues",
		"param_refs": []MMLParamRef{
			{ID: uuid.New(), ParamCode: "PLMNID", Tr069Path: "Device.Services.FAPService.1.PLMNList.{NEW}.PLMNID"},
			{ID: uuid.New(), ParamCode: "Enable", Tr069Path: "Device.Services.FAPService.1.PLMNList.{NEW}.Enable"},
		},
		"parameters": map[string]interface{}{"PLMNID": "46000", "Enable": "true"},
	}
	out := substituteNewInstance(cmd, 5)

	// 原 cmd 不变（深拷贝校验）
	origRefs := cmd["param_refs"].([]MMLParamRef)
	assert.Equal(t, "Device.Services.FAPService.1.PLMNList.{NEW}.PLMNID", origRefs[0].Tr069Path)

	// out.param_refs.Tr069Path 已替换
	outRefs := out["param_refs"].([]MMLParamRef)
	require.Len(t, outRefs, 2)
	assert.Equal(t, "Device.Services.FAPService.1.PLMNList.5.PLMNID", outRefs[0].Tr069Path)
	assert.Equal(t, "Device.Services.FAPService.1.PLMNList.5.Enable", outRefs[1].Tr069Path)

	// parameters 字段不动（mml_code keys，不含 path 占位符）
	assert.Equal(t, cmd["parameters"], out["parameters"])
}

// TestSubstituteNewInstance_JSONRoundTripShape — 模拟 mmlTask 从 DB JSONB
// 反序列化后的形态：param_refs 是 []interface{} of map[string]interface{}。
func TestSubstituteNewInstance_JSONRoundTripShape(t *testing.T) {
	// 模拟 JSONB unmarshal 输出形态
	cmdJSON := `{
		"rpc_method": "SetParameterValues",
		"param_refs": [
			{"id":"00000000-0000-0000-0000-000000000001","param_code":"PLMNID","tr069_path":"Device.X.{NEW}.PLMNID"},
			{"id":"00000000-0000-0000-0000-000000000002","param_code":"Enable","tr069_path":"Device.X.{NEW}.Enable"}
		],
		"parameters": {"PLMNID":"46000"}
	}`
	var cmd map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(cmdJSON), &cmd))

	out := substituteNewInstance(cmd, 7)

	outRefs := out["param_refs"].([]interface{})
	require.Len(t, outRefs, 2)
	r0 := outRefs[0].(map[string]interface{})
	r1 := outRefs[1].(map[string]interface{})
	assert.Equal(t, "Device.X.7.PLMNID", r0["tr069_path"])
	assert.Equal(t, "Device.X.7.Enable", r1["tr069_path"])
	// 其他字段保留
	assert.Equal(t, "PLMNID", r0["param_code"])
}

func TestSubstituteNewInstance_NoPlaceholder_NoOp(t *testing.T) {
	// path 不含 .{NEW}. → ReplaceAll 是 no-op，输出与输入语义等价
	cmd := map[string]interface{}{
		"rpc_method": "SetParameterValues",
		"param_refs": []MMLParamRef{
			{Tr069Path: "Device.Services.FAPService.1.PLMNList.5.PLMNID"},
		},
	}
	out := substituteNewInstance(cmd, 99)
	outRefs := out["param_refs"].([]MMLParamRef)
	assert.Equal(t, "Device.Services.FAPService.1.PLMNList.5.PLMNID", outRefs[0].Tr069Path)
}

func TestSubstituteNewInstance_UnknownShape_PreservedAsIs(t *testing.T) {
	// param_refs 既不是 []MMLParamRef 也不是 []interface{} → 原样保留（防御性）
	cmd := map[string]interface{}{
		"rpc_method": "SetParameterValues",
		"param_refs": "unexpected string",
	}
	out := substituteNewInstance(cmd, 5)
	assert.Equal(t, "unexpected string", out["param_refs"])
}

func TestSubstituteNewInstance_InstanceNumberZero(t *testing.T) {
	// instance_number = 0 也是合法的（某些设备从 0 起编号），替换应成功
	cmd := map[string]interface{}{
		"rpc_method": "SetParameterValues",
		"param_refs": []MMLParamRef{
			{Tr069Path: "Device.X.{NEW}.Foo"},
		},
	}
	out := substituteNewInstance(cmd, 0)
	outRefs := out["param_refs"].([]MMLParamRef)
	assert.Equal(t, "Device.X.0.Foo", outRefs[0].Tr069Path)
}

type sequencerTestEnqueuer struct {
	reqs     []*task.CreateTaskRequest
	onCreate func(req *task.CreateTaskRequest, created *task.Task)
}

func (e *sequencerTestEnqueuer) CreateTask(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	e.reqs = append(e.reqs, req)
	created := &task.Task{
		ID:           uuid.New().String(),
		Source:       req.Source,
		SourceID:     req.SourceID,
		DeviceSN:     req.DeviceSN,
		Method:       req.Method,
		CommandIndex: req.CommandIndex,
		DeviceIndex:  req.DeviceIndex,
		Status:       task.TaskStatusPending,
	}
	if req.FailImmediately {
		created.MarkFailed(0, req.FailReason)
	}
	if e.onCreate != nil {
		e.onCreate(req, created)
	}
	return created, nil
}

func (e *sequencerTestEnqueuer) GetQueueLength(context.Context, string) (int64, error) {
	return 0, nil
}

func TestSequencer_ChainBreakRecordsDependentFailureAndContinues(t *testing.T) {
	mmlID := uuid.New()
	mmlTask := &MMLTask{
		ID:        mmlID,
		Status:    TaskRunning,
		DeviceSNs: []string{"SN001"},
		Commands: []map[string]interface{}{
			{
				"command_code": "ADD CELL",
				"rpc_method":   "AddObject",
			},
			{
				"command_code":   "MOD NEW CELL",
				"rpc_method":     "SetParameterValues",
				"compound_phase": "spv_after_add",
				"param_refs": []MMLParamRef{
					{ParamCode: "Enable", Tr069Path: "Device.Services.FAPService.1.CellConfig.{NEW}.Enable"},
				},
				"parameters": map[string]interface{}{"Enable": true},
			},
			{
				"command_code":   "LST DEVICE_INFO",
				"rpc_method":     "GetParameterValues",
				"operation_type": "LST",
				"param_refs": []MMLParamRef{
					{ParamCode: "SerialNumber", Tr069Path: "Device.DeviceInfo.SerialNumber"},
				},
			},
		},
	}
	repo := &mockTaskRepo{
		getByIDFn: func(context.Context, uuid.UUID) (*MMLTask, error) {
			return mmlTask, nil
		},
	}
	enqueuer := &sequencerTestEnqueuer{}
	fanouter := NewFanouter(nil, nil, nil, nil, zap.NewNop())
	seq := NewSequencer(repo, enqueuer, fanouter, zap.NewNop())
	enqueuer.onCreate = func(req *task.CreateTaskRequest, created *task.Task) {
		if req.FailImmediately {
			seq.OnTaskCompleted(context.Background(), created)
		}
	}

	seq.OnTaskCompleted(context.Background(), &task.Task{
		ID:           uuid.New().String(),
		Source:       task.TaskSourceMML,
		SourceID:     mmlID.String(),
		DeviceSN:     "SN001",
		Method:       "AddObject",
		CommandIndex: 0,
		DeviceIndex:  0,
		Status:       task.TaskStatusFailed,
		Result:       json.RawMessage(`{"fault_code":9005}`),
	})

	require.Len(t, enqueuer.reqs, 2)
	assert.Equal(t, 1, enqueuer.reqs[0].CommandIndex)
	assert.Equal(t, "SetParameterValues", enqueuer.reqs[0].Method)
	assert.True(t, enqueuer.reqs[0].FailImmediately)
	assert.Contains(t, enqueuer.reqs[0].FailReason, "missing instance_number")

	assert.Equal(t, 2, enqueuer.reqs[1].CommandIndex)
	assert.Equal(t, "GetParameterValues", enqueuer.reqs[1].Method)
	assert.False(t, enqueuer.reqs[1].FailImmediately)
}

func TestSequencer_FailedAddSPVRollsBackCreatedInstance(t *testing.T) {
	mmlID := uuid.New()
	mmlTask := &MMLTask{
		ID: mmlID, Status: TaskRunning, DeviceSNs: []string{"SN001"},
		Commands: []map[string]interface{}{
			{"rpc_method": "AddObject"},
			{"rpc_method": "SetParameterValues", "compound_phase": "spv_after_add"},
			{
				"rpc_method": "DeleteObject", "compound_phase": "rollback_after_add",
				"compensation_only": true,
				"parameters":        map[string]interface{}{"object_name": "Device.Cell.{NEW}."},
			},
		},
	}
	repo := &mockTaskRepo{getByIDFn: func(context.Context, uuid.UUID) (*MMLTask, error) { return mmlTask, nil }}
	enqueuer := &sequencerTestEnqueuer{}
	seq := NewSequencer(repo, enqueuer, NewFanouter(nil, nil, nil, nil, zap.NewNop()), zap.NewNop())

	seq.OnTaskCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), Source: task.TaskSourceMML, SourceID: mmlID.String(),
		DeviceSN: "SN001", Method: "SetParameterValues", CommandIndex: 1,
		Status: task.TaskStatusFailed,
		Params: json.RawMessage(`{"values":[],"_mml_rollback_object_name":"Device.Cell.3."}`),
	})

	require.Len(t, enqueuer.reqs, 1)
	assert.Equal(t, "DeleteObject", enqueuer.reqs[0].Method)
	assert.Equal(t, 2, enqueuer.reqs[0].CommandIndex)
	var params map[string]interface{}
	require.NoError(t, json.Unmarshal(enqueuer.reqs[0].Params, &params))
	assert.Equal(t, "Device.Cell.3.", params["object_name"])
}

func TestSequencer_AddObjectPassesRollbackTargetToSPV(t *testing.T) {
	mmlID := uuid.New()
	mmlTask := &MMLTask{
		ID: mmlID, Status: TaskRunning, DeviceSNs: []string{"SN001"},
		Commands: []map[string]interface{}{
			{"rpc_method": "AddObject"},
			{
				"rpc_method": "SetParameterValues", "compound_phase": "spv_after_add",
				"param_refs": []MMLParamRef{{ParamCode: "Enable", Tr069Path: "Device.Cell.{NEW}.Enable", ValueType: "boolean"}},
				"parameters": map[string]interface{}{"Enable": "true"},
			},
		},
	}
	repo := &mockTaskRepo{getByIDFn: func(context.Context, uuid.UUID) (*MMLTask, error) { return mmlTask, nil }}
	enqueuer := &sequencerTestEnqueuer{}
	seq := NewSequencer(repo, enqueuer, NewFanouter(nil, nil, nil, nil, zap.NewNop()), zap.NewNop())

	seq.OnTaskCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), Source: task.TaskSourceMML, SourceID: mmlID.String(), DeviceSN: "SN001",
		Method: "AddObject", CommandIndex: 0, Status: task.TaskStatusCompleted,
		Params: json.RawMessage(`{"object_name":"Device.Cell."}`),
		Result: json.RawMessage(`{"instance_number":3}`),
	})

	require.Len(t, enqueuer.reqs, 1)
	var params map[string]interface{}
	require.NoError(t, json.Unmarshal(enqueuer.reqs[0].Params, &params))
	assert.Equal(t, "Device.Cell.3.", params[rollbackObjectNameKey])
}

func TestSequencer_SuccessfulAddSPVSkipsRollback(t *testing.T) {
	mmlID := uuid.New()
	mmlTask := &MMLTask{
		ID: mmlID, Status: TaskRunning, DeviceSNs: []string{"SN001"},
		Commands: []map[string]interface{}{
			{"rpc_method": "AddObject"},
			{"rpc_method": "SetParameterValues", "compound_phase": "spv_after_add"},
			{"rpc_method": "DeleteObject", "compound_phase": "rollback_after_add", "compensation_only": true},
		},
	}
	repo := &mockTaskRepo{getByIDFn: func(context.Context, uuid.UUID) (*MMLTask, error) { return mmlTask, nil }}
	enqueuer := &sequencerTestEnqueuer{}
	seq := NewSequencer(repo, enqueuer, NewFanouter(nil, nil, nil, nil, zap.NewNop()), zap.NewNop())

	seq.OnTaskCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), Source: task.TaskSourceMML, SourceID: mmlID.String(),
		DeviceSN: "SN001", Method: "SetParameterValues", CommandIndex: 1,
		Status: task.TaskStatusCompleted,
	})

	assert.Empty(t, enqueuer.reqs)
}
