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

export type MMLTaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'paused' | 'cancelled';

export type MMLExecuteType = 'immediate' | 'scheduled' | 'periodic' | 'suspended';

export type MMLTaskResult = 'success' | 'partial' | 'failed';

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

  // Scheduling
  executeType: MMLExecuteType;
  scheduledAt?: string;
  periodStart?: string;
  periodEnd?: string;
  periodTime?: string;

  // Retry strategy
  offlineRetry: boolean;
  offlineRetryWait: number;
  failedRetry: boolean;
  failedRetryCount: number;
  failedRetryInterval: number;

  // Execution timestamps
  startedAt?: string;
  finishedAt?: string;

  // Statistics
  totalDevices: number;
  successCount: number;
  failedCount: number;
  result?: MMLTaskResult;
}

export interface MMLTemplate {
  id: string;
  templateName: string;
  commandCode: string;
  operationType: 'LST' | 'MOD' | 'ADD' | 'RMV';
  templateScope: 'private' | 'public';
  parameters: Record<string, string | number | boolean>;
  paramPaths: string[];
  description: string;
  productTypes: string[];
  creator: string;
  createdAt: string;
  updatedAt: string;
}
