package push

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/reliability"
	"go.uber.org/zap"
)

// ActiveServerInfo 是 push engine 视角的活跃 OSS 服务器最小信息。
//
// 设计：定义在 push 包消费者侧，避免 push → northbound 反向循环依赖
// （northbound 顶层包导入 push）。northbound.ServerService 通过
// GetActiveForPush 适配器返回此结构。
type ActiveServerInfo struct {
	Role string // primary | standby
	Host string
	Port int
}

// ActiveServerProvider 提供当前激活的北向 OSS 服务器配置。
//
// 实现侧：northbound.ServerService（通过 GetActiveForPush 适配）。
// 消费侧：Engine.RefreshActiveTarget 读取后构造 push target。
type ActiveServerProvider interface {
	GetActiveForPush(ctx context.Context) (*ActiveServerInfo, error)
}

// ActiveTargetID 固定的 active target ID，Refresh 时复用此 key。
const ActiveTargetID = "northbound-active"

// SetActiveServerProvider 注入 provider，注入后才会响应
// SubjectNorthboundServerChanged 事件。
func (e *Engine) SetActiveServerProvider(p ActiveServerProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.activeServerProvider = p
}

// RefreshActiveTarget 从 ActiveServerProvider 读取当前激活 OSS 服务器并
// 更新 push targets 中固定 ID 的 ActiveTargetID 条目。
//
// 行为：
//   - provider 未注入 → 直接返 nil（dev 友好，启动期可不强依赖）
//   - GetActive 失败 → metric=failure + 返 error 让调用者决定 retry
//   - 无激活服务器（返 nil） → 移除既有 ActiveTargetID 条目（如果存在）
//   - 拿到 info → 替换/新增 ActiveTargetID 条目
//
// 关闭"切换/编辑即生效"的环：UI 切换主备 → ServerService.SetActive 写 DB +
// publish SubjectNorthboundServerChanged → Engine.handleServerChanged →
// RefreshActiveTarget → 下次推送命中新 host。
func (e *Engine) RefreshActiveTarget(ctx context.Context) error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	p := e.activeServerProvider
	e.mu.RUnlock()
	if p == nil {
		return nil
	}

	info, err := p.GetActiveForPush(ctx)
	if err != nil {
		atomic.AddInt64(&e.refreshFailureCount, 1)
		e.logger.Warn("active_target_refresh_failed",
			zap.Error(err))
		return fmt.Errorf("get active server: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if info == nil {
		// 无激活服务器 — 移除占位条目（如果存在）
		if _, ok := e.targets[ActiveTargetID]; ok {
			delete(e.targets, ActiveTargetID)
			delete(e.circuitBreakers, ActiveTargetID)
			e.logger.Info("active_target_refreshed",
				zap.String("action", "removed"),
				zap.String("reason", "no_active_server"))
		}
		atomic.AddInt64(&e.refreshSuccessCount, 1)
		return nil
	}

	target := &Target{
		ID:         ActiveTargetID,
		URL:        fmt.Sprintf("http://%s:%d", info.Host, info.Port),
		AuthType:   "none",
		DataTypes:  []string{"alarm", "pm", "config"}, // 默认全数据类型；运营商 OSS 自己过滤
		Format:     "json",
		BatchSize:  100,
		RetryCount: 3,
		Enabled:    true,
	}
	e.targets[ActiveTargetID] = target
	if _, ok := e.circuitBreakers[ActiveTargetID]; !ok {
		e.circuitBreakers[ActiveTargetID] = reliability.NewCircuitBreaker(reliability.DefaultCircuitBreakerConfig())
	}

	atomic.AddInt64(&e.refreshSuccessCount, 1)
	e.logger.Info("active_target_refreshed",
		zap.String("action", "upsert"),
		zap.String("role", info.Role),
		zap.String("host", info.Host),
		zap.Int("port", info.Port),
		zap.String("url", target.URL))
	return nil
}

// RefreshSuccessCount / RefreshFailureCount 暴露给 metric 收集器
// （在 monitor 包注册的 prometheus collector 通过这些 getter 拉取）。
func (e *Engine) RefreshSuccessCount() int64 {
	return atomic.LoadInt64(&e.refreshSuccessCount)
}

func (e *Engine) RefreshFailureCount() int64 {
	return atomic.LoadInt64(&e.refreshFailureCount)
}

// handleServerChanged 是订阅 SubjectNorthboundServerChanged 后的 handler。
//
// 任何 server.changed 事件都触发一次 RefreshActiveTarget；ServerService 已
// 在 SetActive / Update 成功后发事件，本方法只需 reload。
//
// Payload 解析仅用于日志可观测性，不参与 reload 逻辑（reload 永远从
// provider 拉最新数据）。
func (e *Engine) handleServerChanged(ctx context.Context, evt event.Event) error {
	var payload struct {
		Role   string `json:"role"`
		Action string `json:"action"`
		Host   string `json:"host"`
		Port   int    `json:"port"`
	}
	_ = json.Unmarshal(evt.Payload, &payload)
	e.logger.Info("northbound_server_changed_received",
		zap.String("role", payload.Role),
		zap.String("action", payload.Action),
		zap.String("host", payload.Host),
		zap.Int("port", payload.Port))
	return e.RefreshActiveTarget(ctx)
}
