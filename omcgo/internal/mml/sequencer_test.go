package mml

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.True(t, isAddObjectMethod("addobject"))       // case-insensitive
	assert.True(t, isAddObjectMethod("  AddObject  "))   // trim space
	assert.False(t, isAddObjectMethod(""))
	assert.False(t, isAddObjectMethod("SetParameterValues"))
	assert.False(t, isAddObjectMethod("GetParameterValues"))
}

func TestIsSpvCmdEntry(t *testing.T) {
	assert.True(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": "SetParameterValues"}))
	assert.True(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": "setparametervalues"}))
	assert.False(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": "AddObject"}))
	assert.False(t, isSpvCmdEntry(map[string]interface{}{}))                          // 缺字段
	assert.False(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": 42}))          // 非 string
	assert.False(t, isSpvCmdEntry(map[string]interface{}{"rpc_method": nil}))         // nil
}

func TestExtractInstanceNumber(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantN   int
		wantOK  bool
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
