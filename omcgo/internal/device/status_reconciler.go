package device

import (
	"context"
	"fmt"
	"strings"
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
//  1. RefreshHeartbeat（ACS Inform 热路径同步调用):写 Redis 心跳 key,TTL = 2×inform_interval。
//  2. Start/Stop（后台扫描):周期遍历"超过自适应阈值未上报"的在线设备,事务性
//     翻转 is_online=false + 累加 cumulative_online_duration + publish device.offline。
//
// 设计要点:
//   - 离线判定**唯一信源**是 PG 的 devices.last_inform_at（durable,重启不丢)。
//     Redis 心跳 key 仅作为 ACS 内部限流 / 会话用途,reconciler 不读它。
//   - 阈值**自适应**:max(2 × inform_interval, MinStaleSec),按设备粒度,在 SQL 里算。
//   - 离线累计**事务性**:repo.MarkOfflineWithAccounting 单 TX 完成 devices + device_info 两表写,
//     避免中间崩溃留下 is_online=false 但 cumulative_online_duration 没累加的脏数据。
//   - 幂等:MarkOfflineWithAccounting 仅在 is_online=true 时翻转;事件只在真翻转后发布。
type DeviceStatusReconciler struct {
	redis     redis.UniversalClient
	repo      statusReconcilerRepo
	eventBus  event.EventBus
	alarmSink offlineAlarmSink
	cache     *DeviceCache
	logger    *zap.Logger

	// onlineIndex acs:online 在线索引（issue #397 根治片）。离线判定在把 last_inform_at
	// 超阈值的候选标离线之前，用此索引（ACS 同步写、免 NATS）做二次确认：候选若 score
	// 仍在其 class 阈值内 → 设备实际在线（last_inform_at 只是 NATS 滞后）→ 不离线。
	// 仅能减少误判离线、绝不新增误判。nil（无 Redis）时退化为不确认（旧行为）。
	onlineIndex *redisx.OnlineIndex

	// issue #203：离线阈值实时配置读取（category=device, key=enbTimeout/cpeTimeout）。
	// nil 时退化到默认阈值（基站 100s / CPE 600s）。每轮扫描读最新值，改配置下一轮即生效。
	thresholdLookup OfflineThresholdLookup

	// 行为参数（默认值见 NewDeviceStatusReconciler;SetXxx 用于测试）。
	// issue #203：checkInterval 不再固定，每轮按 min(阈值/2, 60s) 动态计算；
	// 此字段保留为 SetCheckInterval 显式覆盖时的固定值（>0 时优先于动态计算）。
	checkInterval time.Duration // 显式覆盖扫描周期;0=按配置阈值动态算
	batchSize     int           // 单轮最大处理设备数;默认 1000

	// goroutine 生命周期。
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// statusReconcilerRepo 是 reconciler 消费的窄仓库接口,只包含两个方法。
//
// 故意定义在消费侧:全仓 11 个手写 mockDeviceRepo 都不必新增桩方法,只有
// status_reconciler_test.go 需要 mock 本接口。*PgDeviceRepository 自动满足
// (FindStaleDevicesByClass / MarkOfflineWithAccounting 见 device_repository.go)。
type statusReconcilerRepo interface {
	FindStaleDevicesByClass(ctx context.Context, enbThresholdSec, cpeThresholdSec, upsThresholdSec, limit int) ([]*model.Device, error)
	MarkOfflineWithAccounting(ctx context.Context, deviceID uuid.UUID, reason string, now time.Time) (bool, error)
}

type offlineAlarmSink interface {
	Process(ctx context.Context, alarm *model.Alarm) error
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
		redis:    redisClient,
		repo:     repo,
		eventBus: eventBus,
		logger:   logger,
		// issue #397：从同一 Redis 客户端构造在线索引（redisClient 为 nil 时 client 为 nil，
		// Scores 返回 nil → 退化为不确认）。
		onlineIndex: redisx.NewOnlineIndex(redisClient),
		// issue #203：checkInterval=0 表示不固定，每轮按 min(配置阈值/2, 60s) 动态算。
		// thresholdLookup 由 wiring 层经 SetThresholdLookup 注入；未注入则用默认阈值。
		checkInterval: 0,
		batchSize:     1000,
	}
}

// SetThresholdLookup 注入离线阈值的 sys_configs 读取器（issue #203）。
// 仅在 wiring 阶段调用，nil 安全（不注入则全程用默认阈值 100/600）。
func (r *DeviceStatusReconciler) SetThresholdLookup(lookup OfflineThresholdLookup) {
	r.thresholdLookup = lookup
}

// SetOfflineAlarmSink 注入 OMC 源设备断连告警写入器。nil 时仅维护在线状态和发布事件。
func (r *DeviceStatusReconciler) SetOfflineAlarmSink(sink offlineAlarmSink) {
	r.alarmSink = sink
}

// SetDeviceCache 注入设备缓存，用于离线翻转后失效旧的在线快照。
func (r *DeviceStatusReconciler) SetDeviceCache(cache *DeviceCache) {
	r.cache = cache
}

// currentThresholds 读取本轮扫描使用的两类离线阈值（每轮实时读，改配置即生效）。
func (r *DeviceStatusReconciler) currentThresholds(ctx context.Context) OfflineThresholds {
	return resolveOfflineThresholds(ctx, r.thresholdLookup)
}

// nextScanInterval 返回下一轮扫描周期。SetCheckInterval 显式覆盖（>0）时用固定值，
// 否则按 min(配置阈值/2, 60s) 动态计算（issue #203）。
func (r *DeviceStatusReconciler) nextScanInterval(th OfflineThresholds) time.Duration {
	if r.checkInterval > 0 {
		return r.checkInterval
	}
	return scanIntervalFor(th)
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

	th := r.currentThresholds(ctx)
	r.logger.Info("device status reconciler started",
		zap.Duration("scan_interval", r.nextScanInterval(th)),
		zap.Int("enb_threshold_sec", th.ENBSec),
		zap.Int("cpe_threshold_sec", th.CPESec),
		zap.Int("ups_threshold_sec", th.UPSSec),
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
// 之后按动态周期 = min(配置阈值/2, 60s) 重排 timer（issue #203：阈值改小后
// 扫描频率随之加快，下一轮即按新配置判离线）。
func (r *DeviceStatusReconciler) loop(ctx context.Context) {
	defer r.wg.Done()

	// 首次立即扫,不等第一个 tick；同时拿到本轮阈值用于排下一次 timer。
	th := r.detect(ctx)

	timer := time.NewTimer(r.nextScanInterval(th))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("device status reconciler stopped")
			return
		case <-timer.C:
			th = r.detect(ctx)
			timer.Reset(r.nextScanInterval(th))
		}
	}
}

// detect 执行一轮扫描:读最新阈值,找过期设备,逐个标记离线。
// 返回本轮使用的阈值,供 loop 计算下一次扫描周期。
func (r *DeviceStatusReconciler) detect(ctx context.Context) OfflineThresholds {
	start := time.Now()
	th := r.currentThresholds(ctx)
	devices, err := r.repo.FindStaleDevicesByClass(ctx, th.ENBSec, th.CPESec, th.UPSSec, r.batchSize)
	if err != nil {
		r.logger.Error("find stale devices failed", zap.Error(err))
		return th
	}
	if len(devices) == 0 {
		return th
	}

	// issue #397 根治片：用 acs:online（ACS 同步写、免 NATS）二次确认。候选里 score 仍在
	// 其 class 阈值内的设备，说明实际在上报（last_inform_at 只是 NATS 滞后）→ 从离线名单剔除。
	candidateCount := len(devices)
	rescued := 0
	if r.onlineConfirmEnabled(ctx) {
		devices, rescued = r.vetoAliveByOnlineIndex(ctx, devices, th, start.Unix())
	}

	r.logger.Info("detected stale devices",
		zap.Int("candidates", candidateCount),
		zap.Int("rescued_by_online_index", rescued),
		zap.Int("to_offline", len(devices)))

	if len(devices) == 0 {
		return th // 全部被在线索引确认为在线，无需标离线
	}

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
	return th
}

// onlineConfirmEnabled 报告本轮是否启用 acs:online 二次确认（issue #397）。
// 索引未接线（无 Redis）→ false；sys_configs 显式 false/0/off → false；其余 → 默认 true。
func (r *DeviceStatusReconciler) onlineConfirmEnabled(ctx context.Context) bool {
	if r.onlineIndex == nil {
		return false
	}
	if r.thresholdLookup == nil {
		return true
	}
	v, found := r.thresholdLookup(ctx, offlineConfigCategory, offlineConfigKeyZsetConfirm)
	if !found {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

// vetoAliveByOnlineIndex 从离线候选中剔除「acs:online 确认仍在线」的设备：
// 候选 d 若在索引中的 score ≥ now-该设备类阈值，则其实际在线（last_inform_at 是 NATS
// 滞后造成的假离线）→ 不标离线。返回应继续标离线的子集 + 被救回数。
// 索引读失败 → 回退到不剔除（与未接索引前行为一致，安全：绝不因索引故障漏判真离线）。
func (r *DeviceStatusReconciler) vetoAliveByOnlineIndex(
	ctx context.Context, candidates []*model.Device, th OfflineThresholds, nowUnix int64,
) (keep []*model.Device, rescued int) {
	if r.onlineIndex == nil || len(candidates) == 0 {
		return candidates, 0
	}
	sns := make([]string, len(candidates))
	for i, d := range candidates {
		sns[i] = d.SerialNumber
	}
	scores, err := r.onlineIndex.Scores(ctx, sns...)
	if err != nil || len(scores) != len(candidates) {
		if err != nil {
			r.logger.Warn("online index scores failed; skip veto (fall back to last_inform_at)",
				zap.Error(err))
		}
		return candidates, 0
	}

	keep = make([]*model.Device, 0, len(candidates))
	for i, d := range candidates {
		threshold := th.ENBSec
		if isUPSProductClass(d.ProductClass) {
			threshold = th.UPSSec
		} else if isCPEClass(d.ProductClass) {
			threshold = th.CPESec
		}
		cutoff := nowUnix - int64(threshold)
		// score>0 排除「不在索引中」（ZMSCORE 缺失返回 0）；score≥cutoff 即阈值内有上报。
		if scores[i] > 0 && scores[i] >= float64(cutoff) {
			rescued++
			continue
		}
		keep = append(keep, d)
	}
	return keep, rescued
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

	if r.cache != nil {
		r.cache.Delete(ctx, device.SerialNumber)
	}

	if err := r.raiseOfflineAlarm(ctx, device, now); err != nil {
		r.logger.Warn("raise device disconnected alarm failed",
			zap.Error(err),
			zap.String("device_id", device.ID.String()),
			zap.String("serial", device.SerialNumber),
			zap.String("technology", string(device.Technology)),
		)
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

func (r *DeviceStatusReconciler) raiseOfflineAlarm(ctx context.Context, device *model.Device, raisedAt time.Time) error {
	if r.alarmSink == nil {
		return nil
	}
	if device != nil && isUPSProductClass(device.ProductClass) {
		return nil
	}
	identifier, description, ok := disconnectedAlarmForTechnology(device.Technology)
	if !ok {
		return nil
	}
	omcSource := "OMC"
	eventType := "30000"
	technology := string(device.Technology)
	alarm := &model.Alarm{
		DeviceID:        device.ID,
		DeviceSN:        device.SerialNumber,
		Carrier:         device.Carrier,
		Severity:        model.AlarmCritical,
		AlarmType:       "communication",
		AlarmIdentifier: identifier,
		Description:     description,
		Status:          model.AlarmActive,
		RaisedAt:        raisedAt,
		AlarmSource:     &omcSource,
		EventType:       &eventType,
		Technology:      &technology,
	}
	return r.alarmSink.Process(ctx, alarm)
}

func disconnectedAlarmForTechnology(tech model.Technology) (identifier string, description string, ok bool) {
	switch tech {
	case model.TechLTE:
		return "7", "eNB Disconnected", true
	case model.TechNR:
		return "23", "gNB Disconnected", true
	case model.TechGSM:
		return "4", "GSM Disconnected", true
	default:
		return "", "", false
	}
}

// SetCheckInterval 显式覆盖扫描周期（仅用于测试或运行时调优）。
// 设为 >0 后优先于按阈值动态计算的周期；0 不生效（保持动态）。
func (r *DeviceStatusReconciler) SetCheckInterval(d time.Duration) {
	if d > 0 {
		r.checkInterval = d
	}
}

// SetBatchSize 覆盖单轮处理上限（仅用于测试或运行时调优）。
func (r *DeviceStatusReconciler) SetBatchSize(n int) {
	if n > 0 {
		r.batchSize = n
	}
}
