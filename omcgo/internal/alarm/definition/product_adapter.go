package definition

import (
	"context"

	"github.com/omcgo/omcgo/internal/product"
)

// ProductRegistryAdapter 把 *product.Registry 包装为 ProductResolver。
//
// 单向依赖：alarm/definition → product；保持 alarm/definition 接口纯净，product
// 适配在边界完成。生产 wiring（worker main.go）显式构造此适配器；测试可注入 stub
// ProductResolver 而不依赖 product 包。
type ProductRegistryAdapter struct {
	Registry *product.Registry
}

// ResolveByProductClass 通过 product.Registry.MatchProductClass 反查 product。
//
// 未命中（ErrOrphan 或 nil match）→ 返回 (nil, nil)，调用方按"product 不存在"语义处理。
func (a *ProductRegistryAdapter) ResolveByProductClass(ctx context.Context, productClass string) (*ProductSnapshot, error) {
	if a == nil || a.Registry == nil {
		return nil, nil
	}
	match, err := a.Registry.MatchProductClass(ctx, productClass)
	if err != nil {
		return nil, err
	}
	if match == nil || match.Product == nil {
		return nil, nil
	}
	return &ProductSnapshot{
		ID:                 match.Product.ID,
		Name:               match.Product.Name,
		EnableUnknownAlarm: match.Product.EnableUnknownAlarm,
	}, nil
}
