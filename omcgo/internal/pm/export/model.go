// Package export 实现 KPI 数据导出（KPI-EXPORT）。
//
// 设计文档：~/Documents/notes/PM功能设计/kpi-export-design-20260604.md
//
// 导出走异步任务：
//  1. 建任务（POST /pm/exports）→ 落表 status=pending + 入队 async_jobs（job_type=pm_kpi_export）
//  2. worker 抢 job → 取数 → 流式写 CSV → 传对象存储 → 回填 file_path/file_size/row_count，status=succeeded
//     （T1 本阶段只建 worker 桩：认领 job→标 running→暂不真生成，真生成留 T2）
//  3. 任务管理 Tab 看状态；文件管理 Tab（只列已成功 + 文件就绪）下载
//
// 一张表 pm_kpi_export_tasks 喂两个视图，区别在过滤条件（文件管理只筛 succeeded 且 file_path 非空）。
package export

import (
	"time"

	"github.com/google/uuid"
)

// SourceType 导出来源。
//   - dashboard：仪表盘曲线导出（带当前筛选，全量不受 5000 行上限约束）
//   - device_view：设备性能查看导出（复用 dashboard-like 设备维度取数）
//   - kpi_query：指标查询页导出（复用 dashboard 取数，CSV 固定列贴近查询页表格）
//   - adhoc：adhoc 聚合任务结果导出
type SourceType string

const (
	SourceDashboard  SourceType = "dashboard"
	SourceDeviceView SourceType = "device_view"
	SourceKpiQuery   SourceType = "kpi_query"
	SourceAdhoc      SourceType = "adhoc"
)

// Valid 校验来源是否受支持。
func (s SourceType) Valid() bool {
	return s == SourceDashboard || s == SourceDeviceView || s == SourceKpiQuery || s == SourceAdhoc
}

// Status 导出任务生命周期状态。
type Status string

const (
	StatusPending   Status = "pending"   // 刚建，未被 worker 抢
	StatusRunning   Status = "running"   // worker 处理中
	StatusSucceeded Status = "succeeded" // 终态：文件就绪
	StatusFailed    Status = "failed"    // 终态：失败（error 落库）
)

// FormatCSV 默认导出格式（T1 只支持 CSV）。
const FormatCSV = "csv"

// JobType 是入队 async_jobs 的 job_type，worker 注册对应处理器认领。
const JobType = "pm_kpi_export"

// Task 是 pm_kpi_export_tasks 表一行的 Go 域模型。
type Task struct {
	ID         uuid.UUID
	TaskName   string
	SourceType SourceType
	Params     []byte // 导出范围参数（JSONB 原文）
	Format     string
	Status     Status
	RowCount   int64
	Bucket     string
	FilePath   string
	FileSize   int64
	Error      string
	CreateUser string
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	ExpireAt   *time.Time
}

// CreateRequest 是 Service.Create / handler.Create 的输入（不含 ID / 时间戳 / 状态）。
type CreateRequest struct {
	TaskName   string
	SourceType SourceType
	Params     []byte // JSON 原文（导出范围参数）
	CreateUser string
}

// ListFilter 是 Repository.List 的过滤条件。
type ListFilter struct {
	SourceType *SourceType
	Status     *Status
	// OnlyReady 为 true 时只返回 status='succeeded' 且 file_path 非空的行（文件管理 Tab 视图）。
	OnlyReady bool
	Limit     int
	Offset    int
}
