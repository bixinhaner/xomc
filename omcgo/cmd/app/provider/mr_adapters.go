package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	mrtask "github.com/omcgo/omcgo/internal/mr/task"
	"github.com/omcgo/omcgo/internal/product"
	"go.uber.org/zap"
)

// mr_adapters.go — mr/task 模块在 app 进程的依赖适配。
//
// 与 mml_adapters.go / software_adapters.go 同款思路：把 cmd 层持有的全局
// registry（ProductRegistry / ParamRegistry）+ device.DeviceRepository 包装成
// mrtask 内部定义的小接口（DeviceContext / TranslatorResolver），让 mrtask
// 包不依赖 cmd 层 / 不反向依赖 provision 包。

// ---------- DeviceContext ----------

// mrDeviceContextAdapter 把 device.DeviceRepository 适配为 mrtask.DeviceContext。
// 与 mrtask.DeviceContext 接口一一对应，仅暴露 GetDevice 一个方法。
type mrDeviceContextAdapter struct {
	repo device.DeviceRepository
}

// NewMRDeviceContext 构造适配器。
func NewMRDeviceContext(repo device.DeviceRepository) mrtask.DeviceContext {
	return &mrDeviceContextAdapter{repo: repo}
}

func (a *mrDeviceContextAdapter) GetDevice(ctx context.Context, sn string) (*coremodel.Device, error) {
	if a.repo == nil {
		return nil, fmt.Errorf("mr-device-context: device repo not wired")
	}
	dev, err := a.repo.GetBySerialNumber(ctx, sn)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, fmt.Errorf("mr-device-context: device %s not found", sn)
	}
	return dev, nil
}

// ---------- TranslatorResolver ----------

// mrTranslatorResolverAdapter 把 ProductRegistry + ParamRegistry 适配为
// mrtask.TranslatorResolver。翻译链路与 mml/software 适配器完全一致
// （CLAUDE.md §5.3）：
//
//	productClass → ProductRegistry.MatchProductClass → product.id
//	→ ParamRegistry.Translator(productID, swVersion)
//
// fail-soft：任何环节失败都返回 (nil, false)，mrtask.dispatcher 检测到 nil
// 时降级到 standardPath 下发（与 provision orchestrator 一致策略）。
type mrTranslatorResolverAdapter struct {
	products *product.Registry
	params   *parammodel.Registry
	logger   *zap.Logger
}

// NewMRTranslatorResolver 构造适配器。products / params 任一为 nil → 永远返回
// (nil, false)（mr dispatcher 全程降级；不报错，让单进程 / 无字典部署能跑）。
func NewMRTranslatorResolver(products *product.Registry, params *parammodel.Registry, logger *zap.Logger) mrtask.TranslatorResolver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &mrTranslatorResolverAdapter{
		products: products,
		params:   params,
		logger:   logger.Named("mr-translator-adapter"),
	}
}

func (a *mrTranslatorResolverAdapter) ResolveForDevice(ctx context.Context, dev *coremodel.Device) (*parammodel.Translator, bool) {
	if a.products == nil || a.params == nil || dev == nil || dev.ProductClass == "" {
		return nil, false
	}
	matchRes, err := a.products.MatchProductClass(ctx, dev.ProductClass)
	if err != nil {
		if !errors.Is(err, product.ErrOrphan) {
			a.logger.Debug("MatchProductClass failed",
				zap.String("product_class", dev.ProductClass), zap.Error(err))
		}
		return nil, false
	}
	if matchRes == nil || matchRes.Product == nil {
		return nil, false
	}
	t, err := a.params.Translator(ctx, matchRes.Product.ID, dev.FirmwareVersion)
	if err != nil {
		a.logger.Debug("ParamRegistry.Translator failed",
			zap.String("product_class", dev.ProductClass),
			zap.String("sw_version", dev.FirmwareVersion),
			zap.Error(err))
		return nil, false
	}
	return t, true
}

// ---------- PlatformResolver ----------

// mrPlatformResolverAdapter 把"productClass → param_model.name"链路适配为
// mrtask.PlatformResolver：
//
//	productClass → ProductRegistry.MatchProductClass → product.id + product.ParamModelID
//	             → parammodel.GetParamModelByID(ParamModelID) → ParamModel.Name
//	             → 返回 name（mr/task 内部再校对允许列表 IsSupportedPlatform）
//
// 设计：解析职责与允许列表校对职责分离 — adapter 只负责"productClass → 名字"，
// mr/task 内部决定哪些名字属于"MR 支持平台"。这样允许列表的变更不用动 cmd 层。
type mrPlatformResolverAdapter struct {
	products  *product.Registry
	pmRepo    *parammodel.PgRepository
	logger    *zap.Logger
}

// NewMRPlatformResolver 构造适配器。products / pmRepo 任一为 nil → 永远返回 ""（unsupport）。
func NewMRPlatformResolver(products *product.Registry, pmRepo *parammodel.PgRepository, logger *zap.Logger) mrtask.PlatformResolver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &mrPlatformResolverAdapter{
		products: products,
		pmRepo:   pmRepo,
		logger:   logger.Named("mr-platform-adapter"),
	}
}

func (a *mrPlatformResolverAdapter) ResolvePlatform(ctx context.Context, productClass string) (string, error) {
	if a.products == nil || a.pmRepo == nil || productClass == "" {
		return "", nil
	}
	matchRes, err := a.products.MatchProductClass(ctx, productClass)
	if err != nil {
		if errors.Is(err, product.ErrOrphan) {
			// 设备未在产品中心注册 — 当作"未识别"而非错误
			return "", nil
		}
		return "", fmt.Errorf("MatchProductClass(%s): %w", productClass, err)
	}
	if matchRes == nil || matchRes.Product == nil {
		return "", nil
	}
	if matchRes.Product.ParamModelID == nil {
		// 产品在产品中心，但未挂 param_model — 视为未识别
		a.logger.Debug("product has no param_model_id; mr unsupport",
			zap.String("product_class", productClass),
			zap.String("product_name", matchRes.Product.Name))
		return "", nil
	}
	pm, err := a.pmRepo.GetParamModelByID(ctx, *matchRes.Product.ParamModelID)
	if err != nil {
		if errors.Is(err, parammodel.ErrNoParamModel) {
			return "", nil
		}
		return "", fmt.Errorf("GetParamModelByID(%s): %w", *matchRes.Product.ParamModelID, err)
	}
	return pm.Name, nil
}
