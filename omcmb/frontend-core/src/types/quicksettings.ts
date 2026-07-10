/**
 * 快速设置(Quick Settings)分组元数据类型。
 *
 * 数据来源:后端 GET /api/v1/quicksettings/groups?device_id={uuid}
 * 后端模块:omcgo/internal/quicksettings/(per-paramModel XML 加载,链路 device→product→paramModel)
 * 设计文档:docs/design/参数设置页-设计.md(T-0138)
 */

/**
 * 枚举选项(用于 Param.type === 'enum')。
 */
export interface QuickSettingsEnumOption {
  value: string;
  label: string;
}

/**
 * 单个参数的元数据。
 *
 * 单实例分组(multiInstance=false):使用 standardPath。
 * 多实例分组(multiInstance=true):使用 leaf 叶子名,与 Group.objectPath 拼接。
 */
export interface QuickSettingsParam {
  name: string;
  titleZh: string;
  titleEn: string;
  standardPath?: string;
  leaf?: string;
  /** 控件类型:string | int | enum | multiCheckbox。未指定时按 string 处理。 */
  type?: string;
  required?: boolean;
  readonly?: boolean;
  hint?: string;
  defaultValue?: string;
  minValue?: number;
  maxValue?: number;
  enumOptions?: QuickSettingsEnumOption[];
  checkboxOptions?: string[];
  /**
   * 可选:在该字段下方以小字展示另一个只读参数的当前值。
   * 前端按该路径拉取 schema/currentValue,按 [lo ~ hi] 格式展示
   * (值若是 "24,30" / "24~30" / "24-30" 则拆成两端,其它情况按原值包裹方括号)。
   */
  extraInfoPath?: string;
  /** 可选:参数单位后缀 (如 "dBm")。前端渲染 label 时以 "(unit)" 拼在 extraInfo 之后。 */
  unit?: string;
  /**
   * 可选:为 true 时前端不在 label 后面展示 schema 推导的取值范围提示
   * (如 [0 ~ 2199])。适用于字典范围与业务含义不一致的场景。
   */
  hideRangeHint?: boolean;
}

/**
 * 业务分组(如「小区参数」「邻区列表」)。
 */
export interface QuickSettingsGroup {
  id: string;
  titleZh: string;
  titleEn: string;
  multiInstance: boolean;
  /** 多实例分组的对象路径前缀(含 {i} 占位符,二级子表使用 {j})。 */
  objectPath?: string;
  /** 多实例分组的最大实例数(仅 multiInstance=true 生效);未配置时为 undefined。 */
  maxInstances?: number;
  /** 渲染样式:table(顶层多实例选择器) | form(表单) | subtable(二级子表)。 */
  style?: 'table' | 'form' | 'subtable' | string;
  /** 父级 group id。当本组依赖另一个 multiInstance 分组的实例选择时填写。 */
  parentSelector?: string;
  params: QuickSettingsParam[];
}

/** 后端 GET /quicksettings/groups 响应。groups 为空数组表示该 paramModel 未配置 XML,前端应不显示 tab。 */
export interface QuickSettingsGroupsResponse {
  paramModel: string;
  groups: QuickSettingsGroup[];
}
