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

export interface AggregatedCounter {
  bucket: string;
  deviceId: string;
  cellId: string;
  counterGroup: string;
  counterName: string;
  sumValue: number;
  avgValue: number;
  minValue: number;
  maxValue: number;
  sampleCount: number;
}

export interface KPICalculationRequest {
  device_id: string;
  cell_id?: string;
  start_time: string;
  end_time: string;
  carrier: string;
  technology: string;
}

export interface KPICalculationResult {
  time: string;
  deviceId: string;
  cellId: string;
  kpiName: string;
  kpiValue: number;
  carrier: string;
  technology: string;
}

export interface AggregatedCounterQuery {
  device_id?: string;
  cell_id?: string;
  counter_group?: string;
  start_time?: string;
  end_time?: string;
}
