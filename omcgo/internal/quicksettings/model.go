// Package quicksettings 提供设备详情「快速设置」tab 的业务分组元数据。
//
// 数据源:data/quicksettings/<ParamModelName>.xml(如 BLQ.xml / BaiBNQ.xml);
// 启动期由 dictloader Loader 加载到进程内 Registry,按 paramModel name 索引。
// REST 端点:GET /api/v1/quicksettings/groups?device_id={uuid} 走
// device→product→paramModel name 链路,返回该设备所属 paramModel 的 Group 列表。
//
// 设计文档:docs/design/参数设置页-设计.md(T-0138 per-paramModel 架构)
package quicksettings

// Group 是一组业务相关的参数(如「小区参数」「邻区列表」)。
//
// 单实例分组(multiInstance=false)的 Params 每项含 StandardPath,前端按 path 查 schema 渲染表单。
// 多实例分组(multiInstance=true)的 Params 每项含 Leaf 叶子名,前端拼接 ObjectPath + 实例号 + Leaf
// 形成完整 standardPath,并支持 AddObject / DeleteObject。
//
// Style/ParentSelector 用于 BSC 等需要「实例选择器 + 多面板」UI 的场景:
//   - Style="table" (默认):MultiInstanceTable 行=实例,列=Params
//   - Style="form":InstanceSelectorForm 顶部按 ParentSelector 选实例,中部以 grid 渲染本组 Params
//   - Style="subtable":InstanceSelectorForm 嵌入子表(行=子实例,列=Params),ObjectPath 为 2 层 {i}.<sub>.{j}.
type Group struct {
	ID            string `json:"id"`
	TitleZh       string `json:"titleZh"`
	TitleEn       string `json:"titleEn"`
	MultiInstance bool   `json:"multiInstance"`
	ObjectPath    string `json:"objectPath,omitempty"` // 仅 multiInstance=true 时填,含 {i} 占位符
	// MaxInstances 多实例分组的最大实例数(仅 multiInstance=true 生效)。0 表示未指定。
	MaxInstances int `json:"maxInstances,omitempty"`
	// Style 渲染样式,可选 "table" / "form" / "subtable";空值由前端按 multiInstance 默认 "table"。
	Style string `json:"style,omitempty"`
	// ParentSelector 引用同 paramModel 内另一个 Group ID,UI 由该 group 的实例选择器统一控制。
	ParentSelector string  `json:"parentSelector,omitempty"`
	Params         []Param `json:"params"`
}

// Param 是分组下的单个参数。
//
// 单实例分组用 StandardPath;多实例分组用 Leaf(与 Group.ObjectPath 拼接成完整路径)。
// Type/Required/Min/Max/Hint/Readonly/EnumOptions/CheckboxOptions 为可选 UI 元数据,供 form/subtable
// 渲染器决定输入控件类型与校验规则;table 风格不使用这些字段(由 schema 推断)。
type Param struct {
	Name             string       `json:"name"`
	TitleZh          string       `json:"titleZh"`
	TitleEn          string       `json:"titleEn"`
	StandardPath     string       `json:"standardPath,omitempty"`
	Leaf             string       `json:"leaf,omitempty"`
	Type             string       `json:"type,omitempty"` // string/int/enum/multiCheckbox
	Required         bool         `json:"required,omitempty"`
	Readonly         bool         `json:"readonly,omitempty"`
	Hint             string       `json:"hint,omitempty"`
	MinValue         *int64       `json:"minValue,omitempty"`
	MaxValue         *int64       `json:"maxValue,omitempty"`
	EnumOptions      []EnumOption `json:"enumOptions,omitempty"`
	CheckboxOptions  []string     `json:"checkboxOptions,omitempty"`
	// ExtraInfoPath 可选:用于在该参数下方以小字方式展示另一个只读参数的当前值
	// (如 NR PowerModify 下方提示 Device.DeviceInfo.SupportedPowerRange = "24,30" → "[24 ~ 30]")。
	// 前端在渲染时按该路径拉取 schema/currentValue 并按 [lo ~ hi] 格式展示。
	ExtraInfoPath string `json:"extraInfoPath,omitempty"`
	// Unit 可选:参数单位后缀 (如 "dBm" / "MHz")。
	// 前端渲染 label 时以 "(unit)" 形式拼在 ExtraInfo 之后,以保证范围提示位于单位之前。
	Unit string `json:"unit,omitempty"`
	// HideRangeHint 可选:为 true 时前端不在 label 后面展示 schema 推导的
	// 取值范围提示 (如 [0 ~ 2199])。适用于字典范围与业务含义不一致的场景。
	HideRangeHint bool `json:"hideRangeHint,omitempty"`
}

// EnumOption 单个枚举值（value 是上送 TR069 的字符串，label 是 UI 显示）。
type EnumOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
