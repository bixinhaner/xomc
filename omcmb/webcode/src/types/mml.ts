export type MMLParamType = 'string' | 'number' | 'boolean' | 'enum' | 'range' | 'ipAddress' | 'list';

export interface MMLParam {
  name: string;
  type: MMLParamType;
  required: boolean;
  defaultValue?: string | number | boolean;
  description: string;
  options?: Array<{ label: string; value: string | number }>;
  minValue?: number;
  maxValue?: number;
  pattern?: string;
}

export interface MMLCommand {
  id: string;
  commandName: string;
  commandCode: string;
  category: string;
  description: string;
  params: MMLParam[];
  productTypes: string[];
}

export interface MMLResult {
  success: boolean;
  rawOutput: string;
  parsedData?: Record<string, unknown>;
  executionTime: number;
  timestamp: string;
}

export interface MMLScript {
  id: string;
  scriptName: string;
  description: string;
  content: string;
  deviceType: string;
  creator: string;
  createTime: string;
  updateTime: string;
  tags: string[];
}

export type MMLTaskStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled';

export interface MMLTask {
  id: string;
  taskName: string;
  scriptId?: string;
  deviceSns: string[];
  commands: string[];
  status: MMLTaskStatus;
  results: Array<{ deviceSn: string; result: MMLResult }>;
  createdAt: string;
  updatedAt: string;
  creator: string;
}
