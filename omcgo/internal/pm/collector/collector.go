package collector

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/compress"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/rawarchive"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"github.com/omcgo/omcgo/internal/core/reliability/runner"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// DeviceRegistrationGrace is how long a PM event may wait for the matching
// device registration before it becomes a permanent, replayable failure.
// Queue retry budgets must cover this duration.
const DeviceRegistrationGrace = 30 * time.Minute

// isMinIONotFound 判定 MinIO 错误是否为对象/桶不存在（沿用 internal/core/rawarchive、
// internal/backup/restore_service.go 已有的同款判定：minio-go 的 GetObject 不立即发
// 请求，对象不存在的错误在首次 Read 时才暴露，故调用点在 MaybeGunzip/Parse 读取失败
// 处判定，而不是 GetObject 调用本身的返回值）。
//
// 用 errors.As 而非直接 minio.ToErrorResponse(err).Code：MaybeGunzip/Parse 会用
// fmt.Errorf("...: %w", err) 逐层包装底层读错误（如 "parse pm xml: decode pm xml: %w"），
// minio.ToErrorResponse 内部是裸类型断言，遇到包装过的 error 会直接判定失败——必须先
// errors.As 拆到底层 minio.ErrorResponse 再取 Code，否则本判定在真实调用链路上永远
// 返回 false（2026-07-22 补单测时验证过这个坑，是本次改动最容易踩空的地方）。
func isMinIONotFound(err error) bool {
	if err == nil {
		return false
	}
	var resp minio.ErrorResponse
	if !errors.As(err, &resp) {
		return false
	}
	return resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket"
}

// FileReceivedPayload is the event payload for pm.file.received.
//
// Two publish paths exist with different completeness levels:
//
//   - transfer.Bridge (worker process, T-0164-P3): "fat" payload, all fields
//     pre-filled from device repository lookup before publish.
//   - acs.upload.Handler (ACS process, T-0164 G1 真机闭环): "thin" payload —
//     only MinIOPath / Bucket / DeviceSN / FileSize / FileName. The ACS hot
//     path can't afford a synchronous DB lookup, so DeviceID / DeviceOUI /
//     Carrier / Technology are left empty and the collector resolves them
//     via DeviceLookup (set by worker main).
type FileReceivedPayload struct {
	MinIOPath  string `json:"minio_path"`
	Bucket     string `json:"bucket,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	FileSize   int64  `json:"file_size,omitempty"`
	DeviceID   string `json:"device_id"`
	DeviceOUI  string `json:"device_oui"` // T-0164-P3: TR-069 标准设备唯一标识 (oui, sn) 双键
	DeviceSN   string `json:"device_sn"`
	Carrier    string `json:"carrier"`
	Technology string `json:"technology"`
}

// DeviceLookup resolves a device by serial number. Used by PMCollector to
// fill in DeviceID / DeviceOUI / Carrier / Technology when the upstream
// publisher (acs.upload.Handler) only had the SN at publish time.
//
// Implemented by device.DeviceRepository — but we declare a minimal interface
// here to avoid the worker module taking a hard dep on the entire device
// package internals just for this one method.
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*model.Device, error)
}

// FileMarkerLookup checks whether a PM file already completed ingestion.
// It deliberately exposes only the indexed file-level idempotency query needed
// by the collector, rather than coupling the hot path to the full PMFileStore.
type FileMarkerLookup interface {
	IsFileParsed(ctx context.Context, deviceSN, fileName string) (bool, error)
}

type rawArchiver interface {
	Schedule(bucket, object string, onTerminal func(context.Context, string, string, string))
	RemoveOld(ctx context.Context, bucket, object string)
}

type rawObjectDiscarder interface {
	RemoveObject(context.Context, string, string, minio.RemoveObjectOptions) error
}

// CounterMeta 是 CounterWhitelist 命中后回填给 PMCounter 的元数据（PM-P2）。
//   - IndicatorID：指标编号（perf_indicators_*.id，如 C000060216），落库即编号化的目标。
//   - StatisType：'sum' / 'avg' / 'max' / 'pct'，或空串（indicator 元数据未填），驱动 G5 自然桶聚合。
//   - Unit：指标业务单位文本（如 number/%/Mbps），驱动 15min 入库前结果值规范化。
type CounterMeta struct {
	IndicatorID string
	ReportKey   string
	StatisType  string
	Unit        string
}

// CounterWhitelist 按设备 SN 返回指标库定义的"上报名(report_key) → CounterMeta"映射，
// 用于：(a) 过滤 PM 文件里的孤儿 counter（厂家上报但 perf_indicators_{enb,gnb,gsm}
// 未注册）；(b) 命中后把 PMCounter.CounterName 改写成指标编号（IndicatorID）——
// PM-P2 落库即编号化的唯一翻译入口；(c) 填充 PMCounter.StatisType，下游 counterToMetric
// 透传到 pm_metrics.statis_type 驱动 G5 自然桶聚合。
//
// 建键锚点是 report_key（设备上报名、入库不可改的解析契约），不是 en_name（可改的展示名）。
//
// 真实实现用 router.Router（设备 → product → indicator_platform → counter 子集）。
// worker main 写 adapter 把 *router.Router 包装成此接口，避免 collector 直接耦合 router 包。
//
// 错误处理约定（fail-open，参见 BUG-6 真根因复盘）：
//   - 返回 (nil, err)：collector 跳过过滤，log warn 后照常 BatchInsert 全量 counter；
//     不阻塞 PM 处理。失败保留量比误删数据风险小。
//   - 返回 (empty map, nil)：collector 同样跳过过滤（防误删全部 — 如启动期缓存未就绪）。
//   - 返回 (map, nil)：map 内（按 report_key 命中）的 counter 保留、改写成编号并填充
//     StatisType，其他作为孤儿丢弃。
//
// 注意：#866 接入结果值规范化后，过滤阶段仍保持 fail-open 以避免误删 counter；但写入前
// normalizeResults 会要求每条待写结果必须具备 Unit/StatisType，缺失元数据会让该 PM 文件处理失败。
type CounterWhitelist interface {
	LookupCounters(ctx context.Context, deviceSN string) (map[string]CounterMeta, error)
}

// KnownReportKeyLookup 返回某制式全局指标库中已经登记或明确处置的 Counter report_key。
// 它用于区分“库里存在但未绑定当前产品”和“全局库也不存在”，避免把前者误报成
// 厂家上报名漂移。返回集合只读；真实实现应缓存结果，不能按 PM 文件逐次查询主库。
type KnownReportKeyLookup interface {
	LookupKnownReportKeys(ctx context.Context, technology string) (map[string]struct{}, error)
}

// EnabledIndicatorLookup 读取当前制式下已启用的指标编号集。
//
// 真实实现必须走主库控制面元数据（enabled_pm_indicators_<tech>），不能走时序库。
// 查询失败时返回 error，collector 会让当前文件处理失败并进入既有 retry/DLQ；这样比
// fail-open 写入禁用指标或 fail-closed 静默丢光指标更保守、可观测。
type EnabledIndicatorLookup interface {
	LookupEnabledIndicators(ctx context.Context, technology string) (map[string]struct{}, error)
}

// CopyIngestor 是 copy-direct 写路径（copy 模式）的最小接口：把一个 PM 文件的全部 metric 行
// （counter + KPI）与 pm_files 幂等标记在单事务里 plain COPY 原子入库（去掉每行自然键 UPSERT
// 的写 CPU 大头，幂等下沉到每文件一次 pm_files 唯一约束）。由 metrics.PgRepository 实现
// （CopyIngest）。注入后 collector 走 copy 模式，否则保持默认 counterRepo UPSERT 写路径。
type CopyIngestor interface {
	CopyIngest(ctx context.Context, marker metrics.FileMarker, counters []model.PMCounter, kpis []model.KPIValue) (ingested bool, err error)
}

// PMCollector handles PM file processing: download from MinIO, parse XML, store counters.
//
// The optional runner.Wrapper field enables retry + DLQ instrumentation: when
// SetRunner has been called, Subscribe wires the handler through the wrapper
// so transient failures are retried per RetryConfig and exhausted events land
// in the dead_letters table (T-0012 / R-106). When runner is nil the handler
// is registered directly, preserving legacy behaviour for tests / single-process
// deployments without DLQ infra.
type PMCollector struct {
	minioClient          *minio.Client
	bucket               string
	parser               *PMXMLParser
	kpiEngine            *kpi.KPIEngine
	fileStore            pm.PMFileStore
	eventBus             event.EventBus
	metrics              *pm.PMMetrics
	runner               runner.Wrapper
	deviceLookup         DeviceLookup
	fileMarkerLookup     FileMarkerLookup
	counterWhitelist     CounterWhitelist
	knownReportKeys      KnownReportKeyLookup
	enabledIndicators    EnabledIndicatorLookup
	quarantineStore      QuarantineStore
	numberProcessLookup  NumberProcessLookup
	copyIngestor         CopyIngestor
	archiver             rawArchiver
	rawDiscarder         rawObjectDiscarder
	concurrency          int
	logger               *zap.Logger
	diagnosticMu         sync.Mutex
	lastWhitelistMissLog map[string]time.Time
}

const (
	maxWhitelistMissSample   = 8
	whitelistMissLogInterval = time.Minute
	pmRegistrationWaitQueue  = "pm-registration-wait"
)

// NewPMCollector creates a new PM collector.
func NewPMCollector(
	minioClient *minio.Client, bucket string, parser *PMXMLParser,
	kpiEngine *kpi.KPIEngine,
	fileStore pm.PMFileStore,
	eventBus event.EventBus, logger *zap.Logger,
) *PMCollector {
	collector := &PMCollector{
		minioClient: minioClient, bucket: bucket, parser: parser,
		kpiEngine: kpiEngine,
		fileStore: fileStore,
		eventBus:  eventBus, logger: logger,
	}
	if minioClient != nil {
		collector.rawDiscarder = minioClient
	}
	return collector
}

// SetMetrics attaches Prometheus metrics to the collector.
func (c *PMCollector) SetMetrics(m *pm.PMMetrics) {
	c.metrics = m
}

// SetRunner attaches a retry+DLQ wrapper. When set, Subscribe wires the
// handler through w.Wrap so failures are retried and exhausted events land
// in the dead_letters table. Pass nil to keep the legacy direct-subscribe
// behaviour. Call this before Subscribe; mid-flight changes are not honoured.
func (c *PMCollector) SetRunner(w runner.Wrapper) {
	c.runner = w
}

// SetDeviceLookup wires a device-resolution dependency for the "thin"
// pm.file.received payload (T-0164 G1 真机闭环): when acs.upload.Handler
// publishes the event with only device_sn, the collector uses this to fill
// in DeviceID / DeviceOUI / Carrier / Technology. Nil-safe — when unset,
// thin-payload events fail the existing uuid.Parse(DeviceID) check.
func (c *PMCollector) SetDeviceLookup(lookup DeviceLookup) {
	c.deviceLookup = lookup
}

// SetFileMarkerLookup wires the file-level idempotency lookup. When a completed
// marker already exists, the collector ACKs a redelivered event before touching
// the original MinIO path, which may have been renamed by the raw archiver.
func (c *PMCollector) SetFileMarkerLookup(lookup FileMarkerLookup) {
	c.fileMarkerLookup = lookup
}

// SetCounterWhitelist wires the indicator-library-driven counter whitelist
// (T-0164 G1 BUG-6 真根因复盘 / 方案 D)：解析后用于丢弃指标库未注册的孤儿
// counter。Nil-safe — 未设置时 collector 不过滤，行为退化到注入前。
func (c *PMCollector) SetCounterWhitelist(w CounterWhitelist) {
	c.counterWhitelist = w
}

// SetKnownReportKeyLookup wires the global report-key catalog used to classify
// routed-whitelist misses. Nil keeps the conservative legacy classification.
func (c *PMCollector) SetKnownReportKeyLookup(lookup KnownReportKeyLookup) {
	c.knownReportKeys = lookup
}

// SetEnabledIndicatorLookup wires the enabled-indicator filter for 15min raw ingest.
// Nil-safe — unset keeps the pre-#171 behavior and is used by narrow unit tests.
func (c *PMCollector) SetEnabledIndicatorLookup(lookup EnabledIndicatorLookup) {
	c.enabledIndicators = lookup
}

// SetQuarantineStore wires durable metadata storage for PM files whose XML
// technology conflicts with the resolved device identity.
func (c *PMCollector) SetQuarantineStore(store QuarantineStore) {
	c.quarantineStore = store
}

// SetNumberProcessLookup 注入 indicator.process.number 读取函数，供 15min 入库前结果值规范化使用。
// 未注入时使用 resultnorm 默认策略；读取错误会让当前 PM 文件处理失败并暴露。
func (c *PMCollector) SetNumberProcessLookup(lookup NumberProcessLookup) {
	c.numberProcessLookup = lookup
}

// SetCopyIngestor 注入 copy-direct 写路径（pm_files 标记 + 全部 metric 行单事务 plain COPY 原子
// 入库，见 CopyIngestor / handleFileReceived）。worker 启动期总是注入它——migration 000042 删
// uq_pm_metrics_natural 后 copy 是唯一写路径；未注入时 handleFileReceived fail-fast 返错（仅可能
// 出现在不写库的单测）。
func (c *PMCollector) SetCopyIngestor(ci CopyIngestor) {
	c.copyIngestor = ci
}

// SetArchiver 注入原始文件压缩回写器（issue #836）：入库成功后对明文 XML 尝试一次 gzip 回写
// MinIO 省盘（已是 gzip 的真机文件零成本跳过）。Nil-safe — 未注入时不做压缩回写。
func (c *PMCollector) SetArchiver(a *rawarchive.Archiver) {
	c.archiver = a
}

func (c *PMCollector) SetRawObjectDiscarder(discarder rawObjectDiscarder) {
	c.rawDiscarder = discarder
}

// markRawCompressed 把已压缩回写的 PM 原始对象在 pm_files 标记 raw_compressed=true 并把 minio_path
// 更新为压缩后的新键（issue #836 + 改键 .xml→.xml.gz）。作为 archiver.Schedule 的 onTerminal
// 回调，在压缩 goroutine 内调用；nil-safe。DB 更新成功后删旧明文键（仅改键时）；标记/删除失败只
// warn；旧键残留只占盘，后续由 MinIO 生命周期/运维清理兜底。
func (c *PMCollector) markRawCompressed(ctx context.Context, bucket, oldObject, newObject string) {
	if c.fileStore == nil {
		return
	}
	if err := c.fileStore.MarkCompressed(ctx, map[string]string{oldObject: newObject}); err != nil {
		c.logger.Warn("mark pm_files raw_compressed", zap.String("object", oldObject), zap.Error(err))
		return
	}
	if oldObject != newObject {
		// 用 archiver 实际压缩的 bucket 删旧键（不能假定等于 c.bucket）。
		c.archiver.RemoveOld(ctx, bucket, oldObject)
	}
}

// SetConcurrency 设置 PM 文件入库的进程内并发订阅数。
//
// 动机：NATS push 订阅的 async 回调由 nats.go 单 goroutine 串行投递，单订阅只能用 ~1 核。
// 这里对同一 subject(pm.file.received)+queue(pm-workers, durable) 发起 N 个 QueueSubscribe，
// 它们共享同一个 server-side durable consumer，JetStream 在 N 个订阅间负载均衡（核对
// nats.go@v1.52.0：queue 分支不触发 "consumer already bound"，每条消息的 ack 仍绑定其
// 自身 *nats.Msg，at-least-once / 重试语义不变），从而吃满 worker 多核。
//
// n<=1（或未设置）→ 退化为单订阅，保持旧行为。处理逻辑 handleFileReceived 与 runner.Wrap
// 均无可变共享状态，并发调用安全。
func (c *PMCollector) SetConcurrency(n int) {
	c.concurrency = n
}

// Subscribe registers the collector to listen for PM file received events.
func (c *PMCollector) Subscribe(bus event.EventBus) error {
	mainHandler := c.handleFileReceived
	deferredHandler := c.handleFileReceived
	if c.runner != nil {
		mainHandler = c.runner.Wrap(event.SubjectPMFileReceived, c.handleFileReceived)
		deferredHandler = c.runner.Wrap(event.SubjectPMFileDeferred, c.handleFileReceived)
	}
	mainHandler = c.handoffDeferredDevice(bus, mainHandler)
	n := c.concurrency
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		if _, err := bus.QueueSubscribe(event.SubjectPMFileDeferred, pmRegistrationWaitQueue, deferredHandler); err != nil {
			return fmt.Errorf("subscribe pm.file.deferred (sub %d/%d): %w", i+1, n, err)
		}
	}
	for i := 0; i < n; i++ {
		if _, err := bus.QueueSubscribe(event.SubjectPMFileReceived, "pm-workers", mainHandler); err != nil {
			return fmt.Errorf("subscribe pm.file.received (sub %d/%d): %w", i+1, n, err)
		}
	}
	c.logger.Info("PM collector subscribed to pm.file.received",
		zap.Bool("retry_dlq_wrapped", c.runner != nil),
		zap.Int("concurrency", n),
		zap.String("registration_wait_queue", pmRegistrationWaitQueue),
	)
	return nil
}

// handoffDeferredDevice moves registration-race events out of the main PM
// durable before acknowledging them. The separate durable may wait for the
// full registration grace without consuming every pm-workers ack-pending slot.
// Publish failure is returned so JetStream keeps the original message.
func (c *PMCollector) handoffDeferredDevice(bus event.EventBus, handler event.EventHandler) event.EventHandler {
	return func(ctx context.Context, evt event.Event) error {
		err := handler(ctx, evt)
		if !errors.Is(err, reliability.ErrDeferred) {
			return err
		}
		if publishErr := bus.Publish(ctx, event.SubjectPMFileDeferred, evt); publishErr != nil {
			return fmt.Errorf("handoff deferred PM event: %w", publishErr)
		}
		c.logger.Debug("PM event handed off to registration wait queue",
			zap.String("event_id", evt.ID),
			zap.Time("event_timestamp", evt.Timestamp),
		)
		return nil
	}
}

func (c *PMCollector) handleFileReceived(ctx context.Context, evt event.Event) error {
	startTime := time.Now()

	// issue #20: 给 PM 文件处理热路径补 span，让 PM 处理慢/卡（下载、解析、批量入库、
	// KPI 反算）在 trace 上可定位；tracing 全局 no-op 时零开销（StartSpan 安全）。
	ctx, span := tracing.StartSpan(ctx, tracing.PMTracerName, "PM HandleFileReceived")
	defer span.End()

	var payload FileReceivedPayload
	if err := evt.DecodePayload(&payload); err != nil {
		tracing.RecordError(span, err)
		return fmt.Errorf("decode payload: %w", err)
	}
	span.SetAttributes(
		attribute.String("pm.device_sn", payload.DeviceSN),
		attribute.String("pm.minio_path", payload.MinIOPath),
	)
	c.logger.Info("processing PM file",
		zap.String("object_hash", shortPMIdentityHash(payload.MinIOPath)),
		zap.String("device_hash", shortPMIdentityHash(payload.DeviceSN)),
	)

	if err := c.resolveDevice(ctx, &payload); err != nil {
		if errors.Is(err, reliability.ErrDeferred) &&
			(evt.Timestamp.IsZero() || time.Since(evt.Timestamp) >= DeviceRegistrationGrace) {
			return c.discardUnregisteredRaw(ctx, payload)
		}
		return err
	}

	// A parsed marker no longer short-circuits an existing source object: the
	// content digest must still be compared so same-name replacements can be
	// reparsed safely. It is retained as a fallback when the raw archiver has
	// already renamed the original object and a duplicate event is redelivered.
	knownParsed := false
	if c.fileMarkerLookup != nil {
		parsed, lookupErr := c.fileMarkerLookup.IsFileParsed(
			ctx, payload.DeviceSN, path.Base(payload.MinIOPath),
		)
		if lookupErr != nil {
			return fmt.Errorf("lookup parsed pm file marker: %w", lookupErr)
		}
		knownParsed = parsed
	}

	deviceID, err := uuid.Parse(payload.DeviceID)
	if err != nil {
		return fmt.Errorf("parse device_id: %w", err)
	}

	obj, err := c.minioClient.GetObject(ctx, c.bucket, payload.MinIOPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("download pm file: %w", err)
	}
	defer obj.Close()

	// #168：单文件体积上限，防超大/异常文件单次全量入内存 OOM。Stat 给确切大小，超限直接拒
	// （交 retry/DLQ 包装器）；fileSize 同时复用给下方 pm_files 元数据。
	var fileSize int64
	if stat, statErr := obj.Stat(); statErr == nil {
		fileSize = stat.Size
		if err := ensurePMFileSize(fileSize); err != nil {
			c.logger.Warn("PM file rejected: oversized",
				zap.String("path", payload.MinIOPath),
				zap.Int64("size", fileSize),
				zap.Int64("limit", maxPMFileBytes))
			return err
		}
	}

	// pm_files 元数据由 copy 写路径的 CopyIngest 在入库事务里一并写入（文件级幂等的单一锚点），
	// 不在解析前预存。now 作为 marker 的 collect_time 透传给 ingestViaCopy。
	now := time.Now()

	// issue #321：真机/模拟器按 TR-069 上传 .xml.gz，MinIO 原样存压缩字节；解析前按
	// gzip 魔数嗅探透明解压（明文 .xml 原样透传）。解压在 LimitReader 之前 → 体积
	// 上限作用于解压后内容，兼防 gzip 炸弹。
	decoded, rawCompressed, derr := compress.MaybeGunzip(obj)
	if derr != nil {
		if c.metrics != nil {
			c.metrics.FilesProcessedTotal.WithLabelValues("failed").Inc()
			c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
		}
		tracing.RecordError(span, derr)
		// 2026-07-22 压测实测：20000设备规模下磁盘长期逼近满载，观测到大量 PM 文件在
		// worker 处理前对象已从 MinIO 消失（疑似磁盘压力下的写入/清理异常，根因还在查）。
		// 这类错误重试注定必然失败（同一个已不存在的 key 重试多少次结果都一样），之前
		// 走通用 3 次重试+指数退避（约7秒/条）在队列被大量此类消息淹没时会显著拖慢
		// 真正可处理消息的吞吐——参照 resolveDevice 对「设备不存在」的处理，同样包一层
		// reliability.ErrPermanent 首次即终止，不重试、直接进 DLQ。
		if isMinIONotFound(derr) {
			if knownParsed {
				return nil
			}
			return fmt.Errorf("decompress pm file: %w: %w", derr, reliability.ErrPermanent)
		}
		return fmt.Errorf("decompress pm file: %w", derr)
	}

	// io.LimitReader 兜底：Stat 不可用/谎报时,解析最多读 maxPMFileBytes,截断 → 解析报错被捕获。
	contentHasher := sha256.New()
	content, err := c.parser.Parse(io.LimitReader(io.TeeReader(decoded, contentHasher), maxPMFileBytes), deviceID)
	if err != nil {
		if c.metrics != nil {
			c.metrics.FilesProcessedTotal.WithLabelValues("failed").Inc()
			c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
		}
		tracing.RecordError(span, err)
		// 同上：MinIO 对象不存在的错误也可能延迟到这里（XML 解析器内部持续读取 obj）
		// 才首次暴露，同样短路不重试。
		if isMinIONotFound(err) {
			if knownParsed {
				return nil
			}
			return fmt.Errorf("parse pm xml: %w: %w", err, reliability.ErrPermanent)
		}
		return fmt.Errorf("parse pm xml: %w", err)
	}

	span.SetAttributes(attribute.Int("pm.parsed_counters", len(content.Counters)))
	c.logger.Info("parsed PM file", zap.Int("counters", len(content.Counters)))

	quarantined, err := c.checkTechnologyConsistency(ctx, &payload, content)
	if err != nil {
		return err
	}
	if quarantined {
		if c.metrics != nil {
			c.metrics.FilesProcessedTotal.WithLabelValues("quarantined").Inc()
			c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
		}
		span.SetAttributes(attribute.String("pm.outcome", "technology_mismatch"))
		return nil
	}

	// G4-Gap-1: 上报延迟 = ingest_time - end_time。仅在两值齐全且 end_time 非 zero 时记录；
	// 时钟漂移可能产生负值，Prometheus histogram 不接受负 Observe，需 clamp 到 0。
	if c.metrics != nil && !content.FileEndTime.IsZero() && !content.IngestTime.IsZero() {
		delay := content.IngestTime.Sub(content.FileEndTime).Seconds()
		if delay < 0 {
			delay = 0
		}
		c.metrics.ReportDelaySeconds.WithLabelValues(payload.Carrier, payload.Technology).Observe(delay)
	}

	applyPayloadIdentity(content.Counters, payload.DeviceOUI, payload.DeviceSN)

	// T-0164 G1 BUG-6 方案 D：用产品指标库白名单过滤孤儿 counter。
	// 厂家 PM 文件可能含未注册 counter（如 Baicells `MR.RIPPRB` × 53 PRB 索引把序号
	// 编码在 measType.p 而不是 Name 里），落库会撞 pm_metrics 自然键。
	// 过滤逻辑 fail-open：lookup 失败 / 空白名单时跳过过滤，避免在本阶段误删全部 counter。
	// #866 后续 normalizeResults 会校验每条待写结果的 Unit/StatisType，缺失元数据时失败并暴露。
	var allow map[string]CounterMeta
	content.Counters, allow, content.whitelistMissValues, content.knownDisabledValues, content.whitelistMissSample = c.filterByWhitelistWithAllow(
		ctx, payload.DeviceSN, payload.Technology, content.Counters,
	)

	// PM 入库统一走 copy-direct 原子写路径（pm_files 标记 + counter + 内存算出的 KPI 单事务 plain
	// COPY，见 ingestViaCopy / metrics.CopyIngest）。worker 启动期总是注入 copyIngestor；未注入仅见
	// 于不写库的单测，这里 fail-fast 而非退回已退役的非幂等 UPSERT 旁路。
	if c.copyIngestor == nil {
		return fmt.Errorf("pm collector: copy ingestor not wired (copy is the sole write path)")
	}
	return c.ingestViaCopy(
		ctx, span, startTime, now, fileSize, deviceID, &payload, content, allow,
		rawCompressed, contentHasher.Sum(nil),
	)
}

func (c *PMCollector) discardUnregisteredRaw(
	ctx context.Context,
	payload FileReceivedPayload,
) error {
	if c.rawDiscarder == nil {
		return fmt.Errorf("discard unregistered PM raw object: discarder unavailable: %w", reliability.ErrDeferred)
	}
	bucket := payload.Bucket
	if bucket == "" {
		bucket = c.bucket
	}
	err := c.rawDiscarder.RemoveObject(
		ctx,
		bucket,
		payload.MinIOPath,
		minio.RemoveObjectOptions{},
	)
	if err != nil && !isMinIOObjectNotFound(err) {
		return fmt.Errorf("discard unregistered PM raw object: %v: %w", err, reliability.ErrDeferred)
	}
	if c.metrics != nil {
		c.metrics.FilesDiscardedTotal.WithLabelValues("device_not_registered").Inc()
	}
	c.logger.Warn("discarded PM file for unregistered device",
		zap.String("reason", "device_not_registered"),
		zap.String("object_hash", shortPMIdentityHash(payload.MinIOPath)),
		zap.String("device_hash", shortPMIdentityHash(payload.DeviceSN)),
	)
	return nil
}

func isMinIOObjectNotFound(err error) bool {
	var response minio.ErrorResponse
	return errors.As(err, &response) && response.Code == "NoSuchKey"
}

func shortPMIdentityHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:6])
}

func (c *PMCollector) checkTechnologyConsistency(
	ctx context.Context,
	payload *FileReceivedPayload,
	content *PMFileContent,
) (bool, error) {
	evidence := ClassifyTechnologyWithIdentity(content.DeviceSN, content.Counters)
	declared := strings.ToLower(strings.TrimSpace(payload.Technology))
	if evidence.Technology == "" || declared == "" || evidence.Technology == declared {
		return false, nil
	}
	if c.quarantineStore == nil {
		return false, fmt.Errorf(
			"save PM technology mismatch quarantine: quarantine store not wired",
		)
	}
	record := QuarantineRecord{
		SourceFileID:       quarantineSourceFileID(payload),
		DeviceSN:           payload.DeviceSN,
		DeclaredTechnology: declared,
		DetectedTechnology: evidence.Technology,
		Reason:             technologyMismatchReason,
		MinIOPath:          payload.MinIOPath,
		Evidence:           evidence.Signals,
	}
	inserted, err := c.quarantineStore.Save(ctx, record)
	if err != nil {
		return false, fmt.Errorf("save PM technology mismatch quarantine: %w", err)
	}
	if inserted && c.metrics != nil {
		c.metrics.TechnologyMismatchFilesTotal.
			WithLabelValues(declared, evidence.Technology).
			Inc()
	}
	c.logger.Warn("quarantined PM file with mismatched XML technology",
		zap.String("device_sn", payload.DeviceSN),
		zap.String("declared_technology", declared),
		zap.String("detected_technology", evidence.Technology),
		zap.String("minio_path", payload.MinIOPath),
		zap.Int("evidence_count", len(evidence.Signals)))
	return true, nil
}

// ingestViaCopy 是 copy 模式的写收尾：用内存 counter 算出 KPI（只算不写），把 counter + KPI +
// pm_files 幂等标记交给 CopyIngest 在单事务里 plain COPY 原子入库。与默认路径相比省掉：
//   - counter 的自然键 ON CONFLICT DO UPDATE（每行索引探测 + 更新）；
//   - KPI 单独一次 BatchInsert（又一趟 COPY 暂存表 + UPSERT）；
//   - pm_files 预存 + UpdateFileParsed 两次额外写（marker 一次写定 parsed/counter_count）。
//
// 返回 nil 让该文件 ack（含 marker 冲突=已入库的幂等跳过与迟到压缩 chunk 降级）。
func (c *PMCollector) ingestViaCopy(
	ctx context.Context, span trace.Span, startTime, now time.Time, fileSize int64,
	deviceID uuid.UUID, payload *FileReceivedPayload, content *PMFileContent, allow map[string]CounterMeta,
	rawCompressed bool, contentSHA256 []byte,
) error {
	// KPI：用本文件已过白名单、已编号化的内存 counter 直接算，不落库（随 counter 一起 COPY）。
	var kpis []model.KPIValue
	if c.kpiEngine != nil {
		ks, kerr := c.kpiEngine.CalculateFromCounters(ctx, deviceID, payload.DeviceOUI, payload.DeviceSN,
			content.Counters, content.CollectTime, model.CarrierCode(payload.Carrier), model.Technology(payload.Technology))
		if kerr != nil {
			c.logger.Warn("calculate kpi (copy mode)", zap.Int("counters", len(content.Counters)), zap.Error(kerr))
		} else {
			kpis = ks
		}
	}
	var enabledAllow map[string]CounterMeta
	if c.enabledIndicators != nil {
		enabled, err := c.enabledIndicators.LookupEnabledIndicators(ctx, payload.Technology)
		if err != nil {
			c.logger.Warn("enabled PM indicator lookup failed",
				zap.String("device_sn", payload.DeviceSN),
				zap.String("carrier", payload.Carrier),
				zap.String("technology", payload.Technology),
				zap.Error(err))
			return fmt.Errorf("lookup enabled PM indicators: %w", err)
		}
		beforeCounters, beforeKPIs := len(content.Counters), len(kpis)
		content.Counters = filterCountersByEnabled(content.Counters, enabled)
		kpis = filterKPIsByEnabled(kpis, enabled)
		enabledAllow = filterAllowByEnabled(allow, enabled)
		if droppedCounters, droppedKPIs := beforeCounters-len(content.Counters), beforeKPIs-len(kpis); droppedCounters > 0 || droppedKPIs > 0 {
			content.knownDisabledValues += droppedCounters
			c.logger.Info("filtered disabled PM indicators before 15min ingest",
				zap.String("device_sn", payload.DeviceSN),
				zap.String("carrier", payload.Carrier),
				zap.String("technology", payload.Technology),
				zap.Int("enabled_size", len(enabled)),
				zap.Int("dropped_counters", droppedCounters),
				zap.Int("dropped_kpis", droppedKPIs))
		}
	} else {
		enabledAllow = allow
	}
	if err := c.normalizeResults(ctx, content.Counters, kpis); err != nil {
		return err
	}
	content.Counters = fillMissingSupportedCounters(content.Counters, enabledAllow)

	measurementStart, measurementEnd := pmMeasurementWindow(content)
	marker := metrics.FileMarker{
		DeviceID:         deviceID,
		DeviceSN:         payload.DeviceSN,
		Carrier:          payload.Carrier,
		Technology:       payload.Technology,
		FileName:         path.Base(payload.MinIOPath),
		FileSize:         fileSize,
		CollectTime:      now,
		MeasurementStart: measurementStart,
		MeasurementEnd:   measurementEnd,
		MinioPath:        payload.MinIOPath,
		ContentSHA256:    contentSHA256,
		CounterCount:     len(content.Counters),
		RawCompressed:    rawCompressed,
	}
	ingested, err := c.copyIngestor.CopyIngest(ctx, marker, content.Counters, kpis)
	if err != nil {
		// 同一设备、同一文件名已经存在，但上传内容摘要不同，说明上游复用了
		// 已完成窗口的源身份。这类数据无法安全覆盖既有幂等锚点，也不可能通过
		// 重试恢复；删除当前非法对象并 ACK，避免重复投递持续占用主队列和 DLQ。
		if errors.Is(err, metrics.ErrSourceContentChanged) {
			if discardErr := c.discardChangedSourceRaw(ctx, *payload); discardErr != nil {
				return discardErr
			}
			span.SetAttributes(attribute.String("pm.outcome", "source_content_changed"))
			return nil
		}
		// 迟到补传命中压缩 chunk：与默认路径一致降级（log WARN + 记 metric + 跳过 ack），
		// 不让历史补传反复重试灌 DLQ 阻塞实时 PM。
		if errors.Is(err, metrics.ErrLateArrival) {
			c.logger.Warn("PM file skipped: late-arriving data hit compressed chunk (copy mode)",
				zap.String("path", payload.MinIOPath), zap.String("device_sn", payload.DeviceSN), zap.Error(err))
			if c.metrics != nil {
				c.metrics.LateArrivalFilesTotal.WithLabelValues(payload.Carrier, payload.Technology).Inc()
				c.metrics.FilesProcessedTotal.WithLabelValues("late_arrival").Inc()
				c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
			}
			span.SetAttributes(attribute.String("pm.outcome", "late_arrival"))
			return nil
		}
		if c.metrics != nil {
			c.metrics.FilesProcessedTotal.WithLabelValues("failed").Inc()
			c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
		}
		tracing.RecordError(span, err)
		return fmt.Errorf("copy ingest pm file: %w", err)
	}

	if !ingested {
		// marker 冲突：该文件已入库（NATS 重投 / 并发已写）→ 当作成功跳过，正常 ack。
		c.logger.Info("PM file already ingested (marker conflict), skip",
			zap.String("path", payload.MinIOPath), zap.String("device_sn", payload.DeviceSN))
	} else if !rawCompressed && c.archiver != nil {
		// issue #321：仅新入库时把原始 XML 压缩回写 MinIO 省盘（已 gzip 则零成本跳过）。
		// 异步有界并发，不阻塞 ack；nil-safe。压成功后经 onTerminal 标记 pm_files.raw_compressed=true。
		c.archiver.Schedule(c.bucket, payload.MinIOPath, c.markRawCompressed)
	}
	if ingested && content.whitelistMissValues > 0 &&
		c.shouldLogWhitelistMiss(payload.Carrier, payload.Technology, time.Now()) {
		c.logger.Warn("PM report keys missing from indicator library",
			zap.String("device_sn", payload.DeviceSN),
			zap.String("carrier", payload.Carrier),
			zap.String("technology", payload.Technology),
			zap.Int("whitelist_miss_values", content.whitelistMissValues),
			zap.Strings("report_key_sample", content.whitelistMissSample))
	}
	if ingested && c.metrics != nil {
		c.metrics.RecordIngestSuccess(payload.Carrier, payload.Technology, now)
		if content.whitelistMissValues > 0 {
			c.metrics.WhitelistMissValuesTotal.
				WithLabelValues(payload.Carrier, payload.Technology).
				Add(float64(content.whitelistMissValues))
		}
		if content.knownDisabledValues > 0 {
			c.metrics.KnownDisabledValuesTotal.
				WithLabelValues(payload.Carrier, payload.Technology).
				Add(float64(content.knownDisabledValues))
		}
	}
	if c.metrics != nil {
		c.metrics.FilesProcessedTotal.WithLabelValues("success").Inc()
		c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
	}
	span.SetAttributes(
		attribute.Int("pm.parsed_counters", len(content.Counters)),
		attribute.Int("pm.kpi_rows", len(kpis)),
		attribute.Bool("pm.copy_mode", true),
	)

	parsedEvt, perr := event.NewEvent(event.SubjectPMFileParsed, map[string]interface{}{
		"minio_path": payload.MinIOPath, "device_id": payload.DeviceID, "counter_count": len(content.Counters),
	})
	if perr == nil {
		_ = c.eventBus.Publish(ctx, event.SubjectPMFileParsed, parsedEvt)
	}
	return nil
}

func pmMeasurementWindow(content *PMFileContent) (time.Time, time.Time) {
	if content == nil {
		return time.Time{}, time.Time{}
	}
	end := content.FileEndTime
	if end.IsZero() {
		end = content.CollectTime
	}
	start := content.FileBeginTime
	if start.IsZero() && !end.IsZero() && content.Granularity > 0 {
		start = end.Add(-time.Duration(content.Granularity) * time.Minute)
	}
	return start, end
}

func (c *PMCollector) discardChangedSourceRaw(
	ctx context.Context,
	payload FileReceivedPayload,
) error {
	if c.rawDiscarder == nil {
		return fmt.Errorf("discard changed-source PM raw object: discarder unavailable: %w", reliability.ErrDeferred)
	}
	bucket := payload.Bucket
	if bucket == "" {
		bucket = c.bucket
	}
	err := c.rawDiscarder.RemoveObject(
		ctx,
		bucket,
		payload.MinIOPath,
		minio.RemoveObjectOptions{},
	)
	if err != nil && !isMinIOObjectNotFound(err) {
		return fmt.Errorf("discard changed-source PM raw object: %v: %w", err, reliability.ErrDeferred)
	}
	if c.metrics != nil {
		c.metrics.FilesDiscardedTotal.WithLabelValues("source_content_changed").Inc()
	}
	c.logger.Warn("discarded PM file with reused source identity",
		zap.String("reason", "source_content_changed"),
		zap.String("object_hash", shortPMIdentityHash(payload.MinIOPath)),
		zap.String("device_hash", shortPMIdentityHash(payload.DeviceSN)),
	)
	return nil
}

func filterCountersByEnabled(counters []model.PMCounter, enabled map[string]struct{}) []model.PMCounter {
	if len(counters) == 0 {
		return counters
	}
	if len(enabled) == 0 {
		return counters[:0]
	}
	out := counters[:0]
	for _, ctr := range counters {
		if _, ok := enabled[ctr.CounterName]; ok {
			out = append(out, ctr)
		}
	}
	return out
}

func filterKPIsByEnabled(kpis []model.KPIValue, enabled map[string]struct{}) []model.KPIValue {
	if len(kpis) == 0 {
		return kpis
	}
	if len(enabled) == 0 {
		return kpis[:0]
	}
	out := kpis[:0]
	for _, k := range kpis {
		if _, ok := enabled[k.IndicatorID]; ok {
			out = append(out, k)
		}
	}
	return out
}

func filterAllowByEnabled(allow map[string]CounterMeta, enabled map[string]struct{}) map[string]CounterMeta {
	if len(allow) == 0 || len(enabled) == 0 {
		return nil
	}
	out := make(map[string]CounterMeta, len(allow))
	for reportKey, meta := range allow {
		if _, ok := enabled[meta.IndicatorID]; ok {
			out[reportKey] = meta
		}
	}
	return out
}

// resolveDevice fills in DeviceID / DeviceOUI / Carrier / Technology on the
// payload when the publisher only provided device_sn (T-0164 G1 真机闭环 —
// acs.upload.Handler 发的瘦 payload）。transfer.Bridge 发的胖 payload device_id
// 已填，函数直接 no-op 返回。
//
// 错误分类：
//   - lookup 本身出错（DB 连接等基础设施问题）→ 普通错误，触发 retry+DLQ，通常瞬时问题。
//   - lookup 成功但 dev==nil → ErrDeferred。清洁部署或设备批量重连时，PM 文件可能
//     早于 Inform 注册落库；由 JetStream 持久化退避重投，避免直接 DLQ 形成 KPI 缺口。
//     handleFileReceived 超过 DeviceRegistrationGrace 后会转成 ErrPermanent，确保真正
//     未注册设备最终只产生一条可运维处理的死信，不无限占用队列。
func (c *PMCollector) resolveDevice(ctx context.Context, payload *FileReceivedPayload) error {
	if payload.DeviceID != "" {
		return nil
	}
	if c.deviceLookup == nil {
		return fmt.Errorf("pm.file.received: device_id empty and no DeviceLookup wired (sn=%s)", payload.DeviceSN)
	}
	if payload.DeviceSN == "" {
		return fmt.Errorf("pm.file.received: both device_id and device_sn empty")
	}
	dev, err := c.deviceLookup.GetBySerialNumber(ctx, payload.DeviceSN)
	if err != nil {
		return fmt.Errorf("lookup device by sn %s: %w", payload.DeviceSN, err)
	}
	if dev == nil {
		return fmt.Errorf("pm.file.received: device not found for sn=%s: %w", payload.DeviceSN, reliability.ErrDeferred)
	}
	payload.DeviceID = dev.ID.String()
	payload.DeviceOUI = dev.OUI
	payload.Carrier = string(dev.Carrier)
	payload.Technology = string(dev.Technology)
	return nil
}

// applyPayloadIdentity 把 ACS upload handler 解析的 (OUI, SN) 覆盖到 parser 输出的
// Counter 切片，统一全系统设备身份（T-0164-P3 双键 + BUG-6 续修 device_sn 一致性）。
//
// 为什么不用 parser 输出的 DeviceSN：parser 从 XML `<managedElement localDn=...>`
// 提取的 SN 依赖厂家 XML 规范，如 Baicells 真机用 `Station=eNb-{SN}` 格式，parser
// fallback 取最后一段拿到带 `eNb-` 前缀的脏值。payload.DeviceSN 是 ACS upload
// handler 从 URL `?sn=` / 文件名规范化解析的 TR-069 标准 SN，是全系统权威源
// （与 devices / topology / alarm / KPI 等模块一致）。
//
// OUI 同样由 payload 统一填充（parser 看不到 OUI — XML 内不含）。
func applyPayloadIdentity(counters []model.PMCounter, oui, sn string) {
	for i := range counters {
		counters[i].OUI = oui
		counters[i].DeviceSN = sn
	}
}

// filterByWhitelist 用产品指标库白名单过滤 counter（T-0164 G1 BUG-6 方案 D）。
// fail-open：whitelist 未注入 / 查询失败 / 空集合 → 返回原 counters 不过滤。
// #866 接入结果值规范化后，最终写入前仍会要求每条结果具备 Unit/StatisType。
//
// 当前产品路由外的 counter 缺少完整指标元数据，不能进入结果规范化和写入。本阶段
// 将它们剔除并按全局目录分类：全局已知项计 known_disabled，全局未知项才计
// whitelist_miss。未命中项不会继续进入 enabled filter，因此两类不会重复计数。
func (c *PMCollector) filterByWhitelist(ctx context.Context, deviceSN, carrier, technology string, counters []model.PMCounter) []model.PMCounter {
	out, _, _, _, _ := c.filterByWhitelistWithAllow(ctx, deviceSN, technology, counters)
	return out
}

func (c *PMCollector) filterAndFillByWhitelist(ctx context.Context, deviceSN, carrier, technology string, counters []model.PMCounter) []model.PMCounter {
	filtered, allow, _, _, _ := c.filterByWhitelistWithAllow(ctx, deviceSN, technology, counters)
	if len(allow) == 0 {
		return filtered
	}
	return fillMissingSupportedCounters(filtered, allow)
}

func (c *PMCollector) filterByWhitelistWithAllow(
	ctx context.Context,
	deviceSN string,
	technology string,
	counters []model.PMCounter,
) ([]model.PMCounter, map[string]CounterMeta, int, int, []string) {
	if c.counterWhitelist == nil || len(counters) == 0 {
		return counters, nil, 0, 0, nil
	}
	allow, err := c.counterWhitelist.LookupCounters(ctx, deviceSN)
	if err != nil {
		c.logger.Warn("counter whitelist lookup failed, skip filter",
			zap.String("device_sn", deviceSN), zap.Error(err))
		return counters, nil, 0, 0, nil
	}
	if len(allow) == 0 {
		c.logger.Warn("counter whitelist empty, skip filter (likely cache warming / product not matched)",
			zap.String("device_sn", deviceSN))
		return counters, nil, 0, 0, nil
	}

	var known map[string]struct{}
	if c.knownReportKeys != nil {
		known, err = c.knownReportKeys.LookupKnownReportKeys(ctx, technology)
		if err != nil {
			c.logger.Warn("known report-key catalog lookup failed; classify unrouted keys as whitelist misses",
				zap.String("device_sn", deviceSN),
				zap.String("technology", technology),
				zap.Error(err))
			known = nil
		}
	}

	kept := counters[:0] // 原地 reslice 复用 slice
	unknown := 0
	knownDisabled := 0
	var unknownSample []string
	unknownSampleSet := make(map[string]struct{}, maxWhitelistMissSample)
	for _, ctr := range counters {
		// PM-P2：按 report_key（=上报名 ctr.CounterName）命中白名单。命中后
		// 把 CounterName 改写成指标编号（落库即编号化的唯一翻译入口），并填 statis_type。
		if meta, ok := allow[ctr.CounterName]; ok {
			if meta.ReportKey == "" {
				meta.ReportKey = ctr.CounterName
			}
			reportGroup := measurementGroupFromReportKey(meta.ReportKey)
			if ctr.CounterGroup == "" || !counterGroupMatchesReportKey(ctr.CounterGroup, meta.ReportKey) {
				ctr.CounterGroup = reportGroup
			}
			ctr.CounterName = meta.IndicatorID // 上报名 → 编号
			ctr.StatisType = meta.StatisType   // T-0164-G6 收尾：填充 statis_type 驱动 G5 聚合 (BUG-A)
			ctr.Unit = meta.Unit               // #866：填充单位元数据，入库前规范化 result value
			kept = append(kept, ctr)
		} else {
			if _, ok := known[ctr.CounterName]; ok {
				knownDisabled++
			} else {
				unknown++
				if len(unknownSample) < maxWhitelistMissSample {
					if _, duplicate := unknownSampleSet[ctr.CounterName]; !duplicate {
						unknownSampleSet[ctr.CounterName] = struct{}{}
						unknownSample = append(unknownSample, ctr.CounterName)
					}
				}
			}
		}
	}
	return kept, allow, unknown, knownDisabled, unknownSample
}

func (c *PMCollector) shouldLogWhitelistMiss(carrier, technology string, now time.Time) bool {
	key := carrier + "\x00" + technology
	c.diagnosticMu.Lock()
	defer c.diagnosticMu.Unlock()
	if c.lastWhitelistMissLog == nil {
		c.lastWhitelistMissLog = make(map[string]time.Time)
	}
	if last, ok := c.lastWhitelistMissLog[key]; ok && now.Sub(last) < whitelistMissLogInterval {
		return false
	}
	c.lastWhitelistMissLog[key] = now
	return true
}

type missingCounterAnchor struct {
	oui          string
	sn           string
	objectLDN    string
	counterGroup string
	timeKey      string
	granularity  int
}

type missingCounterGroup struct {
	rep  model.PMCounter
	have map[string]struct{}
}

func fillMissingSupportedCounters(counters []model.PMCounter, allow map[string]CounterMeta) []model.PMCounter {
	if len(counters) == 0 || len(allow) == 0 {
		return counters
	}
	supported := make([]CounterMeta, 0, len(allow))
	for reportKey, meta := range allow {
		if meta.IndicatorID == "" {
			continue
		}
		if meta.ReportKey == "" {
			meta.ReportKey = reportKey
		}
		supported = append(supported, meta)
	}
	sort.Slice(supported, func(i, j int) bool { return supported[i].IndicatorID < supported[j].IndicatorID })
	if len(supported) == 0 {
		return counters
	}

	groups := make(map[missingCounterAnchor]*missingCounterGroup)
	order := make([]missingCounterAnchor, 0)
	for _, ctr := range counters {
		key := missingCounterAnchor{
			oui: ctr.OUI, sn: ctr.DeviceSN, objectLDN: ctr.CellID,
			counterGroup: ctr.CounterGroup, timeKey: ctr.Time.UTC().Format(time.RFC3339Nano), granularity: ctr.Granularity,
		}
		g := groups[key]
		if g == nil {
			g = &missingCounterGroup{rep: ctr, have: make(map[string]struct{})}
			groups[key] = g
			order = append(order, key)
		}
		g.have[ctr.CounterName] = struct{}{}
	}
	for _, key := range order {
		g := groups[key]
		for _, meta := range supported {
			if !counterGroupMatchesReportKey(g.rep.CounterGroup, meta.ReportKey) {
				continue
			}
			if _, ok := g.have[meta.IndicatorID]; ok {
				continue
			}
			ctr := g.rep
			ctr.CounterName = meta.IndicatorID
			ctr.CounterValue = math.NaN()
			ctr.StatisType = meta.StatisType
			ctr.Unit = meta.Unit
			counters = append(counters, ctr)
		}
	}
	return counters
}

func counterGroupMatchesReportKey(counterGroup, reportKey string) bool {
	if counterGroup == "" || reportKey == "" {
		return false
	}
	return counterGroup == measurementGroupFromReportKey(reportKey) ||
		reportKey == counterGroup ||
		strings.HasPrefix(reportKey, counterGroup+".")
}

func measurementGroupFromReportKey(reportKey string) string {
	if reportKey == "" {
		return ""
	}
	if i := strings.IndexByte(reportKey, '.'); i > 0 {
		return reportKey[:i]
	}
	return reportKey
}
