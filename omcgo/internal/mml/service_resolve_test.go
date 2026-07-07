package mml

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---- resolveCmdRepo：最小化 CommandRepository mock ----

// resolveCmdRepo 实现完整 CommandRepository 接口，仅 GetByCode 有逻辑。
type resolveCmdRepo struct {
	data map[string]*MMLCommand // code → command；不存在则返回 ErrNotFound
}

var errResolveNotFound = errors.New("not found")

func (r *resolveCmdRepo) GetByCode(_ context.Context, code string) (*MMLCommand, error) {
	if cmd, ok := r.data[code]; ok {
		return cmd, nil
	}
	return nil, errResolveNotFound
}

func (r *resolveCmdRepo) GetByID(_ context.Context, _ uuid.UUID) (*MMLCommand, error) {
	return nil, errResolveNotFound
}

func (r *resolveCmdRepo) List(_ context.Context, _ CommandFilter) (*model.ListResponse[MMLCommand], error) {
	return &model.ListResponse[MMLCommand]{}, nil
}

func (r *resolveCmdRepo) ListByGroupID(_ context.Context, _ uuid.UUID) ([]MMLCommand, error) {
	return nil, nil
}

// newResolveService 构造仅含 cmdRepo + logger 的最小 *Service。
func newResolveService(data map[string]*MMLCommand) *Service {
	return &Service{
		cmdRepo: &resolveCmdRepo{data: data},
		logger:  zap.NewNop(),
	}
}

// ---- BUG-01：resolveRPCMethods 分号修复测试 ----
//
// 回归：Closes #706
// 场景：脚本用户在 textarea 里写 "LST DEVICE_INFO;" (含尾部分号)；
// 前端直接把该字符串作为 command_code 提交；
// 修复前 resolveRPCMethods 查字典 miss → rpc_method 留空 → fanout 跳过。
func TestResolveRPCMethods_TrimsTrailingSemicolon(t *testing.T) {
	svc := newResolveService(map[string]*MMLCommand{
		"LST DEVICE_INFO": {
			CommandCode: "LST DEVICE_INFO",
			RPCMethod:   "GetParameterValues",
		},
	})

	// 模拟脚本行带分号（前端 splitScriptLines 未 trim 的情形）
	commands := []map[string]interface{}{
		{"command_code": "LST DEVICE_INFO;"},
	}
	result := svc.resolveRPCMethods(context.Background(), commands)

	require.Len(t, result, 1)
	assert.Equal(t, "GetParameterValues", result[0]["rpc_method"],
		"带分号的 command_code 应在 trim 后命中字典")
	assert.Equal(t, "LST DEVICE_INFO", result[0]["command_code"],
		"命中后 command_code 应规范化为字典存储形式（不含分号）")
}

// TestResolveRPCMethods_TrimsTrailingSemicolonWithSpaces 测试 "  LST DEVICE_INFO;  " 带尾部空格。
func TestResolveRPCMethods_TrimsTrailingSemicolonWithSpaces(t *testing.T) {
	svc := newResolveService(map[string]*MMLCommand{
		"LST DEVICE_INFO": {
			CommandCode: "LST DEVICE_INFO",
			RPCMethod:   "GetParameterValues",
		},
	})

	commands := []map[string]interface{}{
		{"command_code": "  LST DEVICE_INFO;  "},
	}
	result := svc.resolveRPCMethods(context.Background(), commands)
	require.Len(t, result, 1)
	assert.Equal(t, "GetParameterValues", result[0]["rpc_method"])
}

// TestResolveRPCMethods_NoSemicolon 确保无分号时行为不变。
func TestResolveRPCMethods_NoSemicolon(t *testing.T) {
	svc := newResolveService(map[string]*MMLCommand{
		"MOD DEVICE_INFO": {
			CommandCode: "MOD DEVICE_INFO",
			RPCMethod:   "SetParameterValues",
		},
	})

	commands := []map[string]interface{}{
		{"command_code": "MOD DEVICE_INFO"},
	}
	result := svc.resolveRPCMethods(context.Background(), commands)
	require.Len(t, result, 1)
	assert.Equal(t, "SetParameterValues", result[0]["rpc_method"])
}

func TestResolveRPCMethods_AddObjectFillsObjectNameFromTargetObject(t *testing.T) {
	svc := newResolveService(map[string]*MMLCommand{
		"ADD DRX_INITIAL_PARAM": {
			CommandCode:   "ADD DRX_INITIAL_PARAM",
			RPCMethod:     "AddObject",
			OperationType: "ADD",
			TargetObject:  "Device.Services.FAPService.CellConfig.LTE.RAN.MAC.DrxInitialParam.",
		},
	})

	commands := []map[string]interface{}{
		{"command_code": "ADD DRX_INITIAL_PARAM"},
	}
	result := svc.resolveRPCMethods(context.Background(), commands)

	require.Len(t, result, 1)
	assert.Equal(t, "AddObject", result[0]["rpc_method"])
	params, ok := result[0]["parameters"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t,
		"Device.Services.FAPService.CellConfig.LTE.RAN.MAC.DrxInitialParam.",
		params["object_name"])
}

func TestResolveRPCMethods_AddObjectKeepsExplicitObjectName(t *testing.T) {
	svc := newResolveService(map[string]*MMLCommand{
		"ADD DRX_INITIAL_PARAM": {
			CommandCode:   "ADD DRX_INITIAL_PARAM",
			RPCMethod:     "AddObject",
			OperationType: "ADD",
			TargetObject:  "Device.Default.",
		},
	})

	commands := []map[string]interface{}{
		{
			"command_code": "ADD DRX_INITIAL_PARAM",
			"parameters": map[string]interface{}{
				"object_name": "Device.Custom.",
			},
		},
	}
	result := svc.resolveRPCMethods(context.Background(), commands)

	require.Len(t, result, 1)
	params, ok := result[0]["parameters"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Device.Custom.", params["object_name"])
}

// TestResolveRPCMethods_AlreadyHasRpcMethod 确保已有 rpc_method 的条目不重复查库。
func TestResolveRPCMethods_AlreadyHasRpcMethod(t *testing.T) {
	// 故意让 repo 里无数据；若 resolveRPCMethods 再查库会返回 errResolveNotFound
	svc := newResolveService(map[string]*MMLCommand{})

	commands := []map[string]interface{}{
		{
			"command_code": "LST DEVICE_INFO",
			"rpc_method":   "GetParameterValues", // 已有，不应再查库
		},
	}
	result := svc.resolveRPCMethods(context.Background(), commands)
	require.Len(t, result, 1)
	assert.Equal(t, "GetParameterValues", result[0]["rpc_method"])
}

// TestResolveRPCMethods_UnknownCode_SkipsGracefully 未知命令码不应 panic，只是留空。
func TestResolveRPCMethods_UnknownCode_SkipsGracefully(t *testing.T) {
	svc := newResolveService(map[string]*MMLCommand{})

	commands := []map[string]interface{}{
		{"command_code": "UNKNOWN_CMD;"},
	}
	result := svc.resolveRPCMethods(context.Background(), commands)
	require.Len(t, result, 1)
	_, hasMethod := result[0]["rpc_method"]
	assert.False(t, hasMethod, "未知命令码不应设置 rpc_method")
}

// ---- BUG-06 smoke test ----
// 完整 ExecuteCommand 集成测试需要数据库，在 e2e_verify.sh 补充。
// 这里仅验证 uuid.Parse 基础路径不 panic（smoke）。
func TestScriptIDParsing_ValidUUID_NoError(t *testing.T) {
	sid := "50e2ba1d-d57a-41a8-a4ba-2624088b7a47"
	parsed, err := uuid.Parse(sid)
	require.NoError(t, err)
	assert.Equal(t, sid, parsed.String())
}
