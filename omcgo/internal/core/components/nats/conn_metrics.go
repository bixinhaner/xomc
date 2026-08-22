package nats

import (
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
)

// RegisterMetrics 为本客户端底层 nats.Conn 在 reg 上注册连接指标
// （nats_conn_status / nats_reconnect_total / nats_msgs_*），同时保留
// NewNATSClient 安装的「NATS reconnected」日志。
//
// RegisterConnMetrics 只装一个「计数」型重连回调，会覆盖 NewNATSClient 的日志回调；
// 这里随后重装一个组合回调（计数 + 日志），让指标与重连可观测性两者都不丢失。
func (c *NATSClient) RegisterMetrics(reg prometheus.Registerer) *ConnMetrics {
	c.metricsMu.Lock()
	if c.metrics != nil {
		cm := c.metrics
		c.metricsMu.Unlock()
		return cm
	}

	cm := RegisterConnMetricsWithSampler(c.sampleConn, reg)
	c.metrics = cm
	c.metricsMu.Unlock()

	c.installMetricsHandlers(c.currentConn())

	queueMetrics := newQueueMetricsWithReader(natsClientQueueMetricsReader{client: c}, reg, c.logger, defaultQueueMetricTargets)
	queueMetrics.Start()
	cm.attachQueueMetrics(queueMetrics)
	return cm
}

func (c *NATSClient) currentConn() *nats.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Conn
}

func (c *NATSClient) currentJS() nats.JetStreamContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.JS
}

func (c *NATSClient) sampleConn() connSnapshot {
	c.mu.RLock()
	conn := c.producerConn
	if conn == nil {
		conn = c.Conn
	}
	c.mu.RUnlock()
	if conn == nil {
		return connSnapshot{State: nats.CLOSED}
	}
	stats := conn.Stats()
	return connSnapshot{
		Connected: conn.IsConnected(),
		State:     conn.Status(),
		MsgsIn:    stats.InMsgs,
		MsgsOut:   stats.OutMsgs,
	}
}

func (c *NATSClient) installMetricsHandlers(conn *nats.Conn) {
	if conn == nil {
		return
	}
	c.metricsMu.Lock()
	cm := c.metrics
	c.metricsMu.Unlock()
	if cm == nil {
		return
	}
	conn.SetReconnectHandler(func(nc *nats.Conn) {
		cm.IncReconnect()
		logNATSReconnected(c.logger, nc)
	})
	conn.SetClosedHandler(func(nc *nats.Conn) {
		cm.IncClosed()
		logNATSClosed(c.logger, nc)
	})
}

type natsClientQueueMetricsReader struct {
	client *NATSClient
}

func (r natsClientQueueMetricsReader) StreamInfo(ctx context.Context, stream string) (*nats.StreamInfo, error) {
	js := r.client.currentJS()
	if js == nil {
		return nil, nats.ErrConnectionClosed
	}
	return js.StreamInfo(stream, nats.Context(ctx))
}

func (r natsClientQueueMetricsReader) ConsumerInfo(ctx context.Context, stream, durable string) (*nats.ConsumerInfo, error) {
	js := r.client.currentJS()
	if js == nil {
		return nil, nats.ErrConnectionClosed
	}
	return js.ConsumerInfo(stream, durable, nats.Context(ctx))
}

func (r natsClientQueueMetricsReader) GetMsg(ctx context.Context, stream string, sequence uint64, subject string) (*nats.RawStreamMsg, error) {
	js := r.client.currentJS()
	if js == nil {
		return nil, nats.ErrConnectionClosed
	}
	return js.GetMsg(stream, sequence, nats.DirectGetNext(subject), nats.Context(ctx))
}

// DefaultConnMetricsInterval 默认采样周期。
const DefaultConnMetricsInterval = 5 * time.Second

// connSnapshot 是 NATS 连接一次采样的结果。
type connSnapshot struct {
	Connected bool
	State     nats.Status
	MsgsIn    uint64 // 累计入站消息（来自 nats.Conn.Stats().InMsgs）
	MsgsOut   uint64 // 累计出站消息
}

// connSampler 在每次轮询时返回一个连接快照。
type connSampler func() connSnapshot

// natsConn 是 nats.Conn 的最小接口子集，便于在测试里替换。
type natsConn interface {
	Stats() nats.Statistics
	IsConnected() bool
	Status() nats.Status
	SetReconnectHandler(cb nats.ConnHandler)
	SetClosedHandler(cb nats.ConnHandler)
}

// ConnMetrics 持有 NATS 连接的 Prometheus 指标集合。
//
// 暴露的指标（章程 W3.F.3 / T-0061）：
//   - nats_conn_status     (gauge)   1 = connected, 0 = disconnected
//   - nats_conn_state      (gauge)   one-hot NATS status labels
//   - nats_reconnect_total (counter) 重连次数（基于 SetReconnectHandler 回调累加）
//   - nats_conn_closed_total (counter) 连接彻底 closed 次数，用于区分可恢复断连和已放弃重连
//   - nats_msgs_in_total   (counter) 累计入站消息数（来自 Stats().InMsgs）
//   - nats_msgs_out_total  (counter) 累计出站消息数（来自 Stats().OutMsgs）
type ConnMetrics struct {
	Status         prometheus.Gauge
	State          *prometheus.GaugeVec
	ReconnectTotal prometheus.Counter
	ClosedTotal    prometheus.Counter
	MsgsIn         prometheus.Counter
	MsgsOut        prometheus.Counter

	sampler  connSampler
	interval time.Duration

	cancel   context.CancelFunc
	stopped  chan struct{}
	stopOnce sync.Once
	queue    *QueueMetrics

	lastIn  uint64
	lastOut uint64
}

func (c *ConnMetrics) attachQueueMetrics(queue *QueueMetrics) {
	c.queue = queue
}

// ConnMetricsOption 配置 ConnMetrics 行为。
type ConnMetricsOption func(*ConnMetrics)

// WithConnMetricsInterval 自定义采样周期，默认 5s。
func WithConnMetricsInterval(interval time.Duration) ConnMetricsOption {
	return func(c *ConnMetrics) {
		if interval > 0 {
			c.interval = interval
		}
	}
}

// RegisterConnMetrics 为 nats.Conn 注册 Prometheus 指标，启动后台采样 goroutine，
// 并通过 SetReconnectHandler 累加 nats_reconnect_total。
//
// 注意：SetReconnectHandler 会覆盖现有 handler；如调用方需要自己的回调，
// 应使用 RegisterConnMetricsWithSampler 自行串联。
func RegisterConnMetrics(conn *nats.Conn, reg prometheus.Registerer, opts ...ConnMetricsOption) *ConnMetrics {
	if conn == nil {
		panic("nats.RegisterConnMetrics: conn is nil")
	}
	return registerConnMetricsWithProvider(connFromNATS(conn), reg, opts...)
}

// connFromNATS 适配 *nats.Conn 到 natsConn 接口。
type natsConnAdapter struct{ c *nats.Conn }

func (a natsConnAdapter) Stats() nats.Statistics                  { return a.c.Stats() }
func (a natsConnAdapter) IsConnected() bool                       { return a.c.IsConnected() }
func (a natsConnAdapter) Status() nats.Status                     { return a.c.Status() }
func (a natsConnAdapter) SetReconnectHandler(cb nats.ConnHandler) { a.c.SetReconnectHandler(cb) }
func (a natsConnAdapter) SetClosedHandler(cb nats.ConnHandler)    { a.c.SetClosedHandler(cb) }

func connFromNATS(c *nats.Conn) natsConn {
	return natsConnAdapter{c: c}
}

func registerConnMetricsWithProvider(conn natsConn, reg prometheus.Registerer, opts ...ConnMetricsOption) *ConnMetrics {
	if conn == nil {
		panic("nats.registerConnMetricsWithProvider: conn is nil")
	}
	if reg == nil {
		panic("nats.RegisterConnMetrics: registerer is nil")
	}

	cm := &ConnMetrics{
		interval: DefaultConnMetricsInterval,
		stopped:  make(chan struct{}),
		Status: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nats_conn_status",
			Help: "NATS connection status: 1 = connected, 0 = disconnected.",
		}),
		State: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nats_conn_state",
			Help: "NATS connection state as one-hot labels.",
		}, []string{"state"}),
		ReconnectTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nats_reconnect_total",
			Help: "Cumulative number of NATS reconnect events observed by the local client.",
		}),
		ClosedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nats_conn_closed_total",
			Help: "Cumulative number of NATS client connections that reached the closed state.",
		}),
		MsgsIn: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nats_msgs_in_total",
			Help: "Cumulative number of NATS messages received by the local client.",
		}),
		MsgsOut: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nats_msgs_out_total",
			Help: "Cumulative number of NATS messages sent by the local client.",
		}),
	}
	for _, opt := range opts {
		opt(cm)
	}

	// sampler 闭包持有 conn 引用。
	cm.sampler = func() connSnapshot {
		stats := conn.Stats()
		return connSnapshot{
			Connected: conn.IsConnected(),
			State:     conn.Status(),
			MsgsIn:    stats.InMsgs,
			MsgsOut:   stats.OutMsgs,
		}
	}

	reg.MustRegister(cm.Status, cm.State, cm.ReconnectTotal, cm.ClosedTotal, cm.MsgsIn, cm.MsgsOut)

	// 注册重连回调（注意会覆盖现有 handler；调用方需要自定义请直接使用 sampler 版本）。
	conn.SetReconnectHandler(func(_ *nats.Conn) {
		cm.ReconnectTotal.Inc()
	})
	conn.SetClosedHandler(func(_ *nats.Conn) {
		cm.ClosedTotal.Inc()
	})

	cm.sample()

	ctx, cancel := context.WithCancel(context.Background())
	cm.cancel = cancel
	go cm.run(ctx)
	return cm
}

// RegisterConnMetricsWithSampler 暴露给调用方完全自定义采样源（含重连计数路径）。
// 通常仅在测试或特殊集成场景使用；普通业务代码请使用 RegisterConnMetrics。
func RegisterConnMetricsWithSampler(sampler connSampler, reg prometheus.Registerer, opts ...ConnMetricsOption) *ConnMetrics {
	if sampler == nil {
		panic("nats.RegisterConnMetricsWithSampler: sampler is nil")
	}
	if reg == nil {
		panic("nats.RegisterConnMetricsWithSampler: registerer is nil")
	}

	cm := &ConnMetrics{
		sampler:  sampler,
		interval: DefaultConnMetricsInterval,
		stopped:  make(chan struct{}),
		Status: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nats_conn_status",
			Help: "NATS connection status: 1 = connected, 0 = disconnected.",
		}),
		State: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nats_conn_state",
			Help: "NATS connection state as one-hot labels.",
		}, []string{"state"}),
		ReconnectTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nats_reconnect_total",
			Help: "Cumulative number of NATS reconnect events observed by the local client.",
		}),
		ClosedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nats_conn_closed_total",
			Help: "Cumulative number of NATS client connections that reached the closed state.",
		}),
		MsgsIn: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nats_msgs_in_total",
			Help: "Cumulative number of NATS messages received by the local client.",
		}),
		MsgsOut: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nats_msgs_out_total",
			Help: "Cumulative number of NATS messages sent by the local client.",
		}),
	}
	for _, opt := range opts {
		opt(cm)
	}
	reg.MustRegister(cm.Status, cm.State, cm.ReconnectTotal, cm.ClosedTotal, cm.MsgsIn, cm.MsgsOut)
	cm.sample()
	ctx, cancel := context.WithCancel(context.Background())
	cm.cancel = cancel
	go cm.run(ctx)
	return cm
}

// IncReconnect 公开给测试或外部回调累加重连次数。
func (c *ConnMetrics) IncReconnect() {
	c.ReconnectTotal.Inc()
}

func (c *ConnMetrics) IncClosed() {
	c.ClosedTotal.Inc()
}

// Stop 停止后台采样并等待 goroutine 退出。可重复调用。
func (c *ConnMetrics) Stop() {
	c.stopOnce.Do(func() {
		if c.queue != nil {
			c.queue.Stop()
		}
		if c.cancel != nil {
			c.cancel()
		}
		<-c.stopped
	})
}

func (c *ConnMetrics) run(ctx context.Context) {
	defer close(c.stopped)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sample()
		}
	}
}

// sample 读取连接快照并刷新 gauge / counter。
//
// MsgsIn / MsgsOut 是单调递增的累计值；以 delta 方式 Add，避免覆盖
// 或在 nats client 重建时计数回退。
func (c *ConnMetrics) sample() {
	if c.sampler == nil {
		return
	}
	snap := c.sampler()
	if snap.Connected {
		c.Status.Set(1)
	} else {
		c.Status.Set(0)
	}
	c.observeState(snap.State)

	if snap.MsgsIn > c.lastIn {
		c.MsgsIn.Add(float64(snap.MsgsIn - c.lastIn))
		c.lastIn = snap.MsgsIn
	} else if snap.MsgsIn < c.lastIn {
		c.lastIn = snap.MsgsIn
	}
	if snap.MsgsOut > c.lastOut {
		c.MsgsOut.Add(float64(snap.MsgsOut - c.lastOut))
		c.lastOut = snap.MsgsOut
	} else if snap.MsgsOut < c.lastOut {
		c.lastOut = snap.MsgsOut
	}
}

func (c *ConnMetrics) observeState(state nats.Status) {
	if c.State == nil {
		return
	}
	current := connStateLabel(state)
	for _, label := range connStateLabels {
		value := 0.0
		if label == current {
			value = 1
		}
		c.State.WithLabelValues(label).Set(value)
	}
}

var connStateLabels = []string{
	"connected",
	"disconnected",
	"reconnecting",
	"connecting",
	"draining",
	"closed",
	"unknown",
}

func connStateLabel(state nats.Status) string {
	switch state {
	case nats.CONNECTED:
		return "connected"
	case nats.DISCONNECTED:
		return "disconnected"
	case nats.RECONNECTING:
		return "reconnecting"
	case nats.CONNECTING:
		return "connecting"
	case nats.DRAINING_SUBS, nats.DRAINING_PUBS:
		return "draining"
	case nats.CLOSED:
		return "closed"
	default:
		return "unknown"
	}
}
