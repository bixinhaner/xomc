package nats

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnMetrics_AllMetricNamesRegistered(t *testing.T) {
	t.Parallel()

	stub := newStubConn(true, 0, 0)
	reg := prometheus.NewRegistry()
	cm := registerConnMetricsWithProvider(stub, reg, WithConnMetricsInterval(time.Hour))
	defer cm.Stop()

	families, err := reg.Gather()
	require.NoError(t, err)
	names := metricNames(families)
	for _, want := range []string{
		"nats_conn_status",
		"nats_conn_state",
		"nats_reconnect_total",
		"nats_conn_closed_total",
		"nats_msgs_in_total",
		"nats_msgs_out_total",
	} {
		assert.Contains(t, names, want, "missing metric %q", want)
	}
}

func TestConnMetrics_StatusReflectsConnected(t *testing.T) {
	t.Parallel()

	stub := newStubConn(true, 0, 0)
	reg := prometheus.NewRegistry()
	cm := registerConnMetricsWithProvider(stub, reg, WithConnMetricsInterval(time.Hour))
	defer cm.Stop()

	require.Equal(t, float64(1), testutil.ToFloat64(cm.Status))

	stub.setConnected(false)
	cm.sample()
	require.Equal(t, float64(0), testutil.ToFloat64(cm.Status))

	stub.setConnected(true)
	cm.sample()
	require.Equal(t, float64(1), testutil.ToFloat64(cm.Status))
}

func TestConnMetrics_StateReflectsNATSStatus(t *testing.T) {
	t.Parallel()

	stub := newStubConn(true, 0, 0)
	stub.setStatus(nats.CONNECTED)
	reg := prometheus.NewRegistry()
	cm := registerConnMetricsWithProvider(stub, reg, WithConnMetricsInterval(time.Hour))
	defer cm.Stop()

	require.Equal(t, float64(1), testutil.ToFloat64(cm.State.WithLabelValues("connected")))
	require.Equal(t, float64(0), testutil.ToFloat64(cm.State.WithLabelValues("reconnecting")))

	stub.setStatus(nats.RECONNECTING)
	cm.sample()
	require.Equal(t, float64(0), testutil.ToFloat64(cm.State.WithLabelValues("connected")))
	require.Equal(t, float64(1), testutil.ToFloat64(cm.State.WithLabelValues("reconnecting")))

	stub.setStatus(nats.CLOSED)
	cm.sample()
	require.Equal(t, float64(0), testutil.ToFloat64(cm.State.WithLabelValues("reconnecting")))
	require.Equal(t, float64(1), testutil.ToFloat64(cm.State.WithLabelValues("closed")))
}

func TestConnMetrics_MsgCountersIncreaseMonotonically(t *testing.T) {
	t.Parallel()

	stub := newStubConn(true, 100, 200)
	reg := prometheus.NewRegistry()
	cm := registerConnMetricsWithProvider(stub, reg, WithConnMetricsInterval(time.Hour))
	defer cm.Stop()

	require.Equal(t, float64(100), testutil.ToFloat64(cm.MsgsIn))
	require.Equal(t, float64(200), testutil.ToFloat64(cm.MsgsOut))

	stub.setStats(150, 280)
	cm.sample()
	require.Equal(t, float64(150), testutil.ToFloat64(cm.MsgsIn))
	require.Equal(t, float64(280), testutil.ToFloat64(cm.MsgsOut))

	// 模拟连接重建：底层计数被重置，但 prom counter 不能下跌。
	stub.setStats(20, 30)
	cm.sample()
	require.Equal(t, float64(150), testutil.ToFloat64(cm.MsgsIn))
	require.Equal(t, float64(280), testutil.ToFloat64(cm.MsgsOut))

	stub.setStats(60, 90)
	cm.sample()
	require.Equal(t, float64(190), testutil.ToFloat64(cm.MsgsIn))
	require.Equal(t, float64(340), testutil.ToFloat64(cm.MsgsOut))
}

func TestConnMetrics_ReconnectHandlerInstalled(t *testing.T) {
	t.Parallel()

	stub := newStubConn(true, 0, 0)
	reg := prometheus.NewRegistry()
	cm := registerConnMetricsWithProvider(stub, reg, WithConnMetricsInterval(time.Hour))
	defer cm.Stop()

	cb := stub.handler()
	require.NotNil(t, cb, "RegisterConnMetrics must install reconnect handler")
	require.Equal(t, float64(0), testutil.ToFloat64(cm.ReconnectTotal))

	// 模拟 nats client 触发回调（回调签名 *nats.Conn 在测试里传 nil）。
	cb(nil)
	cb(nil)
	require.Equal(t, float64(2), testutil.ToFloat64(cm.ReconnectTotal))
}

func TestConnMetrics_ClosedHandlerInstalled(t *testing.T) {
	t.Parallel()

	stub := newStubConn(true, 0, 0)
	reg := prometheus.NewRegistry()
	cm := registerConnMetricsWithProvider(stub, reg, WithConnMetricsInterval(time.Hour))
	defer cm.Stop()

	cb := stub.closedHandler()
	require.NotNil(t, cb, "RegisterConnMetrics must install closed handler")
	require.Equal(t, float64(0), testutil.ToFloat64(cm.ClosedTotal))

	cb(nil)
	cb(nil)
	require.Equal(t, float64(2), testutil.ToFloat64(cm.ClosedTotal))
}

func TestConnMetrics_StopIdempotent(t *testing.T) {
	t.Parallel()

	stub := newStubConn(true, 0, 0)
	reg := prometheus.NewRegistry()
	cm := registerConnMetricsWithProvider(stub, reg, WithConnMetricsInterval(10*time.Millisecond))
	cm.Stop()
	cm.Stop()
}

func TestConnMetrics_BackgroundSampling(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	sampler := func() connSnapshot {
		calls.Add(1)
		return connSnapshot{Connected: true, State: nats.CONNECTED, MsgsIn: uint64(calls.Load()), MsgsOut: uint64(calls.Load())}
	}
	reg := prometheus.NewRegistry()
	cm := RegisterConnMetricsWithSampler(sampler, reg, WithConnMetricsInterval(15*time.Millisecond))
	defer cm.Stop()

	require.Eventually(t, func() bool {
		return calls.Load() >= 3
	}, 2*time.Second, 15*time.Millisecond)
}

func TestRegisterConnMetrics_NilConnPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		require.NotNil(t, recover(), "nil conn should panic")
	}()
	RegisterConnMetrics(nil, prometheus.NewRegistry())
}

func TestRegisterConnMetricsWithSampler_NilSamplerPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		require.NotNil(t, recover(), "nil sampler should panic")
	}()
	RegisterConnMetricsWithSampler(nil, prometheus.NewRegistry())
}

// stubConn 实现 natsConn，便于无 NATS 服务器情况下测试。
type stubConn struct {
	mu          sync.Mutex
	connected   bool
	status      nats.Status
	stats       nats.Statistics
	reconnectCB nats.ConnHandler
	closedCB    nats.ConnHandler
}

func newStubConn(connected bool, in, out uint64) *stubConn {
	return &stubConn{
		connected: connected,
		status:    nats.CONNECTED,
		stats:     nats.Statistics{InMsgs: in, OutMsgs: out},
	}
}

func (s *stubConn) Stats() nats.Statistics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

func (s *stubConn) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}

func (s *stubConn) Status() nats.Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *stubConn) SetReconnectHandler(cb nats.ConnHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconnectCB = cb
}

func (s *stubConn) SetClosedHandler(cb nats.ConnHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closedCB = cb
}

func (s *stubConn) setConnected(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = v
	if v {
		s.status = nats.CONNECTED
	} else {
		s.status = nats.DISCONNECTED
	}
}

func (s *stubConn) setStatus(status nats.Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.connected = status == nats.CONNECTED
}

func (s *stubConn) setStats(in, out uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = nats.Statistics{InMsgs: in, OutMsgs: out}
}

func (s *stubConn) handler() nats.ConnHandler {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reconnectCB
}

func (s *stubConn) closedHandler() nats.ConnHandler {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closedCB
}

// metricNames 提取 gather 出的所有 metric family 名字。
func metricNames(families []*dto.MetricFamily) []string {
	out := make([]string, 0, len(families))
	for _, f := range families {
		out = append(out, f.GetName())
	}
	return out
}
