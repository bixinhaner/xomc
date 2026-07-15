package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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
//
//	→ ParamRegistry.Translator(productID, swVersion) → Translator.ToPrivate(path)
//
// T-0168 激进路线：MatchProductClass 返 ErrOrphan 时**不**返 error，构造
// TranslationOutcome{ProductResolved=false, AggregateSource="orphan_passthrough"}
// 让上层不阻塞 fanout；并触发 Prometheus 告警 mml_path_translation_orphan_total。
type mmlPathTranslatorAdapter struct {
	products *product.Registry
	params   *parammodel.Registry
	metrics  *mml.FanoutMetrics
	logger   *zap.Logger
}

// NewMMLPathTranslator 构造适配器。若 products 或 params 为 nil，TranslateForDevice
// 返回错误（防止运行期空指针）。metrics 为 nil 时退化为匿名 Registry（单测可用）。
func NewMMLPathTranslator(products *product.Registry, params *parammodel.Registry, metrics *mml.FanoutMetrics, logger *zap.Logger) mml.PathTranslator {
	if logger == nil {
		logger = zap.NewNop()
	}
	if metrics == nil {
		metrics = mml.NewFanoutMetrics(nil)
	}
	return &mmlPathTranslatorAdapter{
		products: products,
		params:   params,
		metrics:  metrics,
		logger:   logger.Named("mml-translator-adapter"),
	}
}

// TranslateForDevice 实现 mml.PathTranslator 接口（T-0168 升级版）。
//
//	productClass → ProductRegistry.MatchProductClass → product.id
//	→ ParamRegistry.Translator(productID, swVersion) → ToPrivate
//
// 行为：
//   - product 命中 + 单条 path 未命中 mapping → Source="passthrough"（Private=Standard）
//   - product 命中 + path 命中 → Source=translator.Source()（discovered / default）
//   - **product 未命中（ErrOrphan）→ 全部 path Source="orphan_passthrough"**（激进路线），
//     同时触发 metrics.OrphanInc(productClass) + WARN log
//   - ParamRegistry 失败（ErrNoMapping / ErrNoParamModel）→ 全 passthrough（product 已识别）
//   - 参数模型已去激活 → 返回错误，禁止 MML 通过 passthrough 绕过模型总开关
//
// 仅在以下场景返 error：未注入（products/params==nil）、参数模型已去激活、
// Registry IO 错误（非 ErrOrphan）。
func (a *mmlPathTranslatorAdapter) TranslateForDevice(
	ctx context.Context,
	productClass, softwareVersion string,
	standardPaths []string,
) (*mml.TranslationOutcome, error) {
	if a.products == nil || a.params == nil {
		return nil, fmt.Errorf("mml-translator-adapter: ProductRegistry or ParamRegistry not wired")
	}

	matchRes, err := a.products.MatchProductClass(ctx, productClass)
	orphan := errors.Is(err, product.ErrOrphan) || (err == nil && (matchRes == nil || matchRes.Product == nil))
	if err != nil && !errors.Is(err, product.ErrOrphan) {
		return nil, fmt.Errorf("mml-translator-adapter: match product_class %s: %w", productClass, err)
	}

	if orphan {
		// T-0168 激进路线：productClass 未识别 → 整任务走 orphan_passthrough，不阻塞 fanout
		a.metrics.OrphanInc(productClass)
		a.logger.Warn("product_class unresolved, all paths orphan_passthrough",
			zap.String("product_class", productClass),
			zap.Int("path_count", len(standardPaths)))
		paths := make([]mml.TranslatedPath, 0, len(standardPaths))
		for _, p := range standardPaths {
			paths = append(paths, mml.TranslatedPath{Standard: p, Private: p, Source: "orphan_passthrough"})
		}
		return &mml.TranslationOutcome{
			Paths:           paths,
			ProductResolved: false,
			ProductID:       uuid.Nil,
			ProductClass:    productClass,
			AggregateSource: "orphan_passthrough",
		}, nil
	}

	translator, err := a.params.Translator(ctx, matchRes.Product.ID, softwareVersion)
	if err != nil {
		if errors.Is(err, parammodel.ErrInactiveParamModel) {
			return nil, fmt.Errorf("mml-translator-adapter: parameter model inactive for product_class %s: %w", productClass, err)
		}
		// ParamRegistry 失败：product 已识别，但 mapping 拿不到 → 全 passthrough
		a.logger.Warn("translator unavailable, all paths passthrough",
			zap.String("product_class", productClass),
			zap.String("software_version", softwareVersion),
			zap.Error(err))
		paths := make([]mml.TranslatedPath, 0, len(standardPaths))
		for _, p := range standardPaths {
			paths = append(paths, mml.TranslatedPath{Standard: p, Private: p, Source: "passthrough"})
		}
		return &mml.TranslationOutcome{
			Paths:           paths,
			ProductResolved: true,
			ProductID:       matchRes.Product.ID,
			ProductClass:    productClass,
			AggregateSource: "passthrough",
		}, nil
	}

	// T-0170: product 已识别 + Translator 成功，单 path miss 直接拒绝整 task
	// （CPE 必返 9005 — 让 OMC 在 fanout 前 422 拒绝比让用户等 30 秒 CPE 拒强）。
	// 收集 miss path 清单后统一返 ErrPathUnsupported。
	src := string(translator.Source())
	paths := make([]mml.TranslatedPath, 0, len(standardPaths))
	missPaths := make([]string, 0)
	for _, p := range standardPaths {
		r := translator.ToPrivate(p)
		if r.Found {
			paths = append(paths, mml.TranslatedPath{Standard: p, Private: r.Translated, Source: src})
		} else {
			missPaths = append(missPaths, p)
			paths = append(paths, mml.TranslatedPath{Standard: p, Private: p, Source: "passthrough"})
		}
	}
	if len(missPaths) > 0 {
		a.logger.Warn("path(s) not in param_mappings, rejecting task (T-0170)",
			zap.String("product_class", productClass),
			zap.String("param_model_id", matchRes.Product.ParamModelID.String()),
			zap.Strings("unsupported_paths", missPaths))
		return nil, &mml.ErrPathUnsupported{
			ProductClass: productClass,
			ParamModelID: matchRes.Product.ParamModelID.String(),
			Paths:        missPaths,
		}
	}
	// 全部命中 → 正常返回（T-0170 后 missPaths 非空已在上面提前返错，到这只剩纯命中）
	return &mml.TranslationOutcome{
		Paths:           paths,
		ProductResolved: true,
		ProductID:       matchRes.Product.ID,
		ProductClass:    productClass,
		AggregateSource: src, // 全命中 → 必为单一来源（discovered / default）
	}, nil
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

// resolveDeviceByKey 把 MML console "deviceKey"（SN 或 UUID 字符串）解析为
// *model.Device。
//
// 解析顺序：
//  1. 尝试 uuid.Parse(deviceKey) — 成功则按 UUID 路径走 DeviceRepo.GetByID
//     （未命中再退到 SN 兜底，因 SN 字面也可能是 UUID 形态字串）
//  2. SN 路径走 DeviceService.GetBySerialNumber — 享受 DeviceCache L1 命中
//     （前端 ConsoleDevice 入参仅 SN，是热路径）
//
// 未找到 → 返回 (nil, nil)（与原 SQL "OR + LIMIT 1" 未命中等价语义）。
// 调用方据此 silent skip 到全集 sub_field 行为。
func resolveDeviceByKey(ctx context.Context, c *Container, deviceKey string) (*model.Device, error) {
	if deviceKey == "" {
		return nil, nil
	}
	// 1) UUID 路径
	if id, err := uuid.Parse(deviceKey); err == nil && c.DeviceRepo != nil {
		dev, err := c.DeviceRepo.GetByID(ctx, id)
		if err == nil && dev != nil {
			return dev, nil
		}
		// UUID 路径未命中：可能 deviceKey 字面是 UUID 但实际是 SN（极少见），
		// 不立即失败，继续退到 SN 路径兜底。
	}
	// 2) SN 路径（cache-aware via DeviceService）
	if c.DeviceService == nil {
		return nil, fmt.Errorf("device service not wired")
	}
	return c.DeviceService.GetBySerialNumber(ctx, deviceKey)
}
