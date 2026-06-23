package device

import (
	"context"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ParamSyncStarter 是 device 包消费者驱动的 narrow interface（T-0126 设计 §4.1）。
//
// 抽象"手动触发 Path B 全量参数同步"能力，让 device 包 handler 调用不需 import
// provision 包（避免 device → provision 循环依赖，因为 provision 已 import device）。
//
// 唯一实现者是 *provision.SyncService（通过 SetParamSyncStarter setter 注入到
// DeviceService）。底层调 provision.SyncService.StartManualSync 等价于
// StartPathBSync(WithReason("manual"))，触发完整 Path B 链路：
//   - reason 通道 (T-0123/T-0125 共用 Redis hint provision:syncreason:{deviceID})
//   - 差异日志 (T-0127 BatchUpsert 前后 diff B-A)
//   - 回写口径统一 (T-0124 HandleSyncResultPathB 末端 last_param_sync_at)
//
// 返回 (used, err)：used=true 表示已入队 Path B；used=false 表示 Path B 不可用
// （MappingSet 缺失，设备 productClass 未路由），调用方应返 503/N/A 状态码。
type ParamSyncStarter interface {
	StartManualSync(ctx context.Context, dev *model.Device, sourceID string, parameterPaths []string) (bool, int, error)
}
