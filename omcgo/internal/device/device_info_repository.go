package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// DeviceInfoRepository provides persistence for the device_info table.
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_device_info_repository_test.go -package=device . DeviceInfoRepository
type DeviceInfoRepository interface {
	// GetByDeviceID returns the extended info for a device.
	GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*DeviceInfo, error)

	// Create inserts a new device_info row.
	Create(ctx context.Context, info *DeviceInfo) error

	// UpdateManualFields updates only the operator-editable fields.
	UpdateManualFields(ctx context.Context, deviceID uuid.UUID, req UpdateDeviceInfoRequest, updater string) error

	// UpdateSyncFields updates fields synced from TR069 parameters or events.
	UpdateSyncFields(ctx context.Context, deviceID uuid.UUID, fields map[string]interface{}) error

	// UpdateNameSyncFields 更新设备名称同步相关字段（Issue #758）。
	// pending: name_sync_pending 标记（true=需人工确认）
	// lmtName: lmt_device_name 缓存的 LMT 设备名称
	UpdateNameSyncFields(ctx context.Context, deviceID uuid.UUID, pending bool, lmtName string) error

	// UpdateDeviceName 更新 device_info.device_name（LMT→OMC 自动同步时使用）。
	UpdateDeviceName(ctx context.Context, deviceID uuid.UUID, name string) error

	// GetTopologyAttributes 读取拓扑匹配相关的 device_info 列（LAC / TAC 等）
	// 的当前值，返回 column→value（仅含 NOT NULL 的列）。供 InfoSyncer 在 UPDATE
	// 之前 diff 新旧值用，决定是否要 publish device.attributes.changed 触发
	// GroupMatchEngine 重新匹配。设备不存在返回空 map（不报错）。
	GetTopologyAttributes(ctx context.Context, deviceID uuid.UUID) (map[string]string, error)

	// ListDevicesWithInfo returns a paginated list of devices joined with device_info.
	ListDevicesWithInfo(ctx context.Context, filter DeviceFilter) (*model.ListResponse[DeviceWithInfo], error)

	// GetByIDWithInfo returns a single device joined with device_info (and group).
	// 与 ListDevicesWithInfo 同样的 LEFT JOIN，供设备详情页使用——使详情与列表
	// 的 op_state（激活状态）及 device_info 扩展字段口径完全一致。
	// 设备不存在或已软删返回 (nil, nil)。
	GetByIDWithInfo(ctx context.Context, deviceID uuid.UUID) (*DeviceWithInfo, error)

	// T-0162: ComputeListStats 计算筛选条件下的全量统计（与当前页 items 解耦，
	// 修复 Q2 分析报告里"前端 fallback 用当前页 filter() 算 stats"的偏差）。
	// 在主 list 查询同样的 WHERE 子句下，跑 group-by 拿 lifecycle/online/alarm
	// 三维聚合。
	ComputeListStats(ctx context.Context, filter DeviceFilter) (*DeviceListStats, error)
}

// DeviceListStats 是设备列表筛选条件下的全量统计 payload（T-0162 D5）。
// 由 handler 填入 ListResponse[T].Stats 后回给前端，前端直读 stats 而不是从
// items 自行 filter().length 估算（避免 page-only 偏差）。
type DeviceListStats struct {
	// Total = 筛选条件下符合的设备总数（= ListResponse.Total，便于前端不依赖 items 长度）
	Total int64 `json:"total"`
	// ByLifecycle 按 lifecycle_state 分组计数。key 为 6 个生命周期值之一。
	ByLifecycle map[model.DeviceLifecycle]int64 `json:"by_lifecycle"`
	// OnlineCount = is_online=TRUE 的设备数；OfflineCount = Total - OnlineCount。
	// 不限定 lifecycle（"在线"语义是 is_online=TRUE，与生命周期解耦）。
	OnlineCount  int64 `json:"online_count"`
	OfflineCount int64 `json:"offline_count"`
	// CurrentUECount = 当前在线设备上报的接入 UE 数之和。离线设备的 ue_count
	// 可能是断连前遗留值，因此不计入首页“当前接入 UE 数”。
	CurrentUECount int64 `json:"current_ue_count"`
	// Alarmed = active 告警总条数（任何级别）。#361 已落地：来自
	// SUM(active_alarm_count) FROM alarms_active WHERE status<>'cleared'，
	// 受同一 applyDeviceFilters 约束，与列表行内告警数量加总一致。
	Alarmed int64 `json:"alarmed"`
}
