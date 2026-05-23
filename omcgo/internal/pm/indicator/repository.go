package indicator

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/omcgo/omcgo/internal/core/model"
)

// GroupRepository manages indicator group tree persistence.
type GroupRepository interface {
	// List returns all groups for a device type.
	List(ctx context.Context, dt DeviceType) ([]*IndicatorGroup, error)
	// GetByID returns a single group by ID.
	GetByID(ctx context.Context, dt DeviceType, id string) (*IndicatorGroup, error)
	// Create inserts a new group.
	Create(ctx context.Context, dt DeviceType, group *IndicatorGroup) error
	// Update modifies a group.
	Update(ctx context.Context, dt DeviceType, id string, req *UpdateGroupRequest) error
	// Delete removes a group. If tx is non-nil, participates in the transaction.
	Delete(ctx context.Context, dt DeviceType, id string, tx pgx.Tx) error
	// CountIndicatorsByGroup returns a map of group_id -> indicator count.
	CountIndicatorsByGroup(ctx context.Context, dt DeviceType) (map[string]int64, error)
}

// IndicatorRepository manages indicator persistence.
type IndicatorRepository interface {
	// List returns a paginated list of indicators with enabled status and custom name.
	List(ctx context.Context, filter IndicatorListFilter) (*model.ListResponse[IndicatorListItem], error)
	// GetByID returns a single indicator by ID.
	GetByID(ctx context.Context, dt DeviceType, id string) (*PerfIndicator, error)
	// Create inserts a new indicator. If tx is non-nil, participates in the transaction.
	Create(ctx context.Context, dt DeviceType, indicator *PerfIndicator, tx pgx.Tx) error
	// Update modifies an indicator. If tx is non-nil, participates in the transaction.
	Update(ctx context.Context, dt DeviceType, id string, req *UpdateIndicatorRequest, tx pgx.Tx) error
	// Delete removes an indicator. If tx is non-nil, participates in the transaction.
	Delete(ctx context.Context, dt DeviceType, id string, tx pgx.Tx) error
	// DeleteByGroupID removes all indicators in a group. If tx is non-nil, participates in the transaction.
	DeleteByGroupID(ctx context.Context, dt DeviceType, groupID string, tx pgx.Tx) error
	// GetIDsByGroupID returns all indicator IDs in a group (for cascade delete).
	GetIDsByGroupID(ctx context.Context, dt DeviceType, groupID string) ([]string, error)
	// GetNextKPIID generates the next available KPI ID.
	GetNextKPIID(ctx context.Context, dt DeviceType, operatorCode string) (string, error)
	// GetNextCounterID generates the next available Counter ID.
	GetNextCounterID(ctx context.Context) (string, error)
	// ListByIDs returns indicators matching the given IDs (for formula validation).
	ListByIDs(ctx context.Context, dt DeviceType, ids []string) ([]*PerfIndicator, error)
	// ListAll returns all indicators matching the filter without pagination (for export).
	ListAll(ctx context.Context, filter IndicatorListFilter) ([]IndicatorListItem, error)
}

// PlatformFormulaRepository manages platform-indicator-formula relationships.
type PlatformFormulaRepository interface {
	// ListByIndicatorID returns all formulas for an indicator.
	ListByIndicatorID(ctx context.Context, dt DeviceType, indicatorID string) ([]*PlatformFormula, error)
	// BatchCreate inserts multiple formula records. If tx is non-nil, participates in the transaction.
	BatchCreate(ctx context.Context, dt DeviceType, formulas []*PlatformFormula, tx pgx.Tx) error
	// DeleteByIndicatorID removes all formulas for an indicator. If tx is non-nil, participates in the transaction.
	DeleteByIndicatorID(ctx context.Context, dt DeviceType, indicatorID string, tx pgx.Tx) error
	// DeleteByIndicatorIDs removes formulas for multiple indicators. If tx is non-nil, participates in the transaction.
	DeleteByIndicatorIDs(ctx context.Context, dt DeviceType, indicatorIDs []string, tx pgx.Tx) error
	// ListPlatformNames returns distinct platform names for a device type.
	ListPlatformNames(ctx context.Context, dt DeviceType) ([]string, error)
	// ListByPlatform returns all formula rows for a platform.
	// T-0164-P1: 给 pm/kpi/router 走"产品平台 → KPI 子集"路由用，按 platform_name 一次性
	// 拉全部 (indicator_id, formula)，调用方再用 indicatorRepo.ListByIDs 拼指标元数据。
	ListByPlatform(ctx context.Context, dt DeviceType, platformName string) ([]*PlatformFormula, error)
}

// EnabledIndicatorRepository manages enabled indicator status.
type EnabledIndicatorRepository interface {
	// List returns all enabled indicator IDs for an operator.
	List(ctx context.Context, dt DeviceType, operatorCode string) ([]string, error)
	// BatchCreate enables multiple indicators. If tx is non-nil, participates in the transaction.
	BatchCreate(ctx context.Context, dt DeviceType, operatorCode string, indicatorIDs []string, tx pgx.Tx) error
	// BatchDelete disables multiple indicators. If tx is non-nil, participates in the transaction.
	BatchDelete(ctx context.Context, dt DeviceType, operatorCode string, indicatorIDs []string, tx pgx.Tx) error
	// Exists checks if an indicator is enabled.
	Exists(ctx context.Context, dt DeviceType, operatorCode string, indicatorID string) (bool, error)
}

// TemplateRelRepository checks template associations (delete precondition).
type TemplateRelRepository interface {
	// ExistsByIndicatorID checks if any template references the indicator.
	ExistsByIndicatorID(ctx context.Context, indicatorID string) (bool, error)
}

// CustNameRepository manages custom indicator names.
type CustNameRepository interface {
	// Upsert creates or updates a custom name. If tx is non-nil, participates in the transaction.
	Upsert(ctx context.Context, custName *CustName, tx pgx.Tx) error
	// Get returns the custom name for an indicator.
	Get(ctx context.Context, operatorCode string, perfID string) (*CustName, error)
	// ListByOperator returns all custom names for an operator.
	ListByOperator(ctx context.Context, operatorCode string) ([]*CustName, error)
}

// IndicatorThresholdRepository queries indicator thresholds.
type IndicatorThresholdRepository interface {
	// ListByIndicatorID returns all thresholds for an indicator.
	ListByIndicatorID(ctx context.Context, indicatorID string) ([]*IndicatorThreshold, error)
	// ExistsByIndicatorID checks if any threshold exists for the indicator.
	ExistsByIndicatorID(ctx context.Context, indicatorID string) (bool, error)
}
