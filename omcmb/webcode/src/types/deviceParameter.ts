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
