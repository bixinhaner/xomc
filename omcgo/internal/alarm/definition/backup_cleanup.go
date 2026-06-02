package definition

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// worker BackupCleanup(严格对标 T-0180 indicator/backup_cleanup.go;告警 custom 目录
// 是扁平结构,故只扫单一 customDir,不像 indicator 步进三制式子目录)。
//
// 清理对象(customDir 直接子文件):
//   <customDir>/<file>.xml.deleted.<14位ts>   ← DeleteFile 备份
//   <customDir>/<file>.xml.bak.<14位ts>       ← UploadXML overwrite 备份
//   <customDir>/<file>.xml.tmp.<纳秒ts>       ← UploadXML 进行中 tmp(rename 成功后已不存在)
//
// 保留策略:.deleted/.bak = 30 天;.tmp = 1 小时。时间从文件名 ts 解析(不靠 mtime,
// 防 rsync/cp -p 篡改);单文件失败 continue 不阻塞;Prometheus 指标供 SLO 告警。

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
// 标签:kind = deleted|bak|tmp;result = swept|error|skipped。
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

// BackupCleanup 实现 customDir 扁平目录下备份文件的周期清理。
type BackupCleanup struct {
	customDir string
	maxAge    time.Duration
	tmpMaxAge time.Duration
	now       func() time.Time // 测试可注入
	metrics   *BackupCleanupMetrics
	log       *zap.Logger
}

// NewBackupCleanup 构造清扫器。retentionDays ≤ 0 → 走默认 30 天。
func NewBackupCleanup(customDir string, retentionDays int, metrics *BackupCleanupMetrics, log *zap.Logger) *BackupCleanup {
	if retentionDays <= 0 {
		retentionDays = DefaultAlarmBackupRetentionDays
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &BackupCleanup{
		customDir: customDir,
		maxAge:    time.Duration(retentionDays) * 24 * time.Hour,
		tmpMaxAge: AlarmTmpResidualMaxAge,
		now:       time.Now,
		metrics:   metrics,
		log:       log.Named("alarmdef.backup-cleanup"),
	}
}

// Run 是单次扫描的核函数。返回 (sweptCount, error)。
// customDir 不存在(运维未上传过)→ 静默 (0, nil)。
func (b *BackupCleanup) Run(ctx context.Context) (int, error) {
	cutoff := b.now().Add(-b.maxAge)
	tmpCutoff := b.now().Add(-b.tmpMaxAge)

	entries, err := os.ReadDir(b.customDir)
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
		kind, ok := classifyAlarmBackupFile(name)
		if !ok {
			continue // 非清扫对象(合法 X.xml 或用户私有文件)
		}
		full := filepath.Join(b.customDir, name)
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
		zap.String("custom_dir", b.customDir),
		zap.Int("swept", swept),
		zap.Duration("retention", b.maxAge))
	return swept, nil
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
