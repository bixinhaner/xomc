// Package ops 步骤路由器（T-0101-b）。
//
// PRD §5.3.2 要求 dispatcher 按 step.type 分发到不同 handler：
//   - rpc      → 调 acs/rpc/ (e.g., reboot/factory_reset/get/set parameter)
//   - mml      → 调 internal/mml/ (按 MML 协议下发命令)
//   - wait     → time.Sleep（步骤间延迟）
//   - loop     → 重复执行子步骤序列
//   - branch   → 按条件选择分支
//
// 当前 MVP 阶段所有 handler 都是 stub —— 真 RPC/MML 派发由 T-0102-c
// (RPC 命令支持) + 既有 internal/mml 调用面 future 接入。本文件提供框架：
//   - StepHandler 函数签名（消费者驱动）
//   - StepRouter struct 注册 + 分发
//   - 5 个内建 stub handler 注册到 default router
//
// 上层（TaskExecutor.Run）迭代 template.steps 时调 router.Dispatch(ctx, step)；
// 未知 step.type 返 ErrUnknownStepType 让 TaskExecutor 按 failure_policy 决定处理。
package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// StepType 已定义的 step.type 值（PRD §5.3.2）。
type StepType string

const (
	StepRPC    StepType = "rpc"
	StepMML    StepType = "mml"
	StepWait   StepType = "wait"
	StepLoop   StepType = "loop"
	StepBranch StepType = "branch"
)

// ErrUnknownStepType step.type 未注册到 router。
var ErrUnknownStepType = errors.New("unknown step type")

// Step ops_templates.steps[] 解析后的单步定义（最小骨架）。
// 完整 schema 由 PRD §5.3.2 + JSON Schema 描述；本 struct 仅含 router 所需字段。
type Step struct {
	Type   StepType        `json:"type"`
	Name   string          `json:"name,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// StepHandler dispatcher 处理单步的函数签名。
//   - 返 nil 表示成功；返 error 表示失败（由 TaskExecutor 根据 failure_policy 决定下一步）
//   - 必须响应 ctx 取消（dispatcher 暂停/取消时调用方传 cancelled ctx）
type StepHandler func(ctx context.Context, step Step) error

// StepRouter step.type → handler 注册表（T-0101-b 步骤路由器骨架）。
//
// 线程安全：注册期间不应有 Dispatch；启动后只读，无锁。
type StepRouter struct {
	handlers map[StepType]StepHandler
	logger   *zap.Logger
}

// NewStepRouter 创建路由器并注册默认 5 个 stub handler。
func NewStepRouter(logger *zap.Logger) *StepRouter {
	r := &StepRouter{
		handlers: make(map[StepType]StepHandler),
		logger:   logger.Named("ops.steprouter"),
	}
	// 注册默认 5 个 stub handler（MVP；future 由 T-0102-c 等替换为真实派发）
	r.Register(StepRPC, defaultRPCHandler(r.logger))
	r.Register(StepMML, defaultMMLHandler(r.logger))
	r.Register(StepWait, defaultWaitHandler(r.logger))
	r.Register(StepLoop, defaultLoopHandler(r.logger))
	r.Register(StepBranch, defaultBranchHandler(r.logger))
	return r
}

// Register 注册 / 替换 step.type 的 handler。生产代码应在 wire 阶段调用，
// 不应在 Dispatch 并发期间 mutate（无锁实现）。
func (r *StepRouter) Register(t StepType, h StepHandler) {
	r.handlers[t] = h
}

// Dispatch 按 step.type 路由到对应 handler 并执行。未注册类型返 ErrUnknownStepType。
func (r *StepRouter) Dispatch(ctx context.Context, step Step) error {
	h, ok := r.handlers[step.Type]
	if !ok {
		return fmt.Errorf("step type %q: %w", step.Type, ErrUnknownStepType)
	}
	return h(ctx, step)
}

// ---- 内建默认 stub handlers (MVP) ----

func defaultRPCHandler(logger *zap.Logger) StepHandler {
	return func(_ context.Context, step Step) error {
		// MVP：仅日志，不真发 RPC。T-0102-c 替换为 acs/rpc/ 调用。
		logger.Info("rpc step dispatched (stub)",
			zap.String("name", step.Name),
			zap.ByteString("params", step.Params))
		return nil
	}
}

func defaultMMLHandler(logger *zap.Logger) StepHandler {
	return func(_ context.Context, step Step) error {
		logger.Info("mml step dispatched (stub)",
			zap.String("name", step.Name))
		return nil
	}
}

// defaultWaitHandler 真实有效（time.Sleep with ctx cancel）；future 可保持。
func defaultWaitHandler(logger *zap.Logger) StepHandler {
	return func(ctx context.Context, step Step) error {
		var params struct {
			SecondsRaw json.RawMessage `json:"seconds"`
		}
		if len(step.Params) > 0 {
			if err := json.Unmarshal(step.Params, &params); err != nil {
				return fmt.Errorf("parse wait step params: %w", err)
			}
		}
		seconds := 0
		if len(params.SecondsRaw) > 0 {
			_ = json.Unmarshal(params.SecondsRaw, &seconds)
		}
		if seconds <= 0 {
			logger.Debug("wait step seconds<=0 noop", zap.String("name", step.Name))
			return nil
		}
		logger.Info("wait step sleeping",
			zap.String("name", step.Name),
			zap.Int("seconds", seconds))
		select {
		case <-time.After(time.Duration(seconds) * time.Second):
			return nil
		case <-ctx.Done():
			return fmt.Errorf("wait step cancelled: %w", ctx.Err())
		}
	}
}

func defaultLoopHandler(logger *zap.Logger) StepHandler {
	return func(_ context.Context, step Step) error {
		logger.Info("loop step dispatched (stub)",
			zap.String("name", step.Name))
		return nil
	}
}

func defaultBranchHandler(logger *zap.Logger) StepHandler {
	return func(_ context.Context, step Step) error {
		logger.Info("branch step dispatched (stub)",
			zap.String("name", step.Name))
		return nil
	}
}
