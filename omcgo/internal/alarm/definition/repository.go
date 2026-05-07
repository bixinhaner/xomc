package definition

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrUnknownIdentifier 表示告警 identifier 不在 alarm_definitions 表。
//
// 调用方据此触发 §3.5 fallback 决策（product.enable_unknown_alarm 三态）。
var ErrUnknownIdentifier = errors.New("alarm/definition: unknown identifier")

// Repository 抽象 AlarmDefinition Registry 所需的读访问。
//
// 写路径（XML 加载）由 P1-06 Loader 单独负责，不在本接口范围。
type Repository interface {
	// ListAll 返回所有 alarm_definitions 行 + severity_id → severity_code 反查。
	ListAll(ctx context.Context) ([]ResolvedDefinition, error)
	// ListSeverityLevels 返回 alarm_severity_levels 全部 4 行（启动期一次性）。
	ListSeverityLevels(ctx context.Context) ([]SeverityLevel, error)
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
