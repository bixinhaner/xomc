package upload

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// issue #318：PM 文件上传资源背压 watchdog。
//
// 资源最大化部署（每台独占跑全栈，CPU 不按进程切）下磁盘写满或 CPU 打满仍无脑收 PM 文件会
// 拖垮整机。本 watchdog 周期采集「数据盘占用%」（查 MinIO 指标端点，ACS/MinIO 同栈内网可达，
// 免鉴权）+「CPU 负载」（host /proc/loadavg ÷ 核数），带迟滞地切换背压态：超高水位 → ACS 上传
// handler 对 fileType=PM 返回 503（TR-069 设备会重传，不丢数据）；回落到低水位 → 自动恢复。
//
// 配置全部落 sys_configs（category=acs.backpressure），经 NATS SubjectSysConfigSaved 热刷新。
// 热路径（ServeHTTP）只读一个原子标志，零额外 IO。

const (
	// BackpressureCategory 是 sys_configs 中背压配置的 category。
	BackpressureCategory = "acs.backpressure"

	bpKeyEnabled     = "enabled"
	bpKeyDiskHighPct = "disk_high_pct"
	bpKeyDiskLowPct  = "disk_low_pct"
	bpKeyCPUHigh     = "cpu_high_per_core"
	bpKeyCPULow      = "cpu_low_per_core"
	bpKeyInterval    = "check_interval_sec"

	bpDefaultEnabled     = true
	bpDefaultDiskHighPct = 85.0
	bpDefaultDiskLowPct  = 75.0
	bpDefaultCPUHigh     = 0.90
	bpDefaultCPULow      = 0.70
	bpDefaultIntervalSec = 30
	bpMinInterval        = 5 * time.Second
)

// BackpressureGate 是 PM 上传背压门闸的最小读取接口（便于 handler 测试）。
type BackpressureGate interface {
	// Allowed 返回是否允许接收 PM 上传（false = 当前处于背压）。
	Allowed() bool
	// RecordRejected 记录一次因背压拒收（metric）。
	RecordRejected()
}

// BackpressureConfig 是背压判定阈值，来自 sys_configs，可热刷新。
type BackpressureConfig struct {
	Enabled        bool
	DiskHighPct    float64 // 磁盘高水位%：≥ 则进入背压
	DiskLowPct     float64 // 磁盘低水位%：≤ 才解除（迟滞）
	CPUHighPerCore float64 // 每核负载高水位
	CPULowPerCore  float64 // 每核负载低水位
	Interval       time.Duration
}

// ConfigLookup 读 sys_configs 单值（value, found）。
type ConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// DiskUsageFunc 返回数据盘使用率百分比（0-100）。
type DiskUsageFunc func(ctx context.Context) (pct float64, err error)

// loadBackpressureConfig 从 sys_configs 读全部背压阈值，缺失/非法回落默认值并做迟滞防呆。
func loadBackpressureConfig(ctx context.Context, lookup ConfigLookup) BackpressureConfig {
	cfg := BackpressureConfig{
		Enabled:        bpDefaultEnabled,
		DiskHighPct:    bpDefaultDiskHighPct,
		DiskLowPct:     bpDefaultDiskLowPct,
		CPUHighPerCore: bpDefaultCPUHigh,
		CPULowPerCore:  bpDefaultCPULow,
		Interval:       time.Duration(bpDefaultIntervalSec) * time.Second,
	}
	if lookup != nil {
		cfg.Enabled = bpReadBool(ctx, lookup, bpKeyEnabled, bpDefaultEnabled)
		cfg.DiskHighPct = float64(bpReadInt(ctx, lookup, bpKeyDiskHighPct, int(bpDefaultDiskHighPct), 1, 100))
		cfg.DiskLowPct = float64(bpReadInt(ctx, lookup, bpKeyDiskLowPct, int(bpDefaultDiskLowPct), 0, 100))
		cfg.CPUHighPerCore = bpReadFloat(ctx, lookup, bpKeyCPUHigh, bpDefaultCPUHigh)
		cfg.CPULowPerCore = bpReadFloat(ctx, lookup, bpKeyCPULow, bpDefaultCPULow)
		sec := bpReadInt(ctx, lookup, bpKeyInterval, bpDefaultIntervalSec, 1, 3600)
		cfg.Interval = time.Duration(sec) * time.Second
	}
	if cfg.Interval < bpMinInterval {
		cfg.Interval = bpMinInterval
	}
	// 迟滞防呆：low 必须 ≤ high，否则回落条件与进入条件交叠会抖动。
	if cfg.DiskLowPct > cfg.DiskHighPct {
		cfg.DiskLowPct = cfg.DiskHighPct
	}
	if cfg.CPULowPerCore > cfg.CPUHighPerCore {
		cfg.CPULowPerCore = cfg.CPUHighPerCore
	}
	return cfg
}

func bpReadBool(ctx context.Context, lookup ConfigLookup, key string, def bool) bool {
	v, found := lookup(ctx, BackpressureCategory, key)
	if !found {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}

func bpReadInt(ctx context.Context, lookup ConfigLookup, key string, def, lo, hi int) int {
	v, found := lookup(ctx, BackpressureCategory, key)
	if !found {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < lo || n > hi {
		return def
	}
	return n
}

func bpReadFloat(ctx context.Context, lookup ConfigLookup, key string, def float64) float64 {
	v, found := lookup(ctx, BackpressureCategory, key)
	if !found {
		return def
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || f < 0 {
		return def
	}
	return f
}

// decideBackpressure 是纯函数迟滞决策（便于单测）。diskPct / loadPerCore 为负表示该信号不可用
// （探测失败），不可用信号在进入判定时忽略、在解除判定时视作已达标（fail-open，不长期误堵）。
func decideBackpressure(current bool, diskPct, loadPerCore float64, cfg BackpressureConfig) bool {
	if !cfg.Enabled {
		return false
	}
	if !current {
		// 未背压：任一已知信号越高水位 → 进入背压。
		if diskPct >= 0 && diskPct >= cfg.DiskHighPct {
			return true
		}
		if loadPerCore >= 0 && loadPerCore >= cfg.CPUHighPerCore {
			return true
		}
		return false
	}
	// 已背压：全部已知信号回落到低水位以下才解除（迟滞）。
	diskOK := diskPct < 0 || diskPct <= cfg.DiskLowPct
	loadOK := loadPerCore < 0 || loadPerCore <= cfg.CPULowPerCore
	return !(diskOK && loadOK)
}

// BackpressureMetrics 暴露背压可观测指标。
type BackpressureMetrics struct {
	active      prometheus.Gauge
	rejected    prometheus.Counter
	diskPct     prometheus.Gauge
	loadPerCore prometheus.Gauge
}

// NewBackpressureMetrics 注册并返回背压指标。reg 为 nil 时不注册（测试用）。
func NewBackpressureMetrics(reg prometheus.Registerer) *BackpressureMetrics {
	m := &BackpressureMetrics{
		active: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_pm_upload_backpressure_active",
			Help: "PM 上传背压是否生效（1=拒收中，0=正常），issue #318",
		}),
		rejected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "acs_pm_upload_backpressure_rejected_total",
			Help: "因背压被拒收的 PM 上传总数（设备会重传，不丢数据）",
		}),
		diskPct: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_data_disk_usage_percent",
			Help: "watchdog 观测到的数据盘（MinIO）使用率百分比",
		}),
		loadPerCore: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_host_load_per_core",
			Help: "watchdog 观测到的主机每核 1 分钟负载（loadavg ÷ NumCPU）",
		}),
	}
	if reg != nil {
		reg.MustRegister(m.active, m.rejected, m.diskPct, m.loadPerCore)
	}
	return m
}

// Watchdog 周期采样磁盘/CPU，带迟滞维护背压态，并对 handler 暴露 Allowed()。
type Watchdog struct {
	lookup    ConfigLookup
	diskUsage DiskUsageFunc
	loadFn    func() (float64, error)
	metrics   *BackpressureMetrics
	logger    *zap.Logger

	cfg    atomic.Pointer[BackpressureConfig]
	active atomic.Bool // true = 背压中（拒收 PM）
}

// NewWatchdog 构造 watchdog。diskUsage 为 nil 时跳过磁盘信号（仅看 CPU）。
func NewWatchdog(lookup ConfigLookup, diskUsage DiskUsageFunc, metrics *BackpressureMetrics, logger *zap.Logger) *Watchdog {
	if logger == nil {
		logger = zap.NewNop()
	}
	w := &Watchdog{
		lookup:    lookup,
		diskUsage: diskUsage,
		loadFn:    loadPerCore,
		metrics:   metrics,
		logger:    logger,
	}
	cfg := loadBackpressureConfig(context.Background(), lookup)
	w.cfg.Store(&cfg)
	return w
}

// Allowed 实现 BackpressureGate：非背压态才允许接收 PM 上传。
func (w *Watchdog) Allowed() bool { return !w.active.Load() }

// RecordRejected 记录一次背压拒收。
func (w *Watchdog) RecordRejected() {
	if w.metrics != nil {
		w.metrics.rejected.Inc()
	}
}

// Run 启动周期采样循环，直到 ctx 取消。
func (w *Watchdog) Run(ctx context.Context) {
	interval := w.cfg.Load().Interval
	w.sample(ctx) // 启动即采一次
	t := time.NewTicker(interval)
	defer t.Stop()
	w.logger.Info("backpressure watchdog started", zap.Duration("interval", interval))
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("backpressure watchdog stopped")
			return
		case <-t.C:
			// 先采样（刷新阈值 + 采样周期到 w.cfg），再据最新采样周期校正 ticker——
			// 这样 check_interval_sec 的变更与阈值变更一样在「下一拍」即生效，兑现下方
			// sample 注释「变更 ≤1 个采样周期生效」的契约。若先读旧 cfg 再采样，interval
			// 变更要多等一整拍（旧周期）才被 ticker 采纳，与契约不符。
			w.sample(ctx)
			if cur := w.cfg.Load().Interval; cur != interval {
				interval = cur
				t.Reset(interval)
			}
		}
	}
}

// sample 采一次磁盘/CPU 并更新背压态与指标。每次采样前从 sys_configs 刷新阈值——ACS 的
// JetStream workqueue 流上 SubjectSysConfigSaved 已被 transfercfg 占用（同一 filter subject
// 不允许第二个 consumer），故背压配置走轮询刷新（变更 ≤1 个采样周期生效），不另开订阅。
func (w *Watchdog) sample(ctx context.Context) {
	cfg := loadBackpressureConfig(ctx, w.lookup)
	w.cfg.Store(&cfg)
	if !cfg.Enabled {
		if w.active.Swap(false) {
			w.logger.Info("PM upload backpressure RELEASED (disabled by config)")
		}
		w.setActiveGauge(false)
		return
	}

	diskPct := -1.0
	if w.diskUsage != nil {
		if p, err := w.diskUsage(ctx); err != nil {
			w.logger.Warn("disk usage probe failed; skip disk signal", zap.Error(err))
		} else {
			diskPct = p
			if w.metrics != nil {
				w.metrics.diskPct.Set(p)
			}
		}
	}

	loadPC := -1.0
	if w.loadFn != nil {
		if l, err := w.loadFn(); err != nil {
			w.logger.Warn("cpu load probe failed; skip cpu signal", zap.Error(err))
		} else {
			loadPC = l
			if w.metrics != nil {
				w.metrics.loadPerCore.Set(l)
			}
		}
	}

	prev := w.active.Load()
	next := decideBackpressure(prev, diskPct, loadPC, cfg)
	if next != prev {
		w.active.Store(next)
		if next {
			w.logger.Warn("PM upload backpressure ENGAGED — rejecting PM uploads (devices will retry)",
				zap.Float64("disk_pct", diskPct), zap.Float64("disk_high_pct", cfg.DiskHighPct),
				zap.Float64("load_per_core", loadPC), zap.Float64("cpu_high_per_core", cfg.CPUHighPerCore))
		} else {
			w.logger.Info("PM upload backpressure RELEASED — resuming PM uploads",
				zap.Float64("disk_pct", diskPct), zap.Float64("load_per_core", loadPC))
		}
	}
	w.setActiveGauge(next)
}

func (w *Watchdog) setActiveGauge(active bool) {
	if w.metrics == nil {
		return
	}
	if active {
		w.metrics.active.Set(1)
	} else {
		w.metrics.active.Set(0)
	}
}

// loadPerCore 读 host /proc/loadavg 的 1 分钟负载 ÷ 逻辑核数。非 Linux（如 dev darwin）无
// /proc/loadavg → 返回错误，调用方据此跳过 CPU 信号。
func loadPerCore() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected /proc/loadavg content")
	}
	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parse loadavg: %w", err)
	}
	n := runtime.NumCPU()
	if n < 1 {
		n = 1
	}
	return load1 / float64(n), nil
}

// MinIOMetricsURL 据 MinIO endpoint（host:port）+ UseSSL 拼出集群指标端点 URL。
func MinIOMetricsURL(endpoint string, useSSL bool) string {
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/minio/v2/metrics/cluster", scheme, endpoint)
}

// NewMinIODiskUsage 返回从 MinIO Prometheus 集群指标端点读取磁盘使用率%的 DiskUsageFunc。
// 端点需 MINIO_PROMETHEUS_AUTH_TYPE=public（本栈已配）；非 200 / 解析失败均返回 error，
// 由 watchdog 降级为「磁盘信号不可用」（不误堵）。
func NewMinIODiskUsage(metricsURL string, timeout time.Duration, client *http.Client) DiskUsageFunc {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return func(ctx context.Context) (float64, error) {
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, metricsURL, nil)
		if err != nil {
			return 0, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, fmt.Errorf("fetch minio metrics: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("minio metrics http status %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if err != nil {
			return 0, fmt.Errorf("read minio metrics: %w", err)
		}
		return parseDiskUsagePct(string(body))
	}
}

// parseDiskUsagePct 从 MinIO 集群指标文本计算磁盘使用率%。按指标版本差异依次尝试多组容量指标：
// usable 容量 → raw 容量 → 节点磁盘 free → 节点磁盘 used，取首个可用组。
func parseDiskUsagePct(text string) (float64, error) {
	sums, present := sumMetrics(text, []string{
		"minio_cluster_capacity_usable_total_bytes",
		"minio_cluster_capacity_usable_free_bytes",
		"minio_cluster_capacity_raw_total_bytes",
		"minio_cluster_capacity_raw_free_bytes",
		"minio_node_disk_total_bytes",
		"minio_node_disk_free_bytes",
		"minio_node_disk_used_bytes",
	})

	if t := sums["minio_cluster_capacity_usable_total_bytes"]; t > 0 && present["minio_cluster_capacity_usable_free_bytes"] {
		return usedPctFromFree(t, sums["minio_cluster_capacity_usable_free_bytes"]), nil
	}
	if t := sums["minio_cluster_capacity_raw_total_bytes"]; t > 0 && present["minio_cluster_capacity_raw_free_bytes"] {
		return usedPctFromFree(t, sums["minio_cluster_capacity_raw_free_bytes"]), nil
	}
	if t := sums["minio_node_disk_total_bytes"]; t > 0 {
		if present["minio_node_disk_free_bytes"] {
			return usedPctFromFree(t, sums["minio_node_disk_free_bytes"]), nil
		}
		if present["minio_node_disk_used_bytes"] {
			return clampPct(sums["minio_node_disk_used_bytes"] / t * 100), nil
		}
	}
	return 0, fmt.Errorf("no usable minio capacity metrics found")
}

func usedPctFromFree(total, free float64) float64 {
	if total <= 0 {
		return 0
	}
	return clampPct((total - free) / total * 100)
}

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// sumMetrics 解析 Prometheus 文本，按指定名字汇总各指标全部样本的值之和（跨 label）。
func sumMetrics(text string, names []string) (sums map[string]float64, present map[string]bool) {
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	sums = make(map[string]float64, len(names))
	present = make(map[string]bool, len(names))

	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		name := line
		if i := strings.IndexAny(line, "{ "); i >= 0 {
			name = line[:i]
		}
		if _, ok := want[name]; !ok {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// 格式：`name{labels} value [timestamp]` 或 `name value`，value 恒为第 2 个字段。
		v, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		sums[name] += v
		present[name] = true
	}
	return sums, present
}
