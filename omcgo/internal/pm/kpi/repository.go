package kpi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// KPIFilter defines query parameters for KPI value retrieval.
type KPIFilter struct {
	DeviceID   *uuid.UUID
	CellID     *string
	KPIName    *string
	Carrier    *model.CarrierCode
	Technology *model.Technology
	StartTime  time.Time
	EndTime    time.Time
	// VisibleGroups 是 #64 设备组数据权限的三态可见分组（nil=超管 / []=fail-closed / [g...]=限定）。
	// handler 解析调用者身份后注入，仓库层透传到 metrics.QueryRequest 按 device_sn 收口。
	VisibleGroups []uuid.UUID
	model.ListRequest
}

// KPIRepository defines the interface for KPI value persistence.
type KPIRepository interface {
	BatchInsert(ctx context.Context, values []model.KPIValue) error
	// ReplaceForRecompute 在单个事务内"原子替换"某 (设备 oui+sn, cell, 15min 窗口) 的 KPI 行：
	// 先删既有 KPI 行（metric_type='kpi'）再插 values。migration 000042 删 uq_pm_metrics_natural 后
	// 无 ON CONFLICT DO UPDATE 兜底，重算改用此法替代旧自然键 UPSERT 的"后写覆盖"。
	// 保证：① 原子——删与插同事务，插失败则删回滚，旧值不丢；② 并发安全——按 (oui,sn,cell,窗口)
	// 取 pg advisory xact 锁串行化同范围并发重算，杜绝交错产生重复/丢失。values 为空仅删（路由/
	// 公式变更后该窗口可能不再有 KPI）。
	ReplaceForRecompute(ctx context.Context, oui, deviceSN, cellID string, endTime time.Time, values []model.KPIValue) error
	Query(ctx context.Context, filter KPIFilter) (*model.ListResponse[model.KPIValue], error)
	ListDefinitions(ctx context.Context, carrier *model.CarrierCode, tech *model.Technology) ([]model.KPIDefinition, error)
	SyncDefinitions(ctx context.Context, defs []model.KPIDefinition) error
}
