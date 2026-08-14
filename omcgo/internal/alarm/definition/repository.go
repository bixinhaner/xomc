package definition

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrUnknownIdentifier 表示告警 identifier 不在 alarm_definitions 表。
//
// 调用方据此触发 §3.5 fallback 决策（product.enable_unknown_alarm 三态）。
var ErrUnknownIdentifier = errors.New("alarm/definition: unknown identifier")

// Repository 抽象 AlarmDefinition Registry 所需的读访问。
//
// 写路径（XML 加载）由 P1-06 Loader 单独负责；P3-04 handler 用的 CRUD
// 写路径见 WriteRepository。
type Repository interface {
	// ListAll 返回所有 alarm_definitions 行 + severity_id → severity_code 反查。
	ListAll(ctx context.Context) ([]ResolvedDefinition, error)
	// ListSeverityLevels 返回 alarm_severity_levels 全部 4 行（启动期一次性）。
	ListSeverityLevels(ctx context.Context) ([]SeverityLevel, error)
}

// WriteRepository 是 P3-04 handler CRUD 用的写路径接口。
//
// 与 Repository 分开：Registry 只读不写；Loader 走 UPSERT；handler 走精确语义。
type WriteRepository interface {
	Repository

	// ListWithFilter 支持 ne_type / severity_code / keyword 过滤 + 分页。
	// keyword 在 identifier / cn_name / en_name 上做 ILIKE。
	ListWithFilter(ctx context.Context, f ListFilter) ([]ResolvedDefinition, int, error)

	// GetByIdentifier 单条详情；不存在 → (nil, ErrUnknownIdentifier)。
	GetByIdentifier(ctx context.Context, identifier string) (*ResolvedDefinition, error)

	// Create 新建告警定义；severity_code 用于反查 severity_id。
	Create(ctx context.Context, in CreateInput) (*ResolvedDefinition, error)

	// Update 局部更新；nil 字段表示不变。
	Update(ctx context.Context, identifier string, in UpdateInput) (*ResolvedDefinition, error)

	// Delete 按 identifier 删除；返回是否真正删除（false=不存在）。
	Delete(ctx context.Context, identifier string) (bool, error)

	// UnknownStats 聚合 alarms_active 中 is_unknown=true 的 identifier 频次。
	// productID 为 nil 时不按 product 过滤；days 限制 raised_at 时间窗口。
	UnknownStats(ctx context.Context, productID *uuid.UUID, days int) ([]UnknownAlarmStat, error)

	// ListNeTypes 聚合 alarm_definitions 按 ne_type + loaded_from 双键统计。
	// T-0179 drill-down 一级视图;每个 ne_type 一行,含告警总数与各严重级计数。
	ListNeTypes(ctx context.Context) ([]NeTypeStat, error)

	// DeleteOrphansSince 删除 updated_at < since 的 alarm_definitions 行(reload
	// destructive 语义对齐 parammodel)。返回受影响行数。
	// 用法:reload 入口在 ReloadOne 之前记录 startedAt,Loader UPSERT 完成后
	// 调本方法把"本次未被 UPSERT 触达"的旧定义清掉(BEFORE UPDATE 触发器
	// 保证 UPSERT 写 updated_at = NOW(),所以孤儿的 updated_at 必然 < startedAt)。
	DeleteOrphansSince(ctx context.Context, since time.Time) (int64, error)
}

// ListFilter 控制 ListWithFilter 行为。
type ListFilter struct {
	NeType       *string
	LoadedFrom   *string
	SeverityCode *int
	Keyword      *string
	Page         int
	PageSize     int
}

// CreateInput 是 Create 入参。
type CreateInput struct {
	Identifier      string
	NeType          string
	CnName          string
	EnName          string
	SeverityCode    int // 31001-31004
	EventType       *int
	CnProbableCause string
	EnProbableCause string
	CnSuggestion    string
	EnSuggestion    string
	IsShow          bool
	Description     string
}

// UpdateInput 是 Update 入参；nil 字段保留原值。
type UpdateInput struct {
	NeType          *string
	CnName          *string
	EnName          *string
	SeverityCode    *int
	EventType       *int
	CnProbableCause *string
	EnProbableCause *string
	CnSuggestion    *string
	EnSuggestion    *string
	IsShow          *bool
	Description     *string
}

// UnknownAlarmStat 是 unknown-stats 端点的单条结果。
type UnknownAlarmStat struct {
	Identifier string `json:"identifier"`
	Count      int    `json:"count"`
	LastSeenAt string `json:"last_seen_at"`
	NeType     string `json:"ne_type,omitempty"`
}

// NeTypeStat 是 ne-types 聚合端点的单条结果(T-0179 drill-down 一级视图)。
//
// LoadedFrom 为空表示该 ne_type 下存在历史数据未回填(Loader 未重跑)。前端按
// (ne_type, loaded_from) 双键展示;Total 是按 ne_type 单键的合计。
type NeTypeStat struct {
	NeType      string `json:"ne_type"`
	LoadedFrom  string `json:"loaded_from"`
	Source      string `json:"source"`    // ClassifySource(loaded_from) 派生:builtin/custom/unknown
	Deletable   bool   `json:"deletable"` // IsDeletable(loaded_from) 派生:仅 custom 可删
	Total       int    `json:"total"`
	CriticalCnt int    `json:"critical_cnt"`
	MajorCnt    int    `json:"major_cnt"`
	MinorCnt    int    `json:"minor_cnt"`
	WarningCnt  int    `json:"warning_cnt"`
}

// ResolvedDefinition 是 AlarmDefinition + severity 反查后的合成结构，供 Registry 直接缓存。
//
// 设计取舍：alarm_definitions 表外键 severity_id → alarm_severity_levels(id)；
// 业务消费需要的是 severity_code（31001-31004）。Registry 在加载时一次性 join，
// 把代码下发到内存，避免每次 Lookup 再去查 severity 表。
type ResolvedDefinition struct {
	AlarmDefinition
	SeverityCode int
	SeverityName string
}

// ProductSnapshot 是 Receiver fallback 决策所需的最小 product 视图。
//
// 抽出独立类型避免 alarm 包反向依赖 product 包，保持单向依赖（receiver → 抽象 →
// 适配器，适配器内引 product.Registry）。
type ProductSnapshot struct {
	ID                 uuid.UUID
	Name               string
	EnableUnknownAlarm bool
}

// ProductResolver 由消费方实现：根据 device 的 ProductClass 反查到 product 装配件。
//
// 生产环境由 product.Registry.MatchProductClass + 适配器满足；测试可注入 stub。
type ProductResolver interface {
	ResolveByProductClass(ctx context.Context, productClass string) (*ProductSnapshot, error)
}
