// Package product 提供 T-0098 产品装配件字典（设计 §4）。
//
// P1-06 阶段交付：
//   - model.go    XML 解析结构 + 领域类型
//   - loader.go   实现 dictloader.Loader — 启动期 products.xml 加载 + 三引用校验
//
// Phase 2/3 接力：ProductRegistry / handler / REST API（设计 §4.3 / §4.4）。
package product

import (
	"encoding/xml"

	"github.com/google/uuid"
)

// ── Domain types ──────────────────────────────────────────────────────

// Product 对应 products 表（设计 §4.2.1）。
type Product struct {
	ID                  uuid.UUID
	Name                string
	Vendor              string
	Tech                string
	RadioModes          string
	Description         string
	ParamModelID        *uuid.UUID
	IndicatorDeviceType string
	IndicatorPlatform   string
	AlarmNeType         string
	EnableFileType11    bool
	DeviceAttrsOverride map[string]any // JSONB
	EnableUnknownAlarm  bool
}

// ProductClassPattern 对应 product_class_patterns 表（设计 §4.2.2）。
type ProductClassPattern struct {
	ID           uuid.UUID
	ProductID    uuid.UUID
	ProductClass string
	SortOrder    int
	IsActive     bool
}

// MatchResult 是 ProductRegistry.MatchProductClass 的命中结果（设计 §4.3 核心类型）。
type MatchResult struct {
	Product        *Product
	MatchedPattern string
	GlobalOrder    int
}

// ValidationReport 汇总 ProductRegistry.ValidateReferences 的三引用校验结果（设计 §4.5）。
//
// 语义对齐 P1-06 product Loader：paramModel.id 由 PG FK 强一致已校验；
// indicator_platform / alarm_ne_type 是软引用，本报告对未命中条目仅 WARN，
// 不阻塞 Registry 启动。BlockedProducts 用于未来 handler 层（P3-01）保存校验。
type ValidationReport struct {
	TotalProducts   int
	BlockedProducts []ValidationIssue // 硬阻断（保留，本任务暂不产出，留给 P3-01 handler）
	WarnedProducts  []ValidationIssue
}

// ValidationIssue 单条引用校验问题。
type ValidationIssue struct {
	ProductID   uuid.UUID
	ProductName string
	Field       string // "indicator_platform" / "alarm_ne_type"
	Reason      string
}

// ── XML 解析结构（设计 §4.5）────────────────────────────────────────

type xmlProducts struct {
	XMLName        xml.Name     `xml:"products"`
	TotalProducts  int          `xml:"totalProducts,attr"`
	TotalPatterns  int          `xml:"totalPatterns,attr"`
	GeneratedAt    string       `xml:"generatedAt,attr"`
	Products       []xmlProduct `xml:"product"`
}

type xmlProduct struct {
	Name                string                  `xml:"name,attr"`
	Vendor              string                  `xml:"vendor,attr"`
	Tech                string                  `xml:"tech,attr"`
	RadioModes          string                  `xml:"radioModes,attr"`
	Description         string                  `xml:"description"`
	ParamModel          string                  `xml:"paramModel"`
	EnableFileType11    string                  `xml:"enableFileType11"`
	DeviceAttrsOverride xmlDeviceAttrsOverride  `xml:"deviceAttrsOverride"`
	Indicator           xmlIndicatorRef         `xml:"indicator"`
	Alarm               xmlAlarmRef             `xml:"alarm"`
	Patterns            []xmlPattern            `xml:"patterns>pattern"`
}

type xmlDeviceAttrsOverride struct {
	Access        string `xml:"access,attr"`
	MinValue      string `xml:"min_value,attr"`
	MaxValue      string `xml:"max_value,attr"`
	ChangeApplies string `xml:"change_applies,attr"`
	DataType      string `xml:"data_type,attr"`
}

type xmlIndicatorRef struct {
	DeviceType string `xml:"deviceType,attr"`
	Platform   string `xml:"platform,attr"`
}

type xmlAlarmRef struct {
	NeType              string `xml:"neType,attr"`
	EnableUnknownAlarm  string `xml:"enableUnknownAlarm,attr"` // 缺省 → false（设计 §4.5）
}

type xmlPattern struct {
	GlobalOrder int    `xml:"globalOrder,attr"`
	Value       string `xml:",chardata"`
}
