package product

import (
	"context"

	"github.com/google/uuid"
)

// Repository 抽象 products / product_class_patterns 的读访问，便于
// ProductRegistry 在测试中替换为内存 Fake（设计 §4.3）。
//
// CRUD 写路径（Create / Update / Delete）交给 P3-01 的 handler 层落地；
// 本接口只覆盖 Registry 路由 + 详情查询所需的读路径。
type Repository interface {
	// ListActivePatterns 返回 is_active=true 的全集，按 sort_order ASC 排序。
	// 用于 Registry.Refresh 构建匹配快照。
	ListActivePatterns(ctx context.Context) ([]ProductClassPattern, error)

	// GetProductByID 按主键加载单个 product；不存在 → (nil, nil)。
	GetProductByID(ctx context.Context, id uuid.UUID) (*Product, error)

	// ListProducts 全集；用于 ValidateReferences 与启动期 sanity check。
	ListProducts(ctx context.Context) ([]*Product, error)

	// FetchIndicatorPlatformsByDeviceType 返回 perf_indicators_{enb|gsm|gnb} 表中
	// 已存在的 platform 集合（distinct）。用于校验 product.indicator_platform 软引用。
	// 实现侧负责按 deviceType 路由到正确的物理表名（enb/gsm/gnb）。
	FetchIndicatorPlatformsByDeviceType(ctx context.Context, deviceType string) (map[string]struct{}, error)

	// FetchAlarmNeTypes 返回 alarm_definitions 表中已存在的 ne_type 集合（distinct）。
	// 用于校验 product.alarm_ne_type 软引用。
	FetchAlarmNeTypes(ctx context.Context) (map[string]struct{}, error)
}
