package indicator

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ReloadResult 是 PerformReloadWithOrphans 的返回值。
// Orphans 按 tech 拆分,便于前端 message.success 显示 "已重载 + 删 N+M+K 个孤儿"。
//
// 2026-06-03 用户决策:导入 XML / 重载 XML / 刷新缓存 三功能合并为单一"导入 XML",
// 上传端点内部固定走 destructive 重载(全量 + 删孤儿),不再有 import/reload 模式区分。
type ReloadResult struct {
	Orphans map[string]int `json:"orphans"` // tech → 主表删除行数
}

// PerformReloadWithOrphans 执行 destructive 全量重载的核心 orchestration:
//
//  1. 记录 start = time.Now()(用作孤儿截断时刻)
//  2. 调用 reloader.ReloadOne(LoaderName) — Loader 全量 UPSERT;
//     ON CONFLICT DO UPDATE 路径触发 BEFORE UPDATE trigger,所有命中指标的
//     updated_at >= start;未命中的(即 XML 中已删的孤儿)updated_at < start。
//  3. 三制式各调一次 repo.DeleteOrphansBefore(tech, start),累加返回。
//
// 任一步失败立即返,部分成功不视为成功(前端 message.error 显示具体原因)。
//
// 调用方:UploadXML 上传成功后串联此函数(写文件 → destructive 重载 → 刷新缓存)。
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
