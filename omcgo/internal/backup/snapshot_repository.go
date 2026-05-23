package backup

import (
	"context"
)

// SnapshotRepository 是 config_snapshots 表的契约。
//
// 一设备一行（serial_number 是主键）。Upsert 语义：同 SN 冲突时覆盖
// 元数据（file_name / object_path / md5 / size / source / update_time）。
//
// 调用方：
//   - SnapshotService.PromoteFromBackup（备份链路自动写）
//   - SnapshotService.ImportFromUpload（手动导入写）
//   - RestoreService.CreateBySnapshot（读 BatchGetBySerialNumbers）
//   - SnapshotHandler（List / GetBySerialNumber / Delete）
type SnapshotRepository interface {
	// Upsert 插入或覆盖单台设备的快照行。覆盖语义：同 SN 冲突时刷新所有
	// 元数据列并把 update_time 设为当前时间；created_at 保持首次写入值。
	Upsert(ctx context.Context, snap *ConfigSnapshot) error

	// GetBySerialNumber 按 SN 取最新一行。不存在返回 nil, nil（非错误）。
	GetBySerialNumber(ctx context.Context, sn string) (*ConfigSnapshot, error)

	// BatchGetBySerialNumbers 批量按 SN 取快照。
	// 返回的 map 仅包含查到的 SN，未查到的不会出现在结果中 ——
	// 调用方据此识别"缺失快照"的设备（Restore 创建时整批拒绝的依据）。
	BatchGetBySerialNumbers(ctx context.Context, sns []string) (map[string]*ConfigSnapshot, error)

	// List 按过滤条件分页查询。返回 items + total。
	List(ctx context.Context, filter SnapshotFilter) (items []ConfigSnapshot, total int64, err error)

	// Delete 删除单台设备的快照行。不存在视为成功（幂等）。
	// 注意：本方法只删 DB 行，MinIO 对象的清理由 Service 层负责。
	Delete(ctx context.Context, sn string) error

	// BatchDelete 删除多台设备的快照行。
	// 返回实际删除的 SN 列表（即删除前在表中存在的那些）。
	BatchDelete(ctx context.Context, sns []string) (deleted []string, err error)
}
