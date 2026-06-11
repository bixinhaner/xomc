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
	Query(ctx context.Context, filter KPIFilter) (*model.ListResponse[model.KPIValue], error)
	ListDefinitions(ctx context.Context, carrier *model.CarrierCode, tech *model.Technology) ([]model.KPIDefinition, error)
	SyncDefinitions(ctx context.Context, defs []model.KPIDefinition) error
}
