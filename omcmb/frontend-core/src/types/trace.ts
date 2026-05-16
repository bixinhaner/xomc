// T-0137 / M1: TR069 报文跟踪类型定义
// 设计：docs/design/TR069报文跟踪-设计.md
//
// 后端 BackendXxx (snake_case) → mapBackendXxx → 前端 Xxx (camelCase)，
// 转换在 services/api/traceApi.ts 内完成。

import type { PageResponse } from './pagination';

export type TraceTaskStatus = 'running' | 'stopped' | 'purged';
export type TraceDirection = 'in' | 'out';

export interface TraceTask {
  id: string;
  deviceSn: string;
  operatorCode: string;
  status: TraceTaskStatus;
  startTime: string;
  expiresAt: string;
  stoppedAt?: string;
  purgedAt?: string;
  createdBy: string;
  messageCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface TraceMessage {
  id: string;
  capturedAt: string;
  taskId: string;
  deviceSn: string;
  direction: TraceDirection;
  rpcMethod?: string;
  cwmpId?: string;
  sessionId?: string;
  httpStatus?: number;
  payloadSizeBytes: number;
  payloadInline?: string;
  payloadObjectKey?: string;
}

export interface TraceTaskListParams {
  deviceSn?: string;
  status?: TraceTaskStatus;
  operatorCode?: string;
  page?: number;
  pageSize?: number;
  sortField?: string;
  sortOrder?: 'ascend' | 'descend';
}

export interface TraceMessageListParams {
  direction?: TraceDirection;
  rpcMethod?: string;
  startTime?: string;
  endTime?: string;
  page?: number;
  pageSize?: number;
}

export interface CreateTraceTaskPayload {
  deviceSn: string;
  durationMinutes?: number;
}

export type TraceTaskListResponse = PageResponse<TraceTask>;
export type TraceMessageListResponse = PageResponse<TraceMessage>;

// M2-08: 异步下载任务
export type TraceExportJobStatus = 'queued' | 'running' | 'done' | 'failed';

export interface TraceExportJob {
  id: string;
  taskId: string;
  requestedBy: string;
  status: TraceExportJobStatus;
  objectKey?: string;
  objectBucket?: string;
  messageCount: number;
  sizeBytes: number;
  errorMessage?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
  downloadUrl?: string;
}
