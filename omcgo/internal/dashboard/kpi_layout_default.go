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
		{"title":"dashboard.panel.traffic","metrics":["K900010015","K900010016","K900010040","K900010041"],"x":0,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.availability","metrics":["K900010076"],"x":6,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.utilization","metrics":["K900010014","K900010013"],"x":0,"y":8,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.accessibility","metrics":["K900010006","K900010002","K900010005","K900010029"],"x":6,"y":8,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.retainability","metrics":["K900010027"],"x":0,"y":16,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.mobility","metrics":["K900010017","K900010022","K900010021","K900010026"],"x":6,"y":16,"w":6,"h":8,"chartType":"line"}
	]}`,
	techNR: `{"panels":[
		{"title":"dashboard.panel.traffic","metrics":["KGNB0511","KGNB0510","KGNB0517","KGNB0516"],"x":0,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.utilization","metrics":["KGNB0506","KGNB0505"],"x":6,"y":0,"w":6,"h":8,"chartType":"line"}
	]}`,
	techGSM: `{"panels":[
		{"title":"dashboard.panel.accessibility","metrics":["KGSM0102"],"x":0,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.retainability","metrics":["KGSM0103"],"x":6,"y":0,"w":6,"h":8,"chartType":"line"},
		{"title":"dashboard.panel.mobility","metrics":["KGSM0101"],"x":0,"y":8,"w":12,"h":8,"chartType":"line"}
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
