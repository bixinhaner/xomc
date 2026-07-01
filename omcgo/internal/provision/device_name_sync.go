// Package provision implements device provisioning and parameter synchronization.
//
// device_name_sync.go 实现设备名称同步功能（Issue #758）。
//
// 功能：在 Path B 参数同步完成后，检测 LMT 设备名称（HNBName）与网管 device_info.device_name
// 是否一致，按配置方向（lmt_to_omc / omc_to_lmt）自动同步或标记待人工确认。
//
// 触发时机：
//   - 设备上线（BOOTSTRAP/BOOT Inform）
//   - 周期巡检（PeriodicSyncer）
//   - 手动同步（POST /devices/:id/sync-params）
//
// 三条线路统一复用 Path B 后处理钩子。
package provision

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"go.uber.org/zap"
)

// HNBName 标准路径（取第一个小区 FAPService.1）。
// 4G LTE 和 5G NR 通用：设备私有路径由 Translator 自动翻译。
const hnbNameStandardPath = "Device.Services.FAPService.1.AccessMgmt.LTE.HNBName"

// 配置键名（category = "device"）
//
// 名称同步策略由单一枚举字段 nameSyncMode 描述（四值），页面与存储一一对应，
// 取代旧的 nameSettingEnable + nameSyncDirection + prompt 三字段组合（避免
// "开关关着但方向/提示仍有值" 这类矛盾状态）。
const (
	nameSyncConfigCategory = "device"
	nameSyncConfigMode     = "nameSyncMode" // 名称同步策略（四值枚举，见 NameSyncMode*）
)

// 名称同步策略枚举（nameSyncMode 的取值）。
const (
	NameSyncModeAutoLMTToOMC = "auto_lmt_to_omc" // 自动修改：LMT 名称覆盖网管
	NameSyncModeAutoOMCToLMT = "auto_omc_to_lmt" // 自动修改：网管名称下发到 LMT
	NameSyncModePrompt       = "prompt"           // 仅提示：标记待人工确认，不自动改（默认）
)

// 同步方向常量（内部使用，标准路径翻译等仍按方向区分）。
const (
	SyncDirectionLMTToOMC = "lmt_to_omc" // LMT 名称覆盖网管
	SyncDirectionOMCToLMT = "omc_to_lmt" // 网管名称下发到 LMT
)

// DeviceNameSyncConfig 设备名称同步配置（从 sys_configs 读取）。
type DeviceNameSyncConfig struct {
	Mode string // nameSyncMode 名称同步策略（四值枚举）
}

// NameSyncConfigLookup 消费者驱动接口：按 (category, key) 读取系统配置。
// 由 wiring 层提供 admin.PgSysConfigRepository.GetByKey 的适配器。
type NameSyncConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// DeviceNameSyncUpdater 消费者驱动接口：更新 device_info 的名称同步相关字段。
type DeviceNameSyncUpdater interface {
	// UpdateNameSyncFields 更新 device_info 的 name_sync_pending 和 lmt_device_name 字段。
	UpdateNameSyncFields(ctx context.Context, deviceID uuid.UUID, pending bool, lmtName string) error
	// UpdateDeviceName 更新 device_info.device_name（LMT→OMC 自动同步时使用）。
	UpdateDeviceName(ctx context.Context, deviceID uuid.UUID, name string) error
}

// DeviceNameSiteUpdater 消费者驱动接口：更新 devices.site_name（主表名称，列表展示用）。
// 与 DeviceNameSyncUpdater 分离，保持接口最小化原则。
type DeviceNameSiteUpdater interface {
	// UpdateSiteName 更新 devices.site_name，失败时调用方降级 Warn 不阻断。
	UpdateSiteName(ctx context.Context, id uuid.UUID, name string) error
}

// DeviceNameSPVSender 消费者驱动接口：下发 SPV 命令设置设备名称（OMC→LMT）。
type DeviceNameSPVSender interface {
	// SendNameToDevice 向设备下发 HNBName 参数。
	// path 是设备私有路径（已由 Translator 翻译）。
	SendNameToDevice(ctx context.Context, dev *model.Device, privatePath, name string) error
}

// DeviceNameSyncHook 是 Path B 同步完成后的设备名称同步钩子。
//
// 职责：
//   - 从 device_parameters 读取 HNBName（标准路径）
//   - 比对 device_info.device_name
//   - 按配置方向处理：自动同步或标记待确认
type DeviceNameSyncHook struct {
	configLookup NameSyncConfigLookup
	paramRepo    device.DeviceParameterRepository
	infoRepo     DeviceNameSyncUpdater
	siteUpdater  DeviceNameSiteUpdater // P1-1: 自动路径同步 devices.site_name（列表展示）
	spvSender    DeviceNameSPVSender
	infoGetter   DeviceInfoGetter
	translator   PathTranslator
	logger       *zap.Logger
}

// DeviceInfoGetter 消费者驱动接口：获取 device_info。
type DeviceInfoGetter interface {
	GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*device.DeviceInfo, error)
}

// PathTranslator 消费者驱动接口：标准路径 → 私有路径翻译。
type PathTranslator interface {
	ToPrivate(dev *model.Device, standardPath string) (string, bool)
}

// NewDeviceNameSyncHook 创建设备名称同步钩子。
func NewDeviceNameSyncHook(
	configLookup NameSyncConfigLookup,
	paramRepo device.DeviceParameterRepository,
	infoRepo DeviceNameSyncUpdater,
	infoGetter DeviceInfoGetter,
	logger *zap.Logger,
) *DeviceNameSyncHook {
	return &DeviceNameSyncHook{
		configLookup: configLookup,
		paramRepo:    paramRepo,
		infoRepo:     infoRepo,
		infoGetter:   infoGetter,
		logger:       logger,
	}
}

// SetSPVSender 注入 SPV 发送器（OMC→LMT 方向需要）。
func (h *DeviceNameSyncHook) SetSPVSender(sender DeviceNameSPVSender) *DeviceNameSyncHook {
	h.spvSender = sender
	return h
}

// SetTranslator 注入路径翻译器（OMC→LMT 方向需要）。
func (h *DeviceNameSyncHook) SetTranslator(t PathTranslator) *DeviceNameSyncHook {
	h.translator = t
	return h
}

// SetSiteUpdater 注入 devices.site_name 更新器（LMT→OMC 自动路径补写列表名称用）。
func (h *DeviceNameSyncHook) SetSiteUpdater(u DeviceNameSiteUpdater) *DeviceNameSyncHook {
	h.siteUpdater = u
	return h
}

// Execute 执行设备名称同步钩子。
//
// 调用时机：HandleSyncResultPathB 在 finalizePathBSync 后调用。
// 返回 nil 表示成功或静默跳过（配置关闭、设备不支持等）。
func (h *DeviceNameSyncHook) Execute(ctx context.Context, dev *model.Device) error {
	if dev == nil {
		return nil
	}

	// 1. 读取配置
	cfg := h.loadConfig(ctx)

	// 2. 从 device_parameters 读取 HNBName
	lmtName, err := h.getLMTDeviceName(ctx, dev.ID)
	if err != nil {
		h.logger.Warn("get LMT device name failed (skipped)",
			zap.String("device_sn", dev.SerialNumber),
			zap.Error(err))
		return nil // 静默跳过，不阻塞主流程
	}
	if lmtName == "" {
		h.logger.Debug("LMT device name empty, skipped",
			zap.String("device_sn", dev.SerialNumber))
		return nil
	}

	// 3. 获取 device_info 当前的 device_name
	info, err := h.infoGetter.GetByDeviceID(ctx, dev.ID)
	if err != nil || info == nil {
		h.logger.Warn("get device_info failed (skipped)",
			zap.String("device_sn", dev.SerialNumber),
			zap.Error(err))
		return nil
	}
	omcName := info.DeviceName

	// 4. 比对：一致则清除 pending 标记
	if strings.TrimSpace(lmtName) == strings.TrimSpace(omcName) {
		if info.NameSyncPending {
			if err := h.infoRepo.UpdateNameSyncFields(ctx, dev.ID, false, lmtName); err != nil {
				h.logger.Warn("clear name_sync_pending failed",
					zap.String("device_sn", dev.SerialNumber),
					zap.Error(err))
			}
		}
		h.logger.Debug("device name already in sync",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("name", lmtName))
		return nil
	}

	// 5. 不一致：按策略处理
	h.logger.Info("device name mismatch detected",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("lmt_name", lmtName),
		zap.String("omc_name", omcName),
		zap.String("mode", cfg.Mode))

	switch cfg.Mode {
	case NameSyncModeAutoLMTToOMC:
		// 自动修改：LMT 覆盖网管
		return h.handleLMTToOMC(ctx, dev, lmtName, omcName, false)
	case NameSyncModeAutoOMCToLMT:
		// 自动修改：网管下发到 LMT
		return h.handleOMCToLMT(ctx, dev, lmtName, omcName, false)
	case NameSyncModePrompt:
		// 仅提示：标记待人工确认（复用 LMT→OMC 分支的 prompt=true 路径，仅缓存 LMT 名 + 置 pending）
		return h.handleLMTToOMC(ctx, dev, lmtName, omcName, true)
	default:
		// 兜底：未知策略按不处理，防止误改
		h.logger.Warn("unknown name sync mode, skipped",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("mode", cfg.Mode))
		return nil
	}
}

// loadConfig 从 sys_configs 加载设备名称同步配置。
func (h *DeviceNameSyncHook) loadConfig(ctx context.Context) DeviceNameSyncConfig {
	// 默认 prompt：配置缺失时仅标记不一致，不自动改名（最保守安全底线）。
	cfg := DeviceNameSyncConfig{Mode: NameSyncModePrompt}

	if h.configLookup == nil {
		return cfg
	}

	if v, ok := h.configLookup(ctx, nameSyncConfigCategory, nameSyncConfigMode); ok {
		switch v {
		case NameSyncModeAutoLMTToOMC, NameSyncModeAutoOMCToLMT, NameSyncModePrompt:
			cfg.Mode = v
		}
	}

	return cfg
}

// hnbNamePathSuffix 是 HNBName 路径的固定后缀，用于多实例扫描。
const hnbNamePathSuffix = ".AccessMgmt.LTE.HNBName"

// hnbNamePathPrefix 是扫描所有 FAPService 实例时的路径前缀。
const hnbNamePathPrefix = "Device.Services.FAPService."

// getLMTDeviceName 从 device_parameters 读取 HNBName。
// 先尝试标准实例 FAPService.1；若未命中，扫描所有 FAPService.{i} 实例，
// 取第一个包含 HNBName 的值（P1-2：多实例兼容）。
func (h *DeviceNameSyncHook) getLMTDeviceName(ctx context.Context, deviceID uuid.UUID) (string, error) {
	// 快路径：直接查 .1
	param, err := h.paramRepo.GetByPath(ctx, deviceID, hnbNameStandardPath)
	if err != nil {
		return "", fmt.Errorf("query HNBName: %w", err)
	}
	if param != nil && param.ParameterValue != "" {
		return param.ParameterValue, nil
	}

	// 回退：扫描所有 FAPService 实例，找第一个含 HNBName 的路径
	params, err := h.paramRepo.GetByPathPrefix(ctx, deviceID, hnbNamePathPrefix)
	if err != nil {
		h.logger.Warn("fallback HNBName scan failed",
			zap.String("device_id", deviceID.String()),
			zap.Error(err))
		return "", nil
	}
	for _, p := range params {
		if strings.HasSuffix(p.ParameterPath, hnbNamePathSuffix) && p.ParameterValue != "" {
			h.logger.Debug("HNBName found via fallback scan",
				zap.String("device_id", deviceID.String()),
				zap.String("path", p.ParameterPath))
			return p.ParameterValue, nil
		}
	}
	return "", nil
}

// handleLMTToOMC 处理 LMT→OMC 方向同步。
func (h *DeviceNameSyncHook) handleLMTToOMC(ctx context.Context, dev *model.Device, lmtName, omcName string, prompt bool) error {
	if prompt {
		// 需要人工确认：标记 pending + 缓存 LMT 名称
		if err := h.infoRepo.UpdateNameSyncFields(ctx, dev.ID, true, lmtName); err != nil {
			return fmt.Errorf("set name_sync_pending: %w", err)
		}
		h.logger.Info("name sync pending (lmt_to_omc, prompt=true)",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("lmt_name", lmtName))
		return nil
	}

	// 自动同步：更新 device_info.device_name
	if err := h.infoRepo.UpdateDeviceName(ctx, dev.ID, lmtName); err != nil {
		return fmt.Errorf("update device_name: %w", err)
	}
	// P1-1: 同步写 devices.site_name，保证列表与详情展示一致
	if h.siteUpdater != nil {
		if err := h.siteUpdater.UpdateSiteName(ctx, dev.ID, lmtName); err != nil {
			h.logger.Warn("update devices.site_name failed after auto name sync",
				zap.String("device_sn", dev.SerialNumber),
				zap.Error(err))
			// 降级：不阻断，detail 页已更新
		}
	}
	// 清除 pending 标记并更新 lmt_device_name
	if err := h.infoRepo.UpdateNameSyncFields(ctx, dev.ID, false, lmtName); err != nil {
		h.logger.Warn("clear name_sync_pending after auto sync failed",
			zap.String("device_sn", dev.SerialNumber),
			zap.Error(err))
	}
	h.logger.Info("device name auto synced (lmt_to_omc)",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("new_name", lmtName),
		zap.String("old_name", omcName))
	return nil
}

// handleOMCToLMT 处理 OMC→LMT 方向同步。
func (h *DeviceNameSyncHook) handleOMCToLMT(ctx context.Context, dev *model.Device, lmtName, omcName string, prompt bool) error {
	if prompt {
		// 需要人工确认：标记 pending + 缓存 LMT 名称
		if err := h.infoRepo.UpdateNameSyncFields(ctx, dev.ID, true, lmtName); err != nil {
			return fmt.Errorf("set name_sync_pending: %w", err)
		}
		h.logger.Info("name sync pending (omc_to_lmt, prompt=true)",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("lmt_name", lmtName),
			zap.String("omc_name", omcName))
		return nil
	}

	// 自动下发：SPV 设置 HNBName
	if omcName == "" {
		h.logger.Debug("omc device_name empty, skip SPV",
			zap.String("device_sn", dev.SerialNumber))
		return nil
	}

	if h.spvSender == nil || h.translator == nil {
		h.logger.Warn("SPV sender or translator not configured, skip omc_to_lmt sync",
			zap.String("device_sn", dev.SerialNumber))
		// 降级到标记 pending
		if err := h.infoRepo.UpdateNameSyncFields(ctx, dev.ID, true, lmtName); err != nil {
			return fmt.Errorf("set name_sync_pending (fallback): %w", err)
		}
		return nil
	}

	// 翻译标准路径 → 私有路径
	privatePath, ok := h.translator.ToPrivate(dev, hnbNameStandardPath)
	if !ok {
		h.logger.Warn("translate HNBName path failed, fallback to pending",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("standard_path", hnbNameStandardPath))
		if err := h.infoRepo.UpdateNameSyncFields(ctx, dev.ID, true, lmtName); err != nil {
			return fmt.Errorf("set name_sync_pending (translate fail): %w", err)
		}
		return nil
	}

	// 下发 SPV
	if err := h.spvSender.SendNameToDevice(ctx, dev, privatePath, omcName); err != nil {
		h.logger.Warn("send HNBName SPV failed, fallback to pending",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("private_path", privatePath),
			zap.Error(err))
		if err := h.infoRepo.UpdateNameSyncFields(ctx, dev.ID, true, lmtName); err != nil {
			return fmt.Errorf("set name_sync_pending (spv fail): %w", err)
		}
		return nil
	}

	// 下发成功：清除 pending
	if err := h.infoRepo.UpdateNameSyncFields(ctx, dev.ID, false, lmtName); err != nil {
		h.logger.Warn("clear name_sync_pending after SPV success failed",
			zap.String("device_sn", dev.SerialNumber),
			zap.Error(err))
	}
	h.logger.Info("device name SPV sent (omc_to_lmt)",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("private_path", privatePath),
		zap.String("omc_name", omcName))
	return nil
}
