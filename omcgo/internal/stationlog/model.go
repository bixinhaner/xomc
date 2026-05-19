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

// LogFile 代表一条日志采集文件记录，对应 station_log_files 表。
type LogFile struct {
	ID          uuid.UUID  `json:"id"`
	DeviceID    *uuid.UUID `json:"device_id,omitempty"`
	DeviceSN    string     `json:"device_sn"`
	LogType     LogType    `json:"log_type"`
	FileName    string     `json:"file_name"`
	ObjectPath  string     `json:"object_path"`
	Bucket      string     `json:"bucket"`
	FileSize    int64      `json:"file_size"`
	FaultReason string     `json:"fault_reason,omitempty"`
	FaultDetail string     `json:"fault_detail,omitempty"`
	TaskID      *uuid.UUID `json:"task_id,omitempty"`
	IsDeleted   bool       `json:"is_deleted"`
	CollectedAt time.Time  `json:"collected_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// LogFileFilter 日志文件查询过滤条件
type LogFileFilter struct {
	DeviceID *uuid.UUID
	DeviceSN string
	LogType  LogType
	Page     int
	PageSize int
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
