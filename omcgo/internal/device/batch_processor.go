package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/netutil"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// informUpdate 封装单次 Inform 更新的所有数据。
type informUpdate struct {
	device     *model.Device           // 已查到的设备（含 ID，prepareDeviceUpdate 已写入新字段）
	inform     *tr069.InformMessage    // 原始 Inform 数据
	params     []model.DeviceParameter // 转换后的参数列表
	oldStatus  model.DeviceStatus      // T-0123 batch path：调 prepareDeviceUpdate 前捕获的旧 status
	oldVersion string                  // T-0125 batch path：调 prepareDeviceUpdate 前捕获的旧 firmware_version
}

// BatchInformProcessor 批量处理 Periodic Inform 的设备更新。
// 通过多协程并发 + 定时批量刷新，降低 DB 写入压力。
type BatchInformProcessor struct {
	workers         int
	flushInterval   time.Duration
	maxBatchSize    int
	shutdownTimeout time.Duration

	pool        *pgxpool.Pool
	redisClient redis.UniversalClient
	reconciler  *DeviceStatusReconciler
	cache       *DeviceCache
	stunUpdater StunAddressUpdater
	metrics     *DeviceMetrics
	logger      *zap.Logger

	workerChans []chan *informUpdate
	wg          sync.WaitGroup
	stopCh      chan struct{}

	// transitionPublisher: T-0123 / T-0125 batch path 补完 — flush 成功后发布
	// device.online / device.firmware.changed 事件。允许 nil（test 场景）。
	transitionPublisher TransitionEventPublisher

	// Phase 6/3 (设计文档 §4.3 §4.2 Layer C)：batch path 补完。原 batch flush
	// 旁路了 InfoSyncer + applyProductMetadata,导致 device_info 11 个 Phase 2
	// 新列 + devices.model_name 在 batch=true 下永远不被回填。
	//
	// productMatcher: applyProductMetadata 用,回填 device.ModelName。Submit 时
	//   调一次（batchUpdateDevices SQL 含 model_name 列,会持久化）。
	// infoSyncer: SyncFromParameters 用,把 device_parameters 投影到 device_info
	//   的 11 个新列 + mac / transmit_power 等。flush 成功后异步调用。
	//
	// 两者为 nil 时退化为原 batch path 行为（无回填），与改造前等价。
	productMatcher      ProductClassMatcher
	infoSyncer          deviceInfoParameterSyncer
	upsRuntimeRepo      UPSRuntimeRepository
	infoProjectionSlots chan struct{}
	infoProjectionRetry sync.Map
}

type deviceInfoParameterSyncer interface {
	SyncFromParameters(
		ctx context.Context,
		deviceID uuid.UUID,
		carrier model.CarrierCode,
		technology model.Technology,
		productClass string,
	) ([]string, error)
}

// SetProductMatcher 注入 ProductRegistry 用于 batch path 回填 device.ModelName。
// Phase 6 follow-up — nil 时跳过回填。
func (p *BatchInformProcessor) SetProductMatcher(m ProductClassMatcher) {
	p.productMatcher = m
}

// SetInfoSyncer 注入 InfoSyncer 用于 batch flush 后投影 device_info 扩展列。
// Phase 3 follow-up — nil 时跳过投影。
func (p *BatchInformProcessor) SetInfoSyncer(s *InfoSyncer) {
	p.infoSyncer = s
}

func (p *BatchInformProcessor) SetUPSRuntimeRepository(repo UPSRuntimeRepository) {
	p.upsRuntimeRepo = repo
}

// TransitionEventPublisher 是 BatchInformProcessor 调 DeviceService 发布
// device.online / device.firmware.changed 事件的窄接口（避免反向依赖 DeviceService 整体）。
type TransitionEventPublisher interface {
	ClearDisconnectedAlarmOnOnline(ctx context.Context, device *model.Device)
	PublishDeviceOnlineEvent(ctx context.Context, device *model.Device)
	PublishDeviceFirmwareChangedEvent(ctx context.Context, device *model.Device,
		oldVersion, newVersion string, becameOnline bool)
}

// NewBatchInformProcessor creates a new BatchInformProcessor.
//
// reconciler 可为 nil（dev/test),flush 时跳过 RefreshHeartbeat,不影响 PG 写入。
func NewBatchInformProcessor(
	cfg appconfig.BatchProcessorConfig,
	pool *pgxpool.Pool,
	redisClient redis.UniversalClient,
	reconciler *DeviceStatusReconciler,
	cache *DeviceCache,
	stunUpdater StunAddressUpdater,
	metrics *DeviceMetrics,
	logger *zap.Logger,
) *BatchInformProcessor {
	workers := cfg.Workers
	if workers <= 0 {
		workers = 4
	}
	flushInterval := cfg.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 10 * time.Second
	}
	maxBatchSize := cfg.MaxBatchSize
	if maxBatchSize <= 0 {
		maxBatchSize = 200
	}
	inputBuffer := cfg.InputBuffer
	if inputBuffer <= 0 {
		inputBuffer = 2500
	}
	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 30 * time.Second
	}

	workerChans := make([]chan *informUpdate, workers)
	for i := 0; i < workers; i++ {
		workerChans[i] = make(chan *informUpdate, inputBuffer)
	}

	return &BatchInformProcessor{
		workers:             workers,
		flushInterval:       flushInterval,
		maxBatchSize:        maxBatchSize,
		shutdownTimeout:     shutdownTimeout,
		pool:                pool,
		redisClient:         redisClient,
		reconciler:          reconciler,
		cache:               cache,
		stunUpdater:         stunUpdater,
		metrics:             metrics,
		logger:              logger,
		workerChans:         workerChans,
		stopCh:              make(chan struct{}),
		infoProjectionSlots: make(chan struct{}, 8),
	}
}

// Start launches all worker goroutines.
func (p *BatchInformProcessor) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.runWorker(i)
	}
	p.logger.Info("batch inform processor started",
		zap.Int("workers", p.workers),
		zap.Duration("flush_interval", p.flushInterval),
		zap.Int("max_batch_size", p.maxBatchSize))
}

// Stop gracefully shuts down all workers, flushing remaining buffers.
func (p *BatchInformProcessor) Stop() {
	close(p.stopCh)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("batch inform processor stopped gracefully")
	case <-time.After(p.shutdownTimeout):
		p.logger.Warn("batch inform processor shutdown timed out, some data may be lost")
	}
}

// SetTransitionPublisher 注入 transition 事件发布器（T-0123/T-0125 batch path）。
// nil 表示不发事件（test 场景或灰度关闭）。
func (p *BatchInformProcessor) SetTransitionPublisher(pub TransitionEventPublisher) {
	p.transitionPublisher = pub
}

// Submit dispatches an inform update to the appropriate worker.
// Heartbeat is refreshed immediately (not deferred to flush).
// oldStatus / oldVersion 由 caller（InformHandler）在调 prepareDeviceUpdate 之前捕获，
// 用于 flush 后判断是否发 device.online / firmware.changed 事件。
func (p *BatchInformProcessor) Submit(device *model.Device, inform *tr069.InformMessage,
	params []model.DeviceParameter, oldStatus model.DeviceStatus, oldVersion string) {
	// 心跳立即刷新，不等 flush
	if p.reconciler != nil {
		p.reconciler.RefreshHeartbeat(context.Background(), device.SerialNumber, device.InformInterval)
	}

	// Phase 6 (设计文档 §4.3 方案 X): ProductRegistry 回填 device.ModelName + Technology。
	// 与 UpdateFromInform 非 batch 路径 device_service.go:applyProductMetadata 行为对齐。
	// helper 内部已 nil-safe + 空字段检查，调用代价小（多数情况 cache 命中即 return）。
	// 不在这里早返回 ModelName != "" — Technology 可能还错着,需要 inline 内独立判断。
	if p.productMatcher != nil && device.ProductClass != "" {
		applyProductMetadataInline(context.Background(), p.productMatcher, device, p.logger)
	}

	update := &informUpdate{
		device:     device,
		inform:     inform,
		params:     params,
		oldStatus:  oldStatus,
		oldVersion: oldVersion,
	}

	workerID := p.dispatchToWorker(device.SerialNumber)
	select {
	case p.workerChans[workerID] <- update:
	default:
		if p.metrics != nil {
			p.metrics.BatchDropped.Inc()
		}
		p.logger.Warn("batch processor: worker channel full, dropping update",
			zap.String("device_sn", device.SerialNumber),
			zap.Int("worker_id", workerID))
	}
}

func (p *BatchInformProcessor) dispatchToWorker(sn string) int {
	h := fnv.New32a()
	h.Write([]byte(sn))
	return int(h.Sum32()) % p.workers
}

func (p *BatchInformProcessor) runWorker(id int) {
	defer p.wg.Done()

	buffer := make(map[string]*informUpdate)
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case update := <-p.workerChans[id]:
			// 同一 deviceSN 后续更新覆盖前一条（保留最新）
			buffer[update.device.SerialNumber] = update
			if len(buffer) >= p.maxBatchSize {
				p.flush(id, buffer)
				buffer = make(map[string]*informUpdate)
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				p.flush(id, buffer)
				buffer = make(map[string]*informUpdate)
			}
		case <-p.stopCh:
			// 优雅关闭：flush 残留数据
			if len(buffer) > 0 {
				p.flush(id, buffer)
			}
			return
		}
	}
}

func (p *BatchInformProcessor) flush(workerID int, buffer map[string]*informUpdate) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	count := len(buffer)

	// 最多重试 2 次（共 3 次尝试）
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(1 * time.Second)
			p.logger.Warn("batch flush retry",
				zap.Int("worker_id", workerID),
				zap.Int("attempt", attempt))
		}

		err = p.doFlush(ctx, buffer)
		if err == nil {
			break
		}
	}

	if err != nil {
		if p.metrics != nil {
			p.metrics.BatchFlushFailed.Inc()
		}
		p.logger.Error("batch flush failed after retries, data will recover on next Inform cycle",
			zap.Int("worker_id", workerID),
			zap.Int("device_count", count),
			zap.Error(err))
		return
	}

	elapsed := time.Since(start)
	if p.metrics != nil {
		p.metrics.BatchFlushTotal.Inc()
		p.metrics.BatchFlushSize.Observe(float64(count))
		p.metrics.BatchFlushDuration.Observe(elapsed.Seconds())
	}

	p.logger.Debug("batch flush completed",
		zap.Int("worker_id", workerID),
		zap.Int("device_count", count),
		zap.Duration("elapsed", elapsed))
}

func (p *BatchInformProcessor) doFlush(ctx context.Context, buffer map[string]*informUpdate) error {
	updates := make([]*informUpdate, 0, len(buffer))
	for _, u := range buffer {
		updates = append(updates, u)
	}

	// 1. 批量更新设备表（返回每条 update 的 RowsAffected）
	affected, err := p.batchUpdateDevices(ctx, updates)
	if err != nil {
		return fmt.Errorf("batch update devices: %w", err)
	}

	// Partition：UPDATE 命中的（PG 行还在）走正常 cache write-through；
	// 没命中的（cache 是 stale，PG 行已被删）DEL cache 让下次 inform 走 auto-register。
	hit := make([]*informUpdate, 0, len(updates))
	orphan := make([]*informUpdate, 0)
	for i, u := range updates {
		if affected[i] > 0 {
			hit = append(hit, u)
		} else {
			orphan = append(orphan, u)
		}
	}

	if len(orphan) > 0 {
		p.invalidateOrphanCache(ctx, orphan)
	}

	// 2. 批量更新参数表（仅 hit，避免 device_id 外键悬空）
	paramResult, err := p.batchUpsertParams(ctx, hit)
	if err = p.handleParameterUpsertOutcome(hit, paramResult, err); err != nil {
		return fmt.Errorf("batch upsert params: %w", err)
	}

	// 3. UPS 运行态投影。只处理 ProductClass=UPS* 的设备，失败不阻断既有制式。
	p.projectUPSRuntime(ctx, hit)

	// 4. 批量 Redis 操作（仅 hit，避免把 stale device 写回 cache）
	p.batchRedisOps(ctx, hit)

	// 5. T-0123/T-0125: PG + cache 写入成功后发 transition 事件。
	// 与 UpdateFromInform 非 batch 路径行为对齐（device_service.go §UpdateFromInform 末尾）。
	// 同一 Inform 满足两者时优先发 firmware.changed（不发 device.online），由
	// HandleFirmwareChanged 触发模型刷新 + durable 全量同步覆盖 online 语义，避免双触发。
	if p.transitionPublisher != nil {
		for _, u := range hit {
			p.publishTransitionEvents(ctx, u)
		}
	}

	return nil
}

func (p *BatchInformProcessor) projectUPSRuntime(ctx context.Context, updates []*informUpdate) {
	if p == nil || p.upsRuntimeRepo == nil {
		return
	}
	for _, update := range updates {
		if update == nil || update.device == nil || update.inform == nil || !isUPSProductClass(update.inform.DeviceId.ProductClass) {
			continue
		}
		if err := p.upsRuntimeRepo.UpsertFromInform(ctx, update.device, update.inform); err != nil {
			p.logger.Warn("batch project UPS runtime from Inform failed",
				zap.String("device_id", update.device.ID.String()),
				zap.String("serial_number", update.device.SerialNumber),
				zap.Error(err))
		}
	}
}

// handleParameterUpsertOutcome queues device_info projection for every device
// whose parameter transaction committed, even when a later contended-device
// transaction failed. The retry marker is installed before the asynchronous
// projection starts, so an unchanged retry cannot permanently skip the
// already-committed device.
func (p *BatchInformProcessor) handleParameterUpsertOutcome(
	hit []*informUpdate,
	result deviceParameterUpsertResult,
	upsertErr error,
) error {
	if p.infoSyncer == nil {
		return upsertErr
	}
	changed := make([]*informUpdate, 0, len(result.changedDevices))
	for _, update := range hit {
		if update == nil || update.device == nil || isUPSProductClass(update.device.ProductClass) {
			continue
		}
		_, parameterChanged := result.changedDevices[update.device.ID]
		_, retryPending := p.infoProjectionRetry.Load(update.device.ID)
		if !parameterChanged && !retryPending {
			continue
		}
		// Mark before launching the goroutine. asyncSyncDeviceInfo removes the
		// marker only after a successful, idempotent full-device projection.
		p.infoProjectionRetry.Store(update.device.ID, struct{}{})
		changed = append(changed, update)
	}
	if p.metrics != nil {
		p.metrics.BatchInfoProjection.WithLabelValues("queued").Add(float64(len(changed)))
		p.metrics.BatchInfoProjection.WithLabelValues("skipped").Add(float64(len(hit) - len(changed)))
	}
	if len(changed) > 0 {
		go p.asyncSyncDeviceInfo(changed)
	}
	return upsertErr
}

func (p *BatchInformProcessor) publishTransitionEvents(ctx context.Context, u *informUpdate) {
	if p.transitionPublisher == nil || u == nil || u.device == nil {
		return
	}
	if u.device.Status == model.DeviceActive && u.device.IsOnline {
		p.transitionPublisher.ClearDisconnectedAlarmOnOnline(ctx, u.device)
	}

	newVersion := u.device.FirmwareVersion
	firmwareChanged := u.oldVersion != "" && newVersion != "" && u.oldVersion != newVersion
	becameOnline := u.oldStatus == model.DeviceOffline && u.device.Status == model.DeviceActive

	switch {
	case firmwareChanged:
		p.transitionPublisher.PublishDeviceFirmwareChangedEvent(ctx, u.device, u.oldVersion, newVersion, becameOnline)
	case becameOnline:
		p.transitionPublisher.PublishDeviceOnlineEvent(ctx, u.device)
	}
}

// invalidateOrphanCache DEL stale cache for devices whose PG row vanished.
// 下次 inform 会 cache miss → PG miss → handlePeriodic 走 auto-register（A1 修复后的路径）。
func (p *BatchInformProcessor) invalidateOrphanCache(ctx context.Context, orphans []*informUpdate) {
	if p.cache == nil {
		return
	}
	pipe := p.redisClient.Pipeline()
	for _, u := range orphans {
		pipe.Del(ctx, deviceCacheKey(u.device.SerialNumber))
		p.logger.Warn("orphan inform: device row missing from PG, clearing stale cache (next inform will auto-register)",
			zap.String("device_sn", u.device.SerialNumber),
			zap.String("stale_device_id", u.device.ID.String()))
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		p.logger.Warn("orphan cache invalidate", zap.Error(err))
	}
}

// batchUpdateDevices 批量更新设备表。返回每条 update 的 RowsAffected：
// >0 表示 PG 行存在并已更新；==0 表示 cache 命中但 PG 行已不存在（orphan，
// 由 doFlush 走 invalidateOrphanCache）。
func (p *BatchInformProcessor) batchUpdateDevices(ctx context.Context, updates []*informUpdate) ([]int64, error) {
	if len(updates) == 0 {
		return nil, nil
	}

	batch := &pgx.Batch{}
	for _, u := range updates {
		dev := u.device
		eventsData, err := json.Marshal(dev.LastInformEvents)
		if err != nil {
			eventsData = []byte("[]")
		}

		var ipAddr interface{}
		if dev.IPAddress != "" {
			ipAddr = dev.IPAddress
		}
		var udpAddr interface{}
		if dev.UDPConnectionRequestAddress != "" {
			udpAddr = dev.UDPConnectionRequestAddress
		}

		// T-0162: devices.status 列已 DROP（migrations/000137），改写 lifecycle_state +
		// is_online 双列。prepareDeviceUpdate 已显式维护这两个新字段（收到 Inform 即
		// IsOnline=true、Discovered/Registered 升 Commissioned），这里直接持久化。
		// AND deleted_at IS NULL：防止软删的 device 被 inform 静默复活
		// （cache stale → 这里 UPDATE 仍命中已删行，silent data corruption）
		// Phase 6 follow-up：加 model_name 列让 Submit 中 applyProductMetadataInline 写入
		// 的 device.ModelName 真正持久化。SET model_name 不写空字符串覆盖既有非空值。
		// product_id / param_model_id：按 productClass 命中时回写（随 product_class 变更
		// 实时重算所属产品）；未命中（$15/$16 为 NULL）则保留既有值，不误清管理员手动绑定 / 孤儿态。
		// technology：Submit 中 prepareDeviceUpdate（路径推断）与 applyProductMetadataInline
		// （产品字典覆盖/清空）的最终值在此持久化——缺列会让非无线产品的清空只改内存不落库，
		// 每个周期从 DB 读回旧值反复"cleared"（与导出的 BatchUpdateDevices 写 technology 对齐）。
		query := `UPDATE devices SET
			oui = $1, product_class = $2, manufacturer = $3,
			lifecycle_state = $4, is_online = $5, firmware_version = $6,
			ip_address = $7, connection_request_url = $8,
			nat_detected = $9, udp_connection_request_address = $10,
			last_inform_at = $11, last_inform_events = $12,
			model_name = CASE WHEN $13::text <> '' THEN $13 ELSE model_name END,
			technology = $17,
			product_id = CASE WHEN $15::uuid IS NOT NULL THEN $15 ELSE product_id END,
			param_model_id = CASE WHEN $15::uuid IS NOT NULL THEN $16 ELSE param_model_id END,
			updated_at = NOW()
		WHERE id = $14 AND deleted_at IS NULL`

		batch.Queue(query,
			dev.OUI, dev.ProductClass, dev.Manufacturer,
			dev.LifecycleState, dev.IsOnline, dev.FirmwareVersion,
			ipAddr, dev.ConnectionRequestURL,
			dev.NatDetected, udpAddr,
			dev.LastInformAt, eventsData,
			dev.ModelName,
			dev.ID,
			dev.ProductID, dev.ParamModelID,
			dev.Technology,
		)
	}

	br := p.pool.SendBatch(ctx, batch)
	defer br.Close()

	affected := make([]int64, len(updates))
	for i := 0; i < len(updates); i++ {
		ct, err := br.Exec()
		if err != nil {
			return nil, fmt.Errorf("exec batch update item %d (device %s): %w",
				i, updates[i].device.SerialNumber, err)
		}
		affected[i] = ct.RowsAffected()
	}
	return affected, nil
}

// batchUpsertParams 将所有设备的参数收敛为多行 UPSERT。只有字段实际变化时
// PostgreSQL 才执行 UPDATE，并返回需要刷新 device_info 投影的设备集合。
func (p *BatchInformProcessor) batchUpsertParams(
	ctx context.Context,
	updates []*informUpdate,
) (deviceParameterUpsertResult, error) {
	rows := make([]deviceParameterUpsertRow, 0)
	for _, u := range updates {
		for _, param := range u.params {
			rows = append(rows, deviceParameterUpsertRow{
				deviceID: u.device.ID, parameter: param,
			})
		}
	}
	result, err := bulkUpsertDeviceParameters(ctx, p.pool, rows)
	if err != nil {
		return result, err
	}
	if p.metrics != nil {
		p.metrics.BatchParameterRows.WithLabelValues("attempted").Add(float64(result.attempted))
		p.metrics.BatchParameterRows.WithLabelValues("changed").Add(float64(result.changed))
		p.metrics.BatchParameterRows.WithLabelValues("skipped").Add(float64(result.attempted - result.changed))
	}
	return result, nil
}

// batchRedisOps 批量刷新设备缓存和 STUN 地址。
func (p *BatchInformProcessor) batchRedisOps(ctx context.Context, updates []*informUpdate) {
	pipe := p.redisClient.Pipeline()

	for _, u := range updates {
		dev := u.device
		// 刷新设备缓存
		if p.cache != nil {
			data, err := json.Marshal(dev)
			if err == nil {
				pipe.Set(ctx, deviceCacheKey(dev.SerialNumber), data, deviceCacheTTL)
			}
		}
	}

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		p.logger.Warn("batch redis pipeline", zap.Error(err))
	}

	// STUN 地址单独处理（StunAddressUpdater 接口不支持 pipeline）。
	// 注意：UDPConnectionRequestAddress 可能是 DB 里残留的脏值（历史派生兜底写入）。
	// 这里必须用与 prepareDeviceUpdate 同口径的 unspecified 守卫，否则 0.0.0.0
	// 或派生的 host:port 会被反复刷回 Redis acs:stun:<SN>，污染 dispatcher。
	for _, u := range updates {
		dev := u.device
		if p.stunUpdater == nil {
			continue
		}
		addr := dev.UDPConnectionRequestAddress
		if addr == "" || netutil.IsUnspecifiedUDPAddress(addr) {
			continue
		}
		if err := p.stunUpdater.SetFromInform(ctx, dev.SerialNumber, addr); err != nil {
			p.logger.Warn("batch stun address sync",
				zap.String("device_sn", dev.SerialNumber), zap.Error(err))
		}
	}
}

// prepareDeviceUpdate 从 Inform 消息构建设备更新和参数列表。
// 这是从 DeviceService.UpdateFromInform 提取的核心字段更新逻辑。
func prepareDeviceUpdate(device *model.Device, inform *tr069.InformMessage) ([]model.DeviceParameter, bool) {
	now := time.Now()
	device.OUI = inform.DeviceId.OUI
	device.ProductClass = inform.DeviceId.ProductClass
	device.Manufacturer = inform.DeviceId.Manufacturer
	if inferredTech, ok := detectTechnologyFromPaths(inform.ParameterList); ok {
		device.Technology = inferredTech
	}
	device.FirmwareVersion = informSoftwareVersion(inform.ParameterList)
	updateConnectionRequestSummary(device, inform.ParameterList)
	device.LastInformAt = &now
	device.LastInformEvents = tr069.EventCodes(inform.Event)

	udpAddr := deriveUDPConnectionRequestAddress(inform.ParameterList)
	stunChanged := false
	if udpAddr != "" {
		device.IPAddress = deriveInformIPAddress(udpAddr, device.ConnectionRequestURL)
		device.UDPConnectionRequestAddress = udpAddr
		device.NatDetected = true
		stunChanged = true
	} else {
		// 与 DeviceService.UpdateFromInform 同步：本次 Inform 未上报有效 UDP 地址 →
		// 清空残留脏值，避免 batchRedisOps 把 cache/DB 老值刷回 Redis acs:stun:<SN>。
		device.UDPConnectionRequestAddress = ""
		device.NatDetected = false
		if device.IPAddress == "" && isUPSProductClass(inform.DeviceId.ProductClass) {
			device.IPAddress = informUPSExternalIPAddress(inform.ParameterList)
		}
	}

	// T-0162: 收到 Inform 即视为在线 —— 必须显式写新字段 IsOnline，因为
	// normalizeDeviceForPersist shim 只在 LifecycleState=="" 时才从 Status 派生新字段；
	// 而 scan 出来的设备 LifecycleState 早已是 'commissioned'（非空），shim 不会重做
	// 派生，Update SQL 会拿到 scan 时的旧 IsOnline（false） → DB 永远 is_online=false
	// → executor 判定离线 → 任何任务派发都进 suspended。
	device.IsOnline = true
	// 设备初次入网（Discovered/Registered）→ 升级到 Commissioned。
	if device.LifecycleState == model.LifecycleDiscovered || device.LifecycleState == model.LifecycleRegistered {
		device.LifecycleState = model.LifecycleCommissioned
	}
	// 自动切换到 active 状态（老 Status 字段兼容路径，给读侧未迁移代码用）。
	if device.Status == model.DeviceDiscovered || device.Status == model.DeviceOffline || device.Status == model.DeviceRegistered {
		device.Status = model.DeviceActive
	}
	// T-FIX-OPSTATE: 同步 op_state 给前端"激活状态"列展示（status 任意变更后必须刷新）。
	device.OpState = model.DeriveOpState(device.Status)

	// 构建参数列表
	var params []model.DeviceParameter
	if len(inform.ParameterList) > 0 {
		params = make([]model.DeviceParameter, 0, len(inform.ParameterList))
		for _, p := range inform.ParameterList {
			params = append(params, model.DeviceParameter{
				DeviceID:       device.ID,
				ParameterPath:  p.Name,
				ParameterValue: p.Value,
				ParameterType:  model.ParamString,
				Writable:       false,
				LastUpdatedAt:  now,
			})
		}
	}

	_ = stunChanged
	return params, stunChanged
}

// GetBufferSizes returns the current buffer sizes for each worker (for metrics/debugging).
func (p *BatchInformProcessor) GetBufferSizes() []int {
	sizes := make([]int, p.workers)
	for i := 0; i < p.workers; i++ {
		sizes[i] = len(p.workerChans[i])
	}
	return sizes
}

// GetQueueLen returns the queue length for the given device serial number's worker.
func (p *BatchInformProcessor) GetQueueLen(sn string) int {
	workerID := p.dispatchToWorker(sn)
	return len(p.workerChans[workerID])
}

// MakeDeviceUpdate is a package-level alias for prepareDeviceUpdate, used by InformHandler.
func MakeDeviceUpdate(device *model.Device, inform *tr069.InformMessage) ([]model.DeviceParameter, bool) {
	return prepareDeviceUpdate(device, inform)
}

// BatchUpdateDevices adds a batch update method to PgDeviceRepository.
func (r *PgDeviceRepository) BatchUpdateDevices(ctx context.Context, devices []*model.Device) error {
	if len(devices) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, dev := range devices {
		eventsData, err := json.Marshal(dev.LastInformEvents)
		if err != nil {
			eventsData = []byte("[]")
		}

		var ipAddr interface{}
		if dev.IPAddress != "" {
			ipAddr = dev.IPAddress
		}
		var udpAddr interface{}
		if dev.UDPConnectionRequestAddress != "" {
			udpAddr = dev.UDPConnectionRequestAddress
		}

		// T-0162: devices.status 列已 DROP（migrations/000137），改写 lifecycle_state +
		// is_online 双列。prepareDeviceUpdate 已显式维护这两个新字段（收到 Inform 即
		// IsOnline=true、Discovered/Registered 升 Commissioned），这里直接持久化。
		// AND deleted_at IS NULL：防止软删的 device 被 inform 静默复活
		// （cache stale → 这里 UPDATE 仍命中已删行，silent data corruption）
		// Phase 6 follow-up：与 batchUpdateDevices 对齐,加 model_name。
		query := `UPDATE devices SET
			oui = $1, product_class = $2, manufacturer = $3,
			technology = $4,
			lifecycle_state = $5, is_online = $6, firmware_version = $7,
			ip_address = $8, connection_request_url = $9,
			nat_detected = $10, udp_connection_request_address = $11,
			last_inform_at = $12, last_inform_events = $13,
			model_name = CASE WHEN $14::text <> '' THEN $14 ELSE model_name END,
			updated_at = NOW()
		WHERE id = $15 AND deleted_at IS NULL`

		batch.Queue(query,
			dev.OUI, dev.ProductClass, dev.Manufacturer,
			dev.Technology,
			dev.LifecycleState, dev.IsOnline, dev.FirmwareVersion,
			ipAddr, dev.ConnectionRequestURL,
			dev.NatDetected, udpAddr,
			dev.LastInformAt, eventsData,
			dev.ModelName,
			dev.ID,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(devices); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("exec batch update item %d: %w", i, err)
		}
	}
	return nil
}

// dummy use to avoid uuid import error
var _ = uuid.New

// applyProductMetadataInline 是 batch path Submit 时调的 ModelName 回填 helper。
//
// 复制 DeviceService.applyProductMetadata 的核心算法（避免循环依赖 — batchProcessor
// 已经在 device 包内,直接调 DeviceService 方法会让 batchProcessor 持有 DeviceService
// 指针,在 ModuleGraph 接线层制造循环）。helper 的语义与 device_service.go 中的
// applyProductMetadata 严格对齐（设计文档 §4.3 方案 X）。
//
// 调用方在外侧已经判断了 device.ModelName == "" && device.ProductClass != ""。
// 这里仅做 MatchProductClass + 写入 + 错误降级。
func applyProductMetadataInline(ctx context.Context, matcher ProductClassMatcher, device *model.Device, logger *zap.Logger) {
	matchRes, err := matcher.MatchProductClass(ctx, device.ProductClass)
	if err != nil {
		// ErrOrphan 是合法业务态（productClass 未登记）；其他错误视为 soft fail。
		if !errors.Is(err, product.ErrOrphan) {
			logger.Warn("batch path applyProductMetadata: MatchProductClass failed (non-fatal)",
				zap.String("serial_number", device.SerialNumber),
				zap.String("product_class", device.ProductClass),
				zap.Error(err))
		}
		return
	}
	if matchRes == nil || matchRes.Product == nil {
		return
	}
	// 设备所属产品随 productClass 实时重算：命中后回填 product_id + param_model_id，
	// 由 flush UPDATE 持久化（新增设备 / product_class 变更都会重新匹配）。
	pid := matchRes.Product.ID
	device.ProductID = &pid
	device.ParamModelID = matchRes.Product.ParamModelID
	if device.ModelName == "" && matchRes.Product.Name != "" {
		device.ModelName = matchRes.Product.Name
	}
	// 跟 device_service.applyProductMetadata 严格对齐(设计文档 §4.3 方案 X):
	//   · 产品字典登记了 tech → 以字典覆盖,修正首次 Inform 被错误推断的
	//     Technology(5G 设备被推成 lte 的根因);
	//   · 产品已登记但 tech 为空(非无线产品,如核心网) → 清空,避免兜底推断的
	//     lte 残留(设备列表错误显示 eNB(LTE))。
	if isUPSProductClass(device.ProductClass) {
		if device.Technology != "" {
			logger.Info("batch path applyProductMetadata: technology cleared for UPS product",
				zap.String("serial_number", device.SerialNumber),
				zap.String("product_class", device.ProductClass),
				zap.String("old", string(device.Technology)))
			device.Technology = ""
		}
	} else if normalized := model.NormalizeTechnology(matchRes.Product.Tech); normalized != "" {
		if device.Technology != normalized {
			logger.Info("batch path applyProductMetadata: technology corrected from ProductRegistry",
				zap.String("serial_number", device.SerialNumber),
				zap.String("product_class", device.ProductClass),
				zap.String("old", string(device.Technology)),
				zap.String("new", string(normalized)))
			device.Technology = normalized
		}
	} else if device.Technology != "" {
		logger.Info("batch path applyProductMetadata: technology cleared (product registered without tech)",
			zap.String("serial_number", device.SerialNumber),
			zap.String("product_class", device.ProductClass),
			zap.String("old", string(device.Technology)))
		device.Technology = ""
	}
}

// asyncSyncDeviceInfo 在 batch flush 成功后异步把 device_parameters 投影到
// device_info 表（Phase 3 设计文档 §4.2 Layer C）。
//
// 异步执行避免阻塞 flush 主路径;失败仅 WARN,下次 Inform 周期会自动重试
// (InfoSyncer 内部从 paramRepo.GetByDevice 全量读取,幂等)。
//
// 固定大小 worker pool + processor 级 semaphore 限制短时并发，避免多个 flush
// 同时执行时把设备数直接放大为相同数量的 goroutine 和主库连接。
func (p *BatchInformProcessor) asyncSyncDeviceInfo(hit []*informUpdate) {
	const maxProjectionWorkers = 8
	workerCount := min(maxProjectionWorkers, len(hit))
	slots := p.infoProjectionSlots
	if slots == nil {
		slots = make(chan struct{}, maxProjectionWorkers)
	}
	jobs := make(chan *model.Device)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for device := range jobs {
				slots <- struct{}{}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err := p.infoSyncer.SyncFromParameters(
					ctx, device.ID, device.Carrier, device.Technology, device.ProductClass,
				)
				cancel()
				<-slots
				if err != nil {
					p.infoProjectionRetry.Store(device.ID, struct{}{})
					if p.metrics != nil {
						p.metrics.BatchInfoProjection.WithLabelValues("failed").Inc()
					}
					p.logger.Warn("batch path InfoSyncer.SyncFromParameters failed (non-fatal)",
						zap.String("serial_number", device.SerialNumber),
						zap.Error(err))
					continue
				}
				p.infoProjectionRetry.Delete(device.ID)
				if p.metrics != nil {
					p.metrics.BatchInfoProjection.WithLabelValues("completed").Inc()
				}
			}
		}()
	}
	for _, update := range hit {
		if update == nil || update.device == nil || isUPSProductClass(update.device.ProductClass) {
			continue
		}
		jobs <- update.device
	}
	close(jobs)
	workers.Wait()
}
