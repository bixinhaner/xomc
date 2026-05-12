// Package ops 并发控制与节流（T-0101-c）。
//
// PRD §5.3.2 要求 dispatcher 同时执行的 task 数 ≤ max（默认 20），且每秒
// 新增 RPC 分发数受限（防 ACS 雪崩）。本文件提供 ConcurrencyLimiter 复合
// 控制器（buffered channel 信号量 + golang.org/x/time/rate.Limiter）。
package ops

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"
)

// DefaultMaxConcurrentTasks 默认并发任务数（PRD §5.3.2）。
const DefaultMaxConcurrentTasks = 20

// DefaultRPCRateLimit 默认每秒新增 RPC 分发数（防 ACS 雪崩）。
const DefaultRPCRateLimit = 50

// ConcurrencyLimiter 任务并发控制器（T-0101-c）：
//   - 任务级 max=20：信号量 buffered channel 控制 dispatcher 同时执行的 task
//   - RPC 级节流：rate.Limiter 控制每秒 RPC 分发频率，防 ACS 雪崩
//
// 调用方典型流：
//
//	if err := limiter.AcquireTask(ctx); err != nil { return err }
//	defer limiter.ReleaseTask()
//	for _, step := range steps {
//	    if err := limiter.WaitRPC(ctx); err != nil { return err }
//	    // 发出 RPC
//	}
type ConcurrencyLimiter struct {
	taskSem  chan struct{}  // buffered = MaxConcurrent
	rpcLim   *rate.Limiter
}

// NewConcurrencyLimiter 创建并发控制器；maxTasks 是同时执行的 task 数；
// rpcPerSec 是每秒 RPC 分发速率（burst = rpcPerSec）。
// 任一值 ≤ 0 时使用 DefaultMaxConcurrentTasks / DefaultRPCRateLimit。
func NewConcurrencyLimiter(maxTasks, rpcPerSec int) *ConcurrencyLimiter {
	if maxTasks <= 0 {
		maxTasks = DefaultMaxConcurrentTasks
	}
	if rpcPerSec <= 0 {
		rpcPerSec = DefaultRPCRateLimit
	}
	return &ConcurrencyLimiter{
		taskSem: make(chan struct{}, maxTasks),
		rpcLim:  rate.NewLimiter(rate.Limit(rpcPerSec), rpcPerSec),
	}
}

// AcquireTask 占用一个任务级槽位；当并发达上限时阻塞，直到有 task 释放或
// ctx 取消。返回 ctx.Err() 当超时/取消。
func (l *ConcurrencyLimiter) AcquireTask(ctx context.Context) error {
	select {
	case l.taskSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("acquire task slot: %w", ctx.Err())
	}
}

// ReleaseTask 释放一个任务级槽位。调用方应 defer 调用。
func (l *ConcurrencyLimiter) ReleaseTask() {
	<-l.taskSem
}

// WaitRPC 阻塞等待 RPC 速率限制放行（防 ACS 雪崩）；ctx 取消时立即返。
func (l *ConcurrencyLimiter) WaitRPC(ctx context.Context) error {
	if err := l.rpcLim.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit rpc: %w", err)
	}
	return nil
}

// CurrentTaskCount 当前已占用的任务槽位数（O(1) chan len，主要用于监控）。
func (l *ConcurrencyLimiter) CurrentTaskCount() int {
	return len(l.taskSem)
}
