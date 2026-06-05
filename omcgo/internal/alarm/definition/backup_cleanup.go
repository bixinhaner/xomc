package definition

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

// worker BackupCleanup(三库 XML 导入重构改单目录;对标 ParamModel/backup_cleanup.go)。
//
// 目标:周期清理 alarm-definitions/ 目录下的备份/残留文件 + 孤儿 sidecar(builtin + custom
// XML 同住本目录,sidecar 判来源)。
//
// 清理对象(dir 直接子文件):
//   <dir>/<name>.xml.deleted.<14位ts>   ← DeleteFile 备份
//   <dir>/<name>.xml.bak.<14位ts>       ← (历史)overwrite 备份;重构后 Upload 不再产生
//   <dir>/<name>.xml.tmp.<纳秒ts>       ← UploadXML 进行中 tmp(rename 成功后已不存在)
//   <dir>/<name>.xml.custom             ← sidecar 标记;若对应 <name>.xml 已不存在则为孤儿,立即删
//
// 保留策略:.deleted/.bak = 30 天;.tmp = 1 小时;孤儿 sidecar 即清。时间从文件名 ts 解析
// (不靠 mtime,防 rsync/cp -p 篡改);单文件失败 continue 不阻塞;Prometheus 指标供 SLO 告警。

var alarmBackupNameRe = regexp.MustCompile(`\.(deleted|bak)\.(\d{14})$`)
var alarmTmpNameRe = regexp.MustCompile(`\.tmp\.[A-Za-z0-9-]+$`)

const (
	// DefaultAlarmBackupRetentionDays 是 .deleted/.bak 文件保留天数。
	DefaultAlarmBackupRetentionDays = 30
	// AlarmTmpResidualMaxAge 是 .tmp 残留文件的最大留存。
	AlarmTmpResidualMaxAge = time.Hour
	// DefaultAlarmBackupCleanupCron 是默认 cron 表达式(每天凌晨 3 点;与其他库错峰扫描不同目录)。
	DefaultAlarmBackupCleanupCron = "0 3 * * *"
)

// BackupCleanupMetrics 暴露清扫器指标。
// 标签:kind = deleted|bak|tmp|sidecar;result = swept|error|skipped。
type BackupCleanupMetrics struct {
	total *prometheus.CounterVec
}

// NewBackupCleanupMetrics 注册并返回指标集合;reg nil 用匿名注册(测试场景)。
func NewBackupCleanupMetrics(reg prometheus.Registerer) *BackupCleanupMetrics {
	m := &BackupCleanupMetrics{
		total: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alarm_backup_cleanup_total",
				Help: "Total alarm custom-dir backup files swept by BackupCleanup cron.",
			},
			[]string{"kind", "result"},
		),
	}
	if reg != nil {
		reg.MustRegister(m.total)
	}
	return m
}

func (m *BackupCleanupMetrics) observe(kind, result string) {
	if m == nil || m.total == nil {
		return
	}
	m.total.WithLabelValues(kind, result).Inc()
}

// BackupCleanup 实现 alarm-definitions/ 目录下备份文件 + 孤儿 sidecar 的周期清理。
type BackupCleanup struct {
	dir       string
	maxAge    time.Duration
	tmpMaxAge time.Duration
	now       func() time.Time // 测试可注入
	metrics   *BackupCleanupMetrics
	log       *zap.Logger
}

// NewBackupCleanup 构造清扫器。dir 是 alarm-definitions 目录(builtin + custom XML 同住)。
// retentionDays ≤ 0 → 走默认 30 天。
func NewBackupCleanup(dir string, retentionDays int, metrics *BackupCleanupMetrics, log *zap.Logger) *BackupCleanup {
	if retentionDays <= 0 {
		retentionDays = DefaultAlarmBackupRetentionDays
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &BackupCleanup{
		dir:       dir,
		maxAge:    time.Duration(retentionDays) * 24 * time.Hour,
		tmpMaxAge: AlarmTmpResidualMaxAge,
		now:       time.Now,
		metrics:   metrics,
		log:       log.Named("alarmdef.backup-cleanup"),
	}
}

// Run 是单次扫描的核函数。返回 (sweptCount, error)。
// dir 不存在(空库 / 本地裸跑)→ 静默 (0, nil)。
//
// 清扫两类:
//   - 过期备份 .deleted.<ts> / .bak.<ts>(> retention)、残留 .tmp.<ts>(> 1h)
//   - 孤儿 sidecar X.xml.custom(对应 X.xml 已不存在)→ 立即删(kind="sidecar")
func (b *BackupCleanup) Run(ctx context.Context) (int, error) {
	cutoff := b.now().Add(-b.maxAge)
	tmpCutoff := b.now().Add(-b.tmpMaxAge)

	entries, err := os.ReadDir(b.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

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

		// 孤儿 sidecar:X.xml.custom 但 X.xml 已不存在 → 立即删除。
		if xmlBase, ok := orphanSidecarBase(name); ok {
			if _, stillThere := present[xmlBase]; !stillThere {
				if b.removeSidecar(name) {
					swept++
				}
			}
			continue
		}

		kind, ok := classifyAlarmBackupFile(name)
		if !ok {
			continue // 非清扫对象(合法 X.xml / X.xml.custom 仍有主文件 / 用户私有文件)
		}
		full := filepath.Join(b.dir, name)
		if !b.eligibleForRemoval(name, kind, cutoff, tmpCutoff, full) {
			b.metrics.observe(kind, "skipped")
			continue
		}
		if err := os.Remove(full); err != nil {
			b.log.Warn("remove expired backup failed",
				zap.String("file", full), zap.String("kind", kind), zap.Error(err))
			b.metrics.observe(kind, "error")
			continue
		}
		swept++
		b.metrics.observe(kind, "swept")
	}

	b.log.Info("alarm backup cleanup done",
		zap.String("dir", b.dir),
		zap.Int("swept", swept),
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

// removeSidecar 删一个孤儿 sidecar,记 metric。返回是否成功删除。
func (b *BackupCleanup) removeSidecar(name string) bool {
	if err := os.Remove(filepath.Join(b.dir, name)); err != nil {
		b.log.Warn("remove orphan sidecar failed", zap.String("file", name), zap.Error(err))
		b.metrics.observe("sidecar", "error")
		return false
	}
	b.metrics.observe("sidecar", "swept")
	return true
}

// classifyAlarmBackupFile 判定文件类别。ok=false 时跳过。
func classifyAlarmBackupFile(name string) (kind string, ok bool) {
	if m := alarmBackupNameRe.FindStringSubmatch(name); m != nil {
		return m[1], true // "deleted" or "bak"
	}
	if alarmTmpNameRe.MatchString(name) {
		return "tmp", true
	}
	return "", false
}

// eligibleForRemoval 判定单个文件是否到期。
// .deleted/.bak:文件名 ts 解析(本地时区);早于 cutoff → 清理。
// .tmp:后缀不参与时间判定,改 stat mtime;早于 tmpCutoff → 清理。
func (b *BackupCleanup) eligibleForRemoval(name, kind string, cutoff, tmpCutoff time.Time, full string) bool {
	switch kind {
	case "deleted", "bak":
		m := alarmBackupNameRe.FindStringSubmatch(name)
		if m == nil {
			return false
		}
		ts, err := time.ParseInLocation("20060102150405", m[2], time.Local)
		if err != nil {
			return false
		}
		return !ts.After(cutoff)
	case "tmp":
		info, err := os.Stat(full)
		if err != nil {
			return false
		}
		return !info.ModTime().After(tmpCutoff)
	}
	return false
}
