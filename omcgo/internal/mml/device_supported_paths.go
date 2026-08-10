package mml

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ============================================================
// device_supported_paths.go — T-0172 MML catalog 按产品过滤的"supported set 计算"
//
// 设计依据：docs/design (T-0172 接续 T-0098 P1-P5 + T-0170 sub_field 级过滤)
//
// 职责：根据 productClass 决定一个 product 的"被视为支持"的 standardPath 集合。
//
// 方案 X（已选 — backlog T-0172 见方案 Y 评估遗留项）：
//   - supported set = `param_mappings WHERE param_model_id = product.ParamModelID
//     AND is_active AND is_supported`（即 default 映射，不掺杂 discovered）
//   - 语义："这个产品族按规范应支持的全部 standardPath"
//   - 与 per-device A1 (T-0170 / Registry.GetByProduct) 区分开：那是 per-device 执行
//     时的硬拒；本 set 是 catalog 显示时的过滤
//
// 孤儿处理：productClass 未匹配任何 product → SupportedSet 仍返回
//（ProductResolved=false, Paths=空集），控制台调用方据此返回空树/空参数集，
// 禁止把未过滤的完整 catalog 当成该产品支持的命令。
// ============================================================

// ErrProductClassEmpty 由 ResolveByProductClass 在传入空 productClass 时返回。
// 调用方应直接跳过过滤（不调用本函数）。
var ErrProductClassEmpty = errors.New("mml: product_class is empty (no filter)")

// SupportedSet 是 catalog 过滤时一个 productClass 的"被视为支持"的 path 集合。
type SupportedSet struct {
	// ProductClass 入参回显，便于日志/调试。
	ProductClass string
	// ProductID 是 productClass 经 ProductRegistry 路由的产品 UUID。
	// 孤儿场景为 nil。
	ProductID *uuid.UUID
	// ProductTech 是匹配到的产品制式（如 lte / nr / gsm）。
	// 命令树过滤会用它屏蔽跨制式 catalog 命令；路径翻译仍只看 Paths。
	ProductTech string
	// ParamModelID 是 product 的参数模型 UUID。
	// 孤儿或产品无 paramModel 为 nil。
	ParamModelID *uuid.UUID
	// ProductResolved 标记 productClass 是否成功匹配到产品。
	// false → 孤儿；控制台过滤结果为空。
	ProductResolved bool
	// Paths 是 supported set 主体；O(1) 查询用 map[string]struct{}。
	// 孤儿场景为空 map（非 nil，方便 Contains 直接调用）。
	Paths map[string]struct{}
}

// Contains 判断单条 standardPath 是否在 supported set 内。
func (s *SupportedSet) Contains(path string) bool {
	if s == nil || s.Paths == nil {
		return false
	}
	_, ok := s.Paths[path]
	return ok
}

// SupportsObjectCollection 判断模型是否显式声明了 collection 下的实例对象。
// ADD/RMV 的 target_object 是集合路径（如 Device.Ethernet.Interface.），参数模型中的
// object 映射是实例模板（如 Device.Ethernet.Interface.{i}.）。叶子参数前缀命中不能
// 代表设备支持 AddObject/DeleteObject。
func (s *SupportedSet) SupportsObjectCollection(collection string) bool {
	if s == nil || s.Paths == nil || collection == "" {
		return false
	}
	_, ok := s.Paths[collection+"{i}."]
	return ok
}

// Size 返回 set 大小（用于日志/指标）。
func (s *SupportedSet) Size() int {
	if s == nil {
		return 0
	}
	return len(s.Paths)
}

// SupportedPathsRepository 抽象"按 productClass 取 supported set"的能力。
//
// 实现存活于 router/deps 层（cmd/app/router），内部聚合：
//   - product.Registry.MatchProductClass — productClass 路由到 product
//   - parammodel.Registry.GetByParamModel — 取 default param_mappings
//
// mml 包不直接 import product / parammodel 包，借此接口解耦（同 T-0170
// `resolveParamModelByDevice` 模式）。
type SupportedPathsRepository interface {
	ResolveByProductClass(ctx context.Context, productClass string) (*SupportedSet, error)
}

// SupportedPathsResolverFunc 让 DI 层用 closure 实现 SupportedPathsRepository。
type SupportedPathsResolverFunc func(ctx context.Context, productClass string) (*SupportedSet, error)

// ResolveByProductClass 满足 SupportedPathsRepository 接口。
func (f SupportedPathsResolverFunc) ResolveByProductClass(ctx context.Context, productClass string) (*SupportedSet, error) {
	return f(ctx, productClass)
}
