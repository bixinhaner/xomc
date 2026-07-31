package stationlog

import (
	"time"

	"github.com/google/uuid"
)

// LogType 日志类型：运行日志或故障日志
type LogType string

const (
	LogTypeRunning LogType = "running" // FileType "6"
	LogTypeFault   LogType = "fault"   // FileType "8" / "RL"
)

// FaultLogMaxCount 全局故障日志最大保留文件数
const FaultLogMaxCount = 20

// 故障日志记录生命周期（station_fault_logs.record_status）：
//   - detected           满足异常重启规则即落库的占位记录（无文件）
//   - file_received      设备已上传文件并完成持久化
//   - collection_failed  收集失败（如手动收集超时、磁盘满）
const (
	FaultRecordStatusDetected         = "detected"
	FaultRecordStatusFileReceived     = "file_received"
	FaultRecordStatusCollectionFailed = "collection_failed"
)

// 手动收集任务状态（station_fault_logs.manual_collection_status）：
//   - "0"  未收集 / 已完成
//   - "1"  收集中
//   - "2"  失败
const (
	ManualCollectionIdle    = "0"
	ManualCollectionRunning = "1"
	ManualCollectionFailed  = "2"
)

// LogFile 代表一条日志采集文件记录。
//
// 运行日志（station_running_logs）仅使用前半部分字段；
// 故障日志（station_fault_logs）会用到全部字段（含 RecordStatus / HaltReason 等）。
type LogFile struct {
	ID          uuid.UUID  `json:"id"`
	DeviceID    *uuid.UUID `json:"device_id,omitempty"`
	DeviceSN    string     `json:"device_sn"`
	LogType     LogType    `json:"log_type"`
	FileName    string     `json:"file_name,omitempty"`   // detected 状态下为空
	ObjectPath  string     `json:"object_path,omitempty"` // detected 状态下为空
	Bucket      string     `json:"bucket,omitempty"`      // detected 状态下为空
	FileSize    int64      `json:"file_size"`
	FaultReason string     `json:"fault_reason,omitempty"` // HaltReason.MainReason
	FaultDetail string     `json:"fault_detail,omitempty"` // HaltReason.DetailReason
	TaskID      *uuid.UUID `json:"task_id,omitempty"`
	IsDeleted   bool       `json:"is_deleted"`
	CollectedAt time.Time  `json:"collected_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// 设备快照字段（仅 station_fault_logs）
	DeviceName      string `json:"device_name,omitempty"`
	DeviceType      string `json:"device_type,omitempty"` // eNB / gNB
	IsGNB           bool   `json:"is_gnb"`
	OperateIP       string `json:"operate_ip,omitempty"`
	SoftwareVersion string `json:"software_version,omitempty"`
	// 重启前设备已运行的秒数；0 表示未知
	RuntimeBeforeReboot int64 `json:"runtime_before_reboot,omitempty"`

	// 收集状态字段（仅 station_fault_logs）
	RecordStatus           string `json:"record_status,omitempty"`
	CollectionFailReason   string `json:"collection_fail_reason,omitempty"`
	ManualCollectionStatus string `json:"manual_collection_status,omitempty"`
}

// LogFileFilter 日志文件查询过滤条件
type LogFileFilter struct {
	DeviceID     *uuid.UUID
	DeviceSN     string
	LogType      LogType
	RecordStatus string // 仅故障日志：detected / file_received / collection_failed
	DeviceType   string // 仅故障日志：eNB / gNB
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	PageSize     int
	// VisibleGroups 是 #63 设备组可见性强制层注入的调用者可见分组集合（三态契约见
	// authz 包）：nil=超管不过滤，[]=无权限空集 fail-closed，[ids]=限定。仓库层用
	// authz.ApplyDeviceVisibilityFilter 按 device_id 列收窄结果。
	VisibleGroups []uuid.UUID
}

func (f LogFileFilter) Offset() int {
	if f.Page <= 1 {
		return 0
	}
	return (f.Page - 1) * f.PageSize
}

func (f LogFileFilter) Limit() int {
	if f.PageSize <= 0 {
		return 20
	}
	return f.PageSize
}

// LogFileReceivedPayload 是 SubjectLogFileReceived 事件的 payload 结构。
type LogFileReceivedPayload struct {
	Bucket     string `json:"bucket"`
	ObjectPath string `json:"object_path"`
	FileName   string `json:"file_name"`
	FileType   string `json:"file_type"` // "6" running, "8" fault
	FileSize   int64  `json:"file_size"`
	TaskID8    string `json:"task_id8"`  // 任务 ID 前 8 位十六进制（从文件名解析）
	DeviceSN   string `json:"device_sn"` // 从文件名解析
}

// fileTypeToLogType 将 TR-069 文件类型码转换为 LogType。
func fileTypeToLogType(ft string) LogType {
	switch ft {
	case "8", "RL":
		return LogTypeFault
	default:
		return LogTypeRunning
	}
}
