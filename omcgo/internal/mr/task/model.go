// Package task 实现 F05 MR 测量任务管理：任务 CRUD、SPV 下发开启/关闭、
// 调度器按 start_time/end_time 自动触发、Redis 心跳健康度巡检。
//
// 设计与规范：
//   - PRD：docs/project/prd/F05-mr-task-management.md
//   - 开发计划：docs/project/mr-task-management-development-plan-20260525.md
//   - 协议规范：MR_Feature_Analysis.md（项目根目录）
//
// 架构定位：MR 任务**独立引擎**，UFTE 仅做入口聚合（不扩展 UFTE）。
// 原因：MR 任务是长时持续 + cell 维度 + TTL 心跳的状态机，与 UFTE 一次性
// Download/Upload 模型差异过大。
package task

import (
	"time"

	"github.com/google/uuid"
)

// ----------- 状态枚举 -----------

// TaskStatus 任务生命周期状态。
//
//	waitting    — 已创建，未到 start_time，等待调度
//	on          — 已下发开启 SPV，至少一个 cell 成功
//	off         — 已到达 end_time 或被手动停止，关闭 SPV 已发
//	suspend     — 挂起（保留状态，首版不暴露给用户）
//	termination — 终止中（手动 stop 触发，等关闭 SPV 全部回执）
type TaskStatus string

const (
	StatusWaiting     TaskStatus = "waitting" // 拼写与文档 §10 保持一致
	StatusOn          TaskStatus = "on"
	StatusOff         TaskStatus = "off"
	StatusSuspend     TaskStatus = "suspend"
	StatusTermination TaskStatus = "termination"
)

// ProgressStatus cell 级下发结果。
type ProgressStatus string

const (
	ProgressPending      ProgressStatus = "pending"
	ProgressOpenSuccess  ProgressStatus = "openSuccess"
	ProgressOpenFailure  ProgressStatus = "openFailure"
	ProgressCloseSuccess ProgressStatus = "closeSuccess"
	ProgressCloseFailure ProgressStatus = "closeFailure"
	ProgressUnsupport    ProgressStatus = "unsupport"
	ProgressTimeOut      ProgressStatus = "timeOut"
	ProgressNoPermission ProgressStatus = "noPermission"
)

// HealthStatus cell 级上报健康度（独立于 ProgressStatus）。
type HealthStatus string

const (
	HealthNormal   HealthStatus = "normal"
	HealthAbnormal HealthStatus = "abnormal"
	HealthUnknown  HealthStatus = "unknown"
)

// ----------- 报告周期合法集合 -----------

// 文档 §3.3：ReportPeriod 分钟值 × 60 = UploadPeriod 秒值。
var validReportPeriods = map[string]int{
	"15": 900,
	"30": 1800,
	"60": 3600,
}

// 文档 §3.2：PeriodicReportInterval 下发原值。
var validStatisPeriods = map[string]struct{}{
	"2048": {}, "5120": {}, "10240": {},
	"1": {}, "6": {}, "12": {}, "30": {}, "60": {},
}

// UploadPeriodSeconds 把 UI 的 report_period（分钟字符串）转为 TR-069
// UploadPeriod 秒数。非法值返回 (0, false)。
func UploadPeriodSeconds(reportPeriod string) (int, bool) {
	v, ok := validReportPeriods[reportPeriod]
	return v, ok
}

// IsValidStatisPeriod 校验 statis_period 是否在文档定义的合法集合内。
func IsValidStatisPeriod(v string) bool {
	_, ok := validStatisPeriods[v]
	return ok
}

// heartbeatTTLTable 严格按 MR_Feature_Analysis.md §7.3 给定值。
// 注意 15min 档不是 ×1.5（应为 1350），而是 +600 留更长容差 —— 这是规范明文要求，
// 不要"看上去更对称"地改成 1350。
var heartbeatTTLTable = map[int]int{
	900:  1500,
	1800: 2700,
	3600: 5400,
}

// HeartbeatTTLSeconds 给出 Redis MRFileReport_{cellCode} 的 TTL 秒数。
// 文档 §7.3 表：900→1500 / 1800→2700 / 3600→5400。
// 未在合法集合内的 uploadPeriodSec 返回 0（调用方应跳过设置心跳）。
func HeartbeatTTLSeconds(uploadPeriodSec int) int {
	return heartbeatTTLTable[uploadPeriodSec]
}

// ----------- 实体类型 -----------

// Task 对应 mr_customize_task 行。
//
// 简化模型（2026-05-25）：去除 operator_code / note 字段（不再有多租户隔离 + 不要备注）；
// creator 保留但由后端从 auth ctx 自动填，不再走 UI 入参。
type Task struct {
	TaskID       uuid.UUID  `db:"task_id"       json:"task_id"`
	TaskName     string     `db:"task_name"     json:"task_name"`
	MRType       string     `db:"mr_type"       json:"mr_type"` // "MRS,MRE,MRO"
	StatisPeriod string     `db:"statis_period" json:"statis_period"`
	ReportPeriod string     `db:"report_period" json:"report_period"`
	StartTime    time.Time  `db:"start_time"    json:"start_time"`
	EndTime      *time.Time `db:"end_time"      json:"end_time,omitempty"`
	TaskStatus   TaskStatus `db:"task_status"   json:"task_status"`
	TaskResult   *string    `db:"task_result"   json:"task_result,omitempty"`
	Creator      string     `db:"creator"       json:"creator"`
	// TargetDeviceSNs 用户选中的目标设备 SN 列表（migration 000190）。
	// scheduler 在 task 开启时按这些 SN JOIN mr_device_mappings 展开为 cells。
	TargetDeviceSNs []string `db:"target_device_sns" json:"target_device_sns"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Progress 对应 mr_customize_task_progress 行。
type Progress struct {
	ID              uuid.UUID      `db:"id"               json:"id"`
	TaskID          uuid.UUID      `db:"task_id"          json:"task_id"`
	SmallCellCode   string         `db:"small_cell_code"  json:"small_cell_code"`
	SerialNumber    string         `db:"serial_number"    json:"serial_number"`
	HostName        *string        `db:"host_name"        json:"host_name,omitempty"`
	ProgressStatus  ProgressStatus `db:"progress_status"  json:"progress_status"`
	HealthStatus    HealthStatus   `db:"health_status"    json:"health_status"`
	FaultCode       *string        `db:"fault_code"       json:"fault_code,omitempty"`
	LastHeartbeat   *time.Time     `db:"last_heartbeat"   json:"last_heartbeat,omitempty"`
	MissedHeartbeat int            `db:"missed_heartbeat" json:"missed_heartbeat"`
	CreatedAt       time.Time      `db:"created_at"       json:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"       json:"updated_at"`
}

// CellTarget 是创建任务时的目标小站规格。
// 同时需要 cell code（写心跳 / SPV URL）与 serial_number（下发 SPV）。
type CellTarget struct {
	SmallCellCode string  `json:"small_cell_code" binding:"required"`
	SerialNumber  string  `json:"serial_number"   binding:"required"`
	HostName      *string `json:"host_name,omitempty"`
}

// CreateTaskInput 是 Service.Create 的入参（与 HTTP 请求体解耦）。
//
// 简化版（2026-05-25）：不再接 operator_code / note / targets / creator。
//   - operator_code / note 已彻底下线
//   - creator 由 handler 从 auth ctx 自动填，调用 Service 时由 caller 传入
//   - targets 由 scheduler 在 task 开启时从 mr_device_mappings 动态枚举
type CreateTaskInput struct {
	TaskName     string     `json:"task_name"     binding:"required,max=128"`
	MRType       string     `json:"mr_type"`       // 默认 "MRS,MRE,MRO"
	StatisPeriod string     `json:"statis_period"` // 默认 "5120"
	ReportPeriod string     `json:"report_period"` // 默认 "15"
	StartTime    time.Time  `json:"start_time"    binding:"required"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	// Creator 由 Service.Create 调用方（handler）从 auth ctx 自动注入，不走前端入参
	Creator string `json:"-"`
	// TargetDeviceSNs 用户选中的目标设备 SN 列表，非空（Service 强制校验）
	TargetDeviceSNs []string `json:"target_device_sns"`
}

// TaskListFilter 列表过滤参数。
//
// 简化版：不再支持 operator_code 过滤（无多租户隔离）。
type TaskListFilter struct {
	Status   *TaskStatus
	Keyword  *string
	Page     int
	PageSize int
	SortBy   string
	SortDir  string
}

// ProgressListFilter 进度列表过滤参数。
type ProgressListFilter struct {
	TaskID   uuid.UUID
	Status   *ProgressStatus
	Health   *HealthStatus
	Page     int
	PageSize int
}
