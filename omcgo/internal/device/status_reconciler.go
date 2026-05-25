package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// T-0173: 离线原因常量。落 devices.last_offline_reason,排障 / 审计可读。
const (
	OfflineReasonHeartbeatTimeout = "heartbeat_timeout"
	OfflineReasonManual           = "manual"
	OfflineReasonReboot           = "reboot"
)

// DeviceStatusReconciler 异步维护设备在线状态 + 累计在线时长。
//
// T-0173 单一收口:替代原来的 HeartbeatMonitor + OfflineDetector 双扫描器。
//
//	1) RefreshHeartbeat（ACS Inform 热路径同步调用):写 Redis 心跳 key,TTL = 2×inform_interval。
//	2) Start/Stop（后台扫描):周期遍历"超过自适应阈值未上报"的在线设备,事务性
//	   翻转 is_online=false + 累加 cumulative_online_duration + publish device.offline。
//
// 设计要点:
//   - 离线判定**唯一信源**是 PG 的 devices.last_inform_at（durable,重启不丢)。
//     Redis 心跳 key 仅作为 ACS 内部限流 / 会话用途,reconciler 不读它。
//   - 阈值**自适应**:max(2 × inform_interval, MinStaleSec),按设备粒度,在 SQL 里算。
//   - 离线累计**事务性**:repo.MarkOfflineWithAccounting 单 TX 完成 devices + device_info 两表写,
//     避免中间崩溃留下 is_online=false 但 cumulative_online_duration 没累加的脏数据。
//   - 幂等:MarkOfflineWithAccounting 仅在 is_online=true 时翻转;事件只在真翻转后发布。
type DeviceStatusReconciler struct {
	redis    redis.UniversalClient
	repo     statusReconcilerRepo
	eventBus event.EventBus
	logger   *zap.Logger

	// 行为参数（默认值见 NewDeviceStatusReconciler;SetXxx 用于测试）。
	checkInterval time.Duration // 扫描周期;默认 60s
	minStaleSec   int           // 离线阈值下限（秒);默认 600 (10 分钟)
	batchSize     int           // 单轮最大处理设备数;默认 1000

	// goroutine 生命周期。
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// statusReconcilerRepo 是 reconciler 消费的窄仓库接口,只包含两个方法。
//
// 故意定义在消费侧:全仓 11 个手写 mockDeviceRepo 都不必新增桩方法,只有
// status_reconciler_test.go 需要 mock 本接口。*PgDeviceRepository 自动满足
// (FindStaleDevicesAdaptive / MarkOfflineWithAccounting 见 device_repository.go)。
type statusReconcilerRepo interface {
	FindStaleDevicesAdaptive(ctx context.Context, minStaleSec int, limit int) ([]*model.Device, error)
	MarkOfflineWithAccounting(ctx context.Context, deviceID uuid.UUID, reason string, now time.Time) (bool, error)
}

// DeviceOfflineEvent 设备离线事件载荷。
//
// 与原 OfflineDetector 的同名结构二进制兼容:订阅者 (provision.HandleDeviceOffline
// 等) 不需要调整。事件 subject 沿用 "device.offline"。
type DeviceOfflineEvent struct {
	DeviceID    uuid.UUID         `json:"device_id"`
	Serial      string            `json:"serial_number"`
	Carrier     model.CarrierCode `json:"carrier"`
	Technology  model.Technology  `json:"technology"`
	OfflineTime time.Time         `json:"offline_time"`
	Reason      string            `json:"reason"`
}

// NewDeviceStatusReconciler 构造一个 reconciler 实例。
//
// repo 通常传 *PgDeviceRepository（同时满足 DeviceRepository 与 statusReconcilerRepo);
// eventBus 可为 nil（dev/test),此时离线事件不发布,其它功能正常。
func NewDeviceStatusReconciler(
	redisClient redis.UniversalClient,
	repo statusReconcilerRepo,
	eventBus event.EventBus,
	logger *zap.Logger,
) *DeviceStatusReconciler {
	return &DeviceStatusReconciler{
		redis:         redisClient,
		repo:          repo,
		eventBus:      eventBus,
		logger:        logger,
		checkInterval: 60 * time.Second,
		minStaleSec:   600,
		batchSize:     1000,
	}
}

// RefreshHeartbeat 写 / 续 Redis 心跳 key,由 ACS Inform 处理路径每次同步调用。
//
// Key: acs:heartbeat:{sn}；Value: 当前 Unix 秒；TTL: 2 × informInterval,
// 不足 60s 时强制提升到 600s（10 分钟),与 reconciler 最小阈值对齐。
//
// 注意:reconciler 后台扫描器**不读**这个 key,完全靠 PG.last_inform_at 判定离线。
// 此 key 仅供 ACS 限流 / 会话其他模块使用(如 ratelimit:inform 协同等)。
func (r *DeviceStatusReconciler) RefreshHeartbeat(ctx context.Context, deviceSN string, informInterval int) {
	if r == nil || r.redis == nil {
		return
	}
	key := redisx.Keys.ACSHeartbeat(deviceSN)
	ttl := time.Duration(informInterval*2) * time.Second
	if ttl < 60*time.Second {
		ttl = 600 * time.Second
	}
	if err := r.redis.Set(ctx, key, time.Now().Unix(), ttl).Err(); err != nil {
		r.logger.Error("refresh heartbeat", zap.Error(err), zap.String("device_sn", deviceSN))
	}
}

// Start 启动后台扫描 goroutine。非阻塞:返回后调用方需自行 .Stop() 优雅退出。
//
// 与旧 OfflineDetector.Start(ctx) 阻塞返回 ctx.Err() 的语义不同:
// 这里 fire-and-forget,Stop() 触发 cancel。匹配 provider/device.go GS.Register
// 的关停 hook 模型。
func (r *DeviceStatusReconciler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	r.wg.Add(1)
	go r.loop(ctx)

	r.logger.Info("device status reconciler started",
		zap.Duration("check_interval", r.checkInterval),
		zap.Int("min_stale_sec", r.minStaleSec),
		zap.Int("batch_size", r.batchSize),
	)
}

// Stop 取消 ctx 并等待扫描 goroutine 退出。可多次安全调用。
func (r *DeviceStatusReconciler) Stop() {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.wg.Wait()
}

// loop 是扫描器主循环。首次 tick 立即执行一次（与旧 OfflineDetector 行为对齐),
// 之后按 checkInterval 周期。
func (r *DeviceStatusReconciler) loop(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(r.checkInterval)
	defer ticker.Stop()

	// 首次立即扫,不等第一个 tick。
	r.detect(ctx)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("device status reconciler stopped")
			return
		case <-ticker.C:
			r.detect(ctx)
		}
	}
}

// detect 执行一轮扫描:找过期设备,逐个标记离线。
func (r *DeviceStatusReconciler) detect(ctx context.Context) {
	start := time.Now()
	devices, err := r.repo.FindStaleDevicesAdaptive(ctx, r.minStaleSec, r.batchSize)
	if err != nil {
		r.logger.Error("find stale devices failed", zap.Error(err))
		return
	}
	if len(devices) == 0 {
		return
	}

	r.logger.Info("detected stale devices", zap.Int("count", len(devices)))

	var (
		transitionedCount int
		failureCount      int
	)
	for _, d := range devices {
		ok, err := r.markOffline(ctx, d)
		switch {
		case err != nil:
			failureCount++
			r.logger.Error("mark device offline failed",
				zap.Error(err),
				zap.String("device_id", d.ID.String()),
				zap.String("serial_number", d.SerialNumber),
			)
		case ok:
			transitionedCount++
		}
	}

	r.logger.Info("device status reconcile pass complete",
		zap.Int("scanned", len(devices)),
		zap.Int("transitioned", transitionedCount),
		zap.Int("failures", failureCount),
		zap.Duration("elapsed", time.Since(start)),
	)
}

// markOffline 把单台设备从 online→offline,并发布事件（仅在真翻转时)。
//
// 返回 transitioned=true 表示这次调用确实改了状态;false 表示设备已经离线 / 已删除
// (幂等保护,不重复发事件)。
func (r *DeviceStatusReconciler) markOffline(ctx context.Context, device *model.Device) (bool, error) {
	now := time.Now()
	transitioned, err := r.repo.MarkOfflineWithAccounting(ctx, device.ID, OfflineReasonHeartbeatTimeout, now)
	if err != nil {
		return false, fmt.Errorf("mark offline with accounting: %w", err)
	}
	if !transitioned {
		return false, nil
	}

	if r.eventBus != nil {
		payload := DeviceOfflineEvent{
			DeviceID:    device.ID,
			Serial:      device.SerialNumber,
			Carrier:     device.Carrier,
			Technology:  device.Technology,
			OfflineTime: now,
			Reason:      OfflineReasonHeartbeatTimeout,
		}
		evt, evtErr := event.NewEvent(event.SubjectDeviceOffline, payload)
		if evtErr != nil {
			r.logger.Warn("create device.offline event failed",
				zap.Error(evtErr),
				zap.String("device_id", device.ID.String()))
		} else if pubErr := r.eventBus.Publish(ctx, event.SubjectDeviceOffline, evt); pubErr != nil {
			r.logger.Warn("publish device.offline failed",
				zap.Error(pubErr),
				zap.String("device_id", device.ID.String()))
		}
	}

	r.logger.Info("device marked offline",
		zap.String("device_id", device.ID.String()),
		zap.String("serial", device.SerialNumber),
		zap.String("carrier", string(device.Carrier)),
		zap.String("reason", OfflineReasonHeartbeatTimeout),
		zap.Time("offline_time", now),
	)
	return true, nil
}

// SetCheckInterval 覆盖扫描周期（仅用于测试或运行时调优）。
func (r *DeviceStatusReconciler) SetCheckInterval(d time.Duration) {
	if d > 0 {
		r.checkInterval = d
	}
}

// SetMinStaleSec 覆盖离线阈值下限（仅用于测试或运行时调优）。
func (r *DeviceStatusReconciler) SetMinStaleSec(sec int) {
	if sec > 0 {
		r.minStaleSec = sec
	}
}

// SetBatchSize 覆盖单轮处理上限（仅用于测试或运行时调优）。
func (r *DeviceStatusReconciler) SetBatchSize(n int) {
	if n > 0 {
		r.batchSize = n
	}
}
