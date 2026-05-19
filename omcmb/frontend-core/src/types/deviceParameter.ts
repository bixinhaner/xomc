/** 设备参数 — TR069 参数树节点 */
export interface DeviceParameter {
  id: string;
  deviceId: string;
  parameterPath: string;
  parameterValue: string;
  parameterType: ParameterType;
  writable: boolean;
  lastUpdatedAt: string;
}

export type ParameterType = 'string' | 'int' | 'unsignedInt' | 'boolean' | 'dateTime' | 'base64' | 'hexBinary' | 'object';

/** 参数约束（来自数据模型） */
export interface ParameterConstraints {
  minValue?: number;
  maxValue?: number;
  enumValues?: string[];
  /** T-0158: 与 enumValues 一一对应的 UI 显示标签（前端显示值 ≠ 下发值场景） */
  enumLabels?: string[];
  pattern?: string;
  maxLength?: number;
  minLength?: number;
}

/** 参数树节点 — 后端返回的树形结构 */
export interface ParameterTreeNode {
  name: string;
  fullPath: string;
  isObject: boolean;
  parameterType?: ParameterType;
  parameterValue?: string;
  writable?: boolean;
  lastUpdatedAt?: string;
  children?: ParameterTreeNode[];
  // Model metadata (enriched from data model)
  description?: string;
  multiInstance?: boolean;
  maxInstances?: number;
  minInstances?: number;
  instanceCount?: number;
  canAdd?: boolean;
  canDelete?: boolean;
  changeApplies?: string;
  defaultValue?: string;
  constraints?: ParameterConstraints;
}

/** 参数同步状态 */
export interface ParameterSyncStatus {
  deviceId: string;
  status: 'idle' | 'syncing' | 'completed' | 'failed';
  totalBatches: number;
  completedBatches: number;
  totalParameters: number;
  syncedParameters: number;
  percentage: number;
  startedAt?: string;
  completedAt?: string;
  error?: string;
}

/** 参数查询过滤条件 */
export interface ParameterFilter {
  search?: string;
  writable?: boolean;
  parameterType?: ParameterType;
}

/** 参数更新请求 */
export interface ParameterUpdateRequest {
  parameterPath: string;
  parameterValue: string;
  parameterType: ParameterType;
}

/** 参数同步选项 */
export interface ParameterSyncOptions {
  nextLevel?: boolean;
}

/** 参数 Schema 项（含模型元数据和当前值） */
export interface ParameterSchemaItem {
  path: string;
  type: string;
  writable: boolean;
  description?: string;
  defaultValue?: string;
  notify?: string;
  forcedInform?: boolean;
  changeApplies?: string;
  category?: string;
  isList?: boolean;
  constraints?: ParameterConstraints;
  currentValue?: string | null;
  lastSyncedAt?: string;
}

/** 多实例对象 Schema 项 */
export interface ObjectSchemaItem {
  path: string;
  access: string;
  maxInstances: number;
  minInstances: number;
  currentInstances: number[];
  canAdd: boolean;
  canDeleteAny: boolean;
  isList: boolean;
}

/** 参数 Schema 响应 */
export interface ParameterSchemaResponse {
  parameters: ParameterSchemaItem[];
  objects: ObjectSchemaItem[];
  total: number;
}

/** 子参数（getDirectChildren 返回的叶子参数，含模型元数据） */
export interface ChildParameter {
  parameterPath: string;
  parameterValue: string;
  parameterType: ParameterType;
  writable: boolean;
  lastUpdatedAt: string;
  description?: string;
  defaultValue?: string;
  changeApplies?: string;
  constraints?: ParameterConstraints;
}

/** 子对象摘要（某前缀下的直接子文件夹） */
export interface SubObjectSummary {
  name: string;
  fullPath: string;
  childCount: number;
}

/** getDirectChildren 组合响应（叶子参数 + 子对象摘要） */
export interface DirectChildrenResponse {
  items: ChildParameter[];
  subObjects: SubObjectSummary[];
  total: number;
  page: number;
  pageSize: number;
}

/** 参数更新响应 */
export interface ParameterUpdateResponse {
  message: string;
  parameters: number;
  rebootRequired: boolean;
  /** T-0146:后端任务 ID,前端用 useTaskStatus 轮询真实 CPE 应答状态 */
  taskId?: string;
}
