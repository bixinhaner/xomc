export interface KPI {
  id: string;
  kpiName: string;
  kpiCode: string;
  unit: string;
  description: string;
  category: string;
}

export interface Counter {
  id: string;
  counterName: string;
  counterCode: string;
  unit: string;
  description: string;
  category: string;
  kpiIds: string[];
}

export interface Measurement {
  id: string;
  measurementName: string;
  measurementCode: string;
  deviceSn: string;
  deviceName: string;
  kpiCode: string;
  value: number;
  unit: string;
  timestamp: string;
  granularity: '15min' | '30min' | '1h' | '1d';
}

export type PerformanceTaskStatus = 'pending' | 'running' | 'success' | 'failed' | 'cancelled';

export interface PerformanceTask {
  id: string;
  taskName: string;
  taskType: 'extraction' | 'report' | 'threshold-check';
  deviceSns: string[];
  kpiCodes: string[];
  granularity: '15min' | '30min' | '1h' | '1d';
  timeRange: [string, string];
  status: PerformanceTaskStatus;
  progress: number;
  createdAt: string;
  updatedAt: string;
  creator: string;
}

export interface PerformanceThreshold {
  id: string;
  thresholdName: string;
  kpiCode: string;
  kpiName: string;
  operator: 'gt' | 'lt' | 'gte' | 'lte' | 'eq' | 'ne';
  warningValue: number;
  criticalValue: number;
  unit: string;
  enabled: boolean;
  deviceGroups: string[];
  createTime: string;
  updateTime: string;
}

export interface KPIDataPoint {
  timestamp: string;
  value: number;
}

export interface KPISeries {
  kpiName: string;
  data: KPIDataPoint[];
  unit: string;
}
