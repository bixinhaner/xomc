package device

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AutoRecycleDeleter 能将设备批量软删除的最小接口（由 *PgDeviceRepository 满足）。
type AutoRecycleDeleter interface {
	FindOfflineForRecycle(ctx context.Context, olderThan time.Time, limit int) ([]uuid.UUID, error)
	BatchDelete(ctx context.Context, ids []uuid.UUID, deletedBy string) (int64, error)
}

// SysConfigLookupFn 从 sys_configs 读取 (category, key) → value 的函数签名，
// 与 transfercfg.SysConfigLookup 同形（避免跨包引用）。
type SysConfigLookupFn func(ctx context.Context, category, key string) (value string, found bool)

// AutoRecycleJob 回收站自动移入定时任务。
// 每次运行时：
//  1. 读取 sys_configs device:deviceOfflineEnable（总开关）
//  2. 读取 sys_configs device:deviceOfflineSaveDay（离线天数阈值）
//  3. 开关 false 时跳过
//  4. 查询 is_online=false AND last_inform_at < now()-N days AND deleted_at IS NULL 设备
//  5. 批量软删除（deleted_by='system:auto_recycle'）
//
// 单实例假设：当前仅单 worker 部署，无并发锁保护；横扩前需补 PG advisory lock。
type AutoRecycleJob struct {
	lookup    SysConfigLookupFn
	deviceOps AutoRecycleDeleter
	batchSize int
	logger    *zap.Logger
}

// DefaultAutoRecycleBatchSize 单批最大软删设备数（防大事务）。
const DefaultAutoRecycleBatchSize = 500

// DefaultAutoRecycleCron 每天 00:10 执行（与 UI 上「每天 00:10 检查设备离线时间」一致）。
const DefaultAutoRecycleCron = "10 0 * * *"

// NewAutoRecycleJob 构造 AutoRecycleJob。
// batchSize <= 0 时回退 DefaultAutoRecycleBatchSize。
func NewAutoRecycleJob(
	lookup SysConfigLookupFn,
	deviceOps AutoRecycleDeleter,
	batchSize int,
	logger *zap.Logger,
) *AutoRecycleJob {
	if batchSize <= 0 {
		batchSize = DefaultAutoRecycleBatchSize
	}
	return &AutoRecycleJob{
		lookup:    lookup,
		deviceOps: deviceOps,
		batchSize: batchSize,
		logger:    logger,
	}
}

// Run 执行一次自动回收扫描，返回本次软删的设备总数。
// 可被 cron 调度，也可在启动期 catch-up 时手动调用。
func (j *AutoRecycleJob) Run(ctx context.Context) (int64, error) {
	// 1. 读取总开关
	enableVal, found := j.lookup(ctx, "device", "deviceOfflineEnable")
	if !found || enableVal != "true" {
		j.logger.Debug("auto recycle skipped: deviceOfflineEnable not set or false")
		return 0, nil
	}

	// 2. 读取离线天数阈值
	dayVal, found := j.lookup(ctx, "device", "deviceOfflineSaveDay")
	if !found || dayVal == "" {
		j.logger.Warn("auto recycle skipped: deviceOfflineSaveDay not configured")
		return 0, nil
	}
	days, parseErr := strconv.Atoi(dayVal)
	if parseErr != nil {
		return 0, fmt.Errorf("auto recycle: deviceOfflineSaveDay %q is not a valid integer: %w", dayVal, parseErr)
	}
	if days <= 0 {
		return 0, fmt.Errorf("auto recycle: deviceOfflineSaveDay must be > 0, got %d", days)
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	j.logger.Info("auto recycle starting",
		zap.Int("offline_days", days),
		zap.Time("cutoff", cutoff))

	// 3. 循环批量查询 + 软删除（防止 batchSize 内仍有大量设备时的超时）
	var totalDeleted int64
	for {
		ids, queryErr := j.deviceOps.FindOfflineForRecycle(ctx, cutoff, j.batchSize)
		if queryErr != nil {
			return totalDeleted, fmt.Errorf("find offline devices: %w", queryErr)
		}
		if len(ids) == 0 {
			break
		}

		deleted, delErr := j.deviceOps.BatchDelete(ctx, ids, "system:auto_recycle")
		if delErr != nil {
			return totalDeleted, fmt.Errorf("batch delete offline devices: %w", delErr)
		}
		totalDeleted += deleted

		j.logger.Info("auto recycle batch done",
			zap.Int("queried", len(ids)),
			zap.Int64("deleted", deleted),
			zap.Int64("total_so_far", totalDeleted))

		// 若本批未满说明已无更多数据，退出循环；否则继续下一批
		if len(ids) < j.batchSize {
			break
		}
	}

	j.logger.Info("auto recycle finished", zap.Int64("total_deleted", totalDeleted))
	return totalDeleted, nil
}
