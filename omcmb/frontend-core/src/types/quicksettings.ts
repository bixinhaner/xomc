/**
 * 快速设置(Quick Settings)分组元数据类型。
 *
 * 数据来源:后端 GET /api/v1/quicksettings/groups?device_id={uuid}
 * 后端模块:omcgo/internal/quicksettings/(per-paramModel XML 加载,链路 device→product→paramModel)
 * 设计文档:docs/design/参数设置页-设计.md(T-0138)
 */

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
}

/**
 * 业务分组(如「小区参数」「邻区列表」)。
 */
export interface QuickSettingsGroup {
  id: string;
  titleZh: string;
  titleEn: string;
  multiInstance: boolean;
  /** 多实例分组的对象路径前缀(含 {i} 占位符)。 */
  objectPath?: string;
  params: QuickSettingsParam[];
}

/** 后端 GET /quicksettings/groups 响应。groups 为空数组表示该 paramModel 未配置 XML,前端应不显示 tab。 */
export interface QuickSettingsGroupsResponse {
  paramModel: string;
  groups: QuickSettingsGroup[];
}
