package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/northbound/push"
	"github.com/omcgo/omcgo/internal/notification"
)

type providerLifecycleSubscription struct{}

func (providerLifecycleSubscription) Unsubscribe() error { return nil }

type providerLifecycleBus struct {
	configs map[string]event.KeyedQueueConfig
}

func (*providerLifecycleBus) Publish(context.Context, string, event.Event) error { return nil }
func (*providerLifecycleBus) Subscribe(string, event.EventHandler) (event.Subscription, error) {
	return providerLifecycleSubscription{}, nil
}
func (*providerLifecycleBus) QueueSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return providerLifecycleSubscription{}, nil
}
func (*providerLifecycleBus) PullSubscribe(string, string, event.EventHandler) (event.Subscription, error) {
	return providerLifecycleSubscription{}, nil
}
func (*providerLifecycleBus) Close() error { return nil }
func (b *providerLifecycleBus) KeyedQueueSubscribe(
	_ string,
	config event.KeyedQueueConfig,
	_ event.EventKeyFunc,
	_ event.EventHandler,
) (event.Subscription, error) {
	b.configs[config.Durable] = config
	return providerLifecycleSubscription{}, nil
}
func (*providerLifecycleBus) QueueStats(context.Context, string, string) (event.QueueStats, error) {
	return event.QueueStats{}, nil
}

// TestInitNorthboundModuleOutboxWired 回归 #121：北向死信队列端点恒 503。
//
// initNorthboundModule 必须构造 OutboxRepository 并注入 Router（死信端点）与
// Engine（outbox 投递模式），否则 GET/POST /push/deadletter* 在任何环境下都
// 返回 503 "outbox not configured"。
//
// 测试用不可达地址构造 lazy pgxpool（pgx v5 创建池不连库），只验证装配行为，
// 不依赖真实数据库：
//   - replay 非法 uuid → 400（修复前 outboxRepo==nil 在 uuid 解析前短路 503）
//   - 死信列表读 → 不允许 503 / "outbox not configured"（DB 不可达时为 500）
func TestInitNorthboundModuleOutboxWired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	// pgx v5 池创建为 lazy，不会在此真正连接 127.0.0.1:1（仅查询时报连接拒绝）。
	pool, err := pgxpool.New(context.Background(), "postgres://smoke:smoke@127.0.0.1:1/smoke?connect_timeout=1")
	require.NoError(t, err)
	defer pool.Close()

	c := &Container{
		PgPool:   pool,
		EventBus: event.NewChannelEventBus(16, logger),
		Cfg:      &appconfig.AppConfig{},
		Logger:   logger,
		GS:       components.NewGracefulShutdown(5*time.Second, logger),
	}
	require.NoError(t, initNorthboundModule(c))
	// 停 outbox worker / push engine，避免轮询 goroutine 泄漏到其他测试。
	defer func() { _ = c.GS.Shutdown(context.Background()) }()

	require.NotNil(t, c.miscDeps.nbRouter, "initNorthboundModule 应装配 nbRouter")
	engine := gin.New()
	c.miscDeps.nbRouter.RegisterRoutes(engine.Group("/api/v1"))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int // 0 = 不断言具体码，仅排除 503
	}{
		{
			name:       "死信重放非法 uuid 走 400 而非 503（outbox 已装配，nil 短路在 uuid 解析之前）",
			method:     http.MethodPost,
			path:       "/api/v1/northbound/push/deadletter/not-a-uuid/replay",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "死信列表读不再 503 outbox not configured",
			method: http.MethodGet,
			path:   "/api/v1/northbound/push/deadletter?limit=20&offset=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			engine.ServeHTTP(w, req)

			if tt.wantStatus != 0 {
				require.Equal(t, tt.wantStatus, w.Code, "body: %s", w.Body.String())
			}
			require.NotEqual(t, http.StatusServiceUnavailable, w.Code,
				"死信端点不应 503，body: %s", w.Body.String())
			require.False(t, strings.Contains(w.Body.String(), "outbox not configured"),
				"死信端点不应再返回 outbox not configured：%s", w.Body.String())
		})
	}
}

func TestInitNorthboundModuleWiresIndependentNotificationLifecycleDurable(t *testing.T) {
	logger := zap.NewNop()
	pool, err := pgxpool.New(context.Background(), "postgres://smoke:smoke@127.0.0.1:1/smoke?connect_timeout=1")
	require.NoError(t, err)
	defer pool.Close()
	bus := &providerLifecycleBus{configs: make(map[string]event.KeyedQueueConfig)}
	c := &Container{
		PgPool: pool, EventBus: bus,
		Cfg: &appconfig.AppConfig{Alarm: appconfig.AlarmConfig{
			LifecycleMode: "shadow", LifecycleStartSequence: 42,
		}},
		Logger: logger, GS: components.NewGracefulShutdown(5*time.Second, logger),
	}
	require.NoError(t, initNorthboundModule(c))
	defer func() { _ = c.GS.Shutdown(context.Background()) }()

	require.Contains(t, bus.configs, push.AlarmLifecycleConsumerDurable)
	require.Contains(t, bus.configs, notification.LifecycleConsumerDurable)
	require.Equal(t, uint64(42), bus.configs[notification.LifecycleConsumerDurable].StartSequence)
}
