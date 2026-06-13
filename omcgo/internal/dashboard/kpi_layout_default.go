package dashboard

import "encoding/json"

// kpi_layout_default.go —— 内置默认 KPI 布局（issue #213 S1 回退值）。
//
// 当全局布局表无该制式行时（首次部署前 / 被清空 / 测试无库），读接口回退到这套默认布局，
// 保证首页永不空白。本套与 seed/000002 灌入的初始三行**逐图等价**（同一份前端写死布局的迁移），
// 是同一来源的两处落点（seed 进库 + Go 内置回退），需保持同步。

// defaultLayoutJSON 是三制式默认布局体（与 seed 000002 等价）。
var defaultLayoutJSON = map[string]string{
	techLTE: `{"panels":[
		{"title":"dashboard.panel.traffic","metrics":["LTE_PDCP_VOLUME_DL","LTE_PDCP_VOLUME_UL","LTE_PDCP_RATE_DL","LTE_PDCP_RATE_UL"],"x":0,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.availability","metrics":["LTE_CELL_AVAILABLE"],"x":6,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.utilization","metrics":["LTE_PRB_UTIL_DL","LTE_PRB_UTIL_UL"],"x":0,"y":8,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.accessibility","metrics":["WIRELESS_SETUP_SR","RRC_CONN_SETUP_SR","ERAB_SETUP_SR","CSFB_SR"],"x":6,"y":8,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.retainability","metrics":["ERAB_DROP_RATE"],"x":0,"y":16,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.mobility","metrics":["HO_INTRA_ENB_OUT_SR","HO_INTRA_ENB_IN_SR","HO_INTER_ENB_OUT_SR","HO_INTER_ENB_IN_SR"],"x":6,"y":16,"w":6,"h":8,"chartType":"line"}
	]}`,
	techNR: `{"panels":[
		{"title":"dashboard.panel.traffic","metrics":["NR_PDCP_VOLUME_DL","NR_PDCP_VOLUME_UL","NR_PDCP_RATE_DL","NR_PDCP_RATE_UL"],"x":0,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.utilization","metrics":["NR_PRB_UTIL_DL","NR_PRB_UTIL_UL"],"x":6,"y":0,"w":6,"h":8,"chartType":"line"}
	]}`,
	techGSM: `{"panels":[
		{"title":"dashboard.panel.accessibility","metrics":["GSM_CALL_SETUP_SR"],"x":0,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.retainability","metrics":["GSM_CALL_DROP_RATE"],"x":6,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.mobility","metrics":["GSM_HO_SR"],"x":0,"y":8,"w":12,"h":8,"chartType":"line"}
	]}`,
}

// defaultKPILayout 返回某制式的内置默认布局（回退用）。tech 非法时返回空 panels。
func defaultKPILayout(tech string) *KPILayout {
	raw, ok := defaultLayoutJSON[tech]
	if !ok {
		return &KPILayout{Tech: tech, Layout: json.RawMessage(`{"panels":[]}`)}
	}
	// 压缩成紧凑 JSON（去掉源码里的缩进/换行），保证 Layout 是合法紧凑 JSONB。
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return &KPILayout{Tech: tech, Layout: json.RawMessage(`{"panels":[]}`)}
	}
	compact, err := json.Marshal(v)
	if err != nil {
		return &KPILayout{Tech: tech, Layout: json.RawMessage(`{"panels":[]}`)}
	}
	return &KPILayout{Tech: tech, Layout: json.RawMessage(compact)}
}
