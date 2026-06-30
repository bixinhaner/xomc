package acs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// mockInformPeriodLookup 创建用于测试的 lookup 函数
func mockInformPeriodLookup(data map[string]string) InformPeriodLookup {
	return func(ctx context.Context, category, key string) (string, bool) {
		v, ok := data[category+":"+key]
		return v, ok
	}
}

// mockTaskEnqueuer 实现 task.Enqueuer 接口用于测试
type mockTaskEnqueuer struct {
	tasks       []*task.CreateTaskRequest
	createError error
}

func (m *mockTaskEnqueuer) CreateTask(ctx context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	m.tasks = append(m.tasks, req)
	return &task.Task{ID: "test-task-id", DeviceSN: req.DeviceSN, Method: req.Method}, nil
}

func (m *mockTaskEnqueuer) GetQueueLength(ctx context.Context, deviceSN string) (int64, error) {
	return 0, nil
}

func TestInformPeriodPolicy_Enabled(t *testing.T) {
	logger := zap.NewNop()
	enqueuer := &mockTaskEnqueuer{}
	lookup := mockInformPeriodLookup(map[string]string{})

	tests := []struct {
		name     string
		policy   *InformPeriodPolicy
		expected bool
	}{
		{
			name:     "nil policy",
			policy:   nil,
			expected: false,
		},
		{
			name:     "nil lookup",
			policy:   NewInformPeriodPolicy(nil, enqueuer, logger),
			expected: false,
		},
		{
			name:     "nil taskService",
			policy:   NewInformPeriodPolicy(lookup, nil, logger),
			expected: false,
		},
		{
			name:     "all deps present",
			policy:   NewInformPeriodPolicy(lookup, enqueuer, logger),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.policy.Enabled())
		})
	}
}

func TestInformPeriodPolicy_ShouldTrigger(t *testing.T) {
	logger := zap.NewNop()
	enqueuer := &mockTaskEnqueuer{}
	lookup := mockInformPeriodLookup(map[string]string{})
	policy := NewInformPeriodPolicy(lookup, enqueuer, logger)

	tests := []struct {
		name       string
		eventCodes []string
		expected   bool
	}{
		{
			name:       "BOOTSTRAP event",
			eventCodes: []string{tr069.EventBootstrap},
			expected:   true,
		},
		{
			name:       "BOOT event",
			eventCodes: []string{tr069.EventBoot},
			expected:   true,
		},
		{
			name:       "PERIODIC event - should not trigger",
			eventCodes: []string{tr069.EventPeriodic},
			expected:   false,
		},
		{
			name:       "BOOTSTRAP with other events",
			eventCodes: []string{tr069.EventBootstrap, tr069.EventValueChange},
			expected:   true,
		},
		{
			name:       "BOOT with PERIODIC",
			eventCodes: []string{tr069.EventPeriodic, tr069.EventBoot},
			expected:   true,
		},
		{
			name:       "empty events",
			eventCodes: []string{},
			expected:   false,
		},
		{
			name:       "other events only",
			eventCodes: []string{tr069.EventValueChange, tr069.EventConnectionRequest},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, policy.ShouldTrigger(tt.eventCodes))
		})
	}
}

func TestInformPeriodPolicy_ShouldTrigger_DisabledPolicy(t *testing.T) {
	// nil policy should not trigger
	var nilPolicy *InformPeriodPolicy
	assert.False(t, nilPolicy.ShouldTrigger([]string{tr069.EventBootstrap}))

	// policy with nil lookup should not trigger
	logger := zap.NewNop()
	enqueuer := &mockTaskEnqueuer{}
	policy := NewInformPeriodPolicy(nil, enqueuer, logger)
	assert.False(t, policy.ShouldTrigger([]string{tr069.EventBootstrap}))
}

func TestInformPeriodPolicy_LoadConfig(t *testing.T) {
	logger := zap.NewNop()
	enqueuer := &mockTaskEnqueuer{}

	tests := []struct {
		name     string
		data     map[string]string
		expected InformPeriodConfig
	}{
		{
			name:     "empty config",
			data:     map[string]string{},
			expected: InformPeriodConfig{},
		},
		{
			name: "ENB enabled with period",
			data: map[string]string{
				"device:enbInformPeriodAdjustEnable": "true",
				"device:enbInformPeriod":             "60",
			},
			expected: InformPeriodConfig{
				ENBAdjustEnable: true,
				ENBPeriod:       60,
			},
		},
		{
			name: "CPE enabled with period",
			data: map[string]string{
				"device:cpeInformPeriodAdjustEnable": "1",
				"device:cpeInformPeriod":             "120",
			},
			expected: InformPeriodConfig{
				CPEAdjustEnable: true,
				CPEPeriod:       120,
			},
		},
		{
			name: "both ENB and CPE enabled",
			data: map[string]string{
				"device:enbInformPeriodAdjustEnable": "yes",
				"device:enbInformPeriod":             "61",
				"device:cpeInformPeriodAdjustEnable": "TRUE",
				"device:cpeInformPeriod":             "300",
			},
			expected: InformPeriodConfig{
				ENBAdjustEnable: true,
				ENBPeriod:       61,
				CPEAdjustEnable: true,
				CPEPeriod:       300,
			},
		},
		{
			name: "invalid period values",
			data: map[string]string{
				"device:enbInformPeriodAdjustEnable": "true",
				"device:enbInformPeriod":             "invalid",
				"device:cpeInformPeriod":             "-1",
			},
			expected: InformPeriodConfig{
				ENBAdjustEnable: true,
				ENBPeriod:       0, // invalid -> 0
				CPEPeriod:       0, // negative -> 0
			},
		},
		{
			name: "disabled with false string",
			data: map[string]string{
				"device:enbInformPeriodAdjustEnable": "false",
				"device:cpeInformPeriodAdjustEnable": "0",
			},
			expected: InformPeriodConfig{
				ENBAdjustEnable: false,
				CPEAdjustEnable: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := mockInformPeriodLookup(tt.data)
			policy := NewInformPeriodPolicy(lookup, enqueuer, logger)
			cfg := policy.LoadConfig(context.Background())
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestInformPeriodPolicy_LoadConfig_NilPolicy(t *testing.T) {
	var nilPolicy *InformPeriodPolicy
	cfg := nilPolicy.LoadConfig(context.Background())
	assert.Equal(t, InformPeriodConfig{}, cfg)
}

func TestInformPeriodPolicy_EnqueueGPVTask_ENBEnabled(t *testing.T) {
	logger := zap.NewNop()
	enqueuer := &mockTaskEnqueuer{}
	lookup := mockInformPeriodLookup(map[string]string{
		"device:enbInformPeriodAdjustEnable": "true",
		"device:enbInformPeriod":             "61",
	})
	policy := NewInformPeriodPolicy(lookup, enqueuer, logger)

	err := policy.EnqueueGPVTask(context.Background(), "SN001", "FAP/mBS31001/CA")
	require.NoError(t, err)

	// 验证任务已入队
	require.Len(t, enqueuer.tasks, 1)
	req := enqueuer.tasks[0]
	assert.Equal(t, "SN001", req.DeviceSN)
	assert.Equal(t, "GetParameterValues", req.Method)
	assert.Equal(t, task.TaskSourceSystem, req.Source)
	assert.Contains(t, req.Description, informPeriodGPVDescription)

	// 验证 params —— ENB 设备使用 TR-181 单路径
	var params GPVParams
	err = json.Unmarshal(req.Params, &params)
	require.NoError(t, err)
	assert.Len(t, params.Names, 1)
	assert.Equal(t, "Device.ManagementServer.PeriodicInformInterval", params.Names[0])
}

func TestInformPeriodPolicy_EnqueueGPVTask_CPEEnabled(t *testing.T) {
	logger := zap.NewNop()
	enqueuer := &mockTaskEnqueuer{}
	lookup := mockInformPeriodLookup(map[string]string{
		"device:cpeInformPeriodAdjustEnable": "true",
		"device:cpeInformPeriod":             "120",
	})
	policy := NewInformPeriodPolicy(lookup, enqueuer, logger)

	// CPE 设备（product_class 含 "cpe"）
	err := policy.EnqueueGPVTask(context.Background(), "SN002", "HomeCPE/Model1")
	require.NoError(t, err)

	require.Len(t, enqueuer.tasks, 1)
	assert.Equal(t, "SN002", enqueuer.tasks[0].DeviceSN)
}

func TestInformPeriodPolicy_EnqueueGPVTask_DisabledForDeviceType(t *testing.T) {
	logger := zap.NewNop()
	enqueuer := &mockTaskEnqueuer{}

	// 只启用 CPE，不启用 ENB
	lookup := mockInformPeriodLookup(map[string]string{
		"device:cpeInformPeriodAdjustEnable": "true",
		"device:cpeInformPeriod":             "120",
		"device:enbInformPeriodAdjustEnable": "false",
	})
	policy := NewInformPeriodPolicy(lookup, enqueuer, logger)

	// ENB 设备应该不入队
	err := policy.EnqueueGPVTask(context.Background(), "SN003", "FAP/mBS31001/CA")
	require.NoError(t, err)
	assert.Empty(t, enqueuer.tasks)

	// CPE 设备应该入队
	err = policy.EnqueueGPVTask(context.Background(), "SN004", "IndoorCPE")
	require.NoError(t, err)
	assert.Len(t, enqueuer.tasks, 1)
}

func TestInformPeriodPolicy_EnqueueGPVTask_DisabledPolicy(t *testing.T) {
	// nil policy should not enqueue
	var nilPolicy *InformPeriodPolicy
	err := nilPolicy.EnqueueGPVTask(context.Background(), "SN001", "FAP")
	assert.NoError(t, err)
}

func TestIsCPEClassForInformPeriod(t *testing.T) {
	tests := []struct {
		productClass string
		expected     bool
	}{
		{"FAP/mBS31001/CA", false},
		{"HomeCPE/Model1", true},
		{"CPE-5G", true},
		{"IndoorSmallCell", true},
		{"ResidentialGateway", true},
		{"MacroBase", false},
		{"eNodeB", false},
		{"home-gateway", true}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.productClass, func(t *testing.T) {
			assert.Equal(t, tt.expected, isCPEClassForInformPeriod(tt.productClass))
		})
	}
}

func TestParseBoolString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"invalid", false},
		{"  true  ", true}, // with spaces
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseBoolString(tt.input))
		})
	}
}
