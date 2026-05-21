package device

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AbnormalRebootSnapshot 是 device 包发布给"识别即落库"链路的快照入参。
//
// 字段来源：
//   - DeviceID / DeviceSN / DeviceName / OperateIP / SoftwareVersion ← model.Device 当前状态
//   - HaltMainReason / HaltDetailReason / RuntimeBeforeReboot ← TR-069 Inform 参数列表
//   - DeviceType / IsGNB ← Device.Technology 派生
//   - DetectedAt ← 异常重启识别时刻
//
// 故意定义在 device 包而非 stationlog 包：这是 device 的"语义"，stationlog 只是
// 实现"如何持久化"的细节。stationlog.Service 通过 adapter 把这个 snapshot 转成
// 它自己的 LogFile 记录写入 station_fault_logs。
type AbnormalRebootSnapshot struct {
	DeviceID            uuid.UUID
	DeviceSN            string
	DeviceName          string
	DeviceType          string // eNB / gNB
	IsGNB               bool
	OperateIP           string
	SoftwareVersion     string
	HaltMainReason      string
	HaltDetailReason    string
	RuntimeBeforeReboot int64 // 秒；0 表示未知
	DetectedAt          time.Time
}

// AbnormalRebootRecorder 是 device.DeviceService 用来把异常重启信息落库的窄接口。
// 由 stationlog.Service 在 modules.go wiring 时实现并注入。
type AbnormalRebootRecorder interface {
	RecordAbnormalReboot(ctx context.Context, snap AbnormalRebootSnapshot) error
}

// SetAbnormalRebootRecorder 注入"识别即落库"实现；nil 等价于禁用（保持向后
// 兼容）。在 modules.go 完成 wiring 后由 provider 显式调用。
func (s *DeviceService) SetAbnormalRebootRecorder(r AbnormalRebootRecorder) {
	s.abnormalRecorder = r
}
