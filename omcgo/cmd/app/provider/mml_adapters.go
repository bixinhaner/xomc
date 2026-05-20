package provider

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/mml"
	"github.com/omcgo/omcgo/internal/product"
)

// ============================================================
// mml_adapters.go — R-8.4 / R-9.3 适配器
//
// 方案文档：docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md §6.6
//
// 职责：
//   - mmlPathTranslatorAdapter：把 parammodel.Registry + product.Registry 适配为 mml.PathTranslator
//     接口（消费者驱动接口，定义在 mml 包）
//   - mmlDeviceLookupAdapter：把 device.Service 适配为 mml.DeviceLookup
//     （现有接口签名 GetBySerialNumber → *model.Device 直接复用 fanout.DeviceLookup）
//
// 注入点：cmd/app/router 在 mml.Service 构造完成后调用
//   service.SetPathTranslator(NewMMLPathTranslator(c))
//   service.SetDeviceLookup(NewMMLDeviceLookup(c))
// ============================================================

// mmlPathTranslatorAdapter 把 ProductRegistry + ParamRegistry 适配为 mml.PathTranslator。
//
// 翻译链路：product_class → ProductRegistry.MatchProductClass → product.id
//   → ParamRegistry.Translator(productID, swVersion) → Translator.ToPrivate(path)
type mmlPathTranslatorAdapter struct {
	products *product.Registry
	params   *parammodel.Registry
	logger   *zap.Logger
}

// NewMMLPathTranslator 构造适配器。若 products 或 params 为 nil，TranslateForDevice
// 返回错误（防止运行期空指针）。
func NewMMLPathTranslator(products *product.Registry, params *parammodel.Registry, logger *zap.Logger) mml.PathTranslator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &mmlPathTranslatorAdapter{
		products: products,
		params:   params,
		logger:   logger.Named("mml-translator-adapter"),
	}
}

// TranslateForDevice 实现 mml.PathTranslator 接口。
//
//	productClass → ProductRegistry.MatchProductClass → product.id
//	→ ParamRegistry.Translator(productID, swVersion) → ToPrivate
//
// 未命中（Found=false）→ passthrough（Private = Standard）。
// product_class 无匹配 product → mml.ErrProductClassUnresolved。
// ParamRegistry 失败（ErrNoMapping / ErrNoParamModel）→ 全 passthrough。
func (a *mmlPathTranslatorAdapter) TranslateForDevice(
	ctx context.Context,
	productClass, softwareVersion string,
	standardPaths []string,
) ([]mml.TranslatedPath, error) {
	if a.products == nil || a.params == nil {
		return nil, fmt.Errorf("mml-translator-adapter: ProductRegistry or ParamRegistry not wired")
	}

	out := make([]mml.TranslatedPath, 0, len(standardPaths))

	matchRes, err := a.products.MatchProductClass(ctx, productClass)
	if err != nil {
		if errors.Is(err, product.ErrOrphan) {
			return nil, &mml.ErrProductClassUnresolved{
				ProductClass: productClass,
			}
		}
		return nil, fmt.Errorf("mml-translator-adapter: match product_class %s: %w", productClass, err)
	}
	if matchRes == nil || matchRes.Product == nil {
		return nil, &mml.ErrProductClassUnresolved{
			ProductClass: productClass,
		}
	}

	translator, err := a.params.Translator(ctx, matchRes.Product.ID, softwareVersion)
	if err != nil {
		// ParamRegistry 失败：所有路径走 passthrough；记 WARN 不阻塞执行
		a.logger.Warn("translator unavailable, all paths passthrough",
			zap.String("product_class", productClass),
			zap.String("software_version", softwareVersion),
			zap.Error(err))
		for _, p := range standardPaths {
			out = append(out, mml.TranslatedPath{Standard: p, Private: p, Source: "passthrough"})
		}
		return out, nil
	}

	src := string(translator.Source())
	for _, p := range standardPaths {
		r := translator.ToPrivate(p)
		if r.Found {
			out = append(out, mml.TranslatedPath{
				Standard: p,
				Private:  r.Translated,
				Source:   src,
			})
		} else {
			out = append(out, mml.TranslatedPath{
				Standard: p,
				Private:  p,
				Source:   "passthrough",
			})
		}
	}
	return out, nil
}

// mmlDeviceLookupAdapter 把 device.Service 适配为 mml.DeviceLookup 接口。
//
// mml.DeviceLookup 复用 fanout.go 已有的接口签名 GetBySerialNumber(ctx, sn) → *model.Device。
// device.DeviceService.GetBySerialNumber 已直接符合该签名，但 Service 暴露为方法不是接口，
// 这里包装为 struct 满足显式接口契约。
type mmlDeviceLookupAdapter struct {
	devSvc *device.DeviceService
}

// NewMMLDeviceLookup 构造适配器。
func NewMMLDeviceLookup(devSvc *device.DeviceService) mml.DeviceLookup {
	return &mmlDeviceLookupAdapter{devSvc: devSvc}
}

// GetBySerialNumber 实现 mml.DeviceLookup 接口（其实现位于 fanout.go）。
func (a *mmlDeviceLookupAdapter) GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error) {
	if a.devSvc == nil {
		return nil, fmt.Errorf("mml-device-lookup-adapter: device.Service not wired")
	}
	return a.devSvc.GetBySerialNumber(ctx, sn)
}
