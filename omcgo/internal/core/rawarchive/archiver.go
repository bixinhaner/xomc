// Package rawarchive 在 PM/MR 原始文件入库成功后，把仍是明文的 XML gzip 压缩回写 MinIO
// 省盘（issue #321）。
//
// 背景：真机 CPE / cpe_simulator.py 多按 TR-069 上传 .xml.gz，MinIO 原样存压缩字节——这类
// 对象本包零成本跳过（前 2 字节探到 gzip 魔数即跳过整文件读写）。仅合成压测等明文上传的
// 对象才会被 gzip 后按原 key 覆盖写（设 ContentEncoding=gzip）。入库读取侧由
// core/compress.MaybeGunzip 透明解压，故压缩态对象后续仍可正确再解析。
//
// 设计取舍：压缩是纯省盘优化，绝不影响已入库数据。
//   - 异步 + 有界并发：不阻塞 NATS 入库回调 ack；并发满则直接丢弃本次压缩（ILM 仍兜底，
//     见 #319），不积压 goroutine。
//   - 失败只 warn + 记 metric，不回报错误。
//   - enabled 走 sys_configs（category=raw_archive, key=compress_after_ingest，默认 true），
//     TTL 缓存避免每文件查库。
package rawarchive

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/compress"
)

const (
	// Category / KeyEnabled 是 sys_configs 中控制压缩回写的键。
	Category   = "raw_archive"
	KeyEnabled = "compress_after_ingest"

	defaultEnabled = true
	configTTL      = 30 * time.Second
	probeBytes     = 2                // gzip 魔数长度
	maxObjectBytes = 64 << 20         // 64MiB 安全上限（对齐上传体积限），超限不压缩防 OOM
	archiveTimeout = 30 * time.Second // 单文件压缩回写超时
)

// RawStore 是 archiver 需要的 MinIO 子集（便于单测注入 fake）。
type RawStore interface {
	// ReadHead 返回对象前 n 字节（用于探测 gzip 魔数，避免整文件读）。
	ReadHead(ctx context.Context, bucket, object string, n int) ([]byte, error)
	// Get 返回对象完整字节流。
	Get(ctx context.Context, bucket, object string) (io.ReadCloser, error)
	// Put 按原 key 覆盖写，contentEncoding 落对象元数据。
	Put(ctx context.Context, bucket, object string, r io.Reader, size int64, contentType, contentEncoding string) error
}

// ConfigLookup 读 sys_configs 单值（value, found）。
type ConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// Metrics 暴露压缩回写的可观测指标。
type Metrics struct {
	Total      *prometheus.CounterVec // result=compressed|skipped_gz|disabled|empty|too_large|no_gain|dropped_busy|error
	SavedBytes prometheus.Counter
}

// NewMetrics 注册并返回压缩回写指标。reg 为 nil 时不注册（仅供测试）。
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "omc_raw_archive_total",
			Help: "原始文件入库后压缩回写结果计数（issue #321），label result 区分压缩/跳过/失败原因",
		}, []string{"result"}),
		SavedBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "omc_raw_archive_saved_bytes_total",
			Help: "原始文件压缩回写累计省盘字节数（原始大小 - 压缩后大小）",
		}),
	}
	if reg != nil {
		reg.MustRegister(m.Total, m.SavedBytes)
	}
	return m
}

// Archiver 异步、有界并发地对入库完成的原始对象做压缩回写。零值不可用，须经 New 构造。
type Archiver struct {
	baseCtx context.Context
	store   RawStore
	lookup  ConfigLookup
	metrics *Metrics
	logger  *zap.Logger
	sem     chan struct{}

	mu       sync.Mutex
	cached   bool
	cachedAt time.Time
}

// New 构造 archiver。concurrency<=0 时回落为 1。store 为 nil 时返回的 archiver 是无操作的
// （Schedule 直接返回），便于在未接 MinIO 的场景安全注入。
func New(baseCtx context.Context, store RawStore, lookup ConfigLookup, metrics *Metrics, logger *zap.Logger, concurrency int) *Archiver {
	if concurrency <= 0 {
		concurrency = 1
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Archiver{
		baseCtx: baseCtx,
		store:   store,
		lookup:  lookup,
		metrics: metrics,
		logger:  logger,
		sem:     make(chan struct{}, concurrency),
	}
}

// Schedule 非阻塞地排程一次压缩回写。并发已满则丢弃本次（记 dropped_busy，ILM 仍兜底）。
func (a *Archiver) Schedule(bucket, object string) {
	if a == nil || a.store == nil || bucket == "" || object == "" {
		return
	}
	select {
	case a.sem <- struct{}{}:
		go func() {
			defer func() { <-a.sem }()
			base := a.baseCtx
			if base == nil {
				base = context.Background()
			}
			ctx, cancel := context.WithTimeout(base, archiveTimeout)
			defer cancel()
			a.archive(ctx, bucket, object)
		}()
	default:
		a.record("dropped_busy", 0)
	}
}

// archive 执行一次压缩回写的完整逻辑（同步）。
func (a *Archiver) archive(ctx context.Context, bucket, object string) {
	if !a.enabled(ctx) {
		a.record("disabled", 0)
		return
	}

	// 廉价探测：前 2 字节命中 gzip 魔数 → 对象已压缩（真机常态），跳过整文件读写。
	head, err := a.store.ReadHead(ctx, bucket, object, probeBytes)
	if err != nil {
		a.warn("probe head", bucket, object, err)
		a.record("error", 0)
		return
	}
	if len(head) == 0 {
		a.record("empty", 0)
		return
	}
	if compress.IsGzip(head) {
		a.record("skipped_gz", 0)
		return
	}

	// 明文：整读（受上限约束）→ gzip → 覆盖写。
	rc, err := a.store.Get(ctx, bucket, object)
	if err != nil {
		a.warn("get object", bucket, object, err)
		a.record("error", 0)
		return
	}
	raw, rerr := io.ReadAll(io.LimitReader(rc, maxObjectBytes+1))
	_ = rc.Close()
	if rerr != nil {
		a.warn("read object", bucket, object, rerr)
		a.record("error", 0)
		return
	}
	if len(raw) > maxObjectBytes {
		a.record("too_large", 0)
		return
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, werr := zw.Write(raw); werr != nil {
		_ = zw.Close()
		a.warn("gzip write", bucket, object, werr)
		a.record("error", 0)
		return
	}
	if cerr := zw.Close(); cerr != nil {
		a.warn("gzip close", bucket, object, cerr)
		a.record("error", 0)
		return
	}
	gz := buf.Bytes()
	if len(gz) >= len(raw) {
		// 压不动（极小/高熵）→ 不回写，避免越压越大。
		a.record("no_gain", 0)
		return
	}

	if perr := a.store.Put(ctx, bucket, object, bytes.NewReader(gz), int64(len(gz)), "application/xml", "gzip"); perr != nil {
		a.warn("put compressed", bucket, object, perr)
		a.record("error", 0)
		return
	}

	saved := int64(len(raw) - len(gz))
	a.record("compressed", saved)
	a.logger.Debug("raw file compressed in place",
		zap.String("bucket", bucket), zap.String("object", object),
		zap.Int("orig_bytes", len(raw)), zap.Int("gz_bytes", len(gz)), zap.Int64("saved_bytes", saved))
}

// enabled 返回压缩回写开关，结果 TTL 缓存避免每文件查 sys_configs。
func (a *Archiver) enabled(ctx context.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.cachedAt.IsZero() && time.Since(a.cachedAt) < configTTL {
		return a.cached
	}
	val := defaultEnabled
	if a.lookup != nil {
		if v, found := a.lookup(ctx, Category, KeyEnabled); found {
			val = parseBool(v, defaultEnabled)
		}
	}
	a.cached = val
	a.cachedAt = time.Now()
	return val
}

func (a *Archiver) record(result string, saved int64) {
	if a.metrics == nil {
		return
	}
	a.metrics.Total.WithLabelValues(result).Inc()
	if saved > 0 {
		a.metrics.SavedBytes.Add(float64(saved))
	}
}

func (a *Archiver) warn(stage, bucket, object string, err error) {
	a.logger.Warn("raw archive: "+stage+" failed (skip compress, data already ingested)",
		zap.String("bucket", bucket), zap.String("object", object), zap.Error(err))
}

func parseBool(s string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return def
	}
}
