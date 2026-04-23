package alarm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// 频繁异常重启告警：在滑动窗口内，同一设备累计 >=threshold 次异常重启则触发。
//
// 由 device.InformHandler 在处理 "1 BOOT" 但不含 "M Reboot" 的 Inform 时发布
// SubjectDeviceRebootAbnormal 事件。RebootMonitor 订阅该事件，借助 Redis 有序集合
// 按时间窗口滑动计数，超过阈值时构造 FREQUENT_ABNORMAL_REBOOT 告警交给 AlarmEngine。
const (
	// DefaultFrequentRebootWindow 窗口默认值：5 分钟。
	DefaultFrequentRebootWindow = 5 * time.Minute
	// DefaultFrequentRebootThreshold 窗口内触发阈值默认值：3 次。
	DefaultFrequentRebootThreshold = 3
	// AlarmCodeFrequentReboot 频繁异常重启告警码。
	AlarmCodeFrequentReboot = "FREQUENT_ABNORMAL_REBOOT"

	rebootMonitorQueue  = "alarm-reboot-monitor"
	rebootMonitorKeyTTL = 24 * time.Hour
)

func rebootWindowKey(sn string) string {
	return redisx.Keys.RebootAbnormal(sn)
}

// RebootMonitorOption 用函数式选项覆盖滑动窗口配置。
type RebootMonitorOption func(*RebootMonitor)

// WithRebootWindow 设置滑动窗口长度。
func WithRebootWindow(w time.Duration) RebootMonitorOption {
	return func(m *RebootMonitor) {
		if w > 0 {
			m.window = w
		}
	}
}

// WithRebootThreshold 设置窗口内触发告警的次数阈值（需 >=2）。
func WithRebootThreshold(n int) RebootMonitorOption {
	return func(m *RebootMonitor) {
		if n >= 2 {
			m.threshold = n
		}
	}
}

// RebootAbnormalPayload 与 device.DeviceService.RecordBootFromInform 发布的 payload 对齐。
type RebootAbnormalPayload struct {
	DeviceID     string    `json:"device_id"`
	SerialNumber string    `json:"serial_number"`
	Carrier      string    `json:"carrier"`
	BootCount    int       `json:"boot_count"`
	LastBootAt   time.Time `json:"last_boot_at"`
	Events       []string  `json:"events"`
}

// RebootMonitor 监听 SubjectDeviceRebootAbnormal 并在滑动窗口内累计到阈值后抬升告警。
type RebootMonitor struct {
	engine    *AlarmEngine
	redis     redis.UniversalClient
	window    time.Duration
	threshold int
	logger    *zap.Logger
}

// NewRebootMonitor 构造 RebootMonitor。engine、redis、logger 均不可为 nil。
func NewRebootMonitor(
	engine *AlarmEngine,
	client redis.UniversalClient,
	logger *zap.Logger,
	opts ...RebootMonitorOption,
) *RebootMonitor {
	m := &RebootMonitor{
		engine:    engine,
		redis:     client,
		window:    DefaultFrequentRebootWindow,
		threshold: DefaultFrequentRebootThreshold,
		logger:    logger,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Subscribe 注册到事件总线。
func (m *RebootMonitor) Subscribe(bus event.EventBus) error {
	if _, err := bus.QueueSubscribe(event.SubjectDeviceRebootAbnormal, rebootMonitorQueue, m.handle); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectDeviceRebootAbnormal, err)
	}
	m.logger.Info("reboot monitor subscribed",
		zap.String("subject", event.SubjectDeviceRebootAbnormal),
		zap.Duration("window", m.window),
		zap.Int("threshold", m.threshold),
	)
	return nil
}

func (m *RebootMonitor) handle(ctx context.Context, evt event.Event) error {
	var payload RebootAbnormalPayload
	if err := evt.DecodePayload(&payload); err != nil {
		m.logger.Warn("decode reboot.abnormal payload", zap.Error(err))
		return nil // do not re-deliver on malformed payload
	}
	if payload.SerialNumber == "" {
		return nil
	}

	now := time.Now()
	count, err := m.recordAndCount(ctx, payload.SerialNumber, now)
	if err != nil {
		// Best-effort counting: Redis hiccup shouldn't block alarm processing
		// or the upstream event bus — just log and drop the event.
		m.logger.Warn("count abnormal reboots in window failed",
			zap.String("serial_number", payload.SerialNumber),
			zap.Error(err))
		return nil
	}

	m.logger.Debug("abnormal reboot recorded",
		zap.String("serial_number", payload.SerialNumber),
		zap.Int("window_count", count),
		zap.Int("threshold", m.threshold),
		zap.Duration("window", m.window))

	if count < m.threshold {
		return nil
	}

	deviceID, _ := uuid.Parse(payload.DeviceID) // zero UUID on parse failure is acceptable — store still keyed by sn

	alarm := &model.Alarm{
		DeviceID:    deviceID,
		DeviceSN:    payload.SerialNumber,
		Carrier:     model.CarrierCode(payload.Carrier),
		AlarmIdentifier: AlarmCodeFrequentReboot,
		AlarmType:   "device",
		AlarmSource: strPtr("acs"),
		EventType:   strPtr("abnormal_reboot"),
		Description: fmt.Sprintf("设备在 %s 内异常重启 %d 次（阈值 %d）", m.window, count, m.threshold),
		Severity:    model.AlarmMajor,
		RaisedAt:    now,
		AdditionalInfo: map[string]string{
			"window_seconds": fmt.Sprintf("%.0f", m.window.Seconds()),
			"threshold":      fmt.Sprintf("%d", m.threshold),
			"reboot_count":   fmt.Sprintf("%d", count),
			"last_events":    strings.Join(payload.Events, ","),
		},
	}

	if err := m.engine.Process(ctx, alarm); err != nil {
		return fmt.Errorf("process frequent reboot alarm: %w", err)
	}
	return nil
}

// recordAndCount atomically appends the current event to the device's sliding
// window and returns the number of events remaining inside the window.
func (m *RebootMonitor) recordAndCount(ctx context.Context, sn string, now time.Time) (int, error) {
	key := rebootWindowKey(sn)
	cutoff := now.Add(-m.window).UnixMilli()
	score := now.UnixMilli()

	pipe := m.redis.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(score),
		Member: fmt.Sprintf("%d-%s", score, uuid.New().String()),
	})
	// Exclusive cutoff: strictly older than the window boundary.
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("(%d", cutoff))
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, rebootMonitorKeyTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(countCmd.Val()), nil
}
