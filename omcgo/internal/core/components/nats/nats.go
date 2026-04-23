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
	Name     string
	Subjects []string
}

// DefaultStreams 列出所有 JetStream 流。每条流以一个点分前缀吸纳一类事件，
// 保留策略统一为 WorkQueuePolicy（消费完删除），最大存活 72 小时。
// 扩展新事件前缀时，务必在此注册对应 Stream，否则发布的消息不会落盘、
// 订阅者掉线就丢失。
func DefaultStreams() []StreamDef {
	return []StreamDef{
		{Name: "DEVICE", Subjects: []string{"device.>"}},
		{Name: "COMMAND", Subjects: []string{"command.>"}},
		{Name: "TASK", Subjects: []string{"task.>"}},
		{Name: "PM", Subjects: []string{"pm.>"}},
		{Name: "MR", Subjects: []string{"mr.>"}},
		{Name: "ALARM", Subjects: []string{"alarm.>"}},
		{Name: "OSS", Subjects: []string{"oss.>"}},
		{Name: "PROVISION", Subjects: []string{"provision.>"}},
		{Name: "DATAMODEL", Subjects: []string{"datamodel.>"}},
		{Name: "SOFTWARE", Subjects: []string{"firmware.>", "upgrade.>"}},
		{Name: "BACKUP", Subjects: []string{"backup.>"}},
		{Name: "REPORT", Subjects: []string{"report.>"}},
		{Name: "NEDIRECT", Subjects: []string{"nedirect.>"}},
		{Name: "SYS", Subjects: []string{"sys.>"}},
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
func (c *NATSClient) EnsureStreams(ctx context.Context) error {
	for _, def := range DefaultStreams() {
		_, err := c.JS.StreamInfo(def.Name)
		if err == nats.ErrStreamNotFound {
			_, err = c.JS.AddStream(&nats.StreamConfig{
				Name:      def.Name,
				Subjects:  def.Subjects,
				Retention: nats.WorkQueuePolicy,
				MaxAge:    72 * time.Hour,
				Storage:   nats.FileStorage,
				Replicas:  1, // single node for dev; set 3 for production
			})
			if err != nil {
				return fmt.Errorf("create stream %s: %w", def.Name, err)
			}
			c.logger.Info("created JetStream stream", zap.String("name", def.Name))
		} else if err != nil {
			return fmt.Errorf("get stream info %s: %w", def.Name, err)
		}
	}
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
