// license_params.go — 设备详情页 "License 参数" tab 后端服务。
//
// 业务背景：每个设备的标准参数树里有一个 "LICENSE" 子树（如
// `Device.Services.FAPService.{i}.FAPControl.LTE.LICENSE.*`），承载该设备
// 当前的授权 / 容量 / 过期 等参数。本文件提供 2 个能力：
//
//   1. ListLicenseParams — 取设备 device_parameters 表里所有 path 含 LICENSE
//      关键字的参数行，用 ParamRegistry Translator 反向把 privatePath 翻译为
//      OMC 标准 path，返回给前端展示
//   2. TriggerLicenseRefresh — 收到刷新按钮点击后异步下发 GPV，让 CPE 返回最
//      新 license 子树值；30s Redis 锁防抖
//
// 设计取舍：
//   - GET 端点不返回 privatePath（厂商实现细节，不跨 API 边界，user 决策）
//   - GET 数据源是 device_parameters（已 CPE 上报的 concrete instance），不是
//     param_mappings template — 这样能展示所有 Capacity.{i} 实例值；缺点：设备
//     从未上报时列表为空，需点"刷新"触发拉取
//   - 刷新走 SyncStarter.StartSync 的 partial prefix 模式，CPE 自动展开返回子树
//     所有参数（含 Capacity.1/2/3...）；{i} 替换为 1（FAPService 通常单实例）
package device

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
)

// licenseKeywordPrivate / licenseKeywordStandard — path 包含这两个父对象关键字之一就
// 视为 license 类参数（约定 §LICENSE 父对象命名规则）。标准关键字带前导 dot，
// 避免误命中 `X_LICENSE_AGREEMENT` 等非 license 子树；匹配时大小写不敏感，兼容
// `Device.FAP.License.*` 一类厂商路径。
const (
	licenseKeywordPrivate  = "X_COM_LICENSE."
	licenseKeywordStandard = ".LICENSE."
)

// licenseRefreshLockTTL — POST /refresh 的 Redis 防抖锁 TTL（PRD Q5）。
const licenseRefreshLockTTL = 30 * time.Second

// licenseRefreshLockKeyFmt — `device:license:refresh:{deviceID}`。
const licenseRefreshLockKeyFmt = "device:license:refresh:%s"

// licenseSearchKeywordLimit — paramRepo.SearchByKeyword 的上限保护（一台设备
// 几百个 license param 已经远超过常态，1000 足够留 buffer）。
const licenseSearchKeywordLimit = 1000

// LicenseParamView 是 GET 端点的单条响应记录。
//
// 刻意不带 privatePath — privatePath 是厂商实现细节，user 决策不跨 API 边界。
type LicenseParamView struct {
	StandardPath  string    `json:"standardPath"`
	Value         string    `json:"value"`
	DataType      string    `json:"dataType,omitempty"`
	Access        string    `json:"access,omitempty"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

// LicenseRefreshResult 是 POST /refresh 端点的响应。
//
// 用 path partial prefix 模式给 CPE，所以返的"prefixCount"是去重后的父对象
// 数量；CPE 收到 partial path 会自动展开返回子树所有参数。
type LicenseRefreshResult struct {
	TaskCount   int    `json:"taskCount"`   // 实际入队 GPV 任务批数
	PrefixCount int    `json:"prefixCount"` // 去重后 partial path 数量（GPV target 数）
	Reason      string `json:"reason"`      // sourceID 标签，固定 "manual_license_refresh"
}

// SyncStarter — license_params service 消费侧 narrow interface。
//
// 由 *provision.SyncService 满足；NewLicenseParamService caller 端在 modules.go
// wiring 时把 syncService 包成本接口注入（避免反向依赖 provision 包）。
type SyncStarter interface {
	StartSync(ctx context.Context, dev *model.Device, paramPaths []string, sourceID string) error
}

// LicenseParamService 设备 license 参数业务编排。
type LicenseParamService struct {
	deviceRepo      DeviceRepository
	paramRepo       DeviceParameterRepository
	productRegistry *product.Registry
	paramRegistry   *parammodel.Registry
	syncStarter     SyncStarter
	redis           redis.UniversalClient
	logger          *zap.Logger
}

// NewLicenseParamService 构造服务。所有依赖必填；调用方负责注入。
//
// redis 可为 nil（dev 环境无 Redis）：此时 TriggerLicenseRefresh 跳过防抖锁，
// 允许重复点击触发任务 — 与"无 Redis 即放过"的项目通用约定一致。
func NewLicenseParamService(
	deviceRepo DeviceRepository,
	paramRepo DeviceParameterRepository,
	productRegistry *product.Registry,
	paramRegistry *parammodel.Registry,
	syncStarter SyncStarter,
	redisClient redis.UniversalClient,
	logger *zap.Logger,
) *LicenseParamService {
	return &LicenseParamService{
		deviceRepo:      deviceRepo,
		paramRepo:       paramRepo,
		productRegistry: productRegistry,
		paramRegistry:   paramRegistry,
		syncStarter:     syncStarter,
		redis:           redisClient,
		logger:          logger.Named("license-params"),
	}
}

// ListLicenseParams 返回设备的 license 参数列表。
//
// 流程：
//  1. 加载 device（拿 productClass + firmwareVersion）
//  2. 走 ProductRegistry + ParamRegistry 拿 Translator（不可用时 fallback 到
//     裸 privatePath 直接返回 — degraded 模式）
//  3. paramRepo.SearchByKeyword(deviceID, "LICENSE", limit) — ILIKE %LICENSE%
//     已经能命中 `X_COM_LICENSE` 和 `LICENSE` 两种子串
//  4. 对每条用 Translator.ToStandard 反向翻译；Found=false 时跳过（不是 license
//     param 模型的 path，例如设备厂商自定义私有列）
//  5. 返回 standardPath 列表
func (s *LicenseParamService) ListLicenseParams(ctx context.Context, deviceID uuid.UUID) ([]LicenseParamView, error) {
	dev, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("load device: %w", err)
	}
	if dev == nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeDeviceNotFound, "device not found", commonerrors.ErrNotFound)
	}

	translator := s.resolveTranslator(ctx, dev)
	// translator 可能为 nil（产品未路由 / mapping 未加载）— 仍然继续，
	// 此时 standardPath 兜底用 privatePath（与前端约定 standardPath 为 string，
	// 不在 nil 时 panic）。

	params, err := s.paramRepo.SearchByKeyword(ctx, deviceID, "LICENSE", licenseSearchKeywordLimit)
	if err != nil {
		return nil, fmt.Errorf("search license params: %w", err)
	}

	views := make([]LicenseParamView, 0, len(params))
	for _, p := range params {
		// 二次过滤：SearchByKeyword 用宽松 ILIKE %LICENSE%；这里精确匹配
		// 期望的 LICENSE 子树关键字，避免 path 里偶然含 "LICENSE" 的非授权参数。
		if !isLicensePath(p.ParameterPath) {
			continue
		}
		standard := p.ParameterPath
		var dataType, access string
		if translator != nil {
			res := translator.ToStandard(p.ParameterPath)
			if res.Found {
				standard = res.Translated
				if res.Mapping != nil {
					dataType = res.Mapping.DataType
					access = res.Mapping.Access
				}
			}
		}
		// dataType fallback 用 device_parameters 行自身的 type 字段
		if dataType == "" {
			dataType = string(p.ParameterType)
		}
		views = append(views, LicenseParamView{
			StandardPath:  standard,
			Value:         p.ParameterValue,
			DataType:      dataType,
			Access:        access,
			LastUpdatedAt: p.LastUpdatedAt,
		})
	}
	return views, nil
}

// TriggerLicenseRefresh 异步下发 GPV 让 CPE 返回 license 子树最新值。
//
// 流程：
//  1. Redis SETNX 30s 锁（防止用户连点产生 N 个并发任务）
//  2. 取 device + ProductRegistry/ParamRegistry 拿映射集
//  3. 从 mapping 集合提取 license 子树 partial prefix（截到 `LICENSE.` 父对象）
//  4. {i} 替换为 1（FAPService 通常单实例）+ unique
//  5. 调 syncStarter.StartSync 分批入队 GPV
//
// 错误：
//   - device 不存在 → ErrCodeDeviceNotFound (404)
//   - 30s 内重复触发 → ErrCodeRuleTaskRunning 复用（resource conflict 语义）
//   - 产品未路由 / mapping 集合空 → 返 200 + prefixCount=0（前端体验：toast 提示
//     "无 license 参数可刷新"）
func (s *LicenseParamService) TriggerLicenseRefresh(ctx context.Context, deviceID uuid.UUID, operator string) (*LicenseRefreshResult, error) {
	dev, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("load device: %w", err)
	}
	if dev == nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeDeviceNotFound, "device not found", commonerrors.ErrNotFound)
	}

	lockKey := fmt.Sprintf(licenseRefreshLockKeyFmt, deviceID.String())
	lockHeld := false
	keepLock := false
	defer func() {
		if s.redis != nil && lockHeld && !keepLock {
			if err := s.redis.Del(ctx, lockKey).Err(); err != nil {
				s.logger.Warn("license refresh lock release failed",
					zap.String("device_id", deviceID.String()),
					zap.Error(err))
			}
		}
	}()

	// Redis 防抖锁（nil-safe — dev 环境无 Redis 时跳过）。
	if s.redis != nil {
		ok, lockErr := s.redis.SetNX(ctx, lockKey, operator, licenseRefreshLockTTL).Result()
		if lockErr != nil {
			s.logger.Warn("license refresh lock acquire failed (proceeding without lock)",
				zap.String("device_id", deviceID.String()), zap.Error(lockErr))
		} else if !ok {
			return nil, commonerrors.NewBusinessError(
				global.ErrCodeRuleTaskRunning,
				"license refresh already running for this device, try again in a few seconds",
				commonerrors.ErrAlreadyExists,
			)
		} else {
			lockHeld = true
		}
	}

	// 取 license partial prefixes（基于 ParamRegistry 的 default mapping，因为
	// discovered mapping 在设备首次上报 file11 后才有；首次刷新仍能基于
	// default mapping 找到 LICENSE 子树父对象路径）。
	prefixes, err := s.collectLicensePrefixes(ctx, dev)
	if err != nil {
		return nil, fmt.Errorf("collect license prefixes: %w", err)
	}
	if len(prefixes) == 0 {
		s.logger.Info("no license prefix found for device; nothing to refresh",
			zap.String("device_id", deviceID.String()),
			zap.String("product_class", dev.ProductClass))
		return &LicenseRefreshResult{Reason: "manual_license_refresh"}, nil
	}

	sourceID := newManualLicenseRefreshSourceID()
	if err := s.syncStarter.StartSync(ctx, dev, prefixes, sourceID); err != nil {
		return nil, fmt.Errorf("start GPV for license prefixes: %w", err)
	}
	keepLock = true
	s.logger.Info("license refresh GPV submitted",
		zap.String("device_id", deviceID.String()),
		zap.String("source_id", sourceID),
		zap.Int("prefix_count", len(prefixes)),
		zap.String("operator", operator))

	return &LicenseRefreshResult{
		PrefixCount: len(prefixes),
		TaskCount:   len(prefixes), // 实际批数由 syncStarter 决定；上层 UI 仅展示 prefix 数
		Reason:      "manual_license_refresh",
	}, nil
}

// resolveTranslator 走 ProductRegistry + ParamRegistry 取 Translator。
//
// 返回 nil 表示产品未路由 / mapping 集合空：caller 应 fallback 到裸 privatePath。
// 不返 error — Translator 缺失是常见 dev 数据状态（产品 dictionary 未导入），
// 列表展示仍能用裸 path 凑合，不应整个端点 500。
func (s *LicenseParamService) resolveTranslator(ctx context.Context, dev *model.Device) *parammodel.Translator {
	if s.productRegistry == nil || s.paramRegistry == nil {
		return nil
	}
	mr, err := s.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || mr == nil || mr.Product == nil {
		return nil
	}
	t, err := s.paramRegistry.Translator(ctx, mr.Product.ID, dev.FirmwareVersion)
	if err != nil {
		return nil
	}
	return t
}

// collectLicensePrefixes 从产品 mapping 抽出 license 子树父对象 partial prefix。
//
//	"Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}.RemainDays"
//	                                                            ↑ 截到 `.X_COM_LICENSE.`
//	→ "Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE."
//
// {i} → 1 是 FAPService 单实例假设；嵌套 {i}（如 Capacity.{i}）也替换为 1，但
// partial prefix 本身已截在 `LICENSE.` 之前，所以嵌套 {i} 不会出现在 prefix 里。
// CPE 收到 partial prefix 末尾带 dot，会自动展开返回该 object 下所有参数（含
// Capacity.1 / Capacity.2 / ...），不会漏 multi-instance。
func (s *LicenseParamService) collectLicensePrefixes(ctx context.Context, dev *model.Device) ([]string, error) {
	if s.productRegistry == nil || s.paramRegistry == nil {
		return nil, nil
	}
	mr, err := s.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil {
		if errors.Is(err, product.ErrOrphan) {
			return nil, nil // 产品未路由不是 service 错误，仅是 noop
		}
		return nil, fmt.Errorf("match product class: %w", err)
	}
	if mr == nil || mr.Product == nil {
		return nil, nil
	}
	t, err := s.paramRegistry.Translator(ctx, mr.Product.ID, dev.FirmwareVersion)
	if err != nil {
		return nil, fmt.Errorf("load translator: %w", err)
	}
	if t == nil {
		return nil, nil
	}

	uniq := map[string]struct{}{}
	for _, m := range t.Mappings() {
		// 仅参数行（entry_type='parameter'），与 PRD Q4 一致。
		if m.EntryType != "parameter" {
			continue
		}
		prefix := licensePartialPrefix(m.PrivatePath)
		if prefix == "" {
			continue
		}
		uniq[strings.ReplaceAll(prefix, "{i}", "1")] = struct{}{}
	}
	out := make([]string, 0, len(uniq))
	for k := range uniq {
		out = append(out, k)
	}
	return out, nil
}

// licensePartialPrefix 抽 path 的 license 父对象 partial prefix（含末尾 dot）。
//
// 同时识别私有 `X_COM_LICENSE.` 和 标准 `LICENSE.` 两种关键字，前者优先（更长
// 子串，更精确）。
func licensePartialPrefix(path string) string {
	if idx := indexFold(path, licenseKeywordPrivate); idx > 0 {
		return path[:idx+len(licenseKeywordPrivate)]
	}
	if idx := indexFold(path, licenseKeywordStandard); idx > 0 {
		return path[:idx+len(licenseKeywordStandard)]
	}
	return ""
}

// isLicensePath 判断 device_parameters 一行是否属于 license 子树。
//
// SearchByKeyword 用宽松 ILIKE %LICENSE% 拉初步集，这里精确卡父对象关键字
// 避免误命中（如某厂商私有定义里 path 含 "LICENSE_AGREEMENT" 但与 license
// 子树无关）。
func isLicensePath(path string) bool {
	return licensePartialPrefix(path) != ""
}

func newManualLicenseRefreshSourceID() string {
	return uuid.NewString()
}

func indexFold(s, substr string) int {
	return strings.Index(strings.ToUpper(s), strings.ToUpper(substr))
}
