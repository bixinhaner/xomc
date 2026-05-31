// Package adhoc 实现 T-0164-P7 / G7 自定义聚合任务（oneshot + continuous）。
//
// 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.7
// 实施 plan：docs/project/plan-T-0164-P7-adhoc-aggregation.md
//
// 与 G5 cron 聚合的区别：
//   - G5 自然桶按整点对齐自动跑（hourly/daily/weekly/monthly），结果落 pm_metrics_*
//   - G7 adhoc 用户即兴选 N 设备 + N metric + N 粒度 + 任意时窗，结果落 pm_adhoc_aggregation_results
//   - G5 是产品默认能力；G7 是探查 / 报表 / 临时聚合 / 持续监控 自定义入口
//
// 与 G8 asyncjob 框架的关系：**不进** G8（设计 §4.7 锁定）。复用 pm_tasks per-module 表 +
// LockNextPending 独立实现，避免和 G5 等系统级 cron 在同一 queue 抢资源。
package adhoc

import (
	"time"

	"github.com/google/uuid"
)

// Mode 任务执行模式。
//   - oneshot：执行一次 → 归档（succeeded/failed），不再跑
//   - continuous：按 cron_expr 重复执行，每次完成后状态切回 scheduled 等下次 tick
type Mode string

const (
	ModeOneshot    Mode = "oneshot"
	ModeContinuous Mode = "continuous"
)

// Status 任务生命周期状态。
type Status string

const (
	StatusPending   Status = "pending"   // 刚创建，未被 worker 抢
	StatusRunning   Status = "running"   // worker 持锁中（has lock_owner + started_at）
	StatusSucceeded Status = "succeeded" // oneshot 终态
	StatusFailed    Status = "failed"    // oneshot 终态
	StatusCanceled  Status = "canceled"  // 用户主动 cancel
	StatusScheduled Status = "scheduled" // continuous 等下次 tick
)

// TaskSubtype 标记 pm_tasks 行属于 G7 adhoc 任务（区别于老 extraction/report 等）。
const TaskSubtype = "adhoc_aggregation"

// Dimension 标识 adhoc 任务聚合维度。
//
//   - 'device'：每设备一条结果（既有行为，默认值）
//   - 'aggregate_group'：N 个 SN 临时组聚合成一条（按时间桶 + LDN GROUP BY，不 GROUP BY device_sn）
//   - 'product'：按设备所属产品 (devices.product_id) 分组（T-0182）
//   - 'band'：按小区频段分组（T-0182 仅入枚举，聚合实现见 T-0183）
//   - 'network'：全网汇总成一条总线（T-0184，仅制式过滤，无实体键）
//   - 'device_group'：按设备组分组（T-0184，复用 G5 设备组预聚合 pm_group_metrics_*）
type Dimension string

const (
	DimensionDevice         Dimension = "device"
	DimensionAggregateGroup Dimension = "aggregate_group"
	DimensionProduct        Dimension = "product"
	DimensionBand           Dimension = "band"
	DimensionNetwork        Dimension = "network"
	DimensionDeviceGroup    Dimension = "device_group"
)

// Task 是 pm_tasks 表中 task_subtype='adhoc_aggregation' 行的 Go 域模型。
//
// 与现有 pm.PerformanceTask 共享表但走独立 Repository（避免破坏 pm.TaskRepository 既有 8 个调用方）。
type Task struct {
	ID            uuid.UUID
	Name          string
	Mode          Mode
	CronExpr      *string // continuous 必填，oneshot 为 nil
	DeviceSNs     []string
	MetricPaths   []string
	Granularities []string  // 多粒度多选（如 ['hourly','daily']）
	// ObjectLDNs T-0193：小区/PLMN 白名单（完整 object_ldn 字符串）。空 = 不过滤 = 全小区。
	// 纯查看级过滤，仅 device/aggregate_group 维度承载，不改聚合/落库。
	ObjectLDNs    []string
	WindowStart   time.Time // 单次执行的源数据时窗起
	WindowEnd     time.Time // 源数据时窗止
	Dimension     Dimension // 维度，默认 'device'
	Technology    string    // T-0182：任务制式（lte/nr/gsm），空=不限制式；建后不可改
	IsBuiltin     bool      // T-0182：内置任务标记（T-0184 预置 12 个内置任务）
	ExpireDays    int       // T-0182：非持续型任务过期天数（默认 60，约束任务定义层）
	Status        Status
	Progress      int    // 0-100
	Creator       string // user_id 字符串或用户名（与 pm_tasks 既有 creator 列对齐）
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// 运行期字段（DB 中通过 pm_tasks 其它列承载）
	StartedAt   *time.Time
	FinishedAt  *time.Time
	ErrorMsg    string
	LockOwner   *string
}

// CreateRequest 是 Service.Create / handler.Create 的输入（不含 ID / 时间戳 / 状态）。
type CreateRequest struct {
	Name          string
	Mode          Mode
	CronExpr      *string
	DeviceSNs     []string
	MetricPaths   []string
	Granularities []string
	// ObjectLDNs T-0193：小区/PLMN 白名单（完整 object_ldn 字符串）。空 = 不过滤 = 全小区。
	ObjectLDNs    []string
	WindowStart   time.Time
	WindowEnd     time.Time
	Dimension     Dimension // 默认 device
	Technology    string    // T-0182：lte/nr/gsm，空=不限
	IsBuiltin     bool      // T-0182：内置任务标记
	ExpireDays    int       // T-0182：非持续型过期天数，<=0 时 repository 兜底为 60
	Creator       string
}

// ListFilter 是 Repository.List 的过滤条件。
type ListFilter struct {
	Mode      *Mode
	Status    *Status
	Creator   string
	IsBuiltin *bool // T-0184：内置任务过滤（前端分内置区/自建区）；nil=不过滤
	Limit     int
	Offset    int
}

// TaskRun 是 pm_adhoc_task_runs 表一行的 Go 域模型（T-0186）。
//
// 每次 worker 执行一个 adhoc 任务落一行：跑前 INSERT(status=running)，跑后 UPDATE 终态。
// 与 pm_tasks 的 status 循环解耦——pm_tasks 只反映"任务当前状态"，本表保留逐次执行历史。
type TaskRun struct {
	ID          uuid.UUID
	TaskID      uuid.UUID
	RunSeq      int
	Granularity string
	Dimension   string
	WindowStart *time.Time
	WindowEnd   *time.Time
	Status      Status
	QueuedAt    *time.Time
	StartedAt   time.Time
	FinishedAt  *time.Time
	Error       string
	RowsTotal   int
	CreatedAt   time.Time
}

// ResultRow 是 pm_adhoc_aggregation_results 表的一行（写入用）。
type ResultRow struct {
	TaskID      uuid.UUID
	DeviceOUI   string
	DeviceSN    string
	// ProductID 是 product 维度聚合的分组键（T-0182-fix）。
	// device / aggregate_group 维度为 uuid.Nil（落库 NULL）；product 维度填 devices.product_id。
	ProductID   uuid.UUID
	MetricPath  string
	MetricType  string // 'counter' / 'kpi'
	MetricValue float64
	StatisType  *string
	Granularity string
	Time        time.Time
	StartTime   time.Time
	EndTime     time.Time
	ObjectLDN   *string
	Extra       map[string]any
}
