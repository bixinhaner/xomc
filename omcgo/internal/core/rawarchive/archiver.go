// Package rawarchive 在 PM/MR 原始文件入库成功后，把仍是明文的 XML gzip 压缩回写 MinIO
// 省盘（issue #321）。
//
// 背景：真机 CPE / cpe_simulator.py 多按 TR-069 上传 .xml.gz，MinIO 原样存压缩字节——这类
// 对象本包零成本跳过（前 2 字节探到 gzip 魔数即跳过整文件读写）。仅合成压测等明文上传的
// 对象才会被 gzip 后按原 key 覆盖写（设 ContentEncoding=gzip）。入库读取侧由
// core/compress.MaybeGunzip 透明解压，故压缩态对象后续仍可正确再解析。
//
// 设计取舍：压缩是纯省盘优化，绝不影响已入库数据。
//   - 异步 + 有界并发：不阻塞 NATS 入库回调 ack；并发满则丢弃本次内联压缩（记 dropped_busy），
//     不积压 goroutine —— 丢弃的不会漏压，由 Sweeper 兜底（见下）。
//   - 失败只 warn + 记 metric，不回报错误。
//   - enabled 走 sys_configs（category=raw_archive, key=compress_after_ingest，默认 true），
//     TTL 缓存避免每文件查库。
//
// 保证压到（issue #321 加固）：内联 Schedule 是"快路径"，压成功后在 pm_files/mr_files
// 标记 raw_compressed=true；Sweeper（sweeper.go）是"兜底"——周期性补压所有 raw_compressed=false
// 的残量（内联 dropped_busy / 失败 / 崩溃前未压），直至置真。两者复用同一压缩核心 compress()，
// 终态判定（terminal）一致：成功压缩 / 本就是 gzip / 明确无需压（no_gain/too_large/empty）/
// 对象已删（not_found 孤儿行）皆为终态可标记；disabled / error 非终态，留待重扫。故每个原始
// XML 最终都被压缩，不再有静默漏压，Sweeper 也不会在孤儿行处卡死。
package rawarchive

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
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

	defaultEnabled   = true
	configTTL        = 30 * time.Second
	probeBytes       = 2                // gzip 魔数长度
	maxObjectBytes   = 64 << 20         // 64MiB 安全上限（对齐上传体积限），超限不压缩防 OOM
	archiveTimeout   = 30 * time.Second // 单文件压缩回写超时
	gzSuffix         = ".gz"            // 压缩后对象键追加的后缀（.xml → .xml.gz，自描述）
	removeOldTimeout = 10 * time.Second // 删旧明文键的独立短超时（与压缩/扫描 ctx 解耦）
)

// outcome 是单次压缩回写的结果分类。terminal 的结果意味着"该对象的压缩状态已确定"
// （成功压缩 / 本就是 gzip / 明确决定不压）——可在 pm_files/mr_files 标记 raw_compressed=true
// 并移出 Sweeper 待扫描集；非 terminal（disabled / error）保持未压缩，留待 Sweeper 重试。
// dropped_busy 不进入 compress()（在 Schedule 的并发门处直接丢弃），故不在此枚举。
type outcome int

const (
	outcomeError      outcome = iota // 读/写/压缩失败（非终态，待重扫）
	outcomeDisabled                  // 总开关关闭（非终态，开关恢复后重扫）
	outcomeEmpty                     // 对象 0 字节（终态，无可压）
	outcomeSkippedGz                 // 已是 gzip（终态，真机 .xml.gz 常态）
	outcomeCompressed                // 明文压缩并覆盖回写成功（终态）
	outcomeNoGain                    // 压不动（gzip 头开销 ≥ 收益，终态，保留明文）
	outcomeTooLarge                  // 超 maxObjectBytes 上限（终态，保留明文防 OOM）
	outcomeNotFound                  // 对象已不存在（孤儿行，终态，标记后移出待扫集）
)

// terminal 报告该结果是否表示"压缩状态已确定"，可标记 raw_compressed=true。
func (o outcome) terminal() bool {
	switch o {
	case outcomeCompressed, outcomeSkippedGz, outcomeNoGain, outcomeTooLarge, outcomeEmpty, outcomeNotFound:
		return true
	default: // outcomeError / outcomeDisabled
		return false
	}
}

// label 返回 omc_raw_archive_total 的 result 标签（与历史标签字符串保持一致）。
func (o outcome) label() string {
	switch o {
	case outcomeCompressed:
		return "compressed"
	case outcomeSkippedGz:
		return "skipped_gz"
	case outcomeNoGain:
		return "no_gain"
	case outcomeTooLarge:
		return "too_large"
	case outcomeEmpty:
		return "empty"
	case outcomeNotFound:
		return "not_found"
	case outcomeDisabled:
		return "disabled"
	default:
		return "error"
	}
}

// RawStore 是 archiver 需要的 MinIO 子集（便于单测注入 fake）。
type RawStore interface {
	// ReadHead 返回对象前 n 字节（用于探测 gzip 魔数，避免整文件读）。
	ReadHead(ctx context.Context, bucket, object string, n int) ([]byte, error)
	// Get 返回对象完整字节流。
	Get(ctx context.Context, bucket, object string) (io.ReadCloser, error)
	// Put 写对象（压缩回写到新键 object+".gz"），contentEncoding 落对象元数据。
	Put(ctx context.Context, bucket, object string, r io.Reader, size int64, contentType, contentEncoding string) error
	// Remove 删除对象（改键后清理旧明文键）。对象不存在应视为成功（幂等）。
	Remove(ctx context.Context, bucket, object string) error
}

// ConfigLookup 读 sys_configs 单值（value, found）。
type ConfigLookup func(ctx context.Context, category, key string) (value string, found bool)

// Metrics 暴露压缩回写的可观测指标。
type Metrics struct {
	Total      *prometheus.CounterVec // result=compressed|skipped_gz|disabled|empty|too_large|no_gain|not_found|dropped_busy|error
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

// Schedule 非阻塞地排程一次压缩回写（内联快路径）。并发已满则丢弃本次（记 dropped_busy）——
// 丢弃的对象不会漏压，Sweeper 会兜底补压（见包注释）。onTerminal 在压缩抵达终态后于压缩
// goroutine 内回调，传入 (bucket, 旧键, 新键)：压缩成功改键时新键为 object+".gz"，其余终态新键==旧键。
// 透传 bucket 是为了让调用方对正确的桶删旧键（调用方的 c.bucket 未必等于本次 Schedule 的 bucket，
// 如 MR 用 payload.Bucket）。供调用方在 pm_files/mr_files 标记 raw_compressed=true 并把 minio_path
// 更新为新键、再删旧键；nil-safe，非终态（error/disabled）或 dropped_busy 不回调。
func (a *Archiver) Schedule(bucket, object string, onTerminal func(ctx context.Context, bucket, oldObject, newObject string)) {
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
			oc, saved, newObject := a.compress(ctx, bucket, object)
			a.record(oc.label(), saved)
			if oc.terminal() && onTerminal != nil {
				onTerminal(ctx, bucket, object, newObject)
			}
		}()
	default:
		a.record("dropped_busy", 0)
	}
}

// CompressNow 同步压缩一个对象并记 metric（供 Sweeper 补偿扫描用），返回 (是否终态, 压缩后对象键)。
// 终态即可在 pm_files/mr_files 标记 raw_compressed=true 并把 minio_path 更新为返回的新键（压缩成功
// 改键时为 object+".gz"，否则==object），随后删旧键；非终态（禁用 / 读写错误）返回 false，保持
// raw_compressed=false 待下一轮重扫。
func (a *Archiver) CompressNow(ctx context.Context, bucket, object string) (bool, string) {
	if a == nil || a.store == nil || bucket == "" || object == "" {
		return false, object
	}
	oc, saved, newObject := a.compress(ctx, bucket, object)
	a.record(oc.label(), saved)
	return oc.terminal(), newObject
}

// Enabled 返回压缩回写总开关（供 Sweeper 每轮扫描前短路，禁用时不空转扫描）。
func (a *Archiver) Enabled(ctx context.Context) bool {
	if a == nil {
		return false
	}
	return a.enabled(ctx)
}

// archive 同步执行一次压缩回写并记 metric（保留供内部直调与单测）。
func (a *Archiver) archive(ctx context.Context, bucket, object string) {
	oc, saved, _ := a.compress(ctx, bucket, object)
	a.record(oc.label(), saved)
}

// RemoveOld 删除改键后遗留的旧明文对象（object 压缩改名为 object+".gz" 后的清理）。必须在 DB 的
// minio_path 已更新为新键之后调用，否则会删掉仍被引用的对象。失败仅 warn + 记 metric——残留旧对象
// 只是占盘、不影响功能（DB 已指向新键），可被后续 retention/GC 清理。nil-safe。
func (a *Archiver) RemoveOld(ctx context.Context, bucket, object string) {
	if a == nil || a.store == nil || bucket == "" || object == "" {
		return
	}
	// 删旧键是 best-effort 清理，但必须与压缩/扫描的 ctx 生命周期解耦：压缩 ctx 临近 30s 到期、
	// 或 worker 优雅关停 cancel(baseCtx)，都不应在「DB 已切新键」之后打断删旧键（否则留旧明文孤儿，
	// 虽有 MinIO ILM 兜底回收，仍应尽量避免）。WithoutCancel 保留 ctx 的 trace/值、仅解除取消传播，
	// 再叠加独立短超时。代价：关停期最多延后一个 removeOldTimeout 退出，可接受。
	rmCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), removeOldTimeout)
	defer cancel()
	if err := a.store.Remove(rmCtx, bucket, object); err != nil {
		a.warn("remove old object", bucket, object, err)
		a.record("remove_old_error", 0)
	}
}

// compress 是压缩回写的核心逻辑（同步，返回结果分类、省下字节、压缩后对象键）——由
// Schedule（内联快路径）、CompressNow（Sweeper 兜底）、archive（单测）共用，保证三路语义一致。
// 压缩成功时回写到新键 object+".gz"（自描述，不覆盖原明文键），第三个返回值即新键；其余情况
// 返回值==object（未改键）。原明文键由调用方在 DB minio_path 更新成功后经 RemoveOld 删除。
func (a *Archiver) compress(ctx context.Context, bucket, object string) (outcome, int64, string) {
	if !a.enabled(ctx) {
		return outcomeDisabled, 0, object
	}

	// 廉价探测：前 2 字节命中 gzip 魔数 → 对象已压缩（真机常态），跳过整文件读写。
	head, err := a.store.ReadHead(ctx, bucket, object, probeBytes)
	if err != nil {
		// 对象已被删除（孤儿行）→ 终态，让 Sweeper 标记移出，不永久重扫卡死。
		if errors.Is(err, ErrObjectNotFound) {
			return outcomeNotFound, 0, object
		}
		a.warn("probe head", bucket, object, err)
		return outcomeError, 0, object
	}
	if len(head) == 0 {
		return outcomeEmpty, 0, object
	}
	if compress.IsGzip(head) {
		return outcomeSkippedGz, 0, object
	}

	// 明文：整读（受上限约束）→ gzip → 写到新键。
	rc, err := a.store.Get(ctx, bucket, object)
	if err != nil {
		a.warn("get object", bucket, object, err)
		return outcomeError, 0, object
	}
	raw, rerr := io.ReadAll(io.LimitReader(rc, maxObjectBytes+1))
	_ = rc.Close()
	if rerr != nil {
		a.warn("read object", bucket, object, rerr)
		return outcomeError, 0, object
	}
	if len(raw) > maxObjectBytes {
		return outcomeTooLarge, 0, object
	}

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, werr := zw.Write(raw); werr != nil {
		_ = zw.Close()
		a.warn("gzip write", bucket, object, werr)
		return outcomeError, 0, object
	}
	if cerr := zw.Close(); cerr != nil {
		a.warn("gzip close", bucket, object, cerr)
		return outcomeError, 0, object
	}
	gz := buf.Bytes()
	if len(gz) >= len(raw) {
		// 压不动（极小/高熵）→ 不回写，避免越压越大。
		return outcomeNoGain, 0, object
	}

	// 改键回写：写到 object+".gz"（自描述），不覆盖原明文键。原键由调用方在 DB minio_path 更新
	// 成功后删除（RemoveOld），保证「DB 永远指向已存在的对象」、崩溃可由 Sweeper 重扫自愈。
	// 防御：object 已以 .gz 结尾（理论上明文不会）则不再追加，覆盖原键。
	newObject := object
	if !strings.HasSuffix(strings.ToLower(object), gzSuffix) {
		newObject = object + gzSuffix
	}
	if perr := a.store.Put(ctx, bucket, newObject, bytes.NewReader(gz), int64(len(gz)), "application/xml", "gzip"); perr != nil {
		a.warn("put compressed", bucket, newObject, perr)
		return outcomeError, 0, object
	}

	saved := int64(len(raw) - len(gz))
	a.logger.Debug("raw file compressed",
		zap.String("bucket", bucket), zap.String("object", object), zap.String("new_object", newObject),
		zap.Int("orig_bytes", len(raw)), zap.Int("gz_bytes", len(gz)), zap.Int64("saved_bytes", saved))
	return outcomeCompressed, saved, newObject
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
