package provision

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// LeaderElector 多副本协调抽象（T-0124 设计 §2.5）。
//
// 主要用途：在 RC 多副本 app 部署下，保证同一时刻只有一个副本执行周期性任务
// （如 PeriodicSyncer.runOnce），避免同一设备被多个副本同时入队浪费 GPV 配额
// 并错乱 last_param_sync_at。
//
// 接口刻意保持小（2 方法），便于后续 T-G post-RC 让 OfflineDetector 复用同一抽象。
type LeaderElector interface {
	// TryAcquire 尝试拿到 leader 锁。返回 (true, nil) 即本副本是 leader；
	// (false, nil) 表示其他副本是 leader（应本轮跳过）；(_, err) 表示协调器
	// 访问失败（实现应自行决定调用方行为，建议保守跳过本轮）。
	TryAcquire(ctx context.Context) (bool, error)
	// Release 显式释放 leader 锁。通常进程退出时调用；PG 实现允许直接由
	// 连接归还自动释放（session 结束），Release 仅为正常路径优化。
	Release(ctx context.Context) error
}

// PGAdvisoryLeaderElector 基于 PostgreSQL `pg_try_advisory_lock` 的 LeaderElector
// 实现（T-0124 设计 §2.5）。
//
// 关键不变量：
//   - Acquire 成功必须 hold 同一连接到 Release（advisory_lock 是 session-scoped）；
//     不能用每次借连接的方式调 pg_advisory_unlock —— 不同连接不见其他 session 的锁
//   - conn.Release() 归还 pool 时 PG 自动 release 锁（session 结束），因此异常退出 /
//     进程崩溃 / pgxpool 健康检查驱逐都天然不会残留死锁
//   - 多副本：第一个 TryAcquire 成功的副本成为 leader 全程持有；leader 副本退出时
//     conn 归还 → PG session 结束 → 锁自动释放 → 下一副本下一 tick TryAcquire 成功
//
// lockName 通过 `hashtext()` 转 int4 作为 pg_advisory_lock 的 key（advisory_lock
// 仅支持 bigint/int4 入参，不接受字符串），同 lockName 进程间天然协调。
type PGAdvisoryLeaderElector struct {
	pool     *pgxpool.Pool
	lockName string
	held     atomic.Bool
	connMu   sync.Mutex
	conn     *pgxpool.Conn // 持锁期间 hold 一个连接；Release 时归还
	logger   *zap.Logger
}

// NewPGAdvisoryLeaderElector 创建一个 PG advisory lock 协调器。
//
// lockName 推荐使用稳定的英文标识符（如 "periodic_param_syncer"），避免 hash 碰撞。
func NewPGAdvisoryLeaderElector(pool *pgxpool.Pool, lockName string, logger *zap.Logger) *PGAdvisoryLeaderElector {
	return &PGAdvisoryLeaderElector{
		pool:     pool,
		lockName: lockName,
		logger:   logger,
	}
}

// TryAcquire 调 pg_try_advisory_lock(hashtext(lockName))。
//
// 已持锁 → 直接返 (true, nil)（幂等）；未持锁但拿到 → hold 连接并标记 held；
// 未拿到 → 归还连接 + 返 (false, nil)；连接异常 → 返 (false, err)。
func (e *PGAdvisoryLeaderElector) TryAcquire(ctx context.Context) (bool, error) {
	if e.held.Load() {
		return true, nil
	}
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("acquire pg conn: %w", err)
	}
	var ok bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtext($1))", e.lockName).Scan(&ok); err != nil {
		conn.Release()
		return false, fmt.Errorf("pg_try_advisory_lock: %w", err)
	}
	if !ok {
		conn.Release()
		return false, nil
	}
	e.connMu.Lock()
	e.conn = conn
	e.connMu.Unlock()
	e.held.Store(true)
	if e.logger != nil {
		e.logger.Info("PG advisory lock acquired",
			zap.String("lock_name", e.lockName))
	}
	return true, nil
}

// Release 显式调 pg_advisory_unlock 并归还连接。即使返 error 也清状态避免卡死。
// 重复 Release 是安全的 no-op。
func (e *PGAdvisoryLeaderElector) Release(ctx context.Context) error {
	if !e.held.Load() {
		return nil
	}
	e.connMu.Lock()
	conn := e.conn
	e.conn = nil
	e.connMu.Unlock()
	e.held.Store(false)
	if conn == nil {
		return nil
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_unlock(hashtext($1))", e.lockName); err != nil {
		// 退出场景容错：日志记录后归还连接（pool.Release 会让 PG session 结束自动 unlock）
		if e.logger != nil {
			e.logger.Warn("pg_advisory_unlock failed (lock will be released on conn return)",
				zap.String("lock_name", e.lockName),
				zap.Error(err))
		}
		return fmt.Errorf("pg_advisory_unlock: %w", err)
	}
	if e.logger != nil {
		e.logger.Info("PG advisory lock released",
			zap.String("lock_name", e.lockName))
	}
	return nil
}
