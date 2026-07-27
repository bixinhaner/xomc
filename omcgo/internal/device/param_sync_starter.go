package device

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

type ManualParamSyncStart struct {
	Used       bool
	TaskCount  int
	RequestID  uuid.UUID
	RunID      *uuid.UUID
	Status     string
	ResultCode string
}

// DetailedParamSyncStarter is implemented by the durable request/run path.
// Legacy starters continue to satisfy ParamSyncStarter unchanged.
type DetailedParamSyncStarter interface {
	StartManualSyncDetailed(ctx context.Context, dev *model.Device, sourceID string, parameterPaths []string) (*ManualParamSyncStart, error)
}

// ParamSyncStarter 是 device 包消费者驱动的 narrow interface（T-0126 设计 §4.1）。
//
// 抽象"手动触发全量参数同步"能力，让 device 包 handler 调用不需 import
// provision 包（避免 device → provision 循环依赖，因为 provision 已 import device）。
//
// 当前优先由 durable parameter_sync_* 数据面处理；旧 sync-gpv Path B 仅作为
// 临时兜底，待 param_sync_running 稳定后删除。
//
// 主要实现者是 provider.paramSyncStarter / *provision.SyncService（通过
// SetParamSyncStarter setter 注入到 DeviceService）。底层等价于
// StartPathBSync(WithReason("manual"))，统一进入参数同步链路：
//   - reason 通道 (T-0123/T-0125 共用 Redis hint provision:syncreason:{deviceID})
//   - 差异日志 (T-0127 BatchUpsert 前后 diff B-A)
//   - 回写口径统一 (T-0124 HandleSyncResultPathB 末端 last_param_sync_at)
//
// 返回 (used, err)：used=true 表示已入队；used=false 表示 durable 与临时 legacy
// 兜底均不可用，调用方应返 503/N/A 状态码。
type ParamSyncStarter interface {
	StartManualSync(ctx context.Context, dev *model.Device, sourceID string, parameterPaths []string) (bool, int, error)
}
