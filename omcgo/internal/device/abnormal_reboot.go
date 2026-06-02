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

// BootEventSnapshot 是普通 1 BOOT 事件（无 HaltReason）落库到 event_logs 的快照入参。
// 与 AbnormalRebootSnapshot 故意分开：异常重启需要 HaltReason / 关联文件等丰富信息走
// station_fault_logs；普通 BOOT 只做轻量审计流水，字段更瘦。
type BootEventSnapshot struct {
	DeviceID        uuid.UUID
	DeviceSN        string
	DeviceName      string
	DeviceType      string // eNB / gNB
	IsGNB           bool
	OperateIP       string
	SoftwareVersion string
	// 重启前设备已运行的秒数；0 表示未知（与异常重启 AbnormalRebootSnapshot 对齐）
	RuntimeBeforeReboot int64
	BootCount           int
	Events              []string // 原始 Inform 事件码列表，便于排错
	OccurredAt          time.Time
}

// BootEventRecorder 是 DeviceService 用来把普通 1 BOOT 写入 event_logs 的窄接口。
// 由 eventlog.Service 在 modules.go wiring 时实现并注入。
type BootEventRecorder interface {
	RecordBootEvent(ctx context.Context, snap BootEventSnapshot) error
}

// SetBootEventRecorder 注入事件日志写入实现；nil 等价于禁用。
func (s *DeviceService) SetBootEventRecorder(r BootEventRecorder) {
	s.bootEventRecorder = r
}
