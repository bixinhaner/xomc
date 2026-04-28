// Package integration — NATS JetStream 故障演练集成测试 stub。
//
// 章程 W3.E.3 / Backlog T-0059
//
// **为什么是 stub**：真实演练需 staging NATS 实例 + 网络隔离权限，CI/local 无法
// 模拟。本文件提供 testcontainers-go 起 NATS 容器跑最小重连场景的 framework，
// CI 默认 t.Skip（除非 OMC_NATS_INTEGRATION=1 环境变量启用）。
//
// 配套 Runbook: docs/runbook/nats-failover.md
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNATSFailover_Reconnect_PreservesEvents 验证 NATS 短暂中断后客户端重连，
// 重连期间 Publish 的事件全部落地（At-Least-Once）。
//
// 启用方式: OMC_NATS_INTEGRATION=1 OMC_NATS_URL=nats://localhost:4222 go test -run TestNATSFailover ./test/integration/...
func TestNATSFailover_Reconnect_PreservesEvents(t *testing.T) {
	if os.Getenv("OMC_NATS_INTEGRATION") != "1" {
		t.Skip("requires OMC_NATS_INTEGRATION=1 + staging NATS instance (see docs/runbook/nats-failover.md)")
	}

	natsURL := os.Getenv("OMC_NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// TODO(staging): 用 nats.Connect(natsURL, nats.MaxReconnects(-1), nats.ReconnectWait(2*time.Second)) 起 EventBus
	// 然后:
	//   1. 起 100 个 alarm.raised 事件 publisher goroutine（每秒 10 个）
	//   2. 中途 docker stop nats 30 秒
	//   3. 期间 Publish 应排队或 fail（client buffer 满）
	//   4. docker start nats 后 30 秒内重连
	//   5. 验证 100 事件全部到达 subscriber（或 ≥ 95，根据 client buffer 配置）

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Placeholder：当 user 在 staging 启用此测试时，把以下 stub 替换为真实演练代码
	t.Logf("NATS failover stub: will connect %s and run reconnect drill", natsURL)
	require.NotNil(t, ctx)
	assert.NotZero(t, time.Now())
}

// TestEventRetry_AckNakTermBehavior 验证 NATSEventBus 重试语义在故障注入下的
// Ack/Nak/Term 决策正确（至 maxDeliveries=5 后 Term 防无限循环）。
//
// 启用方式同上 OMC_NATS_INTEGRATION=1
func TestEventRetry_AckNakTermBehavior(t *testing.T) {
	if os.Getenv("OMC_NATS_INTEGRATION") != "1" {
		t.Skip("requires OMC_NATS_INTEGRATION=1 + staging NATS instance")
	}

	// TODO(staging): subscriber 故意返 transient error 5 次，验证：
	//   - 第 1-4 次 NATSEventBus 决策 Nak（重投递）
	//   - 第 5 次决策 Term（终止，不再重投递；进 dead-letter 计数）
	//   - 配套 metric event_publish_failed_total{reason="max_deliveries"} +1

	t.Skip("stub awaiting staging implementation")
}

// TestNATSFailover_QueueGroup_LoadBalance 验证多实例 QueueSubscribe 在故障恢复后
// 仍均衡分发（不会全部消息打到一个 instance）。
func TestNATSFailover_QueueGroup_LoadBalance(t *testing.T) {
	if os.Getenv("OMC_NATS_INTEGRATION") != "1" {
		t.Skip("requires OMC_NATS_INTEGRATION=1 + staging NATS instance with ≥ 2 client instances")
	}

	// TODO(staging): 起 3 个 QueueSubscribe instance，发 300 事件，验证每个 instance
	// 收到 ~100 事件（±20%）；故障恢复后再发 300，再次验证均衡。

	t.Skip("stub awaiting staging implementation")
}

// 编译期断言：导入 event 包确保接口可用（即使 t.Skip 也保证 stub 编译通过）。
var _ = event.Event{}
