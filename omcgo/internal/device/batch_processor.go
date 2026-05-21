package device

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
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
	heartbeat   *HeartbeatMonitor
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
}

// TransitionEventPublisher 是 BatchInformProcessor 调 DeviceService 发布
// device.online / device.firmware.changed 事件的窄接口（避免反向依赖 DeviceService 整体）。
type TransitionEventPublisher interface {
	PublishDeviceOnlineEvent(ctx context.Context, device *model.Device)
	PublishDeviceFirmwareChangedEvent(ctx context.Context, device *model.Device,
		oldVersion, newVersion string, becameOnline bool)
}

// NewBatchInformProcessor creates a new BatchInformProcessor.
func NewBatchInformProcessor(
	cfg appconfig.BatchProcessorConfig,
	pool *pgxpool.Pool,
	redisClient redis.UniversalClient,
	heartbeat *HeartbeatMonitor,
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
		workers:         workers,
		flushInterval:   flushInterval,
		maxBatchSize:    maxBatchSize,
		shutdownTimeout: shutdownTimeout,
		pool:            pool,
		redisClient:     redisClient,
		heartbeat:       heartbeat,
		cache:           cache,
		stunUpdater:     stunUpdater,
		metrics:         metrics,
		logger:          logger,
		workerChans:     workerChans,
		stopCh:          make(chan struct{}),
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
	if p.heartbeat != nil {
		p.heartbeat.RefreshHeartbeat(context.Background(), device.SerialNumber, device.InformInterval)
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
	if err := p.batchUpsertParams(ctx, hit); err != nil {
		return fmt.Errorf("batch upsert params: %w", err)
	}

	// 3. 批量 Redis 操作（仅 hit，避免把 stale device 写回 cache）
	p.batchRedisOps(ctx, hit)

	// 4. T-0123/T-0125: PG + cache 写入成功后发 transition 事件。
	// 与 UpdateFromInform 非 batch 路径行为对齐（device_service.go §UpdateFromInform 末尾）。
	// 同一 Inform 满足两者时优先发 firmware.changed（不发 device.online），由
	// HandleFirmwareChanged 触发的重新交集 + Path B 覆盖 online 的能力，避免双 Path B。
	if p.transitionPublisher != nil {
		for _, u := range hit {
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
	}

	return nil
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
		query := `UPDATE devices SET
			oui = $1, product_class = $2, manufacturer = $3,
			lifecycle_state = $4, is_online = $5, firmware_version = $6,
			ip_address = $7, connection_request_url = $8,
			nat_detected = $9, udp_connection_request_address = $10,
			last_inform_at = $11, last_inform_events = $12,
			updated_at = NOW()
		WHERE id = $13 AND deleted_at IS NULL`

		batch.Queue(query,
			dev.OUI, dev.ProductClass, dev.Manufacturer,
			dev.LifecycleState, dev.IsOnline, dev.FirmwareVersion,
			ipAddr, dev.ConnectionRequestURL,
			dev.NatDetected, udpAddr,
			dev.LastInformAt, eventsData,
			dev.ID,
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

// batchUpsertParams 将所有设备的参数合并到一个 pgx.Batch 中执行。
func (p *BatchInformProcessor) batchUpsertParams(ctx context.Context, updates []*informUpdate) error {
	batch := &pgx.Batch{}
	totalParams := 0

	for _, u := range updates {
		if len(u.params) == 0 {
			continue
		}
		now := time.Now()
		for _, param := range u.params {
			query, args, err := storage.Psql.Insert("device_parameters").
				Columns("device_id", "parameter_path", "parameter_value", "parameter_type", "writable", "last_updated_at").
				Values(u.device.ID, param.ParameterPath, param.ParameterValue, param.ParameterType, param.Writable, now).
				Suffix("ON CONFLICT (device_id, parameter_path) DO UPDATE SET parameter_value = EXCLUDED.parameter_value, parameter_type = EXCLUDED.parameter_type, writable = EXCLUDED.writable, last_updated_at = EXCLUDED.last_updated_at").
				ToSql()
			if err != nil {
				return fmt.Errorf("build upsert query: %w", err)
			}
			batch.Queue(query, args...)
			totalParams++
		}
	}

	if totalParams == 0 {
		return nil
	}

	br := p.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < totalParams; i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("exec batch param upsert item %d: %w", i, err)
		}
	}
	return nil
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

		// STUN 地址同步
		if dev.UDPConnectionRequestAddress != "" && p.stunUpdater != nil {
			// STUN 通过接口更新，不走 pipeline（接口不暴露 pipeline）
			// 延迟到 pipeline 执行后单独处理
		}
	}

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		p.logger.Warn("batch redis pipeline", zap.Error(err))
	}

	// STUN 地址单独处理（StunAddressUpdater 接口不支持 pipeline）
	for _, u := range updates {
		dev := u.device
		if dev.UDPConnectionRequestAddress != "" && p.stunUpdater != nil {
			if err := p.stunUpdater.SetFromInform(ctx, dev.SerialNumber, dev.UDPConnectionRequestAddress); err != nil {
				p.logger.Warn("batch stun address sync",
					zap.String("device_sn", dev.SerialNumber), zap.Error(err))
			}
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
	device.FirmwareVersion = findParamValue(inform.ParameterList, "Device.DeviceInfo.SoftwareVersion")
	device.ConnectionRequestURL = findParamValue(inform.ParameterList, "Device.ManagementServer.ConnectionRequestURL")
	device.LastInformAt = &now
	device.LastInformEvents = tr069.EventCodes(inform.Event)

	udpAddr := findParamValue(inform.ParameterList, "Device.ManagementServer.UDPConnectionRequestAddress")
	stunChanged := false
	if udpAddr != "" {
		device.IPAddress = udpAddr
		device.UDPConnectionRequestAddress = udpAddr
		device.NatDetected = true
		stunChanged = true
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
		query := `UPDATE devices SET
			oui = $1, product_class = $2, manufacturer = $3,
			lifecycle_state = $4, is_online = $5, firmware_version = $6,
			ip_address = $7, connection_request_url = $8,
			nat_detected = $9, udp_connection_request_address = $10,
			last_inform_at = $11, last_inform_events = $12,
			updated_at = NOW()
		WHERE id = $13 AND deleted_at IS NULL`

		batch.Queue(query,
			dev.OUI, dev.ProductClass, dev.Manufacturer,
			dev.LifecycleState, dev.IsOnline, dev.FirmwareVersion,
			ipAddr, dev.ConnectionRequestURL,
			dev.NatDetected, udpAddr,
			dev.LastInformAt, eventsData,
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
