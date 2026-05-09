package nats

import (
	"context"
	"fmt"
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
}

// StreamDef defines a JetStream stream.
type StreamDef struct {
	Name      string
	Subjects  []string
	Retention nats.RetentionPolicy
}

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
	return []StreamDef{
		{Name: "DEVICE", Subjects: []string{"device.>"}, Retention: nats.InterestPolicy},
		{Name: "COMMAND", Subjects: []string{"command.>"}, Retention: nats.InterestPolicy},
		{Name: "TASK", Subjects: []string{"task.>"}, Retention: nats.InterestPolicy},
		{Name: "PM", Subjects: []string{"pm.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "MR", Subjects: []string{"mr.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "ALARM", Subjects: []string{"alarm.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "OSS", Subjects: []string{"oss.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "PROVISION", Subjects: []string{"provision.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "DATAMODEL", Subjects: []string{"datamodel.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "SOFTWARE", Subjects: []string{"firmware.>", "upgrade.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "BACKUP", Subjects: []string{"backup.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "REPORT", Subjects: []string{"report.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "NEDIRECT", Subjects: []string{"nedirect.>"}, Retention: nats.WorkQueuePolicy},
		{Name: "SYS", Subjects: []string{"sys.>"}, Retention: nats.WorkQueuePolicy},
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

// EnsureStreams creates all default JetStream streams if they don't exist.
// 当现有 stream 的 Retention 与 DefaultStreams 不一致时：
//   - allowRebuild=true：删除重建（in-flight 消息丢失，仅适合 dev/test）
//   - allowRebuild=false：仅 WARN，不破坏现有 stream（生产默认）
func (c *NATSClient) EnsureStreams(ctx context.Context, allowRebuild bool) error {
	for _, def := range DefaultStreams() {
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
	return nil
}

func (c *NATSClient) createStream(def StreamDef) error {
	_, err := c.JS.AddStream(&nats.StreamConfig{
		Name:      def.Name,
		Subjects:  def.Subjects,
		Retention: def.Retention,
		MaxAge:    72 * time.Hour,
		Storage:   nats.FileStorage,
		Replicas:  1, // single node for dev; set 3 for production
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
	if c.Conn != nil {
		c.Conn.Drain()
	}
}
