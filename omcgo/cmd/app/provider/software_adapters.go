package provider

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/software"
)

// software_adapters.go — software.ParamPathTranslator 适配器
//
// 与 mml_adapters.go NewMMLPathTranslator 同款翻译链路（CLAUDE.md §5.3）：
//   productClass → ProductRegistry.MatchProductClass → product.id
//   → ParamRegistry.Translator(productID, swVersion) → ToPrivate(standardPath)
//
// 注入点：cmd/app/provider/modules.go 在 ProductRegistry/ParamRegistry 就绪后
//   softwareService.SetParamPathTranslator(NewSoftwareParamPathTranslator(...))
//
// 没单独抽 helper 是因为两边接口签名 / 实现都几乎一模一样，但反向依赖
// （software → mml）不能引入，所以这里单独维护一份。后续若有第三方调用方
// 可以再考虑抽到 parammodel 包内做通用 helper。

type softwareParamPathTranslator struct {
	products *product.Registry
	params   *parammodel.Registry
	logger   *zap.Logger
}

// NewSoftwareParamPathTranslator 构造适配器；products / params 为 nil 时
// TranslateForDevice 返回 error（防止运行期空指针）。
func NewSoftwareParamPathTranslator(products *product.Registry, params *parammodel.Registry, logger *zap.Logger) software.ParamPathTranslator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &softwareParamPathTranslator{
		products: products,
		params:   params,
		logger:   logger.Named("software-translator-adapter"),
	}
}

// TranslateForDevice 实现 software.ParamPathTranslator 接口。未命中 → passthrough。
// product_class 解析失败 → 返回 error；ParamRegistry 失败 → 整批 passthrough（不阻塞执行）。
func (a *softwareParamPathTranslator) TranslateForDevice(
	ctx context.Context,
	productClass, softwareVersion string,
	standardPaths []string,
) ([]software.TranslatedParamPath, error) {
	if a.products == nil || a.params == nil {
		return nil, fmt.Errorf("software-translator-adapter: ProductRegistry or ParamRegistry not wired")
	}

	out := make([]software.TranslatedParamPath, 0, len(standardPaths))

	matchRes, err := a.products.MatchProductClass(ctx, productClass)
	if err != nil {
		if errors.Is(err, product.ErrOrphan) {
			return nil, fmt.Errorf("software-translator-adapter: product_class %q has no matching product", productClass)
		}
		return nil, fmt.Errorf("software-translator-adapter: match product_class %s: %w", productClass, err)
	}
	if matchRes == nil || matchRes.Product == nil {
		return nil, fmt.Errorf("software-translator-adapter: product_class %q has no matching product", productClass)
	}

	translator, err := a.params.Translator(ctx, matchRes.Product.ID, softwareVersion)
	if err != nil {
		a.logger.Warn("translator unavailable, all paths passthrough",
			zap.String("product_class", productClass),
			zap.String("software_version", softwareVersion),
			zap.Error(err))
		for _, p := range standardPaths {
			out = append(out, software.TranslatedParamPath{Standard: p, Private: p, Source: "passthrough"})
		}
		return out, nil
	}

	src := string(translator.Source())
	for _, p := range standardPaths {
		r := translator.ToPrivate(p)
		if r.Found {
			out = append(out, software.TranslatedParamPath{
				Standard: p,
				Private:  r.Translated,
				Source:   src,
			})
		} else {
			out = append(out, software.TranslatedParamPath{
				Standard: p,
				Private:  p,
				Source:   "passthrough",
			})
		}
	}
	return out, nil
}
