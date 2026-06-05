package parammodel

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// worker BackupCleanup(PRD §9.6;三库 XML 导入重构改单目录)。
//
// 目标:周期清理 param-mappings 目录下的 .deleted.<ts> / .bak.<ts> / .tmp.<uuid> 残留,
// 默认保留 .deleted / .bak 30 天;.tmp 残留 1 小时即清(写盘中断兜底);
// 并清扫孤儿 sidecar(X.xml.custom 而 X.xml 已不存在 → 删 sidecar)。
//
// 设计决策:
//   - 时间从文件名 ts 解析,不从 os.Stat().ModTime()。
//     mtime 易被 rsync/cp -p 等改,文件名不可篡改(rename 是单步原子)。
//   - 单文件失败 continue 不阻塞后续清理(错误隔离)。
//   - cron 表达式由调用方控制;Run 是一次扫描的核函数。
//   - 启动期延迟一次 catch-up:防 worker 长期宕机后备份堆积,首次拉起补清扫。
//   - Prometheus 指标 parammodel_backup_cleanup_total{kind, result} 给 SLO 告警。
//
// 文件名约定:
//   <name>.xml.deleted.<14位ts>     ← DELETE handler 备份
//   <name>.xml.bak.<14位ts>         ← Upload overwrite 备份
//   <name>.xml.tmp.<uuid>           ← Upload 进行中的 tmp(rename 成功后已不存在)

// backupNameRe 匹配 .deleted/.bak + 14 位时间戳(yyyymmddHHMMSS)。
var backupNameRe = regexp.MustCompile(`\.(deleted|bak)\.(\d{14})$`)

// tmpNameRe 匹配 .tmp.<uuid> — uuid 不参与时间判定,改看 mtime。
var tmpNameRe = regexp.MustCompile(`\.tmp\.[A-Za-z0-9-]+$`)

const (
	// DefaultBackupRetentionDays 是 .deleted/.bak 文件保留天数(用户决策 3)。
	DefaultBackupRetentionDays = 30

	// TmpResidualMaxAge 是 .tmp 残留文件的最大留存(写盘中断 → 容器 OOM 等场景)。
	// 1 小时足以覆盖任何健康 Upload 周期,而避免长期占盘。
	TmpResidualMaxAge = time.Hour

	// DefaultBackupCleanupCron 是默认 cron 表达式(每天凌晨 3 点)。
	// 避开高峰,与 worker 其他 cron(PM 聚合等)错峰。
	DefaultBackupCleanupCron = "0 3 * * *"
)

// BackupCleanupMetrics 暴露清扫器指标(PRD §9.11 第 3 个 counter)。
//
// 标签:
//
//	kind   = "deleted" | "bak" | "tmp" | "sidecar" — 清理对象类别(sidecar=孤儿 .custom 标记)
//	result = "swept" | "error" | "skipped" — swept=成功删除,error=单文件失败,skipped=未到期/解析失败
type BackupCleanupMetrics struct {
	total *prometheus.CounterVec
}

// NewBackupCleanupMetrics 注册并返回指标集合;reg nil 用匿名 Registry(测试)。
func NewBackupCleanupMetrics(reg prometheus.Registerer) *BackupCleanupMetrics {
	m := &BackupCleanupMetrics{
		total: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "parammodel_backup_cleanup_total",
				Help: "Total parammodel custom-dir backup files swept by BackupCleanup cron.",
			},
			[]string{"kind", "result"},
		),
	}
	if reg != nil {
		reg.MustRegister(m.total)
	}
	return m
}

// observe 记录一次清扫结果。
func (m *BackupCleanupMetrics) observe(kind, result string) {
	if m == nil || m.total == nil {
		return
	}
	m.total.WithLabelValues(kind, result).Inc()
}

// BackupCleanup 实现 param-mappings 目录下备份文件 + 孤儿 sidecar 的周期清理(PRD §9.6)。
type BackupCleanup struct {
	dir       string
	maxAge    time.Duration
	tmpMaxAge time.Duration
	now       func() time.Time // 测试可注入
	metrics   *BackupCleanupMetrics
	log       *zap.Logger
}

// NewBackupCleanup 构造清扫器。dir 是 param-mappings 目录(builtin + custom XML 同住)。
// retentionDays ≤ 0 → 走 DefaultBackupRetentionDays 兜底。
func NewBackupCleanup(dir string, retentionDays int, metrics *BackupCleanupMetrics, log *zap.Logger) *BackupCleanup {
	if retentionDays <= 0 {
		retentionDays = DefaultBackupRetentionDays
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &BackupCleanup{
		dir:       dir,
		maxAge:    time.Duration(retentionDays) * 24 * time.Hour,
		tmpMaxAge: TmpResidualMaxAge,
		now:       time.Now,
		metrics:   metrics,
		log:       log.Named("parammodel.backup-cleanup"),
	}
}

// Run 是单次扫描的核函数。返回 (sweptCount, error)。
// error 仅在打开目录失败时返回;单文件错误 → log + metric,不阻塞后续。
//
// 清扫两类:
//   - 过期备份 .deleted.<ts> / .bak.<ts>(> retention)、残留 .tmp.<uuid>(> 1h)
//   - 孤儿 sidecar X.xml.custom(对应 X.xml 已不存在)→ 立即删(kind="sidecar")
//
// 目录不存在(ENOENT)视为合法(首次部署未上传过任何 XML),返 nil。
func (b *BackupCleanup) Run(ctx context.Context) (int, error) {
	entries, err := os.ReadDir(b.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	cutoff := b.now().Add(-b.maxAge)
	tmpCutoff := b.now().Add(-b.tmpMaxAge)

	// 预扫:收集本目录现存普通文件名,供孤儿 sidecar 判定(X.xml 是否仍在)。
	present := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			present[e.Name()] = struct{}{}
		}
	}

	swept := 0
	for _, e := range entries {
		if ctx.Err() != nil {
			return swept, ctx.Err()
		}
		if e.IsDir() {
			continue
		}
		name := e.Name()

		// 孤儿 sidecar:X.xml.custom 但 X.xml 已不存在 → 立即删除
		if base, ok := orphanSidecarBase(name); ok {
			if _, stillThere := present[base]; !stillThere {
				if b.removeFile(name) {
					swept++
				}
			}
			continue
		}

		kind, ok := classifyBackupFile(name)
		if !ok {
			continue // 非清扫对象(可能是合法的 X.xml / 用户私有 .X.bak.manual 等)
		}

		full := filepath.Join(b.dir, name)
		eligible, reason := b.eligibleForRemoval(name, kind, cutoff, tmpCutoff, full)
		if !eligible {
			b.metrics.observe(kind, "skipped")
			continue
		}
		_ = reason // 记 metric 已足,详细原因不写日志,避免高频噪声

		if err := os.Remove(full); err != nil {
			b.log.Warn("remove expired backup failed",
				zap.String("file", name),
				zap.String("kind", kind),
				zap.Error(err))
			b.metrics.observe(kind, "error")
			continue
		}
		swept++
		b.metrics.observe(kind, "swept")
	}

	b.log.Info("backup cleanup done",
		zap.String("dir", b.dir),
		zap.Int("swept", swept),
		zap.Int("scanned", len(entries)),
		zap.Duration("retention", b.maxAge))
	return swept, nil
}

// orphanSidecarBase 判定 name 是否为 sidecar(X.xml.custom),返回其对应的 XML basename(X.xml)。
func orphanSidecarBase(name string) (string, bool) {
	if !strings.HasSuffix(name, CustomMarkerSuffix) {
		return "", false
	}
	return strings.TrimSuffix(name, CustomMarkerSuffix), true
}

// removeFile 删一个 sidecar 孤儿,记 metric。返回是否成功删除。
func (b *BackupCleanup) removeFile(name string) bool {
	if err := os.Remove(filepath.Join(b.dir, name)); err != nil {
		b.log.Warn("remove orphan sidecar failed",
			zap.String("file", name),
			zap.Error(err))
		b.metrics.observe("sidecar", "error")
		return false
	}
	b.metrics.observe("sidecar", "swept")
	return true
}

// classifyBackupFile 判定文件类别。返回 (kind, ok)。
// ok=false 时跳过(不是清扫对象)。
func classifyBackupFile(name string) (kind string, ok bool) {
	if m := backupNameRe.FindStringSubmatch(name); m != nil {
		return m[1], true // "deleted" or "bak"
	}
	if tmpNameRe.MatchString(name) {
		return "tmp", true
	}
	return "", false
}

// eligibleForRemoval 判定单个文件是否到期。
// .deleted / .bak: 文件名 ts 解析(本地时区);ts 早于 cutoff → 清理
// .tmp: 文件名无 ts,改 stat mtime;早于 tmpCutoff → 清理
func (b *BackupCleanup) eligibleForRemoval(
	name, kind string, cutoff, tmpCutoff time.Time, full string,
) (bool, string) {
	switch kind {
	case "deleted", "bak":
		m := backupNameRe.FindStringSubmatch(name)
		if m == nil {
			return false, "name parse failed"
		}
		ts, err := time.ParseInLocation("20060102150405", m[2], time.Local)
		if err != nil {
			return false, "ts parse failed"
		}
		if ts.After(cutoff) {
			return false, "within retention"
		}
		return true, "expired"
	case "tmp":
		info, err := os.Stat(full)
		if err != nil {
			return false, "stat failed"
		}
		if info.ModTime().After(tmpCutoff) {
			return false, "tmp still fresh"
		}
		return true, "tmp residual"
	}
	return false, "unknown kind"
}
