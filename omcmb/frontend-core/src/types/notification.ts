/**
 * Notification module types — 通知模板与历史记录
 *
 * 与后端 internal/notification 模块对齐：
 *  - NotificationTemplate / NotificationHistory 为前端域模型（camelCase）
 *  - Axios 响应不做 body 转换，因此 service 层提供 mapBackend* 进行 snake_case → camelCase 映射
 */

export type NotificationChannel = 'email' | 'sms' | 'webhook';
export type NotificationLanguage = 'zh-CN' | 'en-US';
export type NotificationHistoryStatus =
  | 'pending'
  | 'sent'
  | 'failed'
  | 'dead_letter';

// ---------------------------------------------------------------------------
// 通知模板
// ---------------------------------------------------------------------------

export interface NotificationTemplate {
  id: string;
  name: string;
  channel: NotificationChannel;
  language: NotificationLanguage;
  subject: string;
  body: string;
  variables: string[];
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface NotificationTemplateListParams {
  channel?: NotificationChannel;
  language?: NotificationLanguage;
  enabled?: boolean;
  page?: number;
  pageSize?: number;
}

export interface NotificationTemplateCreatePayload {
  name: string;
  channel: NotificationChannel;
  language: NotificationLanguage;
  subject: string;
  body: string;
  variables?: string[];
  enabled?: boolean;
}

export type NotificationTemplateUpdatePayload =
  Partial<NotificationTemplateCreatePayload>;

// ---------------------------------------------------------------------------
// 通知历史
// ---------------------------------------------------------------------------

export interface NotificationHistory {
  id: string;
  templateId?: string;
  channel: NotificationChannel;
  recipients: string[];
  subject: string;
  body: string;
  status: NotificationHistoryStatus;
  errorMessage?: string;
  alarmId?: string;
  retryCount: number;
  sentAt?: string;
  createdAt: string;
}

export interface NotificationHistoryListParams {
  channel?: NotificationChannel;
  status?: NotificationHistoryStatus;
  templateId?: string;
  alarmId?: string;
  page?: number;
  pageSize?: number;
}

// ---------------------------------------------------------------------------
// 列表分页响应（与 PageResponse<T> 一致，但单独导出便于直接消费）
// ---------------------------------------------------------------------------

export interface NotificationTemplateListResponse {
  items: NotificationTemplate[];
  total: number;
  page: number;
  pageSize: number;
}

export interface NotificationHistoryListResponse {
  items: NotificationHistory[];
  total: number;
  page: number;
  pageSize: number;
}

export type EmailRunBusinessType = 'alarm' | 'kpi';
export type EmailRunStatus =
  | 'pending'
  | 'processing'
  | 'sent'
  | 'partial_failed'
  | 'failed';

export interface EmailRunHistory {
  id: string;
  businessType: EmailRunBusinessType;
  templateName: string;
  period?: string;
  scheduledAt: string;
  startedAt?: string;
  finishedAt?: string;
  jobAttempt: number;
  windowStart: string;
  windowEnd: string;
  status: EmailRunStatus;
  subject?: string;
  lastError?: string;
  recipientCount: number;
  sentCount: number;
  failedCount: number;
  totalAttempts: number;
  createdAt: string;
  updatedAt: string;
}

export interface EmailRunDeliveryHistory {
  id: string;
  recipient: string;
  status: 'pending' | 'processing' | 'sent' | 'failed';
  attempt: number;
  lastError?: string;
  sentAt?: string;
  updatedAt: string;
}

export interface EmailRunHistoryListParams {
  businessType?: EmailRunBusinessType;
  status?: EmailRunStatus;
  page?: number;
  pageSize?: number;
}

export interface EmailRunHistoryListResponse {
  items: EmailRunHistory[];
  total: number;
  page: number;
  pageSize: number;
}
