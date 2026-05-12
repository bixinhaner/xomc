// Package ops 失败策略（T-0101-f）。
//
// PRD §5.3.2 定义任务步骤失败后的 4 种处理策略：
//   - abort     — 立即中止任务，剩余步骤不执行
//   - continue  — 跳过当前步骤，继续下一个
//   - retry-N   — 重试当前步骤最多 N 次，仍失败则按 abort 处理
//   - rollback  — 触发 template.rollback_steps 反向序列（T-0101-h 实现）
//
// 本文件提供策略决策器 PolicyDecider —— dispatcher 在步骤失败后调
// Decide(policy, currentRetry, err) 获取下一动作；不耦合 dispatcher 主循环。
package ops

import (
	"fmt"
	"math"
	"time"
)

// FailureAction PolicyDecider 决策结果（dispatcher 据此走下一步）。
type FailureAction string

const (
	ActionAbort    FailureAction = "abort"
	ActionContinue FailureAction = "continue"
	ActionRetry    FailureAction = "retry"
	ActionRollback FailureAction = "rollback"
)

// PolicyDecider 失败策略决策器（无状态、纯函数）。
//
// 用法：
//
//	decider := NewPolicyDecider(FailureRetry, 3)
//	for attempt := 0; ; attempt++ {
//	    err := dispatchStep(step)
//	    if err == nil { break }
//	    action, delay := decider.Decide(attempt)
//	    if delay > 0 { time.Sleep(delay) }
//	    switch action {
//	    case ActionRetry: continue
//	    case ActionAbort, ActionRollback: return action  // dispatcher 退出循环
//	    case ActionContinue: break                       // 跳过此步
//	    }
//	}
type PolicyDecider struct {
	policy   FailurePolicy
	maxRetry int
}

// NewPolicyDecider 创建决策器；policy=FailureRetry 时 maxRetry 必须 >=1，否则用 3。
func NewPolicyDecider(policy FailurePolicy, maxRetry int) *PolicyDecider {
	if policy == FailureRetry && maxRetry < 1 {
		maxRetry = 3
	}
	return &PolicyDecider{policy: policy, maxRetry: maxRetry}
}

// Decide 给定当前已重试次数（0-based 首次失败后调），返下一动作 + 重试延迟。
// 延迟仅在 ActionRetry 时 > 0；其它动作返 0。
//
//	policy=FailureAbort    → ActionAbort 立即返
//	policy=FailureContinue → ActionContinue 跳过此步
//	policy=FailureRetry    → attempt<maxRetry 时返 ActionRetry+exp backoff；
//	                         attempt>=maxRetry 时返 ActionAbort（重试用尽）
//	policy=FailureRollback → ActionRollback 触发回滚
//
// 指数退避：delay = min(2^attempt * 1s, 30s)（防长尾阻塞）。
func (d *PolicyDecider) Decide(attempt int) (FailureAction, time.Duration) {
	switch d.policy {
	case FailureAbort:
		return ActionAbort, 0
	case FailureContinue:
		return ActionContinue, 0
	case FailureRollback:
		return ActionRollback, 0
	case FailureRetry:
		if attempt < d.maxRetry {
			// 指数退避：2^attempt 秒，封顶 30s
			backoffSecs := math.Pow(2, float64(attempt))
			if backoffSecs > 30 {
				backoffSecs = 30
			}
			return ActionRetry, time.Duration(backoffSecs) * time.Second
		}
		// 重试用尽 → abort
		return ActionAbort, 0
	default:
		// 未知 policy 走最保守 abort
		return ActionAbort, 0
	}
}

// String FailureAction 实现 Stringer 便于 zap log。
func (a FailureAction) String() string { return string(a) }

// Describe 返决策器配置描述（便于 audit / log）。
func (d *PolicyDecider) Describe() string {
	if d.policy == FailureRetry {
		return fmt.Sprintf("retry-max=%d", d.maxRetry)
	}
	return string(d.policy)
}
