/**
 * 快速设置（Quick Settings）分组元数据类型。
 *
 * 数据来源：后端 GET /api/v1/quicksettings/groups?tech={lte|nr}
 * 后端模块：omcgo/internal/quicksettings/（XML 启动期加载到进程内 Registry）
 * 设计文档：docs/design/参数设置页-设计.md（T-0138）
 */

export type TechCode = 'lte' | 'nr';

/**
 * 单个参数的元数据。
 *
 * 单实例分组（multiInstance=false）：使用 standardPath。
 * 多实例分组（multiInstance=true）：使用 leaf 叶子名，与 Group.objectPath 拼接。
 */
export interface QuickSettingsParam {
  name: string;
  titleZh: string;
  titleEn: string;
  standardPath?: string;
  leaf?: string;
}

/**
 * 业务分组（如「小区参数」「邻区列表」）。
 */
export interface QuickSettingsGroup {
  id: string;
  titleZh: string;
  titleEn: string;
  multiInstance: boolean;
  /** 多实例分组的对象路径前缀（含 {i} 占位符）。 */
  objectPath?: string;
  params: QuickSettingsParam[];
}

export interface QuickSettingsGroupsResponse {
  groups: QuickSettingsGroup[];
}
