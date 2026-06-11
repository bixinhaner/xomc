package software

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// taskCancelRegistry 是「紧急叫停真正生效」（#59 Problem 3）的核心：把每个正在执行
// 的主任务（升级 / 回退）映射到一个可取消的 context.CancelFunc。
//
// 背景 bug：旧 startExecution / startRollbackExecution 给每个子任务起的 goroutine
// 都用 context.Background()，与主任务生命周期脱钩。SuspendUpgrade / 阈值自动暂停
// 只改 DB status，对已经在飞的 goroutine 毫无约束——它们照样把 Download 命令推到
// 设备。结果：用户点了「暂停 / 急停」，待派发与在飞的设备仍继续收固件下发。
//
// 修复思路：执行入口（startExecution / startRollbackExecution）从一个可取消的
// 父 context 派生所有子 goroutine 的 ctx，并把该任务的 CancelFunc 登记进本注册表。
// SuspendUpgrade / TerminateUpgrade / 阈值自动暂停在改 DB 之外，额外调用 cancel()，
// 在飞的 goroutine 在派发前（ExecuteOne 顶部 + 锁后派发前的 select on ctx.Done）
// 看到取消信号即提前返回，不再下发 Download。
//
// 注册表只是「尽力而为的提前止血」：DB status 仍是权威终态，ExecuteOne 还会做
// 派发前 DB 复查（见 executor.go），所以即便 cancel 时机错过某个 goroutine，
// DB 复查仍兜底拦截。两道防线叠加，避免单点竞态。
type taskCancelRegistry struct {
	mu      sync.Mutex
	cancels map[uuid.UUID]context.CancelFunc
}

func newTaskCancelRegistry() *taskCancelRegistry {
	return &taskCancelRegistry{cancels: make(map[uuid.UUID]context.CancelFunc)}
}

// derive 为给定主任务派生一个可取消的子 context，并登记其 CancelFunc。
//
// 返回的 ctx 用于该任务下所有子任务 goroutine；调用方必须在该任务全部 goroutine
// 收尾后调用 remove(taskID) 清理注册表项（通常用 sync.WaitGroup 在监管 goroutine
// 里等齐再 remove，见 startExecution）。
//
// 若同一任务已有在册 cancel（例如 Resume 复用同一 taskID 再次启动），先取消旧的
// 再覆盖，避免泄漏：旧那批 goroutine 应当已随上一轮收尾，这里只是防御性兜底。
func (r *taskCancelRegistry) derive(parent context.Context, taskID uuid.UUID) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	r.mu.Lock()
	if old, ok := r.cancels[taskID]; ok {
		old()
	}
	r.cancels[taskID] = cancel
	r.mu.Unlock()
	return ctx, cancel
}

// remove 清理某任务的注册表项。幂等。不主动触发 cancel——cancel 的调用由
// startExecution 收尾 goroutine 的 defer 负责（释放 context 资源）。
func (r *taskCancelRegistry) remove(taskID uuid.UUID) {
	r.mu.Lock()
	delete(r.cancels, taskID)
	r.mu.Unlock()
}

// cancel 触发某任务的提前止血：调用其 CancelFunc（若在册）并从注册表移除。
// 幂等：任务不在册（已完成 / 从未启动 / 已取消）时静默返回，不报错。
// 返回是否命中过一个在册的 cancel（仅供日志 / 测试观测，不影响正确性）。
func (r *taskCancelRegistry) cancel(taskID uuid.UUID) bool {
	r.mu.Lock()
	cancel, ok := r.cancels[taskID]
	if ok {
		delete(r.cancels, taskID)
	}
	r.mu.Unlock()
	if ok {
		cancel()
	}
	return ok
}
