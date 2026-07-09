package mml

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// 回归 to-do-list Q4 根因：Fanouter 必须把 mml task 翻成 TR-069 wire 格式，
// 而不是把前端表单 map 原样塞进 device_tasks.params。
//
// 历史失败形态（必须不再出现）：
//   {"LTE_BTSNUM":"", "LTE_GSM_IP":"", ...}
//
// 期望形态：
//   {"names":["Device.X_BAICELLS.LTE.BtsNum", "Device.X_BAICELLS.LTE.GsmIP", ...]}

func TestFanouter_BuildDeviceTaskRequests_GetParameterValues_TR069Shape(t *testing.T) {
	stub := &stubDeviceTaskCreator{}
	f := NewFanouter(stub, nil, nil, nil, zap.NewNop())

	// 模拟 service 已经把 param_refs 挂到 entry 上
	mmlTask := &MMLTask{
		ID:        uuid.New(),
		TaskName:  "LST DEVICE_INFO",
		DeviceSNs: []string{"SN-A", "SN-B"},
		Commands: []map[string]interface{}{
			{
				"command_code":   "LST DEVICE_INFO",
				"rpc_method":     "GetParameterValues",
				"operation_type": "LST",
				// 用户没填表单，前端按 param_code 列了空值
				"parameters": map[string]interface{}{
					"LTE_BTSNUM":                  "",
					"LTE_GSM_IP":                  "",
					"DEVICEGSM_MCC":               "",
					"LTE_DEVICE_HARDWARE_VERSION": "",
				},
				"param_refs": []MMLParamRef{
					{ParamCode: "LTE_BTSNUM", Tr069Path: "Device.X_BAICELLS.LTE.BtsNum", ValueType: "unsignedInt"},
					{ParamCode: "LTE_GSM_IP", Tr069Path: "Device.X_BAICELLS.LTE.GsmIP", ValueType: "string"},
					{ParamCode: "DEVICEGSM_MCC", Tr069Path: "Device.X_BAICELLS.LTE.GsmMcc", ValueType: "string"},
					{ParamCode: "LTE_DEVICE_HARDWARE_VERSION", Tr069Path: "Device.DeviceInfo.HardwareVersion", ValueType: "string"},
				},
			},
		},
		Creator: "admin",
	}

	n, err := f.Fanout(context.Background(), mmlTask)
	require.NoError(t, err)
	assert.Equal(t, 2, n, "1 command × 2 devices = 2 device_tasks")
	require.Len(t, stub.calls, 1)
	require.Len(t, stub.calls[0], 2)

	// 全部 device_tasks.params 必须是 TR-069 wire 格式
	for i, req := range stub.calls[0] {
		var got struct {
			Names []string `json:"names"`
		}
		require.NoError(t, json.Unmarshal(req.Params, &got), "device_task[%d]", i)
		require.Len(t, got.Names, 4, "device_task[%d] names count", i)
		assert.Contains(t, got.Names, "Device.DeviceInfo.HardwareVersion")
		assert.Equal(t, "GetParameterValues", req.Method)

		// 关键回归：禁止出现 param_code 字符串
		assert.NotContains(t, string(req.Params), "LTE_BTSNUM")
		assert.NotContains(t, string(req.Params), "DEVICEGSM_MCC")
	}
}

func TestFanouter_BuildDeviceTaskRequests_SetParameterValues(t *testing.T) {
	stub := &stubDeviceTaskCreator{}
	f := NewFanouter(stub, nil, nil, nil, zap.NewNop())

	mmlTask := &MMLTask{
		ID:        uuid.New(),
		DeviceSNs: []string{"SN-A"},
		Commands: []map[string]interface{}{
			{
				"command_code":   "MOD DEVICE_INFO",
				"rpc_method":     "SetParameterValues",
				"operation_type": "MOD",
				"parameters": map[string]interface{}{
					"DEVICEGSM_MCC":       "460",
					"DEVICEGSM_NRIBITLEN": float64(8),
					"LTE_BTSNUM":          "", // 应被过滤
				},
				"param_refs": []MMLParamRef{
					{ParamCode: "DEVICEGSM_MCC", Tr069Path: "Device.X.GsmMcc", ValueType: "string", IsWritable: true},
					{ParamCode: "DEVICEGSM_NRIBITLEN", Tr069Path: "Device.X.NriBitLen", ValueType: "int", IsWritable: true},
					{ParamCode: "LTE_BTSNUM", Tr069Path: "Device.X.BtsNum", ValueType: "unsignedInt", IsWritable: true},
				},
			},
		},
	}

	_, err := f.Fanout(context.Background(), mmlTask)
	require.NoError(t, err)
	require.Len(t, stub.calls[0], 1)

	var got struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"values"`
	}
	require.NoError(t, json.Unmarshal(stub.calls[0][0].Params, &got))
	require.Len(t, got.Values, 2, "空值参数应被过滤")
}

func TestFanouter_BuildDeviceTaskRequests_MapsFailedRetryStrategy(t *testing.T) {
	t.Run("disabled disables device task retries", func(t *testing.T) {
		stub := &stubDeviceTaskCreator{}
		f := NewFanouter(stub, nil, nil, nil, zap.NewNop())

		mmlTask := &MMLTask{
			ID:          uuid.New(),
			DeviceSNs:   []string{"SN-A"},
			FailedRetry: false,
			Commands: []map[string]interface{}{
				{"command_code": "REBOOT", "rpc_method": "Reboot"},
			},
		}

		_, err := f.Fanout(context.Background(), mmlTask)
		require.NoError(t, err)
		require.Len(t, stub.calls, 1)
		require.Len(t, stub.calls[0], 1)
		require.NotNil(t, stub.calls[0][0].MaxRetries)
		assert.Equal(t, 0, *stub.calls[0][0].MaxRetries)
	})

	t.Run("enabled forwards retry count", func(t *testing.T) {
		stub := &stubDeviceTaskCreator{}
		f := NewFanouter(stub, nil, nil, nil, zap.NewNop())

		mmlTask := &MMLTask{
			ID:               uuid.New(),
			DeviceSNs:        []string{"SN-A"},
			FailedRetry:      true,
			FailedRetryCount: 5,
			Commands: []map[string]interface{}{
				{"command_code": "REBOOT", "rpc_method": "Reboot"},
			},
		}

		_, err := f.Fanout(context.Background(), mmlTask)
		require.NoError(t, err)
		require.Len(t, stub.calls, 1)
		require.Len(t, stub.calls[0], 1)
		require.NotNil(t, stub.calls[0][0].MaxRetries)
		assert.Equal(t, 5, *stub.calls[0][0].MaxRetries)
	})

	t.Run("forwards offline wait and retry interval", func(t *testing.T) {
		stub := &stubDeviceTaskCreator{}
		f := NewFanouter(stub, nil, nil, nil, zap.NewNop())

		mmlTask := &MMLTask{
			ID:                  uuid.New(),
			DeviceSNs:           []string{"SN-A"},
			OfflineRetry:        true,
			OfflineRetryWait:    7,
			FailedRetry:         true,
			FailedRetryCount:    4,
			FailedRetryInterval: 9,
			Commands: []map[string]interface{}{
				{"command_code": "REBOOT", "rpc_method": "Reboot"},
			},
		}

		_, err := f.Fanout(context.Background(), mmlTask)
		require.NoError(t, err)
		require.Len(t, stub.calls, 1)
		require.Len(t, stub.calls[0], 1)
		assert.Equal(t, 7*60, stub.calls[0][0].ExpiresIn)
		assert.Equal(t, 9*60, stub.calls[0][0].RetryIntervalSeconds)
	})
}

func TestFanouter_BuildDeviceTaskRequests_NoParamRefs_GetSkipped(t *testing.T) {
	stub := &stubDeviceTaskCreator{}
	f := NewFanouter(stub, nil, nil, nil, zap.NewNop())

	mmlTask := &MMLTask{
		ID:        uuid.New(),
		DeviceSNs: []string{"SN-A"},
		Commands: []map[string]interface{}{
			{
				"command_code":   "LST DEVICE_INFO",
				"rpc_method":     "GetParameterValues",
				"operation_type": "LST",
				"parameters":     map[string]interface{}{},
				// 故意不挂 param_refs：BuildTR069Params 应返回 ErrNoUsableParams,
				// Fanouter 跳过这条 command，保护 device_tasks 不被脏数据污染
			},
		},
	}

	n, err := f.Fanout(context.Background(), mmlTask)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "无法翻译的 command 应被跳过，0 device_tasks 创建")
}

func TestFanouter_ParamRefsFromEntry_JSONRoundtrip(t *testing.T) {
	// 模拟 mml_tasks.commands 经过 JSON 序列化后再读出来的形态
	original := []MMLParamRef{
		{ParamCode: "X", Tr069Path: "Device.X", ValueType: "string"},
	}
	bs, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded []interface{}
	require.NoError(t, json.Unmarshal(bs, &decoded))

	cmd := map[string]interface{}{"param_refs": decoded}
	got := paramRefsFromEntry(cmd)
	require.Len(t, got, 1)
	assert.Equal(t, "X", got[0].ParamCode)
	assert.Equal(t, "Device.X", got[0].Tr069Path)
}

func TestFanouter_ParamRefsFromEntry_TypedPath(t *testing.T) {
	original := []MMLParamRef{{ParamCode: "X", Tr069Path: "Device.X"}}
	cmd := map[string]interface{}{"param_refs": original}
	got := paramRefsFromEntry(cmd)
	require.Len(t, got, 1)
	assert.Equal(t, "X", got[0].ParamCode)
}

// 显式调用 task.TaskSourceMML 让 import 不变成 unused（fanout_test 主要靠
// stubDeviceTaskCreator 间接引用 task）。
var _ = task.TaskSourceMML
