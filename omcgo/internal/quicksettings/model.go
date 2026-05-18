// Package quicksettings 提供设备详情「快速设置」tab 的业务分组元数据。
//
// 数据源：data/quicksettings/{enb,gnb}.xml；启动期由 dictloader Loader 加载到进程内 Registry。
// REST 端点：GET /api/v1/quicksettings/groups?tech={lte|nr} 返回 Group 列表。
//
// 设计文档：docs/design/参数设置页-设计.md（T-0138 v0.7.x）
package quicksettings

// TechCode 标识快速设置分组所属的设备制式。
type TechCode string

const (
	TechLTE TechCode = "lte" // ENB（4G）
	TechNR  TechCode = "nr"  // GNB（5G NR）
)

// Group 是一组业务相关的参数（如「小区参数」「邻区列表」）。
//
// 单实例分组（multiInstance=false）的 Params 每项含 StandardPath，前端按 path 查 schema 渲染表单。
// 多实例分组（multiInstance=true）的 Params 每项含 Leaf 叶子名，前端拼接 ObjectPath + 实例号 + Leaf
// 形成完整 standardPath，并支持 AddObject / DeleteObject。
type Group struct {
	ID            string  `json:"id"`
	TitleZh       string  `json:"titleZh"`
	TitleEn       string  `json:"titleEn"`
	MultiInstance bool    `json:"multiInstance"`
	ObjectPath    string  `json:"objectPath,omitempty"` // 仅 multiInstance=true 时填，含 {i} 占位符
	Params        []Param `json:"params"`
}

// Param 是分组下的单个参数。
//
// 单实例分组用 StandardPath；多实例分组用 Leaf（与 Group.ObjectPath 拼接成完整路径）。
type Param struct {
	Name         string `json:"name"`
	TitleZh      string `json:"titleZh"`
	TitleEn      string `json:"titleEn"`
	StandardPath string `json:"standardPath,omitempty"`
	Leaf         string `json:"leaf,omitempty"`
}
