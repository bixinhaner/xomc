package upload

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// issue #318：PM 文件上传资源背压 watchdog。
//
// 资源最大化部署（每台独占跑全栈）下磁盘写满仍无脑收 PM 文件会拖垮整机。本 watchdog 周期采集
// 「数据盘占用%」（查 MinIO 指标端点，ACS/MinIO 同栈内网可达，免鉴权），带迟滞地切换背压态：
// 超高水位 → ACS 上传 handler 对 fileType=PM 返回 503（TR-069 设备会重传，不丢数据）；回落到
// 低水位 → 自动恢复。CPU 负载仍周期探测并上报 Prometheus 指标（acs_host_load_per_core），
// 供监控告警使用，但不参与背压决策（#827）。
//
// 配置全部落 sys_configs（category=acs.backpressure），经 NATS SubjectSysConfigSaved 热刷新。
// 热路径（ServeHTTP）只读一个原子标志，零额外 IO。

const (
	// BackpressureCategory 是 sys_configs 中背压配置的 category。
	BackpressureCategory = "acs.backpressure"

	bpKeyEnabled          = "enabled"
	bpKeyDiskHighPct      = "disk_high_pct"
	bpKeyDiskLowPct       = "disk_low_pct"
	bpKeyIOSomeHigh       = "io_some_high_pct"
	bpKeyIOSomeLow        = "io_some_low_pct"
	bpKeyMaxInflight      = "max_inflight"
	bpKeyInterval         = "check_interval_sec"
	bpKeyQueuePendingHigh = "queue_pending_high"
	bpKeyQueuePendingLow  = "queue_pending_low"
	bpKeyQueueOldestHigh  = "queue_oldest_high_sec"
	bpKeyQueueOldestLow   = "queue_oldest_low_sec"
	bpKeyQueueSlopeWindow = "queue_slope_window_sec"

	bpDefaultEnabled     = true
	bpDefaultDiskHighPct = 70.0
	bpDefaultDiskLowPct  = 60.0
	bpDefaultIOSomeHigh  = 70.0
	bpDefaultIOSomeLow   = 20.0
	bpDefaultMaxInflight = 2000
	bpDefaultIntervalSec = 30
	// 20k devices report PM files in a synchronized quarter-hour burst. Production
	// observations show a healthy, fast-draining peak around 2.7k pending with low
	// I/O pressure; the old 2k threshold returned avoidable 503s every hour.
	bpDefaultQueuePendingHigh = 5000
	bpDefaultQueuePendingLow  = 1000
	bpDefaultQueueOldestHigh  = 10 * time.Minute
	bpDefaultQueueOldestLow   = 2 * time.Minute
	bpDefaultQueueSlopeWindow = 5 * time.Minute
	bpMinQueueSlopeWindow     = 2 * event.QueueHealthSampleInterval
	bpMinInterval             = 5 * time.Second

	rejectReasonResourcePressure = "resource_pressure"
	rejectReasonInflightLimit    = "inflight_limit"

	pressureReasonDisabled          = "disabled"
	pressureReasonDisk              = "disk"
	pressureReasonIO                = "io"
	pressureReasonQueuePending      = "queue_pending"
	pressureReasonQueueOldest       = "queue_oldest"
	pressureReasonQueueGrowing      = "queue_growing"
	pressureReasonQueueSlopeUnknown = "queue_slope_unknown"
	pressureReasonQueueFailure      = "queue_sample_failure"
	pressureReasonRecovered         = "recovered"
)

// BackpressureGate 是 PM 上传背压门闸的最小读取接口（便于 handler 测试）。
type BackpressureGate interface {
	// Acquire 尝试获取一个 PM 上传在途槽位。watchdog 背压生效或达到并发上限时返回 false。
	Acquire() (allowed bool, rejectReason string)
	// Release 释放 Acquire 成功取得的槽位。
	Release()
	// RecordRejected 记录一次因背压拒收（metric）。
	RecordRejected(reason string)
	// RecordAccepted 记录一次已经成功写入对象存储的 PM 上传。
	RecordAccepted(at time.Time)
}

// BackpressureConfig 是背压判定阈值，来自 sys_configs，可热刷新。
type BackpressureConfig struct {
	Enabled          bool
	DiskHighPct      float64 // 磁盘高水位%：≥ 则进入背压
	DiskLowPct       float64 // 磁盘低水位%：≤ 才解除（迟滞）
	IOSomeHighPct    float64 // /proc/pressure/io some avg10 高水位
	IOSomeLowPct     float64 // /proc/pressure/io some avg10 低水位
	MaxInflight      int64   // PM 上传最大在途数
	Interval         time.Duration
	QueuePendingHigh int
	QueuePendingLow  int
	QueueOldestHigh  time.Duration
	QueueOldestLow   time.Duration
	QueueSlopeWindow time.Duration
}

// QueueRates are derived only from two fresh, monotonic queue samples.
type QueueRates struct {
	PendingPerSecond    float64
	AckAdvancePerSecond float64
	DeliveryPerSecond   float64
}

// QueueSignal is a point-in-time input to the pure backpressure decision.
// Configured distinguishes an absent queue source from a configured source
// whose latest read failed or became stale.
type QueueSignal struct {
	Configured     bool
	Available      bool
	RatesAvailable bool
	Stats          event.QueueStats
	Rates          QueueRates
}

type BackpressureDecision struct {
	Active bool
	Reason string
}

type pressureState uint32

const (
	pressureDisk pressureState = 1 << iota
	pressureIO
	pressureQueue
)

func (s pressureState) has(flag pressureState) bool {
	return s&flag != 0
}

// QueueStatsSource exposes only the independent sampler cache. Implementations
// must not perform a new JetStream management query here.
type QueueStatsSource interface {
	LatestStats() (event.QueueStats, bool)
}

type queueSampleAttemptSource interface {
	LastSampleAttempt() (time.Time, bool)
}

// ConfigLookup 读 sys_configs 单值（value, found）。
type ConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// DiskUsageFunc 返回数据盘使用率百分比（0-100）。
type DiskUsageFunc func(ctx context.Context) (pct float64, err error)

// ProjectedBytesFunc estimates bytes that are already accepted but not yet
// materialized in the database (for example pending raw PM files).
type ProjectedBytesFunc func(ctx context.Context) (bytes float64, err error)

// PendingCountFunc returns the number of PM messages accepted by the durable
// queue but not fully acknowledged by workers yet.
type PendingCountFunc func(ctx context.Context) (count uint64, err error)

// loadBackpressureConfig 从 sys_configs 读全部背压阈值，缺失/非法回落默认值并做迟滞防呆。
func loadBackpressureConfig(ctx context.Context, lookup ConfigLookup) BackpressureConfig {
	return loadBackpressureConfigWithDefaults(ctx, lookup, defaultBackpressureConfig())
}

func defaultBackpressureConfig() BackpressureConfig {
	return BackpressureConfig{
		Enabled:          bpDefaultEnabled,
		DiskHighPct:      bpDefaultDiskHighPct,
		DiskLowPct:       bpDefaultDiskLowPct,
		IOSomeHighPct:    bpDefaultIOSomeHigh,
		IOSomeLowPct:     bpDefaultIOSomeLow,
		MaxInflight:      bpDefaultMaxInflight,
		Interval:         time.Duration(bpDefaultIntervalSec) * time.Second,
		QueuePendingHigh: bpDefaultQueuePendingHigh,
		QueuePendingLow:  bpDefaultQueuePendingLow,
		QueueOldestHigh:  bpDefaultQueueOldestHigh,
		QueueOldestLow:   bpDefaultQueueOldestLow,
		QueueSlopeWindow: bpDefaultQueueSlopeWindow,
	}
}

func loadBackpressureConfigWithDefaults(
	ctx context.Context,
	lookup ConfigLookup,
	defaults BackpressureConfig,
) BackpressureConfig {
	cfg := defaults
	if lookup != nil {
		cfg.Enabled = bpReadBool(ctx, lookup, bpKeyEnabled, defaults.Enabled)
		cfg.DiskHighPct = float64(bpReadInt(ctx, lookup, bpKeyDiskHighPct, int(defaults.DiskHighPct), 1, 100))
		cfg.DiskLowPct = float64(bpReadInt(ctx, lookup, bpKeyDiskLowPct, int(defaults.DiskLowPct), 0, 100))
		cfg.IOSomeHighPct = float64(bpReadInt(ctx, lookup, bpKeyIOSomeHigh, int(defaults.IOSomeHighPct), 1, 100))
		cfg.IOSomeLowPct = float64(bpReadInt(ctx, lookup, bpKeyIOSomeLow, int(defaults.IOSomeLowPct), 0, 100))
		cfg.MaxInflight = int64(bpReadInt(ctx, lookup, bpKeyMaxInflight, int(defaults.MaxInflight), 1, 10000))
		sec := bpReadInt(ctx, lookup, bpKeyInterval, int(defaults.Interval/time.Second), 1, 3600)
		cfg.Interval = time.Duration(sec) * time.Second
		cfg.QueuePendingHigh = bpReadInt(ctx, lookup, bpKeyQueuePendingHigh, defaults.QueuePendingHigh, 1, 10000000)
		cfg.QueuePendingLow = bpReadInt(ctx, lookup, bpKeyQueuePendingLow, defaults.QueuePendingLow, 0, 10000000)
		cfg.QueueOldestHigh = time.Duration(bpReadInt(ctx, lookup, bpKeyQueueOldestHigh, int(defaults.QueueOldestHigh/time.Second), 1, 86400)) * time.Second
		cfg.QueueOldestLow = time.Duration(bpReadInt(ctx, lookup, bpKeyQueueOldestLow, int(defaults.QueueOldestLow/time.Second), 0, 86400)) * time.Second
		cfg.QueueSlopeWindow = time.Duration(bpReadInt(ctx, lookup, bpKeyQueueSlopeWindow, int(defaults.QueueSlopeWindow/time.Second), 1, 86400)) * time.Second
	}
	if cfg.Interval < bpMinInterval {
		cfg.Interval = bpMinInterval
	}
	// 迟滞防呆：low 必须 ≤ high，否则回落条件与进入条件交叠会抖动。
	if cfg.DiskLowPct > cfg.DiskHighPct {
		cfg.DiskLowPct = cfg.DiskHighPct
	}
	if cfg.IOSomeLowPct > cfg.IOSomeHighPct {
		cfg.IOSomeLowPct = cfg.IOSomeHighPct
	}
	if cfg.QueuePendingLow > cfg.QueuePendingHigh {
		cfg.QueuePendingLow = cfg.QueuePendingHigh
	}
	if cfg.QueueOldestLow > cfg.QueueOldestHigh {
		cfg.QueueOldestLow = cfg.QueueOldestHigh
	}
	if cfg.QueueSlopeWindow < bpMinQueueSlopeWindow {
		cfg.QueueSlopeWindow = bpMinQueueSlopeWindow
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

// decideBackpressure 是纯函数迟滞决策（便于单测）。diskPct 为负表示该信号不可用
// （探测失败），不可用时进入判定忽略、解除判定视作已达标（fail-open，不长期误堵）。
func decideBackpressure(current bool, diskPct, ioSomePct float64, cfg BackpressureConfig) bool {
	return decideBackpressureWithQueue(current, diskPct, ioSomePct, QueueSignal{}, cfg).Active
}

func decideBackpressureWithQueue(
	current bool,
	diskPct, ioSomePct float64,
	queue QueueSignal,
	cfg BackpressureConfig,
) BackpressureDecision {
	if !cfg.Enabled {
		return BackpressureDecision{Reason: pressureReasonDisabled}
	}
	if !current {
		if diskPct >= 0 && diskPct >= cfg.DiskHighPct {
			return BackpressureDecision{Active: true, Reason: pressureReasonDisk}
		}
		if ioSomePct >= 0 && ioSomePct >= cfg.IOSomeHighPct {
			return BackpressureDecision{Active: true, Reason: pressureReasonIO}
		}
		if queue.Configured && queue.Available {
			if queue.Stats.OldestPendingAge >= cfg.QueueOldestHigh {
				return BackpressureDecision{Active: true, Reason: pressureReasonQueueOldest}
			}
			// 2 万设备会在槽位边界形成短时同步突发。pending 只是 durable queue
			// 正在吸收流量，单独越线不能证明 worker 已失速；至少持续到最老消息超过
			// 低年龄水位后才启动 503 背压。磁盘、IO 和最大在途数仍独立保护资源。
			if queue.Stats.Pending >= uint64(cfg.QueuePendingHigh) &&
				queue.Stats.OldestPendingAge >= cfg.QueueOldestLow {
				return BackpressureDecision{Active: true, Reason: pressureReasonQueuePending}
			}
		}
		return BackpressureDecision{}
	}

	diskOK := diskPct < 0 || diskPct <= cfg.DiskLowPct
	ioOK := ioSomePct < 0 || ioSomePct <= cfg.IOSomeLowPct
	if !diskOK {
		return BackpressureDecision{Active: true, Reason: pressureReasonDisk}
	}
	if !ioOK {
		return BackpressureDecision{Active: true, Reason: pressureReasonIO}
	}
	if queue.Configured {
		if !queue.Available {
			return BackpressureDecision{Active: true, Reason: pressureReasonQueueFailure}
		}
		if queue.Stats.Pending > uint64(cfg.QueuePendingLow) {
			return BackpressureDecision{Active: true, Reason: pressureReasonQueuePending}
		}
		if queue.Stats.OldestPendingAge > cfg.QueueOldestLow {
			return BackpressureDecision{Active: true, Reason: pressureReasonQueueOldest}
		}
		if !queue.RatesAvailable {
			return BackpressureDecision{Active: true, Reason: pressureReasonQueueSlopeUnknown}
		}
		if queue.Rates.PendingPerSecond > 0 {
			return BackpressureDecision{Active: true, Reason: pressureReasonQueueGrowing}
		}
	}
	return BackpressureDecision{Reason: pressureReasonRecovered}
}

// decideBackpressureState independently applies hysteresis to disk, I/O and
// queue pressure. A signal in its neutral band only keeps its own previously
// latched bit; it cannot keep an unrelated pressure source active.
func decideBackpressureState(
	previous pressureState,
	diskPct, ioSomePct float64,
	queue QueueSignal,
	cfg BackpressureConfig,
) (pressureState, BackpressureDecision) {
	if !cfg.Enabled {
		return 0, BackpressureDecision{Reason: pressureReasonDisabled}
	}

	next := previous
	if diskPct >= 0 {
		if next.has(pressureDisk) && diskPct <= cfg.DiskLowPct {
			next &^= pressureDisk
		} else if diskPct >= cfg.DiskHighPct {
			next |= pressureDisk
		}
	}
	if ioSomePct >= 0 {
		if next.has(pressureIO) && ioSomePct <= cfg.IOSomeLowPct {
			next &^= pressureIO
		} else if ioSomePct >= cfg.IOSomeHighPct {
			next |= pressureIO
		}
	}

	queueReason := ""
	if !queue.Configured {
		next &^= pressureQueue
	} else if next.has(pressureQueue) {
		switch {
		case !queue.Available:
			queueReason = pressureReasonQueueFailure
		case queue.Stats.Pending > uint64(cfg.QueuePendingLow):
			queueReason = pressureReasonQueuePending
		case queue.Stats.OldestPendingAge > cfg.QueueOldestLow:
			queueReason = pressureReasonQueueOldest
		case !queue.RatesAvailable:
			queueReason = pressureReasonQueueSlopeUnknown
		case queue.Rates.PendingPerSecond > 0:
			queueReason = pressureReasonQueueGrowing
		default:
			next &^= pressureQueue
		}
	} else if queue.Available {
		switch {
		case queue.Stats.OldestPendingAge >= cfg.QueueOldestHigh:
			next |= pressureQueue
			queueReason = pressureReasonQueueOldest
		case queue.Stats.Pending >= uint64(cfg.QueuePendingHigh) &&
			queue.Stats.OldestPendingAge >= cfg.QueueOldestLow:
			next |= pressureQueue
			queueReason = pressureReasonQueuePending
		}
	}

	if next == 0 {
		return 0, BackpressureDecision{Reason: pressureReasonRecovered}
	}
	reason := queueReason
	if next.has(pressureDisk) {
		reason = pressureReasonDisk
	} else if next.has(pressureIO) {
		reason = pressureReasonIO
	} else if reason == "" {
		reason = pressureReasonQueuePending
	}
	return next, BackpressureDecision{Active: true, Reason: reason}
}

func deriveQueueRates(previous, current event.QueueStats, window time.Duration) (QueueRates, bool) {
	elapsed := current.SampledAt.Sub(previous.SampledAt)
	if previous.SampledAt.IsZero() || current.SampledAt.IsZero() ||
		elapsed <= 0 || window <= 0 || elapsed > window ||
		current.DeliverySequence < previous.DeliverySequence ||
		current.AckConsumerSequence < previous.AckConsumerSequence {
		return QueueRates{}, false
	}
	seconds := elapsed.Seconds()
	return QueueRates{
		PendingPerSecond:    (float64(current.Pending) - float64(previous.Pending)) / seconds,
		AckAdvancePerSecond: float64(current.AckConsumerSequence-previous.AckConsumerSequence) / seconds,
		DeliveryPerSecond:   float64(current.DeliverySequence-previous.DeliverySequence) / seconds,
	}, true
}

// BackpressureMetrics 暴露背压可观测指标。
type BackpressureMetrics struct {
	active            prometheus.Gauge
	transitions       *prometheus.CounterVec
	rejected          prometheus.Counter
	rejectedByReason  *prometheus.CounterVec
	diskPct           prometheus.Gauge
	ioSomePct         prometheus.Gauge
	inflight          prometheus.Gauge
	loadPerCore       prometheus.Gauge
	queueDeliveryRate prometheus.Gauge
	queueAckRate      prometheus.Gauge
	queueBacklogSlope prometheus.Gauge
	lastAccepted      prometheus.Gauge
}

// NewBackpressureMetrics 注册并返回背压指标。reg 为 nil 时不注册（测试用）。
func NewBackpressureMetrics(reg prometheus.Registerer) *BackpressureMetrics {
	m := &BackpressureMetrics{
		active: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_pm_upload_backpressure_active",
			Help: "PM 上传背压是否生效（1=拒收中，0=正常），issue #318",
		}),
		transitions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acs_pm_upload_backpressure_transitions_total",
			Help: "PM upload backpressure state transitions by target state and deciding reason.",
		}, []string{"state", "reason"}),
		rejected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "acs_pm_upload_backpressure_rejected_total",
			Help: "因背压被拒收的 PM 上传总数（设备会重传，不丢数据）",
		}),
		rejectedByReason: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acs_pm_upload_backpressure_rejected_by_reason_total",
			Help: "因背压被拒收的 PM 上传总数，按 resource_pressure/inflight_limit 分类",
		}, []string{"reason"}),
		diskPct: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_data_disk_usage_percent",
			Help: "watchdog 观测到的数据盘（MinIO）使用率百分比",
		}),
		ioSomePct: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_host_io_pressure_some_avg10_percent",
			Help: "watchdog 观测到的 /proc/pressure/io some avg10 百分比",
		}),
		inflight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_pm_upload_inflight",
			Help: "当前已准入且尚未完成的 PM 上传数",
		}),
		loadPerCore: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_host_load_per_core",
			Help: "watchdog 观测到的主机每核 1 分钟负载（loadavg ÷ NumCPU）",
		}),
		queueDeliveryRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_pm_queue_delivery_rate",
			Help: "PM durable consumer delivery sequence advance per second across fresh monotonic samples.",
		}),
		queueAckRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_pm_queue_ack_rate",
			Help: "PM durable acknowledgement sequence advance per second across fresh monotonic samples.",
		}),
		queueBacklogSlope: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_pm_queue_backlog_slope",
			Help: "PM pending backlog change per second across fresh monotonic samples.",
		}),
		lastAccepted: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acs_pm_upload_last_accepted_timestamp_seconds",
			Help: "Unix timestamp of the latest PM upload successfully accepted into object storage.",
		}),
	}
	for _, label := range [][2]string{
		{"engaged", pressureReasonDisk},
		{"engaged", pressureReasonIO},
		{"engaged", pressureReasonQueuePending},
		{"engaged", pressureReasonQueueOldest},
		{"released", pressureReasonRecovered},
		{"released", pressureReasonDisabled},
	} {
		m.transitions.WithLabelValues(label[0], label[1])
	}
	if reg != nil {
		reg.MustRegister(
			m.active, m.transitions, m.rejected, m.rejectedByReason,
			m.diskPct, m.ioSomePct, m.inflight, m.loadPerCore,
			m.queueDeliveryRate, m.queueAckRate, m.queueBacklogSlope,
			m.lastAccepted,
		)
	}
	return m
}

// Watchdog 周期采样磁盘使用率，带迟滞维护背压态，并对 handler 暴露 Allowed()。
// CPU 负载另行探测并上报 acs_host_load_per_core 指标，但不参与背压决策（#827）。
type Watchdog struct {
	lookup        ConfigLookup
	diskUsage     DiskUsageFunc
	ioPressure    func() (float64, error)
	loadFn        func() (float64, error)
	metrics       *BackpressureMetrics
	logger        *zap.Logger
	queueStats    QueueStatsSource
	previousQueue *event.QueueStats
	queueRates    QueueRates
	queueRatesAt  time.Time
	queueRatesOK  bool
	defaults      BackpressureConfig
	stateMu       sync.Mutex

	cfg                atomic.Pointer[BackpressureConfig]
	active             atomic.Bool   // true = 背压中（拒收 PM）
	rememberedPressure atomic.Uint32 // pressureState retained while Enabled=false
	inflight           atomic.Int64
}

// SetQueueStatsSource wires the independent queue sampler cache into the
// watchdog. The source must not issue queue management I/O from LatestStats.
func (w *Watchdog) SetQueueStatsSource(source QueueStatsSource) {
	if w != nil {
		w.stateMu.Lock()
		defer w.stateMu.Unlock()
		w.queueStats = source
		w.previousQueue = nil
		w.invalidateQueueRates()
	}
}

// SetQueueThresholdDefaults applies deployment-level queue defaults while
// retaining sys_configs as the hot-reload override layer.
func (w *Watchdog) SetQueueThresholdDefaults(
	pendingHigh, pendingLow int,
	oldestHigh, oldestLow, slopeWindow time.Duration,
) {
	if w == nil {
		return
	}
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	defaults := w.defaults
	if pendingHigh > 0 {
		defaults.QueuePendingHigh = pendingHigh
	}
	if pendingLow >= 0 {
		defaults.QueuePendingLow = pendingLow
	}
	if oldestHigh > 0 {
		defaults.QueueOldestHigh = oldestHigh
	}
	if oldestLow >= 0 {
		defaults.QueueOldestLow = oldestLow
	}
	if slopeWindow > 0 {
		defaults.QueueSlopeWindow = slopeWindow
	}
	w.defaults = defaults
	cfg := loadBackpressureConfigWithDefaults(context.Background(), w.lookup, defaults)
	w.cfg.Store(&cfg)
}

// NewWatchdog 构造 watchdog。diskUsage 为 nil 时跳过磁盘空间信号，Linux IO PSI 和
// 最大在途数仍继续提供保护。
func NewWatchdog(lookup ConfigLookup, diskUsage DiskUsageFunc, metrics *BackpressureMetrics, logger *zap.Logger) *Watchdog {
	if logger == nil {
		logger = zap.NewNop()
	}
	w := &Watchdog{
		lookup:     lookup,
		diskUsage:  diskUsage,
		ioPressure: ioSomeAvg10,
		loadFn:     loadPerCore,
		metrics:    metrics,
		logger:     logger,
		defaults:   defaultBackpressureConfig(),
	}
	cfg := loadBackpressureConfigWithDefaults(context.Background(), lookup, w.defaults)
	w.cfg.Store(&cfg)
	return w
}

// Acquire 实现 BackpressureGate：在 watchdog 未背压时用 CAS 获取一个有界在途槽位。
func (w *Watchdog) Acquire() (bool, string) {
	if w == nil || w.active.Load() {
		return false, rejectReasonResourcePressure
	}
	limit := w.cfg.Load().MaxInflight
	for {
		cur := w.inflight.Load()
		if cur >= limit {
			return false, rejectReasonInflightLimit
		}
		if w.inflight.CompareAndSwap(cur, cur+1) {
			if w.metrics != nil {
				w.metrics.inflight.Set(float64(cur + 1))
			}
			// 采样态可能在 CAS 期间切换；撤销该槽位，避免新流量穿透已生效的背压。
			if w.active.Load() {
				w.Release()
				return false, rejectReasonResourcePressure
			}
			return true, ""
		}
	}
}

// Release 释放一个成功取得的槽位；重复释放被钳制在 0，避免监控出现负数。
func (w *Watchdog) Release() {
	if w == nil {
		return
	}
	for {
		cur := w.inflight.Load()
		if cur <= 0 {
			return
		}
		if w.inflight.CompareAndSwap(cur, cur-1) {
			if w.metrics != nil {
				w.metrics.inflight.Set(float64(cur - 1))
			}
			return
		}
	}
}

// RecordRejected 记录一次背压拒收。
func (w *Watchdog) RecordRejected(reason string) {
	if w.metrics != nil {
		w.metrics.rejected.Inc()
		if reason == "" {
			reason = rejectReasonResourcePressure
		}
		w.metrics.rejectedByReason.WithLabelValues(reason).Inc()
	}
}

func (w *Watchdog) RecordAccepted(at time.Time) {
	if w != nil && w.metrics != nil {
		w.metrics.lastAccepted.Set(float64(at.Unix()))
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
	w.stateMu.Lock()
	cfg := loadBackpressureConfigWithDefaults(ctx, w.lookup, w.defaults)
	w.cfg.Store(&cfg)
	w.stateMu.Unlock()

	diskPct := -1.0
	if cfg.Enabled && w.diskUsage != nil {
		if p, err := w.diskUsage(ctx); err != nil {
			w.logger.Warn("disk usage probe failed; skip disk signal", zap.Error(err))
		} else {
			diskPct = p
			if w.metrics != nil {
				w.metrics.diskPct.Set(p)
			}
		}
	}
	ioSomePct := -1.0
	if cfg.Enabled && w.ioPressure != nil {
		if p, err := w.ioPressure(); err != nil {
			w.logger.Warn("io pressure probe failed; skip PSI signal", zap.Error(err))
		} else {
			ioSomePct = p
			if w.metrics != nil {
				w.metrics.ioSomePct.Set(p)
			}
		}
	}

	// CPU 负载只更新监控指标，不参与背压决策（#827）。
	if w.loadFn != nil {
		if l, err := w.loadFn(); err != nil {
			w.logger.Warn("cpu load probe failed", zap.Error(err))
		} else if w.metrics != nil {
			w.metrics.loadPerCore.Set(l)
		}
	}

	queueSignal := w.sampleQueueSignal(time.Now(), cfg)
	prevActive := w.active.Load()
	decision := BackpressureDecision{Reason: pressureReasonDisabled}
	next := false
	if cfg.Enabled {
		previous := pressureState(w.rememberedPressure.Load())
		// Compatibility for tests and any caller that restored only the public
		// active bit before the independent latch state existed.
		if previous == 0 && prevActive {
			if queueSignal.Configured {
				previous = pressureQueue
			} else {
				previous = pressureDisk | pressureIO
			}
		}
		state, stateDecision := decideBackpressureState(
			previous, diskPct, ioSomePct, queueSignal, cfg,
		)
		decision = stateDecision
		next = state != 0
		w.rememberedPressure.Store(uint32(state))
	}
	if next != prevActive {
		w.active.Store(next)
		if w.metrics != nil {
			state := "released"
			if next {
				state = "engaged"
			}
			w.metrics.transitions.WithLabelValues(state, decision.Reason).Inc()
		}
		if next {
			w.logger.Warn("PM upload backpressure ENGAGED — rejecting PM uploads (devices will retry)",
				zap.Float64("disk_pct", diskPct), zap.Float64("io_some_avg10_pct", ioSomePct),
				zap.Float64("disk_high_pct", cfg.DiskHighPct), zap.Float64("io_some_high_pct", cfg.IOSomeHighPct),
				zap.String("reason", decision.Reason))
		} else if decision.Reason == pressureReasonDisabled {
			w.logger.Info("PM upload backpressure RELEASED (disabled by config)")
		} else {
			w.logger.Info("PM upload backpressure RELEASED — resuming PM uploads",
				zap.Float64("disk_pct", diskPct), zap.String("reason", decision.Reason))
		}
	}
	w.setActiveGauge(next)
}

func (w *Watchdog) sampleQueueSignal(now time.Time, cfg BackpressureConfig) QueueSignal {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	signal := QueueSignal{Configured: w.queueStats != nil}
	if w.queueStats == nil {
		w.invalidateQueueRates()
		return signal
	}
	stats, ok := w.queueStats.LatestStats()
	age := now.Sub(stats.SampledAt)
	newerAttemptFailed := false
	if attempts, exposesAttempts := w.queueStats.(queueSampleAttemptSource); exposesAttempts {
		attemptedAt, succeeded := attempts.LastSampleAttempt()
		newerAttemptFailed = !attemptedAt.IsZero() && attemptedAt.After(stats.SampledAt) && !succeeded
	}
	if !ok || stats.SampledAt.IsZero() || age > cfg.QueueSlopeWindow || newerAttemptFailed {
		w.invalidateQueueRates()
		return signal
	}
	if w.previousQueue != nil && stats.SampledAt.Before(w.previousQueue.SampledAt) {
		w.invalidateQueueRates()
		return signal
	}
	signal.Available = true
	signal.Stats = stats
	if w.previousQueue != nil && stats.SampledAt.After(w.previousQueue.SampledAt) {
		signal.Rates, signal.RatesAvailable = deriveQueueRates(*w.previousQueue, stats, cfg.QueueSlopeWindow)
		if signal.RatesAvailable {
			w.queueRates = signal.Rates
			w.queueRatesAt = stats.SampledAt
			w.queueRatesOK = true
		} else {
			w.invalidateQueueRates()
		}
	} else if w.queueRatesOK && w.queueRatesAt.Equal(stats.SampledAt) {
		signal.Rates = w.queueRates
		signal.RatesAvailable = true
	}
	if w.previousQueue == nil || stats.SampledAt.After(w.previousQueue.SampledAt) {
		snapshot := stats
		w.previousQueue = &snapshot
	}
	if w.metrics != nil {
		if signal.RatesAvailable {
			w.metrics.queueDeliveryRate.Set(signal.Rates.DeliveryPerSecond)
			w.metrics.queueAckRate.Set(signal.Rates.AckAdvancePerSecond)
			w.metrics.queueBacklogSlope.Set(signal.Rates.PendingPerSecond)
		} else {
			w.invalidateQueueRates()
		}
	}
	return signal
}

func (w *Watchdog) invalidateQueueRates() {
	w.queueRates = QueueRates{}
	w.queueRatesAt = time.Time{}
	w.queueRatesOK = false
	if w.metrics == nil {
		return
	}
	w.metrics.queueDeliveryRate.Set(math.NaN())
	w.metrics.queueAckRate.Set(math.NaN())
	w.metrics.queueBacklogSlope.Set(math.NaN())
}

// ioSomeAvg10 读取 Linux PSI 的 IO some avg10。非 Linux 或未启用 PSI 时返回错误，
// watchdog 对该信号 fail-open，仍保留磁盘空间和在途并发保护。
func ioSomeAvg10() (float64, error) {
	data, err := os.ReadFile("/proc/pressure/io")
	if err != nil {
		return 0, err
	}
	return parseIOSomeAvg10(string(data))
}

func parseIOSomeAvg10(text string) (float64, error) {
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "some" {
			continue
		}
		for _, field := range fields[1:] {
			if !strings.HasPrefix(field, "avg10=") {
				continue
			}
			value, err := strconv.ParseFloat(strings.TrimPrefix(field, "avg10="), 64)
			if err != nil {
				return 0, fmt.Errorf("parse io pressure avg10: %w", err)
			}
			return value, nil
		}
	}
	return 0, fmt.Errorf("io pressure some avg10 not found")
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
	return NewProjectedMinIODiskUsage(metricsURL, timeout, client, nil)
}

// NewProjectedMinIODiskUsage adds accepted-but-not-yet-materialized bytes to
// the filesystem usage reported by MinIO. Database temp files, WAL and
// filesystem preallocation already consume that same filesystem and are
// therefore included in the reported used bytes rather than double-counted.
func NewProjectedMinIODiskUsage(
	metricsURL string,
	timeout time.Duration,
	client *http.Client,
	projected ProjectedBytesFunc,
) DiskUsageFunc {
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
		total, used, err := parseDiskCapacity(string(body))
		if err != nil {
			return 0, err
		}
		if projected != nil {
			if pending, projectionErr := projected(ctx); projectionErr == nil && pending > 0 {
				used += pending
			}
		}
		return projectedUsagePct(total, used, 0), nil
	}
}

// NewDatabasePendingProjection estimates future TSDB bytes from PM work that
// has already been accepted. JetStream backlog is projected using the recent
// average PM file size; legacy pm_files rows not parsed yet are added directly.
// amplification should be derived from the latest sparse-storage observation;
// invalid values use a conservative 1.0.
func NewDatabasePendingProjection(
	tsdb *pgxpool.Pool,
	pendingCount PendingCountFunc,
	amplification float64,
) ProjectedBytesFunc {
	if amplification <= 0 {
		amplification = 1
	}
	return func(ctx context.Context) (float64, error) {
		var pending float64
		if tsdb != nil && pendingCount != nil {
			if count, err := pendingCount(ctx); err == nil && count > 0 {
				var averageFileSize float64
				err = tsdb.QueryRow(ctx, `
					SELECT COALESCE(NULLIF(avg(file_size),0), $1)::float8
					  FROM (
					    SELECT file_size
					      FROM pm_files
					     WHERE parsed = true
					       AND file_size > 0
					     ORDER BY created_at DESC
					     LIMIT 1000
					  ) recent`,
					float64(1<<20),
				).Scan(&averageFileSize)
				if err == nil {
					pending += float64(count) * averageFileSize
				}
			}
		}
		if tsdb != nil {
			var bytes float64
			err := tsdb.QueryRow(ctx,
				`SELECT COALESCE(sum(file_size),0)::float8 FROM pm_files WHERE parsed=false`,
			).Scan(&bytes)
			if err == nil {
				pending += bytes
			}
		}
		return pending * amplification, nil
	}
}

// parseDiskUsagePct 从 MinIO 集群指标文本计算磁盘使用率%。按指标版本差异依次尝试多组容量指标：
// usable 容量 → raw 容量 → 节点磁盘 free → 节点磁盘 used，取首个可用组。
func parseDiskUsagePct(text string) (float64, error) {
	total, used, err := parseDiskCapacity(text)
	if err != nil {
		return 0, err
	}
	return projectedUsagePct(total, used, 0), nil
}

func projectedUsagePct(total, used, pending float64) float64 {
	if total <= 0 {
		return 0
	}
	return clampPct((used + pending) / total * 100)
}

func parseDiskCapacity(text string) (total, used float64, err error) {
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
		return t, t - sums["minio_cluster_capacity_usable_free_bytes"], nil
	}
	if t := sums["minio_cluster_capacity_raw_total_bytes"]; t > 0 && present["minio_cluster_capacity_raw_free_bytes"] {
		return t, t - sums["minio_cluster_capacity_raw_free_bytes"], nil
	}
	if t := sums["minio_node_disk_total_bytes"]; t > 0 {
		if present["minio_node_disk_free_bytes"] {
			return t, t - sums["minio_node_disk_free_bytes"], nil
		}
		if present["minio_node_disk_used_bytes"] {
			return t, sums["minio_node_disk_used_bytes"], nil
		}
	}
	return 0, 0, fmt.Errorf("no usable minio capacity metrics found")
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
