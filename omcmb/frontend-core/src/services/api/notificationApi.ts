import http from '../http';
import type {
  NotificationTemplate,
  NotificationTemplateListParams,
  NotificationTemplateListResponse,
  NotificationTemplateCreatePayload,
  NotificationTemplateUpdatePayload,
  NotificationHistory,
  NotificationHistoryListParams,
  NotificationHistoryListResponse,
  NotificationChannel,
  NotificationLanguage,
  NotificationHistoryStatus,
  EmailRunHistory,
  EmailRunDeliveryHistory,
  EmailRunHistoryListParams,
  EmailRunHistoryListResponse,
} from '../../types/notification';

// ---------------------------------------------------------------------------
// Backend models (snake_case) — Axios 响应不做 body 转换
// ---------------------------------------------------------------------------

interface BackendNotificationTemplate {
  id: string;
  name: string;
  channel: string;
  language: string;
  subject: string;
  body: string;
  variables: string[] | null;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

interface BackendNotificationHistory {
  id: string;
  template_id?: string;
  channel: string;
  recipients: string[] | null;
  subject: string;
  body: string;
  status: string;
  error_message?: string;
  alarm_id?: string;
  retry_count: number;
  sent_at?: string;
  created_at: string;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages?: number;
}

interface BackendEmailRunHistory {
  id: string;
  business_type: 'alarm' | 'kpi';
  template_name: string;
  period?: string;
  scheduled_at: string;
  started_at?: string;
  finished_at?: string;
  job_attempt: number;
  window_start: string;
  window_end: string;
  status: EmailRunHistory['status'];
  subject?: string;
  last_error?: string;
  recipient_count: number;
  sent_count: number;
  failed_count: number;
  total_attempts: number;
  created_at: string;
  updated_at: string;
}

interface BackendEmailRunDeliveryHistory {
  id: string;
  recipient: string;
  status: EmailRunDeliveryHistory['status'];
  attempt: number;
  last_error?: string;
  sent_at?: string;
  updated_at: string;
}

// ---------------------------------------------------------------------------
// Mappers
// ---------------------------------------------------------------------------

function asChannel(v: string): NotificationChannel {
  if (v === 'email' || v === 'sms' || v === 'webhook') return v;
  // 兜底：未知值落到 webhook，避免 UI 渲染崩溃
  return 'webhook';
}

function asLanguage(v: string): NotificationLanguage {
  return v === 'en-US' ? 'en-US' : 'zh-CN';
}

function asHistoryStatus(v: string): NotificationHistoryStatus {
  if (v === 'pending' || v === 'sent' || v === 'failed' || v === 'dead_letter') {
    return v;
  }
  return 'pending';
}

function mapBackendTemplate(bt: BackendNotificationTemplate): NotificationTemplate {
  return {
    id: bt.id,
    name: bt.name,
    channel: asChannel(bt.channel),
    language: asLanguage(bt.language),
    subject: bt.subject ?? '',
    body: bt.body ?? '',
    variables: bt.variables ?? [],
    enabled: Boolean(bt.enabled),
    createdAt: bt.created_at,
    updatedAt: bt.updated_at,
  };
}

function mapBackendHistory(bh: BackendNotificationHistory): NotificationHistory {
  return {
    id: bh.id,
    templateId: bh.template_id,
    channel: asChannel(bh.channel),
    recipients: bh.recipients ?? [],
    subject: bh.subject ?? '',
    body: bh.body ?? '',
    status: asHistoryStatus(bh.status),
    errorMessage: bh.error_message,
    alarmId: bh.alarm_id,
    retryCount: bh.retry_count ?? 0,
    sentAt: bh.sent_at,
    createdAt: bh.created_at,
  };
}

function mapTemplateList(
  resp: BackendListResponse<BackendNotificationTemplate>
): NotificationTemplateListResponse {
  return {
    items: (resp.items || []).map(mapBackendTemplate),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

function mapHistoryList(
  resp: BackendListResponse<BackendNotificationHistory>
): NotificationHistoryListResponse {
  return {
    items: (resp.items || []).map(mapBackendHistory),
    total: resp.total,
    page: resp.page,
    pageSize: resp.page_size,
  };
}

function mapEmailRun(item: BackendEmailRunHistory): EmailRunHistory {
  return {
    id: item.id,
    businessType: item.business_type,
    templateName: item.template_name,
    period: item.period,
    scheduledAt: item.scheduled_at,
    startedAt: item.started_at,
    finishedAt: item.finished_at,
    jobAttempt: item.job_attempt ?? 0,
    windowStart: item.window_start,
    windowEnd: item.window_end,
    status: item.status,
    subject: item.subject,
    lastError: item.last_error,
    recipientCount: item.recipient_count,
    sentCount: item.sent_count,
    failedCount: item.failed_count,
    totalAttempts: item.total_attempts,
    createdAt: item.created_at,
    updatedAt: item.updated_at,
  };
}

// ---------------------------------------------------------------------------
// Template API
// ---------------------------------------------------------------------------

export const notificationTemplateApi = {
  async list(
    params: NotificationTemplateListParams = {}
  ): Promise<NotificationTemplateListResponse> {
    const query: Record<string, unknown> = {};
    if (params.channel) query.channel = params.channel;
    if (params.language) query.language = params.language;
    if (typeof params.enabled === 'boolean') query.enabled = params.enabled;
    if (params.page) query.page = params.page;
    if (params.pageSize) query.pageSize = params.pageSize;

    const { data } = await http.get<BackendListResponse<BackendNotificationTemplate>>(
      '/notifications/templates',
      { params: query }
    );
    return mapTemplateList(data);
  },

  async get(id: string): Promise<NotificationTemplate | null> {
    try {
      const { data } = await http.get<BackendNotificationTemplate>(
        `/notifications/templates/${id}`
      );
      return mapBackendTemplate(data);
    } catch {
      return null;
    }
  },

  async create(
    payload: NotificationTemplateCreatePayload
  ): Promise<NotificationTemplate> {
    const body: Record<string, unknown> = {
      name: payload.name,
      channel: payload.channel,
      language: payload.language,
      subject: payload.subject,
      body: payload.body,
      variables: payload.variables ?? [],
      enabled: payload.enabled ?? true,
    };
    const { data } = await http.post<BackendNotificationTemplate>(
      '/notifications/templates',
      body
    );
    return mapBackendTemplate(data);
  },

  async update(
    id: string,
    payload: NotificationTemplateUpdatePayload
  ): Promise<NotificationTemplate> {
    const body: Record<string, unknown> = {};
    if (payload.name !== undefined) body.name = payload.name;
    if (payload.channel !== undefined) body.channel = payload.channel;
    if (payload.language !== undefined) body.language = payload.language;
    if (payload.subject !== undefined) body.subject = payload.subject;
    if (payload.body !== undefined) body.body = payload.body;
    if (payload.variables !== undefined) body.variables = payload.variables;
    if (payload.enabled !== undefined) body.enabled = payload.enabled;

    const { data } = await http.put<BackendNotificationTemplate>(
      `/notifications/templates/${id}`,
      body
    );
    return mapBackendTemplate(data);
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/notifications/templates/${id}`);
  },
};

// ---------------------------------------------------------------------------
// History API
// ---------------------------------------------------------------------------

export const notificationHistoryApi = {
  async list(
    params: NotificationHistoryListParams = {}
  ): Promise<NotificationHistoryListResponse> {
    const query: Record<string, unknown> = {};
    if (params.channel) query.channel = params.channel;
    if (params.status) query.status = params.status;
    if (params.templateId) query.template_id = params.templateId;
    if (params.alarmId) query.alarm_id = params.alarmId;
    if (params.page) query.page = params.page;
    if (params.pageSize) query.pageSize = params.pageSize;

    const { data } = await http.get<BackendListResponse<BackendNotificationHistory>>(
      '/notifications/history',
      { params: query }
    );
    return mapHistoryList(data);
  },

  async get(id: string): Promise<NotificationHistory | null> {
    try {
      const { data } = await http.get<BackendNotificationHistory>(
        `/notifications/history/${id}`
      );
      return mapBackendHistory(data);
    } catch {
      return null;
    }
  },
};

export const emailRunHistoryApi = {
  async list(params: EmailRunHistoryListParams = {}): Promise<EmailRunHistoryListResponse> {
    const query: Record<string, unknown> = {};
    if (params.businessType) query.business_type = params.businessType;
    if (params.status) query.status = params.status;
    if (params.page) query.page = params.page;
    if (params.pageSize) query.page_size = params.pageSize;
    const { data } = await http.get<BackendListResponse<BackendEmailRunHistory>>(
      '/notifications/email-runs',
      { params: query }
    );
    return {
      items: (data.items ?? []).map(mapEmailRun),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async deliveries(businessType: 'alarm' | 'kpi', runId: string): Promise<EmailRunDeliveryHistory[]> {
    const { data } = await http.get<{ items: BackendEmailRunDeliveryHistory[] }>(
      `/notifications/email-runs/${businessType}/${runId}/deliveries`
    );
    return (data.items ?? []).map((item) => ({
      id: item.id,
      recipient: item.recipient,
      status: item.status,
      attempt: item.attempt,
      lastError: item.last_error,
      sentAt: item.sent_at,
      updatedAt: item.updated_at,
    }));
  },
};
