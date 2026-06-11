// Package backup — restore cross-version schema check (issue #70 task 3).
//
// 跨版本恢复风险：把 A 版本下导出的配置快照恢复到 B 版本的设备，可能因参数模型
// （ParamModel）差异导致部分参数不被识别 / 语义漂移 —— 设备可能"恢复成功"却跑在
// 一份并不完整或不兼容的配置上。本文件在下发前比对"快照来源设备软件版本"与"目标
// 设备当前软件版本"，给出显式策略：
//
//   - CrossVersionWarnAudit（默认）：版本不一致 → 记 warn 日志 + 写审计（不阻断）。
//   - CrossVersionBlock：版本不一致 → 拒绝该设备的恢复（整批语义由调用方决定）。
//   - CrossVersionAllow：完全放行（仅留 debug 日志），用于明确接受风险的场景。
//
// 数据现实：config_snapshots 表当前未持久化"快照捕获时的设备软件版本"，故
// source_version 取快照来源设备**当前**软件版本（by-snapshot 自恢复时即同一设备，
// 是最贴近的可得信号）。这是已知近似，后续若给快照补 captured_sw_version 列可
// 收紧为精确比对——接口与审计字段已按"源版本 vs 目标版本"建模，届时仅换数据源。
package backup

import (
	"context"
	"strings"

	"go.uber.org/zap"
)

// snapshotSourceVersion 返回快照"捕获时的设备软件版本"，作为跨版本检查的源版本。
//
// 数据源：config_snapshots.source_version（#70，由 promote / import 写入路径在抓取
// 快照时回填设备当前固件版本）。存量行 / 未知来源为空 —— CrossVersionChecker.Check
// 对空源版本一律放行不告警，避免对缺数据的设备误报。
func snapshotSourceVersion(snap *ConfigSnapshot) string {
	if snap == nil || snap.SourceVersion == nil {
		return ""
	}
	return *snap.SourceVersion
}

// CrossVersionStrategy 控制跨版本恢复的处置方式。
type CrossVersionStrategy string

const (
	// CrossVersionWarnAudit（默认）：不一致仅告警 + 审计，不阻断。
	CrossVersionWarnAudit CrossVersionStrategy = "warn_audit"
	// CrossVersionBlock：不一致则拒绝该设备恢复。
	CrossVersionBlock CrossVersionStrategy = "block"
	// CrossVersionAllow：放行（仅 debug 日志）。
	CrossVersionAllow CrossVersionStrategy = "allow"
)

// RestoreAuditSink 是跨版本检查写审计留痕的窄接口（nil-safe 由调用处保证）。
// 故意不依赖 internal/ops，避免 backup → ops 反向耦合；生产可注入一个把
// RestoreAuditEvent 转写到 ops_audit_logs 的适配器，测试注入内存桩。
type RestoreAuditSink interface {
	RecordRestoreAudit(ctx context.Context, evt RestoreAuditEvent)
}

// RestoreAuditEvent 是一条跨版本恢复审计记录。
type RestoreAuditEvent struct {
	RestoreID     string
	DeviceSN      string
	SourceVersion string
	TargetVersion string
	Strategy      CrossVersionStrategy
	Blocked       bool
	Message       string
}

// CrossVersionResult 是单设备跨版本检查结论。
type CrossVersionResult struct {
	Mismatch bool // 版本可比且不一致
	Blocked  bool // 因 Block 策略被拒
	// SourceVersion / TargetVersion 回填供调用方写 restore_tasks.source_version。
	SourceVersion string
	TargetVersion string
}

// CrossVersionChecker 执行版本比对 + 告警 + 审计。audit 可为 nil。
type CrossVersionChecker struct {
	strategy CrossVersionStrategy
	audit    RestoreAuditSink
	logger   *zap.Logger
}

// NewCrossVersionChecker 构造检查器。strategy 为空时取默认 warn_audit；audit 可空。
func NewCrossVersionChecker(strategy CrossVersionStrategy, audit RestoreAuditSink, logger *zap.Logger) *CrossVersionChecker {
	if strategy == "" {
		strategy = CrossVersionWarnAudit
	}
	return &CrossVersionChecker{
		strategy: strategy,
		audit:    audit,
		logger:   logger.Named("restore-crossver"),
	}
}

// Check 比对源/目标版本并按策略处置。两个版本任一为空（未知）→ 视为不可比，
// 不告警不阻断（返回 Mismatch=false）。返回的 Blocked 仅在 Block 策略 + 不一致
// 时为 true，调用方据此决定是否跳过该设备。
func (c *CrossVersionChecker) Check(ctx context.Context, restoreID, deviceSN, sourceVersion, targetVersion string) CrossVersionResult {
	sv := strings.TrimSpace(sourceVersion)
	tv := strings.TrimSpace(targetVersion)
	res := CrossVersionResult{SourceVersion: sv, TargetVersion: tv}

	// 版本未知 → 不可比，放行（避免对缺数据的设备误报）。
	if sv == "" || tv == "" {
		return res
	}
	if strings.EqualFold(sv, tv) {
		// 一致，无需处置。
		return res
	}

	res.Mismatch = true
	blocked := c.strategy == CrossVersionBlock
	res.Blocked = blocked
	msg := "restore source config version differs from target device version"

	switch c.strategy {
	case CrossVersionAllow:
		c.logger.Debug(msg+" (allowed)",
			zap.String("restore_id", restoreID), zap.String("device_sn", deviceSN),
			zap.String("source_version", sv), zap.String("target_version", tv))
	default: // warn_audit / block 都告警
		c.logger.Warn(msg,
			zap.String("restore_id", restoreID), zap.String("device_sn", deviceSN),
			zap.String("source_version", sv), zap.String("target_version", tv),
			zap.String("strategy", string(c.strategy)), zap.Bool("blocked", blocked))
	}

	if c.audit != nil && c.strategy != CrossVersionAllow {
		c.audit.RecordRestoreAudit(ctx, RestoreAuditEvent{
			RestoreID:     restoreID,
			DeviceSN:      deviceSN,
			SourceVersion: sv,
			TargetVersion: tv,
			Strategy:      c.strategy,
			Blocked:       blocked,
			Message:       msg,
		})
	}
	return res
}
