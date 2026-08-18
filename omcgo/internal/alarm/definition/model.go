// Package definition 提供 T-0098 告警库字典（设计 §3）。
//
// P1-06 阶段交付：
//   - model.go    XML 解析结构 + 领域类型
//   - loader.go   实现 dictloader.Loader — 启动期 8 个 ne_type XML 加载
//
// Phase 2/3 接力：Registry sync.Map + 接收路径 fallback + handler（设计 §3.3-§3.5）。
package definition

import (
	"encoding/xml"

	"github.com/google/uuid"
)

// AlarmDefinition 对应 alarm_definitions 表（设计 §3.2.2）。
type AlarmDefinition struct {
	ID              uuid.UUID
	Identifier      string
	NeType          string
	CnName          string
	EnName          string
	SeverityID      uuid.UUID
	EventType       *int
	CnProbableCause string
	EnProbableCause string
	IsShow          bool
	Description     string // 用户可编辑的描述/备注（不来自 XML）
}

// SeverityLevel 对应 alarm_severity_levels 表（设计 §3.2.1，4 行种子由 P1-04 写入）。
// JSON tag 必填：前端 alarmDefinitionApi.ts::BackendSeverityLevel 期望 snake_case
// 字段名（id/code/name/display_order），缺 tag 时 encoding/json 输出 PascalCase，
// 前端读 cn_name/en_name 全 undefined → 下拉为空（2026-05-29 用户报告）。
type SeverityLevel struct {
	ID           uuid.UUID `json:"id"`
	Code         int       `json:"code"`
	Name         string    `json:"name"`
	DisplayOrder int       `json:"display_order"`
}

// xmlAlarmModel 解析 8 个 ne_type 的 alarm xml（设计 §3.4）。
// DeviceType 带 omitempty 仅影响 marshal:手工新增导出(#268)未入库 deviceType,
// 省略该属性;解析侧 encoding/xml 忽略 omitempty,行为不变。
type xmlAlarmModel struct {
	XMLName    xml.Name   `xml:"alarmModel"`
	NeType     string     `xml:"neType,attr"`
	DeviceType string     `xml:"deviceType,attr,omitempty"`
	TotalCount int        `xml:"totalCount,attr"`
	Alarms     []xmlAlarm `xml:"alarms>alarm"`
}

type xmlAlarm struct {
	Identifier      string `xml:"identifier,attr"`
	CnName          string `xml:"cnName,attr"`
	EnName          string `xml:"enName,attr"`
	Severity        string `xml:"severity,attr"`
	EventType       string `xml:"eventType,attr"`
	CnProbableCause string `xml:"cnProbableCause,attr"`
	EnProbableCause string `xml:"enProbableCause,attr"`
	IsShow          string `xml:"isShow,attr"` // "Y" / "N"
}
