// Package dashboard 实现 T-0164-P6 / G6 PM 性能查看仪表盘 + 面板模型 + 分享 + 派生 + 用户偏好。
//
// 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.6
// 实施 plan：docs/project/plan-T-0164-P6-frontend-dashboard.md
//
// 与 internal/dashboard（运营总览 — 设备/告警统计）不同：
//   - 表名前缀 pm_ 区分
//   - 业务域是"用户可定制 PM 性能仪表盘"，对应 wave-3 PM 模块改造
//
// 概念：
//   Dashboard {id, name, owner_id, shared_with[], technology, layout, panels[]}
//   Panel    {id, dashboard_id, panel_type, title, metric_paths[], granularity, dimension, ...}
//   UserPref {user_id, kpi_card_layout}
package dashboard

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Technology 顶层制式切换。
type Technology string

const (
	TechLTE Technology = "lte"
	TechNR  Technology = "nr"
	TechGSM Technology = "gsm"
)

// PanelType 5 种受支持图表类型。
type PanelType string

const (
	PanelKPICard   PanelType = "kpi_card"
	PanelLineChart PanelType = "line_chart"
	PanelBarChart  PanelType = "bar_chart"
	PanelTable     PanelType = "table"
	PanelGauge     PanelType = "gauge"
)

// Dimension 数据维度。
type Dimension string

const (
	DimensionDevice      Dimension = "device"
	DimensionDeviceGroup Dimension = "device_group"
)

// CompareMode 对比双模式。nil = 不对比。
type CompareMode string

const (
	CompareSameWindowOtherDevices CompareMode = "same_window_other_devices"
	ComparePreviousWindow         CompareMode = "previous_window"
)

// Dashboard 是 pm_dashboards 表的 Go 域模型。
type Dashboard struct {
	ID                uuid.UUID
	Name              string
	Description       string
	OwnerID           uuid.UUID
	SharedWith        []uuid.UUID
	ParentDashboardID *uuid.UUID
	Technology        Technology
	Layout            json.RawMessage // react-grid-layout 配置（前端透传）
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Panel 是 pm_panels 表的 Go 域模型。
type Panel struct {
	ID             uuid.UUID
	DashboardID    uuid.UUID
	PanelType      PanelType
	Title          string
	MetricPaths    []string
	Granularity    string
	Dimension      Dimension
	DeviceSNs      []string
	DeviceGroupIDs []uuid.UUID
	TimeRange      json.RawMessage // {start_offset, end_offset} or {absolute_start, absolute_end}
	CompareMode    *CompareMode
	AdhocTaskID    *uuid.UUID
	Config         json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UserPreferences 是 pm_user_dashboard_preferences 行。
//
// T-0164 收尾 G6-Gap-3 + G6-Gap-13：制式按主键拆分（每个用户在每种制式下持有独立偏好），
// 包含当前选中仪表盘 + 全局筛选条 + KPI 卡片 layout。
type UserPreferences struct {
	UserID              uuid.UUID
	Technology          Technology      // 制式主键的第二维（lte/nr/gsm）
	KPICardLayout       json.RawMessage // KPI 卡片 layout（按制式独立）
	CurrentDashboardID  *uuid.UUID      // 切回该制式时自动打开的仪表盘
	SharedFilters       json.RawMessage // 全局筛选条快照（时间窗 / 设备组 / 设备多选）
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ── Request DTOs ─────────────────────────────────────────────────────────

// CreateDashboardRequest 创建 dashboard 输入。
type CreateDashboardRequest struct {
	Name        string
	Description string
	Technology  Technology
	Layout      json.RawMessage // 可空，默认 '{"panels":[]}'
}

// UpdateDashboardRequest 更新 dashboard 输入（仅 owner）。
type UpdateDashboardRequest struct {
	Name        *string
	Description *string
	Technology  *Technology
	Layout      json.RawMessage
}

// CreatePanelRequest 创建 panel 输入。
type CreatePanelRequest struct {
	DashboardID    uuid.UUID
	PanelType      PanelType
	Title          string
	MetricPaths    []string
	Granularity    string
	Dimension      Dimension
	DeviceSNs      []string
	DeviceGroupIDs []uuid.UUID
	TimeRange      json.RawMessage
	CompareMode    *CompareMode
	AdhocTaskID    *uuid.UUID
	Config         json.RawMessage
}

// ForkRequest 派生新 dashboard 的输入。
type ForkRequest struct {
	SourceID uuid.UUID
	NewName  string
}

// ShareRequest / UnshareRequest 分享 ACL 操作。
type ShareRequest struct {
	DashboardID uuid.UUID
	UserIDs     []uuid.UUID
}

type UnshareRequest struct {
	DashboardID uuid.UUID
	UserID      uuid.UUID
}
