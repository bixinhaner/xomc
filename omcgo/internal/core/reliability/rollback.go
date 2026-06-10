// Package reliability 已在 circuit_breaker.go 中声明包注释。
package reliability

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// txRollbacker 抽象 pgx.Tx 的 Rollback，便于在 defer 中统一处理回滚错误，
// 也方便单元测试注入桩。生产由 pgx.Tx / pgxpool.Tx 满足。
type txRollbacker interface {
	Rollback(ctx context.Context) error
}

// RollbackTx 在 defer 中安全回滚事务并记录回滚失败。
//
// 背景（MEDIUM-18）：大量代码用 `defer func() { _ = tx.Rollback(ctx) }()`，
// 即使 Rollback 因连接断裂 / 死锁失败也被静默吞掉，导致事务挂起、连接池泄漏
// 无法观测。本 helper 检查回滚错误：
//   - 成功提交后 Rollback 返回 pgx.ErrTxClosed（no-op，正常路径）→ 不记录；
//   - 其它错误 → 以 warn 记录，附带调用点 op 标签，让运维可见连接泄漏风险。
//
// logger 为 nil 时安全降级为不记录（仅保留"检查并消费错误"语义）。
// 典型用法：
//
//	tx, err := pool.Begin(ctx)
//	...
//	defer reliability.RollbackTx(ctx, tx, logger, "loadParamModelFile")
func RollbackTx(ctx context.Context, tx txRollbacker, logger *zap.Logger, op string) {
	err := tx.Rollback(ctx)
	if err == nil || errors.Is(err, pgx.ErrTxClosed) {
		return
	}
	if logger != nil {
		logger.Warn("transaction rollback failed",
			zap.String("op", op),
			zap.Error(err))
	}
}
