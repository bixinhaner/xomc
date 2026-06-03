package collector

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/reliability/runner"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"go.uber.org/zap"
)

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

// CounterMeta 是 CounterWhitelist 命中后回填给 PMCounter 的元数据（PM-P2）。
//   - IndicatorID：指标编号（perf_indicators_*.id，如 C000060216），落库即编号化的目标。
//   - StatisType：'sum' / 'avg' / 'max' / 'pct'，或空串（indicator 元数据未填），驱动 G5 自然桶聚合。
type CounterMeta struct {
	IndicatorID string
	StatisType  string
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
type CounterWhitelist interface {
	LookupCounters(ctx context.Context, deviceSN string) (map[string]CounterMeta, error)
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
	minioClient      *minio.Client
	bucket           string
	parser           *PMXMLParser
	counterRepo      counter.CounterRepository
	kpiEngine        *kpi.KPIEngine
	fileStore        pm.PMFileStore
	eventBus         event.EventBus
	metrics          *pm.PMMetrics
	runner           runner.Wrapper
	deviceLookup     DeviceLookup
	counterWhitelist CounterWhitelist
	logger           *zap.Logger
}

// NewPMCollector creates a new PM collector.
func NewPMCollector(
	minioClient *minio.Client, bucket string, parser *PMXMLParser,
	counterRepo counter.CounterRepository, kpiEngine *kpi.KPIEngine,
	fileStore pm.PMFileStore,
	eventBus event.EventBus, logger *zap.Logger,
) *PMCollector {
	return &PMCollector{
		minioClient: minioClient, bucket: bucket, parser: parser,
		counterRepo: counterRepo, kpiEngine: kpiEngine,
		fileStore: fileStore,
		eventBus: eventBus, logger: logger,
	}
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

// SetCounterWhitelist wires the indicator-library-driven counter whitelist
// (T-0164 G1 BUG-6 真根因复盘 / 方案 D)：解析后用于丢弃指标库未注册的孤儿
// counter。Nil-safe — 未设置时 collector 不过滤，行为退化到注入前。
func (c *PMCollector) SetCounterWhitelist(w CounterWhitelist) {
	c.counterWhitelist = w
}

// Subscribe registers the collector to listen for PM file received events.
func (c *PMCollector) Subscribe(bus event.EventBus) error {
	handler := c.handleFileReceived
	if c.runner != nil {
		handler = c.runner.Wrap(event.SubjectPMFileReceived, c.handleFileReceived)
	}
	_, err := bus.QueueSubscribe(event.SubjectPMFileReceived, "pm-workers", handler)
	if err != nil {
		return fmt.Errorf("subscribe pm.file.received: %w", err)
	}
	c.logger.Info("PM collector subscribed to pm.file.received",
		zap.Bool("retry_dlq_wrapped", c.runner != nil),
	)
	return nil
}

func (c *PMCollector) handleFileReceived(ctx context.Context, evt event.Event) error {
	startTime := time.Now()

	var payload FileReceivedPayload
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	c.logger.Info("processing PM file",
		zap.String("path", payload.MinIOPath),
		zap.String("device_sn", payload.DeviceSN),
	)

	if err := c.resolveDevice(ctx, &payload); err != nil {
		return err
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

	// Save file metadata to pm_files table before parsing
	var fileID uuid.UUID
	if c.fileStore != nil {
		var fileSize int64
		if stat, statErr := obj.Stat(); statErr == nil {
			fileSize = stat.Size
		}
		now := time.Now()
		pmFile := &pm.PMFileInfo{
			DeviceID:    deviceID,
			DeviceSN:    payload.DeviceSN,
			Carrier:     payload.Carrier,
			Technology:  payload.Technology,
			FileName:    path.Base(payload.MinIOPath),
			FileSize:    fileSize,
			CollectTime: now,
			MinioPath:   payload.MinIOPath,
		}
		if err := c.fileStore.SaveFile(ctx, pmFile); err != nil {
			c.logger.Warn("save PM file metadata", zap.Error(err))
		} else {
			fileID = pmFile.ID
		}
	}

	content, err := c.parser.Parse(obj, deviceID)
	if err != nil {
		if c.metrics != nil {
			c.metrics.FilesProcessedTotal.WithLabelValues("failed").Inc()
			c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
		}
		return fmt.Errorf("parse pm xml: %w", err)
	}

	c.logger.Info("parsed PM file", zap.Int("counters", len(content.Counters)))

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
	// 过滤逻辑 fail-open：lookup 失败 / 空白名单时跳过过滤，保留原行为。
	content.Counters = c.filterByWhitelist(ctx, payload.DeviceSN, content.Counters)

	if err := c.counterRepo.BatchInsert(ctx, content.Counters); err != nil {
		if c.metrics != nil {
			c.metrics.FilesProcessedTotal.WithLabelValues("failed").Inc()
			c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
		}
		return fmt.Errorf("batch insert counters: %w", err)
	}

	// Record success metrics
	if c.metrics != nil {
		c.metrics.FilesProcessedTotal.WithLabelValues("success").Inc()
		c.metrics.ProcessingDurationSecs.Observe(time.Since(startTime).Seconds())
	}

	// Update file parsed status
	if c.fileStore != nil && fileID != uuid.Nil {
		if err := c.fileStore.UpdateFileParsed(ctx, fileID, len(content.Counters)); err != nil {
			c.logger.Warn("update PM file parsed status", zap.Error(err))
		}
	}

	// Calculate KPIs
	if c.kpiEngine != nil {
		cellIDs := extractUniqueCellIDs(content.Counters)
		carrier := model.CarrierCode(payload.Carrier)
		tech := model.Technology(payload.Technology)
		for _, cellID := range cellIDs {
			if _, err := c.kpiEngine.CalculateAndStore(ctx, deviceID, payload.DeviceOUI, payload.DeviceSN, cellID, content.CollectTime, carrier, tech); err != nil {
				c.logger.Warn("calculate kpi", zap.String("cell_id", cellID), zap.Error(err))
			}
		}
	}

	parsedEvt, err := event.NewEvent(event.SubjectPMFileParsed, map[string]interface{}{
		"minio_path": payload.MinIOPath, "device_id": payload.DeviceID, "counter_count": len(content.Counters),
	})
	if err == nil {
		_ = c.eventBus.Publish(ctx, event.SubjectPMFileParsed, parsedEvt)
	}
	return nil
}

// resolveDevice fills in DeviceID / DeviceOUI / Carrier / Technology on the
// payload when the publisher only provided device_sn (T-0164 G1 真机闭环 —
// acs.upload.Handler 发的瘦 payload）。transfer.Bridge 发的胖 payload device_id
// 已填，函数直接 no-op 返回。Returning an error here triggers retry+DLQ in
// the wrapping runner — transient cases (device row not yet inserted because
// the inform/registration race) get retried and usually succeed.
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
		return fmt.Errorf("pm.file.received: device not found for sn=%s", payload.DeviceSN)
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
func (c *PMCollector) filterByWhitelist(ctx context.Context, deviceSN string, counters []model.PMCounter) []model.PMCounter {
	if c.counterWhitelist == nil || len(counters) == 0 {
		return counters
	}
	allow, err := c.counterWhitelist.LookupCounters(ctx, deviceSN)
	if err != nil {
		c.logger.Warn("counter whitelist lookup failed, skip filter",
			zap.String("device_sn", deviceSN), zap.Error(err))
		return counters
	}
	if len(allow) == 0 {
		c.logger.Warn("counter whitelist empty, skip filter (likely cache warming / product not matched)",
			zap.String("device_sn", deviceSN))
		return counters
	}

	kept := counters[:0] // 原地 reslice 复用 slice
	dropped := 0
	for _, ctr := range counters {
		// PM-P2：按 report_key（=上报名 ctr.CounterName）命中白名单。命中后
		// 把 CounterName 改写成指标编号（落库即编号化的唯一翻译入口），并填 statis_type。
		if meta, ok := allow[ctr.CounterName]; ok {
			ctr.CounterName = meta.IndicatorID // 上报名 → 编号
			ctr.StatisType = meta.StatisType   // T-0164-G6 收尾：填充 statis_type 驱动 G5 聚合 (BUG-A)
			kept = append(kept, ctr)
		} else {
			dropped++
		}
	}
	if dropped > 0 {
		c.logger.Info("filtered orphan counters not in indicator library",
			zap.String("device_sn", deviceSN),
			zap.Int("kept", len(kept)),
			zap.Int("dropped_orphans", dropped),
			zap.Int("whitelist_size", len(allow)))
	}
	return kept
}

func extractUniqueCellIDs(counters []model.PMCounter) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, c := range counters {
		if _, ok := seen[c.CellID]; !ok {
			seen[c.CellID] = struct{}{}
			result = append(result, c.CellID)
		}
	}
	return result
}
