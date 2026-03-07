export type ParamType = 'string' | 'number' | 'boolean' | 'enum' | 'ipAddress' | 'range';

export interface ConfigParam {
  id: string;
  paramName: string;
  paramCode: string;
  paramValue: string | number | boolean;
  defaultValue: string | number | boolean;
  paramType: ParamType;
  category: string;
  description: string;
  readonly: boolean;
  unit?: string;
  minValue?: number;
  maxValue?: number;
  enumOptions?: Array<{ label: string; value: string | number }>;
}

export interface ConfigTemplate {
  id: string;
  templateName: string;
  description: string;
  params: ConfigParam[];
  createTime: string;
  creator: string;
}

export interface BaselineConfig {
  id: string;
  baselineName: string;
  description: string;
  deviceType: string;
  version: string;
  params: ConfigParam[];
  createTime: string;
  updateTime: string;
  creator: string;
  status: 'draft' | 'active' | 'deprecated';
}

export interface NeighborParam {
  id: string;
  sourceCellId: string;
  sourceCellName: string;
  targetCellId: string;
  targetCellName: string;
  neighborType: 'intra-freq' | 'inter-freq' | 'inter-rat';
  params: Record<string, string | number | boolean>;
  createTime: string;
  updateTime: string;
}

export type ConfigTaskStatus = 'pending' | 'running' | 'success' | 'failed' | 'partial' | 'cancelled';

export interface ConfigTask {
  id: string;
  taskName: string;
  taskType: 'param-sync' | 'batch-config' | 'baseline-apply' | 'neighbor-update';
  deviceSns: string[];
  templateId?: string;
  baselineId?: string;
  params?: ConfigParam[];
  status: ConfigTaskStatus;
  progress: number;
  successCount: number;
  failCount: number;
  totalCount: number;
  createdAt: string;
  updatedAt: string;
  creator: string;
  message?: string;
}
