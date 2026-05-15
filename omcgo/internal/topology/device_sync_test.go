package topology

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

// TestDeviceSyncServiceConfig_Defaults 测试默认配置。
func TestDeviceSyncServiceConfig_Defaults(t *testing.T) {
	config := DefaultDeviceSyncServiceConfig()

	assert.True(t, config.Enabled, "默认应启用同步")
	assert.True(t, config.InitialSync, "默认应执行启动时同步")
	assert.Equal(t, 1*time.Hour, config.FallbackInterval, "默认兜底间隔应为 1 小时")
	assert.Equal(t, 10*time.Second, config.InitialSyncDelay, "默认启动延迟应为 10 秒")
	assert.Equal(t, 100, config.BatchSize, "默认批量大小应为 100")
}

// TestDeviceSyncService_SetConfig 测试配置设置。
func TestDeviceSyncService_SetConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))

	customConfig := DeviceSyncServiceConfig{
		Enabled:          false,
		InitialSync:      false,
		FallbackInterval: 30 * time.Minute,
		InitialSyncDelay: 5 * time.Second,
		BatchSize:        50,
	}

	service.SetConfig(customConfig)

	assert.Equal(t, customConfig, service.config)
}

// TestDeviceSyncService_Start_Disabled 测试禁用状态下的启动。
func TestDeviceSyncService_Start_Disabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))
	service.SetConfig(DeviceSyncServiceConfig{Enabled: false})

	ctx := context.Background()
	err := service.Start(ctx)

	assert.NoError(t, err, "禁用状态下启动应不返回错误")
	assert.Nil(t, service.subscription, "禁用状态不应订阅事件")
	assert.Nil(t, service.cronTicker, "禁用状态不应启动定时任务")
}

// TestDeviceSyncService_Start_Enabled_NoEventBus 测试没有 EventBus 时的启动。
func TestDeviceSyncService_Start_Enabled_NoEventBus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, nil, zaptest.NewLogger(t))
	service.SetConfig(DeviceSyncServiceConfig{
		Enabled:          true,
		InitialSync:      false,
		FallbackInterval: 0, // 不启用定时同步
	})

	ctx := context.Background()
	err := service.Start(ctx)

	assert.NoError(t, err)
	assert.Nil(t, service.subscription, "没有 EventBus 不应订阅")
	assert.Nil(t, service.cronTicker, "未配置定时同步不应启动 ticker")
}

// TestDeviceSyncService_Start_SubscribeEvents 测试事件订阅。
func TestDeviceSyncService_Start_SubscribeEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)
	mockSub := NewMockSubscription(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))
	service.SetConfig(DeviceSyncServiceConfig{
		Enabled:          true,
		InitialSync:      false,
		FallbackInterval: 0,
	})

	// 期望订阅 device.registered 事件
	mockEventBus.EXPECT().QueueSubscribe(
		event.SubjectDeviceRegistered,
		"topology-device-sync",
		gomock.Any(),
	).Return(mockSub, nil)

	ctx := context.Background()
	err := service.Start(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, service.subscription, "应成功订阅事件")
}

// TestDeviceSyncService_Start_SubscribeError 测试订阅失败时的错误处理。
func TestDeviceSyncService_Start_SubscribeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))
	service.SetConfig(DeviceSyncServiceConfig{
		Enabled:          true,
		InitialSync:      false,
		FallbackInterval: 0,
	})

	// 期望订阅失败
	mockEventBus.EXPECT().QueueSubscribe(
		event.SubjectDeviceRegistered,
		"topology-device-sync",
		gomock.Any(),
	).Return(nil, errors.New("subscription failed"))

	ctx := context.Background()
	err := service.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscribe device events")
}

// TestDeviceSyncService_handleDeviceRegistered 测试设备注册事件处理。
func TestDeviceSyncService_handleDeviceRegistered(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPool := NewMockPgPool(ctrl)
	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)

	service := NewDeviceSyncService(mockPool, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))

	deviceID := uuid.New()
	serialNumber := "TEST-SN-001"

	// 创建测试事件
	payload := map[string]interface{}{
		"device_id":     deviceID,
		"serial_number": serialNumber,
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	testEvent := &testEvent{payload: payloadBytes}

	// Mock 设备查询
	mockPool.EXPECT().QueryRow(gomock.Any(), gomock.Any(), serialNumber).Return(
		&mockRow{
			values: []interface{}{
				deviceID, serialNumber, "TEST-OUI", "TestProduct", "TestManu",
				"TestModel", "cmcc", "lte", "active", "1.0",
				"192.168.1.1", "TestSite", "site-001", 39.9, 116.4,
				time.Now(), time.Now(), nil, nil,
			},
		},
		nil,
	)

	// Mock 查询现有节点（空列表）
	mockNodeRepo.EXPECT().ListAll(gomock.Any(), (*uuid.UUID)(nil), (*string)(nil), (*string)(nil)).
		Return([]TopoNode{}, nil)

	// Mock 创建节点
	mockNodeRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	ctx := context.Background()
	err = service.handleDeviceRegistered(ctx, testEvent)

	assert.NoError(t, err)
}

// TestDeviceSyncService_handleDeviceRegistered_InvalidPayload 测试无效负载处理。
func TestDeviceSyncService_handleDeviceRegistered_InvalidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))

	testEvent := &testEvent{payload: []byte("invalid json")}

	ctx := context.Background()
	err := service.handleDeviceRegistered(ctx, testEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode device.registered payload")
}

// TestDeviceSyncService_handleDeviceRegistered_MissingDeviceID 测试缺少 device_id 的负载。
func TestDeviceSyncService_handleDeviceRegistered_MissingDeviceID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))

	payload := map[string]interface{}{
		"serial_number": "TEST-SN-001",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	testEvent := &testEvent{payload: payloadBytes}

	ctx := context.Background()
	err = service.handleDeviceRegistered(ctx, testEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing device_id")
}

// TestDeviceSyncService_Stop 测试停止服务。
func TestDeviceSyncService_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)
	mockSub := NewMockSubscription(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))
	service.SetConfig(DeviceSyncServiceConfig{
		Enabled:          true,
		InitialSync:      false,
		FallbackInterval: 0,
	})

	mockEventBus.EXPECT().QueueSubscribe(
		event.SubjectDeviceRegistered,
		"topology-device-sync",
		gomock.Any(),
	).Return(mockSub, nil)

	ctx := context.Background()
	_ = service.Start(ctx)

	// Mock Unsubscribe
	mockSub.EXPECT().Unsubscribe().Return(nil)

	// 停止服务
	service.Stop()

	assert.Nil(t, service.subscription, "停止后订阅应为 nil")
	assert.Nil(t, service.cronTicker, "停止后 ticker 应为 nil")
}

// TestDeviceSyncService_inferNodeType 测试节点类型推断。
func TestDeviceSyncService_inferNodeType(t *testing.T) {
	tests := []struct {
		name     string
		device   model.Device
		expected string
	}{
		{
			name: "eNodeB - LTE 产品类",
			device: model.Device{
				ProductClass: "eNB-LTE",
				Technology:   model.TechLTE,
			},
			expected: "eNB",
		},
		{
			name: "gNodeB - NR 产品类",
			device: model.Device{
				ProductClass: "gNB-NR",
				Technology:   model.TechNR,
			},
			expected: "gNB",
		},
		{
			name: "CPE - 住宅型",
			device: model.Device{
				ProductClass: "CPE-Indoor",
				Technology:   model.TechLTE,
			},
			expected: "CPE",
		},
		{
			name: "CPE - 默认",
			device: model.Device{
				ProductClass: "Unknown",
				Technology:   "unknown",
			},
			expected: "CPE",
		},
		{
			name: "eGW - 网关",
			device: model.Device{
				ProductClass: "EGW-Core",
				Technology:   model.TechLTE,
			},
			expected: "eGW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferNodeType(tt.device)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDeviceSyncService_mapDeviceStatusToNodeStatus 测试设备状态到节点状态映射。
func TestDeviceSyncService_mapDeviceStatusToNodeStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   model.DeviceStatus
		expected NodeStatus
	}{
		{
			name:     "Active -> Online",
			status:   model.DeviceActive,
			expected: NodeOnline,
		},
		{
			name:     "Offline -> Offline",
			status:   model.DeviceOffline,
			expected: NodeOffline,
		},
		{
			name:     "Maintenance -> Maintenance",
			status:   model.DeviceMaintenance,
			expected: NodeMaintenance,
		},
		{
			name:     "Unknown -> Offline",
			status:   "unknown",
			expected: NodeOffline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapDeviceStatusToNodeStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDeviceSyncService_CalculateStatistics 测试统计计算。
func TestDeviceSyncService_CalculateStatistics(t *testing.T) {
	nodes := []TopoNode{
		{ID: uuid.New(), NodeType: "eNB", Status: NodeOnline},
		{ID: uuid.New(), NodeType: "eNB", Status: NodeOnline},
		{ID: uuid.New(), NodeType: "gNB", Status: NodeOffline},
		{ID: uuid.New(), NodeType: "CPE", Status: NodeAlarm},
		{ID: uuid.New(), NodeType: "CPE", Status: NodeMaintenance},
	}

	edges := []TopoEdge{
		{ID: uuid.New(), Status: EdgeActive},
		{ID: uuid.New(), Status: EdgeInactive},
		{ID: uuid.New(), Status: EdgeDegraded},
	}

	stats := CalculateStatistics(nodes, edges)

	assert.Equal(t, 5, stats.TotalNodes)
	assert.Equal(t, 3, stats.TotalEdges)
	assert.Equal(t, 2, stats.OnlineNodes)
	assert.Equal(t, 1, stats.OfflineNodes)
	assert.Equal(t, 1, stats.AlarmNodes)
	assert.Equal(t, 1, stats.MaintenanceNodes)
	assert.Equal(t, 1, stats.ActiveEdges)
	assert.Equal(t, 1, stats.InactiveEdges)
	assert.Equal(t, 1, stats.DegradedEdges)
	assert.Equal(t, 2, stats.NodeTypeCounts["eNB"])
	assert.Equal(t, 1, stats.NodeTypeCounts["gNB"])
	assert.Equal(t, 2, stats.NodeTypeCounts["CPE"])
}

// === Mock 辅助类型 ===

// testEvent 实现 event.Event 接口用于测试
type testEvent struct {
	payload []byte
}

func (e *testEvent) DecodePayload(v interface{}) error {
	return json.Unmarshal(e.payload, v)
}

// mockRow 实现 pgx.Row 用于测试
type mockRow struct {
	values []interface{}
}

func (r *mockRow) Scan(dest ...interface{}) error {
	if len(r.values) != len(dest) {
		return errors.New("column count mismatch")
	}
	for i, v := range r.values {
		switch d := dest[i].(type) {
		case *uuid.UUID:
			if val, ok := v.(uuid.UUID); ok {
				*d = val
			}
		case *string:
			if val, ok := v.(string); ok {
				*d = val
			}
		case *time.Time:
			if val, ok := v.(time.Time); ok {
				*d = val
			}
		case *float64:
			if val, ok := v.(float64); ok {
				*d = val
			}
		default:
			return errors.New("unsupported type")
		}
	}
	return nil
}

// MockPgPool 是 pgxpool.Pool 的 mock 接口
type MockPgPool struct {
	ctrl *gomock.Controller
}

func NewMockPgPool(ctrl *gomock.Controller) *MockPgPool {
	return &MockPgPool{ctrl: ctrl}
}

func (m *MockPgPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgxRow {
	return &mockRow{}
}

func (m *MockPgPool) Query(ctx context.Context, sql string, args ...interface{}) pgxRows {
	return &mockRows{}
}

type pgxRow interface {
	Scan(dest ...interface{}) error
}

type pgxRows interface {
	Close() error
	Next() bool
	Scan(...interface{}) error
}

type mockRows struct{}

func (m *mockRows) Close() error                     { return nil }
func (m *mockRows) Next() bool                       { return false }
func (m *mockRows) Scan(...interface{}) error       { return nil }

// PgPool 接口定义
type PgPool interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgxRow
	Query(ctx context.Context, sql string, args ...interface{}) pgxRows
}
