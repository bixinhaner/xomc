package parammodel

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrNoMapping 表示按 productId/paramModelId 查不到任何映射（discovered 与 default 均空）。
//
// 调用方可据此决定行为：
//   - Path A 模板下发：跳过该参数，记 INFO 日志
//   - Path B 自动同步：直接放过（无可同步的 privatePath）
//
// 注意：与 ErrNoParamModel 的语义区别——ErrNoParamModel 表示 product 上根本未配置 paramModelID，
// 是产品配置错误；ErrNoMapping 表示配置正确但 XML/数据库中无对应记录。
var ErrNoMapping = errors.New("parammodel: no mapping found")

// ErrBuiltinMappingNotDeletable 表示尝试删除 source='builtin' 的映射(来自 XML,不可删);
// handler 据此返回 403 + ErrCodeParamMappingBuiltinNotDeletable。
var ErrBuiltinMappingNotDeletable = errors.New("parammodel: builtin mapping not deletable")

// ErrNoParamModel 表示 product.ParamModelID 为 nil（产品未绑定参数模型）。
// 通常是 products.xml 漏配或孤儿设备绑定了空 product。
var ErrNoParamModel = errors.New("parammodel: product has no paramModelID")

// ErrDuplicateStandardPath 表示同一 paramModel 下 standard_path 已存在；
// handler 据此返回 409，防止同一标准 PATH 被重复添加。
var ErrDuplicateStandardPath = errors.New("parammodel: standard_path already exists in this model")

// ErrProductGetterUnset 在 Registry 未注入 productGetter 但被调用 GetByProduct/Translator 时返回。
// 仅在测试/降级容器中可能出现；生产 provider 必须注入。
var ErrProductGetterUnset = errors.New("parammodel: productGetter not configured")

// Repository 抽象 ParamRegistry 所需的读访问。写路径（XML 加载、Intersect 写 discovered）
// 由 Loader（P1-06，已实现）和 IntersectService（P2-03，未来）各自负责，不在本接口范围。
type Repository interface {
	// ListMappingsByParamModel 返回某个 paramModel 的全量默认映射（is_active=true）。
	// 不存在 → 返回空切片 + nil err。
	ListMappingsByParamModel(ctx context.Context, paramModelID uuid.UUID) ([]ParamMapping, error)

	// ListDiscoveredMappings 返回某个 product+softwareVersion 的全量发现映射（is_active=true）。
	// 不存在 → 返回空切片 + nil err（用于在 cache 层做 negative 缓存）。
	ListDiscoveredMappings(ctx context.Context, productID uuid.UUID, swVersion string) ([]ParamMapping, error)
}
