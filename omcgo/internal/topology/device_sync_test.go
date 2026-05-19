package topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)

	deviceID := uuid.New()
	serialNumber := "TEST-SN-001"

	// fake pool：getDeviceBySN 的 QueryRow 返回一行设备数据。
	// 列顺序与 device_sync.go getDeviceBySN 的 SELECT 完全一致（17 列）。
	pool := &fakeDBPool{row: &mockRow{values: []any{
		deviceID, serialNumber, "TEST-OUI", "TestProduct", "TestManu",
		"TestModel", "cmcc", "lte", "active", "1.0",
		"192.168.1.1", "TestSite", "site-001",
		time.Now(), time.Now(),
		(*time.Time)(nil), (*time.Time)(nil),
	}}}

	service := NewDeviceSyncService(pool, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))

	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]any{
		"device_id":     deviceID,
		"serial_number": serialNumber,
	})
	require.NoError(t, err)

	// 无现有节点 → syncDevice 走创建分支
	mockNodeRepo.EXPECT().ListAll(gomock.Any(), (*uuid.UUID)(nil), (*string)(nil), (*string)(nil)).
		Return([]TopoNode{}, nil)
	mockNodeRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	require.NoError(t, service.handleDeviceRegistered(context.Background(), evt))
}

// TestDeviceSyncService_handleDeviceRegistered_InvalidPayload 测试无效负载处理。
func TestDeviceSyncService_handleDeviceRegistered_InvalidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNodeRepo := NewMockTopoNodeRepository(ctrl)
	mockEdgeRepo := NewMockTopoEdgeRepository(ctrl)
	mockEventBus := NewMockEventBus(ctrl)

	service := NewDeviceSyncService(nil, mockNodeRepo, mockEdgeRepo, mockEventBus, zaptest.NewLogger(t))

	// Payload 为非法 JSON → DecodePayload 失败
	evt := event.Event{Subject: event.SubjectDeviceRegistered, Payload: json.RawMessage("invalid json")}

	err := service.handleDeviceRegistered(context.Background(), evt)

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

	// Payload 缺 device_id → handleDeviceRegistered 返回 missing device_id 错误
	evt, err := event.NewEvent(event.SubjectDeviceRegistered, map[string]any{
		"serial_number": "TEST-SN-001",
	})
	require.NoError(t, err)

	err = service.handleDeviceRegistered(context.Background(), evt)

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

// === 测试辅助类型 ===

// fakeDBPool 实现 topology.DBPool，供 device_sync 单测注入。
// 仅覆盖被测路径用到的方法：getDeviceBySN 走 QueryRow，Query 不被调用。
type fakeDBPool struct {
	row pgx.Row
}

func (f *fakeDBPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return f.row
}

func (f *fakeDBPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("fakeDBPool.Query not implemented")
}

// mockRow 实现 pgx.Row。Scan 用反射把 values 逐列写入 dest，
// 自动处理 string → 具名字符串类型（DeviceStatus / Technology 等）的转换，
// 以及 nil → 零值，避免为每个字段类型手写 type switch。
type mockRow struct {
	values []any
}

func (r *mockRow) Scan(dest ...any) error {
	if len(r.values) != len(dest) {
		return fmt.Errorf("mockRow.Scan: 列数不匹配 dest=%d values=%d", len(dest), len(r.values))
	}
	for i, v := range r.values {
		dptr := reflect.ValueOf(dest[i])
		if dptr.Kind() != reflect.Ptr || dptr.IsNil() {
			return fmt.Errorf("mockRow.Scan: dest[%d] 不是非空指针", i)
		}
		target := dptr.Elem()
		if v == nil {
			target.Set(reflect.Zero(target.Type()))
			continue
		}
		val := reflect.ValueOf(v)
		switch {
		case val.Type().AssignableTo(target.Type()):
			target.Set(val)
		case val.Type().ConvertibleTo(target.Type()):
			target.Set(val.Convert(target.Type()))
		default:
			return fmt.Errorf("mockRow.Scan: dest[%d] 类型 %s 无法由 %s 赋值", i, target.Type(), val.Type())
		}
	}
	return nil
}
