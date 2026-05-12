// Package ops 并发控制与节流（T-0101-c + T-0102-e）。
//
// PRD §5.3.2 要求 dispatcher 同时执行的 task 数 ≤ max（默认 20），每秒
// 新增 RPC 分发数受限（防 ACS 雪崩），单设备分发数也受限（PRD §4.2.3
// "批量场景设备级独立超时" + per-device 限流）。本文件提供
// ConcurrencyLimiter 复合控制器（buffered channel 信号量 + 全局
// rate.Limiter + per-device rate.Limiter map）。
package ops

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/time/rate"
)

// DefaultMaxConcurrentTasks 默认并发任务数（PRD §5.3.2）。
const DefaultMaxConcurrentTasks = 20

// DefaultRPCRateLimit 默认全局每秒新增 RPC 分发数（防 ACS 雪崩）。
const DefaultRPCRateLimit = 50

// DefaultPerDeviceRPCRateLimit T-0102-e 默认单设备每秒 RPC 分发数；
// 防同设备短时间被同任务多次轰炸（PRD §4.2.3）。低于全局节流值，
// 因单设备的资源（CWMP 会话）窗口比全局窄。
const DefaultPerDeviceRPCRateLimit = 5

// ConcurrencyLimiter 任务并发控制器（T-0101-c + T-0102-e）：
//   - 任务级 max=20：信号量 buffered channel 控制 dispatcher 同时执行的 task
//   - 全局 RPC 节流：rate.Limiter 控制每秒 RPC 分发频率，防 ACS 雪崩
//   - 单设备 RPC 节流（T-0102-e）：per-device rate.Limiter map，
//     防同一 device_sn 短时间被同任务多次 RPC 轰炸
//
// 调用方典型流：
//
//	if err := limiter.AcquireTask(ctx); err != nil { return err }
//	defer limiter.ReleaseTask()
//	for _, step := range steps {
//	    if err := limiter.WaitRPC(ctx); err != nil { return err }
//	    if err := limiter.WaitForDevice(ctx, deviceSN); err != nil { return err }
//	    // 发出 RPC
//	}
type ConcurrencyLimiter struct {
	taskSem       chan struct{} // buffered = MaxConcurrent
	rpcLim        *rate.Limiter
	devLimiters   sync.Map // map[string]*rate.Limiter — lazy-created per device_sn
	perDeviceRate int      // RPC/sec quota for each device limiter
}

// NewConcurrencyLimiter 创建并发控制器；maxTasks 是同时执行的 task 数；
// rpcPerSec 是全局每秒 RPC 分发速率（burst = rpcPerSec）。
// 任一值 ≤ 0 时使用 DefaultMaxConcurrentTasks / DefaultRPCRateLimit。
// 单设备速率默认 DefaultPerDeviceRPCRateLimit，可通过 SetPerDeviceRate 覆盖。
func NewConcurrencyLimiter(maxTasks, rpcPerSec int) *ConcurrencyLimiter {
	if maxTasks <= 0 {
		maxTasks = DefaultMaxConcurrentTasks
	}
	if rpcPerSec <= 0 {
		rpcPerSec = DefaultRPCRateLimit
	}
	return &ConcurrencyLimiter{
		taskSem:       make(chan struct{}, maxTasks),
		rpcLim:        rate.NewLimiter(rate.Limit(rpcPerSec), rpcPerSec),
		perDeviceRate: DefaultPerDeviceRPCRateLimit,
	}
}

// SetPerDeviceRate 覆盖默认单设备 RPC 速率。≤0 时回落默认值。
// 必须在第一次 WaitForDevice 调用前设置（既有 limiter 不重建）。
func (l *ConcurrencyLimiter) SetPerDeviceRate(perSec int) {
	if perSec <= 0 {
		perSec = DefaultPerDeviceRPCRateLimit
	}
	l.perDeviceRate = perSec
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

// WaitRPC 阻塞等待全局 RPC 速率限制放行（防 ACS 雪崩）；ctx 取消时立即返。
func (l *ConcurrencyLimiter) WaitRPC(ctx context.Context) error {
	if err := l.rpcLim.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit rpc: %w", err)
	}
	return nil
}

// WaitForDevice 阻塞等待该 device_sn 的单设备 RPC 速率限制放行（T-0102-e）；
// limiter 按 device_sn 懒创建并通过 sync.Map 缓存复用；ctx 取消时立即返。
//
// 不做主动 GC：单设备 limiter 体积小（rate.Limiter 约 48 字节），且
// dispatcher 调用面里 device_sn 集合在一段时间内基本稳定；过期清理
// 可作为 future 监控指标推动改动（当前最简化 MVP）。
func (l *ConcurrencyLimiter) WaitForDevice(ctx context.Context, deviceSN string) error {
	if deviceSN == "" {
		return nil
	}
	v, _ := l.devLimiters.LoadOrStore(deviceSN,
		rate.NewLimiter(rate.Limit(l.perDeviceRate), l.perDeviceRate))
	lim := v.(*rate.Limiter)
	if err := lim.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit device %s: %w", deviceSN, err)
	}
	return nil
}

// CurrentTaskCount 当前已占用的任务槽位数（O(1) chan len，主要用于监控）。
func (l *ConcurrencyLimiter) CurrentTaskCount() int {
	return len(l.taskSem)
}

// DeviceLimiterCount 当前已懒创建的 per-device limiter 数（监控用）。
// 主要用作内存占用观察点。
func (l *ConcurrencyLimiter) DeviceLimiterCount() int {
	var n int
	l.devLimiters.Range(func(_, _ interface{}) bool { n++; return true })
	return n
}
