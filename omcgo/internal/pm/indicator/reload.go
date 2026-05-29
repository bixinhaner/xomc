package indicator

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ReloadMode 是 POST /api/v1/indicators/import-directory?mode= 的合法值。
type ReloadMode string

const (
	// ReloadModeImport: 加法 UPSERT — 仅刷新 XML 中已有指标的 DB 行,
	// 不删除 XML 文件已移除但 DB 仍残留的"孤儿"指标(向后兼容,默认行为)。
	ReloadModeImport ReloadMode = "import"
	// ReloadModeReload: destructive 全量重载 — 触发 Loader 全量 UPSERT
	// (BEFORE UPDATE trigger 自动刷 updated_at),然后删除 updated_at <
	// 重载开始时刻 的孤儿指标 + 级联 formula/enabled 行。
	ReloadModeReload ReloadMode = "reload"
)

// ParseReloadMode 解析 ?mode= query 值;空串视为 import(向后兼容老接口)。
//
// 非空但非 import/reload 返错(供 handler 转 400)。
func ParseReloadMode(s string) (ReloadMode, error) {
	switch s {
	case "", string(ReloadModeImport):
		return ReloadModeImport, nil
	case string(ReloadModeReload):
		return ReloadModeReload, nil
	default:
		return "", fmt.Errorf("invalid mode %q: must be one of import|reload", s)
	}
}

// ReloadResult 是 PerformReloadWithOrphans 的返回值。
// Orphans 按 tech 拆分,便于前端 message.success 显示 "已重载 + 删 N+M+K 个孤儿"。
type ReloadResult struct {
	Mode    ReloadMode      `json:"mode"`
	Orphans map[string]int  `json:"orphans"` // tech → 主表删除行数(仅 reload 模式非空)
}

// PerformReloadWithOrphans 执行 mode=reload 的核心 orchestration:
//
//  1. 记录 start = time.Now()(用作孤儿截断时刻)
//  2. 调用 reloader.ReloadOne(LoaderName) — Loader 全量 UPSERT;
//     ON CONFLICT DO UPDATE 路径触发 BEFORE UPDATE trigger,所有命中指标的
//     updated_at >= start;未命中的(即 XML 中已删的孤儿)updated_at < start。
//  3. 三制式各调一次 repo.DeleteOrphansBefore(tech, start),累加返回。
//
// 任一步失败立即返,部分成功不视为成功(前端 message.error 显示具体原因)。
//
// 与 ReloadModeImport 的区别只在于"是否做 step 3";老 import-directory 端点
// 保持仅 step 2 行为兼容(import 模式)。
func PerformReloadWithOrphans(
	ctx context.Context,
	repo FileRepository,
	reloader Reloader,
	logger *zap.Logger,
) (ReloadResult, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if reloader == nil {
		return ReloadResult{}, fmt.Errorf("reloader not wired (dictloader registry nil)")
	}
	if repo == nil {
		return ReloadResult{}, fmt.Errorf("file repository not wired")
	}

	start := time.Now()
	if err := reloader.ReloadOne(ctx, LoaderName); err != nil {
		return ReloadResult{}, fmt.Errorf("reload loader %s: %w", LoaderName, err)
	}

	result := ReloadResult{
		Mode:    ReloadModeReload,
		Orphans: make(map[string]int, 3),
	}
	for _, tech := range []string{"enb", "gsm", "gnb"} {
		deleted, err := repo.DeleteOrphansBefore(ctx, tech, start)
		if err != nil {
			return result, fmt.Errorf("delete orphans (%s): %w", tech, err)
		}
		result.Orphans[tech] = deleted
	}

	logger.Info("audit: indicator reload + orphan cleanup",
		zap.String("audit_action", "indicator.reload.with_orphans"),
		zap.Time("start", start),
		zap.Int("orphans_enb", result.Orphans["enb"]),
		zap.Int("orphans_gsm", result.Orphans["gsm"]),
		zap.Int("orphans_gnb", result.Orphans["gnb"]))

	return result, nil
}
