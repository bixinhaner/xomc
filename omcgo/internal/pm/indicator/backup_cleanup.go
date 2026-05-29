package indicator

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// T-0180 P2 worker BackupCleanup(PRD §9.6,对标 T-0178 ParamModel)。
//
// 目标:周期清理 customDir 下三制式子目录中的备份残留:
//   <customDir>/<tech>/<file>.xml.deleted.<14位ts>     ← DeleteFile 备份
//   <customDir>/<tech>/<file>.xml.bak.<14位ts>         ← UploadXML overwrite 备份
//   <customDir>/<tech>/<file>.xml.tmp.<纳秒ts>         ← UploadXML 进行中 tmp(rename 成功后已不存在)
//
// 保留策略:.deleted/.bak = 30 天(用户决策 3);.tmp = 1 小时(写盘中断兜底)。
//
// 设计决策(沿用 T-0178):
//   - 时间从文件名 ts 解析,不从 os.Stat().ModTime()
//     (mtime 易被 rsync/cp -p 改;文件名不可篡改 — rename 单步原子)
//   - 单文件失败 continue 不阻塞后续(错误隔离)
//   - cron 表达式由调用方控制;Run 是一次扫描核函数
//   - 启动期延迟一次 catch-up(防 worker 长期宕机后备份堆积)
//   - Prometheus 指标 indicator_backup_cleanup_total{kind, result} 给 SLO 告警
//
// 与 T-0178 ParamModel 的关键差异:
//   - customDir 是三制式子目录的根(例如 /etc/omcgo/data/indicator-library-custom),
//     Run() 顺序遍历 Subdirs(默认 enb/gsm/gnb)各自的内容,不递归扫描更深层。
//   - 子目录不存在(运维未上传过该制式)→ 静默跳过。

// indicatorBackupNameRe 匹配 .deleted/.bak + 14 位时间戳(yyyymmddHHMMSS)。
var indicatorBackupNameRe = regexp.MustCompile(`\.(deleted|bak)\.(\d{14})$`)

// indicatorTmpNameRe 匹配 .tmp.<suffix> — UploadXML 用纳秒时间戳,但保持宽松
// 兼容 uuid 形式。后缀不参与时间判定(改看 mtime)。
var indicatorTmpNameRe = regexp.MustCompile(`\.tmp\.[A-Za-z0-9-]+$`)

const (
	// DefaultIndicatorBackupRetentionDays 是 .deleted/.bak 文件保留天数(用户决策 3 一致)。
	DefaultIndicatorBackupRetentionDays = 30

	// IndicatorTmpResidualMaxAge 是 .tmp 残留文件的最大留存。
	// 1 小时足以覆盖任何健康 Upload 周期。
	IndicatorTmpResidualMaxAge = time.Hour

	// DefaultIndicatorBackupCleanupCron 是默认 cron 表达式(每天凌晨 3 点)。
	// 与 ParamModel cron 同点;两者扫描目录不同,无 IO 冲突。
	DefaultIndicatorBackupCleanupCron = "0 3 * * *"
)

// defaultBackupSubdirs 是 customDir 下要扫描的三制式子目录名(对标 file_handler 的合法 tech 集合)。
var defaultBackupSubdirs = []string{"enb", "gsm", "gnb"}

// BackupCleanupMetrics 暴露清扫器指标(PRD §7 第 4 个 counter)。
//
// 标签:
//
//	kind   = "deleted" | "bak" | "tmp"     — 清理对象类别
//	result = "swept" | "error" | "skipped" — swept=成功删除,error=单文件失败,skipped=未到期/解析失败
type BackupCleanupMetrics struct {
	total *prometheus.CounterVec
}

// NewBackupCleanupMetrics 注册并返回指标集合;reg nil 用匿名 Registry(测试场景)。
func NewBackupCleanupMetrics(reg prometheus.Registerer) *BackupCleanupMetrics {
	m := &BackupCleanupMetrics{
		total: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "indicator_backup_cleanup_total",
				Help: "Total indicator custom-dir backup files swept by BackupCleanup cron.",
			},
			[]string{"kind", "result"},
		),
	}
	if reg != nil {
		reg.MustRegister(m.total)
	}
	return m
}

// observe 记录一次清扫结果(nil 安全)。
func (m *BackupCleanupMetrics) observe(kind, result string) {
	if m == nil || m.total == nil {
		return
	}
	m.total.WithLabelValues(kind, result).Inc()
}

// BackupCleanup 实现 customDir 下三制式子目录备份文件的周期清理(T-0180 §9.6)。
type BackupCleanup struct {
	customDir string
	subdirs   []string // 要扫描的子目录(默认 enb/gsm/gnb)
	maxAge    time.Duration
	tmpMaxAge time.Duration
	now       func() time.Time // 测试可注入
	metrics   *BackupCleanupMetrics
	log       *zap.Logger
}

// NewBackupCleanup 构造清扫器。
// retentionDays ≤ 0 → 走 DefaultIndicatorBackupRetentionDays 兜底。
func NewBackupCleanup(customDir string, retentionDays int, metrics *BackupCleanupMetrics, log *zap.Logger) *BackupCleanup {
	if retentionDays <= 0 {
		retentionDays = DefaultIndicatorBackupRetentionDays
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &BackupCleanup{
		customDir: customDir,
		subdirs:   defaultBackupSubdirs,
		maxAge:    time.Duration(retentionDays) * 24 * time.Hour,
		tmpMaxAge: IndicatorTmpResidualMaxAge,
		now:       time.Now,
		metrics:   metrics,
		log:       log.Named("indicator.backup-cleanup"),
	}
}

// Run 是单次扫描的核函数。返回 (sweptCount, error)。
//
// 行为:
//  1. 顺序扫 subdirs(默认 enb/gsm/gnb)各自的 customDir/<tech>/ 目录
//  2. 每子目录:目录不存在(ENOENT)→ 静默跳过;打开失败 → 记 log 跳过该子目录,不中断
//  3. 每个 entry:按 classifyBackupFile 判 kind;eligibleForRemoval 判到期;os.Remove
//  4. error 仅在扫描整个 customDir 失败时(罕见)返;单文件失败 → log + metric
func (b *BackupCleanup) Run(ctx context.Context) (int, error) {
	cutoff := b.now().Add(-b.maxAge)
	tmpCutoff := b.now().Add(-b.tmpMaxAge)

	swept := 0
	for _, sub := range b.subdirs {
		if ctx.Err() != nil {
			return swept, ctx.Err()
		}
		dir := filepath.Join(b.customDir, sub)
		n, err := b.runOnDir(ctx, dir, cutoff, tmpCutoff)
		swept += n
		if err != nil {
			// 单子目录失败不阻塞其他制式
			b.log.Warn("backup cleanup subdir failed",
				zap.String("subdir", dir),
				zap.Error(err))
			continue
		}
	}

	b.log.Info("indicator backup cleanup done",
		zap.String("custom_dir", b.customDir),
		zap.Strings("subdirs", b.subdirs),
		zap.Int("swept", swept),
		zap.Duration("retention", b.maxAge))
	return swept, nil
}

// runOnDir 扫描单个子目录;目录不存在 → 静默(0, nil)。
func (b *BackupCleanup) runOnDir(ctx context.Context, dir string, cutoff, tmpCutoff time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
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
		kind, ok := classifyIndicatorBackupFile(name)
		if !ok {
			continue // 非清扫对象(合法 X.xml 或用户私有文件)
		}
		full := filepath.Join(dir, name)
		eligible, _ := b.eligibleForRemoval(name, kind, cutoff, tmpCutoff, full)
		if !eligible {
			b.metrics.observe(kind, "skipped")
			continue
		}
		if err := os.Remove(full); err != nil {
			b.log.Warn("remove expired backup failed",
				zap.String("file", full),
				zap.String("kind", kind),
				zap.Error(err))
			b.metrics.observe(kind, "error")
			continue
		}
		swept++
		b.metrics.observe(kind, "swept")
	}
	return swept, nil
}

// classifyIndicatorBackupFile 判定文件类别。返回 (kind, ok)。
// ok=false 时跳过(不是清扫对象)。
func classifyIndicatorBackupFile(name string) (kind string, ok bool) {
	if m := indicatorBackupNameRe.FindStringSubmatch(name); m != nil {
		return m[1], true // "deleted" or "bak"
	}
	if indicatorTmpNameRe.MatchString(name) {
		return "tmp", true
	}
	return "", false
}

// eligibleForRemoval 判定单个文件是否到期。
// .deleted / .bak:文件名 ts 解析(本地时区);ts 早于 cutoff → 清理
// .tmp:文件名后缀不参与时间判定,改 stat mtime;早于 tmpCutoff → 清理
func (b *BackupCleanup) eligibleForRemoval(
	name, kind string, cutoff, tmpCutoff time.Time, full string,
) (bool, string) {
	switch kind {
	case "deleted", "bak":
		m := indicatorBackupNameRe.FindStringSubmatch(name)
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
