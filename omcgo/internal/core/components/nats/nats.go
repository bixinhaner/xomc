package nats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
)

// NATSClient wraps a NATS connection and JetStream context.
type NATSClient struct {
	Conn   *nats.Conn
	JS     nats.JetStreamContext
	logger *zap.Logger

	metricsMu sync.Mutex
	metrics   *ConnMetrics
}

// StreamDef defines a JetStream stream.
type StreamDef struct {
	Name        string
	Subjects    []string
	Retention   nats.RetentionPolicy
	AllowDirect bool
	MaxAge      time.Duration
	MaxBytes    int64
	Compression nats.StoreCompression
}

const (
	pmAggregationStreamMaxBytes  int64 = 10 << 30
	alarmLifecycleStreamMaxBytes int64 = 10 << 30
)

// DefaultStreams 列出所有 JetStream 流。每条流以一个点分前缀吸纳一类事件，
// 最大存活 72 小时。扩展新事件前缀时，务必在此注册对应 Stream，否则发布
// 的消息不会落盘、订阅者掉线就丢失。
//
// Retention 选择规则：
//   - InterestPolicy（DEVICE / COMMAND / TASK）：消息保留到所有 active
//     consumer 都 ack 后删除。支持多模块独立 fan-out 订阅同一 subject
//     （如 device.registered 同时触发 provisioning + topology rule）。
//   - WorkQueuePolicy（其余）：消息被任一 consumer ack 后立即删除，
//     适合"任务派发"语义（PM/MR/ALARM 文件处理、OSS 导出等）。
//
// 切换 Retention 需 EnsureStreams 删除重建（不可原地修改），由
// NATSConfig.AllowStreamRebuild 控制（生产默认 false）。
func DefaultStreams() []StreamDef {
	return DefaultStreamsWithAlarmLifecycleMaxBytes(alarmLifecycleStreamMaxBytes)
}

// DefaultStreamsWithAlarmLifecycleMaxBytes preserves the bounded default but
// permits deployment-specific sizing of the independent lifecycle stream.
func DefaultStreamsWithAlarmLifecycleMaxBytes(maxBytes int64) []StreamDef {
	if maxBytes <= 0 {
		maxBytes = alarmLifecycleStreamMaxBytes
	}
	return []StreamDef{
		{Name: "DEVICE", Subjects: []string{"device.>"}, Retention: nats.InterestPolicy},
		{Name: "COMMAND", Subjects: []string{"command.>"}, Retention: nats.InterestPolicy},
		{Name: "TASK", Subjects: []string{"task.>"}, Retention: nats.InterestPolicy},
		// Geofence lifecycle and evaluation events fan out to the coordinator,
		// control monitor, and alarm monitor. They must survive process restarts
		// and remain available to every independent durable consumer.
		{Name: "GEOFENCE", Subjects: []string{"geofence.>"}, Retention: nats.InterestPolicy},
		// Parameter-sync task results and run terminal events fan out to the
		// result processor, provisioning bindings, and operational consumers.
		{Name: "PARAM_SYNC", Subjects: []string{"param_sync.>"}, Retention: nats.InterestPolicy},
		// Queue health needs one subject-filtered, read-only raw-message lookup
		// to calculate oldest pm.file.received age without confusing it with
		// other pm.> subjects in this shared stream.
		{Name: "PM", Subjects: []string{"pm.>"}, Retention: nats.WorkQueuePolicy, AllowDirect: true},
		{
			Name:        "PM_AGG_15M",
			Subjects:    []string{"pmaggregation.15m.>"},
			Retention:   nats.LimitsPolicy,
			AllowDirect: true,
			MaxAge:      2 * time.Hour,
			MaxBytes:    pmAggregationStreamMaxBytes,
			Compression: nats.S2Compression,
		},
		{
			Name:        "PM_AGG_HOURLY",
			Subjects:    []string{"pmaggregation.hourly.>"},
			Retention:   nats.LimitsPolicy,
			AllowDirect: true,
			MaxAge:      48 * time.Hour,
			MaxBytes:    pmAggregationStreamMaxBytes,
			Compression: nats.S2Compression,
		},
		{
			Name:        "PM_AGG_DAILY",
			Subjects:    []string{"pmaggregation.daily.>"},
			Retention:   nats.LimitsPolicy,
			AllowDirect: true,
			MaxAge:      40 * 24 * time.Hour,
			MaxBytes:    pmAggregationStreamMaxBytes,
			Compression: nats.S2Compression,
		},
		{
			Name:      "PM_AGG_CONTROL",
			Subjects:  []string{"pmaggregation.control.>"},
			Retention: nats.InterestPolicy,
		},
		{Name: "MR", Subjects: []string{"mr.>"}, Retention: nats.WorkQueuePolicy, AllowDirect: true},
		{Name: "ALARM", Subjects: []string{"alarm.>"}, Retention: nats.WorkQueuePolicy},
		{
			Name:        "DOMAIN_ALARM",
			Subjects:    []string{"domain.alarm.>"},
			Retention:   nats.LimitsPolicy,
			AllowDirect: true,
			MaxAge:      7 * 24 * time.Hour,
			MaxBytes:    maxBytes,
			Compression: nats.S2Compression,
		},
		{Name: "OSS", Subjects: []string{"oss.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "PROVISION", Subjects: []string{"provision.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "DATAMODEL", Subjects: []string{"datamodel.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "SOFTWARE", Subjects: []string{"firmware.>", "upgrade.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "BACKUP", Subjects: []string{"backup.>"}, Retention: nats.WorkQueuePolicy, AllowDirect: true},
		{Name: "REPORT", Subjects: []string{"report.>"}, Retention: nats.WorkQueuePolicy, AllowDirect: true},
		{Name: "NEDIRECT", Subjects: []string{"nedirect.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "SYS", Subjects: []string{"sys.>"}, Retention: nats.WorkQueuePolicy},
		// 基站日志采集（运行日志 FileType 6 / 故障日志 8）：stationlog 单 consumer
		// 订阅 log.file.received 写库。WorkQueuePolicy（与 PM/MR 文件处理同档：
		// 任一 consumer ack 后即删）。缺这条流时 publish log.file.received 在
		// NATS 部署态下报 "no response from stream"，记录永不入库（issue #178/#222）。
		{Name: "LOG", Subjects: []string{"log.>"}, Retention: nats.WorkQueuePolicy},
		// T-0137 M2: TR069 报文跟踪。
		// TRACE_TASK 走 InterestPolicy — ACS 实例 + app SSE 推送 + worker（purge 处理）三类订阅者都需独立 fan-out。
		// TRACE_MSG 走 WorkQueuePolicy — worker QueueSubscribe 群组消费 + 批量落库。
		// TRACE_EXPORT 走 WorkQueuePolicy — worker 单 consumer 顺序执行异步导出。
		{Name: "TRACE_TASK", Subjects: []string{"trace.task.>"}, Retention: nats.InterestPolicy},
		{Name: "TRACE_MSG", Subjects: []string{"trace.message.>"}, Retention: nats.WorkQueuePolicy, AllowDirect: true},
		{Name: "TRACE_EXPORT", Subjects: []string{"trace.export.>"}, Retention: nats.WorkQueuePolicy, AllowDirect: true},
	}
}

// NewNATSClient connects to a NATS server and obtains a JetStream context.
func NewNATSClient(cfg appconfig.NATSConfig, logger *zap.Logger) (*NATSClient, error) {
	opts := []nats.Option{
		nats.Name("omcgo"),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	}

	if cfg.MaxReconnect != 0 {
		opts = append(opts, nats.MaxReconnects(cfg.MaxReconnect))
	}

	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("get JetStream context: %w", err)
	}

	return &NATSClient{
		Conn:   conn,
		JS:     js,
		logger: logger,
	}, nil
}

// EnsureStreams creates all default JetStream streams if they don't exist and
// reconciles subjects on existing streams before consumers bind to them.
// 当现有 stream 的 Retention 与 DefaultStreams 不一致时：
//   - allowRebuild=true：删除重建（in-flight 消息丢失，仅适合 dev/test）
//   - allowRebuild=false：仅 WARN，不破坏现有 stream（生产默认）
func (c *NATSClient) EnsureStreams(ctx context.Context, allowRebuild bool) error {
	return c.EnsureStreamsWithAlarmLifecycleMaxBytes(ctx, allowRebuild, alarmLifecycleStreamMaxBytes)
}

// EnsureStreamsWithAlarmLifecycleMaxBytes reconciles the configured DOMAIN_ALARM
// capacity without changing stream names, subjects, retention, or other domains.
func (c *NATSClient) EnsureStreamsWithAlarmLifecycleMaxBytes(ctx context.Context, allowRebuild bool, maxBytes int64) error {
	streams := DefaultStreamsWithAlarmLifecycleMaxBytes(maxBytes)
	for _, def := range streams {
		info, err := c.JS.StreamInfo(def.Name)
		if err == nats.ErrStreamNotFound {
			if err := c.createStream(def); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("get stream info %s: %w", def.Name, err)
		}

		if err := reconcileStreamSubjects(def, info, func(config *nats.StreamConfig) (*nats.StreamInfo, error) {
			return c.JS.UpdateStream(config)
		}); err != nil {
			return err
		}

		if err := enableDirectLookup(def, info, func(config *nats.StreamConfig) (*nats.StreamInfo, error) {
			return c.JS.UpdateStream(config)
		}); err != nil {
			return err
		}
		if err := reconcileStreamLimits(def, info, func(config *nats.StreamConfig) (*nats.StreamInfo, error) {
			return c.JS.UpdateStream(config)
		}); err != nil {
			return err
		}

		if info.Config.Retention == def.Retention {
			continue
		}

		if !allowRebuild {
			c.logger.Warn("JetStream retention mismatch (rebuild disabled)",
				zap.String("name", def.Name),
				zap.String("current", info.Config.Retention.String()),
				zap.String("expected", def.Retention.String()))
			continue
		}

		c.logger.Warn("JetStream retention mismatch — rebuilding stream (in-flight messages will be lost)",
			zap.String("name", def.Name),
			zap.String("from", info.Config.Retention.String()),
			zap.String("to", def.Retention.String()),
			zap.Uint64("messages_lost", info.State.Msgs))
		if err := c.JS.DeleteStream(def.Name); err != nil {
			return fmt.Errorf("delete stream %s for rebuild: %w", def.Name, err)
		}
		if err := c.createStream(def); err != nil {
			return err
		}
	}

	// 消费者位点健康检查（issue #567）：
	// stream rebuild 后消费者可能保留旧 stream 实例的位点（delivered.stream_seq
	// 远大于新 stream 的 last_seq），导致 NATS 认为所有消息已处理、新消息被静默丢弃。
	// 检测到错位消费者后删除，QueueSubscribe 会自动重建。
	for _, def := range streams {
		info, err := c.JS.StreamInfo(def.Name)
		if err != nil {
			c.logger.Warn("consumer health check: cannot get stream info, skipping",
				zap.String("stream", def.Name), zap.Error(err))
			continue
		}

		lastSeq := info.State.LastSeq
		for cn := range c.JS.ConsumerNames(def.Name) {
			ci, err := c.JS.ConsumerInfo(def.Name, cn)
			if err != nil {
				c.logger.Warn("consumer health check: cannot get consumer info, skipping",
					zap.String("stream", def.Name), zap.String("consumer", cn), zap.Error(err))
				continue
			}
			if ci.Delivered.Stream > lastSeq {
				c.logger.Warn("consumer position ahead of stream last_seq — deleting stale consumer (will auto-recreate on next subscribe)",
					zap.String("stream", def.Name),
					zap.String("consumer", cn),
					zap.Uint64("delivered_stream_seq", ci.Delivered.Stream),
					zap.Uint64("stream_last_seq", lastSeq))
				if err := c.JS.DeleteConsumer(def.Name, cn); err != nil {
					c.logger.Error("failed to delete stale consumer",
						zap.String("stream", def.Name), zap.String("consumer", cn), zap.Error(err))
				}
			}
		}
	}

	return nil
}

func reconcileStreamLimits(
	def StreamDef,
	info *nats.StreamInfo,
	update func(*nats.StreamConfig) (*nats.StreamInfo, error),
) error {
	if info == nil {
		return nil
	}
	config := info.Config
	changed := false
	if def.MaxAge > 0 && config.MaxAge != def.MaxAge {
		config.MaxAge = def.MaxAge
		changed = true
	}
	if def.MaxBytes > 0 && config.MaxBytes != def.MaxBytes {
		config.MaxBytes = def.MaxBytes
		changed = true
	}
	if def.Compression != nats.NoCompression && config.Compression != def.Compression {
		config.Compression = def.Compression
		changed = true
	}
	if !changed {
		return nil
	}
	if _, err := update(&config); err != nil {
		return fmt.Errorf("update stream limits for %s: %w", def.Name, err)
	}
	return nil
}

// enableDirectLookup is intentionally independent of retention reconciliation:
// AllowDirect is a safe in-place read capability, while a retention mismatch
// may be left unchanged in production when stream rebuilding is disabled.
func enableDirectLookup(def StreamDef, info *nats.StreamInfo, update func(*nats.StreamConfig) (*nats.StreamInfo, error)) error {
	if !def.AllowDirect || info == nil || info.Config.AllowDirect {
		return nil
	}
	config := info.Config
	config.AllowDirect = true
	if _, err := update(&config); err != nil {
		return fmt.Errorf("enable direct message lookup for stream %s: %w", def.Name, err)
	}
	return nil
}

func reconcileStreamSubjects(
	def StreamDef,
	info *nats.StreamInfo,
	update func(*nats.StreamConfig) (*nats.StreamInfo, error),
) error {
	if info == nil || sameSubjects(info.Config.Subjects, def.Subjects) {
		return nil
	}
	config := info.Config
	config.Subjects = append([]string(nil), def.Subjects...)
	if _, err := update(&config); err != nil {
		return fmt.Errorf("reconcile subjects for stream %s: %w", def.Name, err)
	}
	return nil
}

func sameSubjects(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, subject := range left {
		counts[subject]++
	}
	for _, subject := range right {
		counts[subject]--
		if counts[subject] < 0 {
			return false
		}
	}
	return true
}

func (c *NATSClient) createStream(def StreamDef) error {
	maxAge := def.MaxAge
	if maxAge <= 0 {
		maxAge = 72 * time.Hour
	}
	_, err := c.JS.AddStream(&nats.StreamConfig{
		Name:        def.Name,
		Subjects:    def.Subjects,
		Retention:   def.Retention,
		MaxAge:      maxAge,
		MaxBytes:    def.MaxBytes,
		Storage:     nats.FileStorage,
		Replicas:    1, // single node for dev; set 3 for production
		AllowDirect: def.AllowDirect,
		Compression: def.Compression,
	})
	if err != nil {
		return fmt.Errorf("create stream %s: %w", def.Name, err)
	}
	c.logger.Info("created JetStream stream",
		zap.String("name", def.Name),
		zap.String("retention", def.Retention.String()))
	return nil
}

// HealthCheck verifies the NATS connection is alive.
func (c *NATSClient) HealthCheck() error {
	if !c.Conn.IsConnected() {
		return fmt.Errorf("NATS not connected")
	}
	return nil
}

// Close drains and closes the NATS connection.
func (c *NATSClient) Close() {
	c.metricsMu.Lock()
	metrics := c.metrics
	c.metricsMu.Unlock()
	if metrics != nil {
		metrics.Stop()
	}
	if c.Conn != nil {
		c.Conn.Drain()
	}
}
