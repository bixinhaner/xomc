package dashboard

import (
	"context"

	"github.com/omcgo/omcgo/internal/pm/indicator"
	"go.uber.org/zap"
)

// kpi_definitions.go —— Dashboard KPI 动态定义端点（issue #213 Phase1）。
//
// 返回首页全部 KPI 的完整定义：symbolic key（前端可读、稳定）+ K 编号（落库 metric_path）
// + 中文名 + 单位，按制式 / Panel 分组。前端据此动态加载指标列表（取代 kpi-config.ts 里
// 硬编码的“key→后端名”清单），Panel 布局结构本轮仍沿用前端静态定义（Phase2/3 再做可配置）。
//
// 数据源 = 别名表（kpi_alias.go，symbolic↔K编号↔Panel 归类）叠加 indicator 库
// （perf_indicators_{enb,gsm,gnb} 的 cnName / unit）。indicator 库不可用时退化为仅别名表
// 静态元数据，端点仍可用（前端至少拿到 symbolic↔K编号↔Panel 结构）。

// KPIDefinitionItem 是单个 KPI 的完整定义（一条 symbolic key）。
type KPIDefinitionItem struct {
	// Key 是前端可读 symbolic key（kpi-config.ts 的 KPIConfig.key）。
	Key string `json:"key"`
	// KCode 是指标库编号（pm_metrics.metric_path 落库值）；none 项为空字符串。
	KCode string `json:"k_code"`
	// CnName 是中文名（来自 indicator 库 cn_name；缺则空）。
	CnName string `json:"cn_name"`
	// Unit 是单位（来自 indicator 库 unit_id，如 % / Mbps / MByte；缺则空）。
	Unit string `json:"unit"`
	// Panel 是 Dashboard Panel 归类（traffic / availability / ...）。
	Panel string `json:"panel"`
	// NeedsReview 标记该映射为待领域复核（medium 置信度或 none 硬缺口）。
	NeedsReview bool `json:"needs_review"`
	// Available 表示库内是否有对应 KPI（false = none 项，前端可据此置灰 / 隐藏）。
	Available bool `json:"available"`
}

// KPITechDefinitions 是单个制式的 KPI 定义集合（扁平 items，Panel 字段标注归属）。
type KPITechDefinitions struct {
	// Tech 是制式：lte / nr / gsm。
	Tech string `json:"tech"`
	// Items 是该制式全部 KPI 定义（按别名表声明顺序，与 Panel 归并）。
	Items []KPIDefinitionItem `json:"items"`
}

// KPIDefinitionsResponse 是 GET /dashboard/kpi/definitions 的响应体。
type KPIDefinitionsResponse struct {
	// Technologies 按制式分组（lte / nr / gsm），各含其 KPI 定义。
	Technologies []KPITechDefinitions `json:"technologies"`
	// Total 是全部 KPI 定义条数。
	Total int `json:"total"`
}

// GetKPIDefinitions 返回 Dashboard 首页全部 KPI 的动态定义（issue #213 Phase1）。
//
// 以别名表（kpi_alias.go）为骨架，按 K 编号批量从 indicator 库富化 cnName / unit。
// indicatorRepo 为 nil 或某制式查询失败时，对应条目退化为仅别名表静态字段（不致整端点失败）。
func (s *Service) GetKPIDefinitions(ctx context.Context) (*KPIDefinitionsResponse, error) {
	// 1. 按制式分桶 K 编号，一次性批量查 indicator 库（每个 DeviceType 一次 ListByIDs）。
	//    enrich[KCode] = (cnName, unit)。
	type meta struct {
		cnName string
		unit   string
	}
	enrich := make(map[string]meta)

	if s.indicatorRepo != nil {
		idsByDT := make(map[indicator.DeviceType][]string)
		for _, a := range dashboardKPIAliases {
			if a.KCode == "" {
				continue
			}
			idsByDT[a.DeviceType] = append(idsByDT[a.DeviceType], a.KCode)
		}
		for dt, ids := range idsByDT {
			rows, err := s.indicatorRepo.ListByIDs(ctx, dt, ids)
			if err != nil {
				// 单制式富化失败不致命：记日志，该制式条目退化为仅静态字段。
				s.logger.Warn("dashboard: enrich kpi definitions failed",
					zap.String("device_type", string(dt)), zap.Error(err))
				continue
			}
			for _, r := range rows {
				enrich[r.ID] = meta{
					cnName: derefStr(r.CnName),
					unit:   derefStr(r.UnitID),
				}
			}
		}
	}

	// 2. 按制式归并别名表为响应（保留别名表声明顺序）。
	byTech := map[string]*KPITechDefinitions{}
	order := []string{} // 制式首次出现顺序
	total := 0
	for _, a := range dashboardKPIAliases {
		td, ok := byTech[a.Tech]
		if !ok {
			td = &KPITechDefinitions{Tech: a.Tech}
			byTech[a.Tech] = td
			order = append(order, a.Tech)
		}
		m := enrich[a.KCode] // KCode 为空 / 未命中 → 零值（空 cnName / unit）
		td.Items = append(td.Items, KPIDefinitionItem{
			Key:         a.Symbolic,
			KCode:       a.KCode,
			CnName:      m.cnName,
			Unit:        m.unit,
			Panel:       a.Panel,
			NeedsReview: a.NeedsReview,
			Available:   a.KCode != "",
		})
		total++
	}

	resp := &KPIDefinitionsResponse{
		Technologies: make([]KPITechDefinitions, 0, len(order)),
		Total:        total,
	}
	for _, tech := range order {
		resp.Technologies = append(resp.Technologies, *byTech[tech])
	}
	return resp, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
