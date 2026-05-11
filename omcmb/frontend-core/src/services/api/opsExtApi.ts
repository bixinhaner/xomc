// F06 运维管理扩展 API（对应后端 internal/ops/handler_ext.go 25 个端点）
// 来源 PRD: docs/project/prd/F06-ops-management.md
// 推进计划: docs/project/F06-ops-management-implementation-plan.md

import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';

// ============================================================
// 类型定义（与后端 model_ext.go 1:1 对照）
// ============================================================

export type RiskLevel = 'safe' | 'cautious' | 'dangerous';
export type ApprovalState = 'not_required' | 'pending' | 'approved' | 'rejected';
export type DiagnosticStatus = 'pending' | 'running' | 'complete' | 'failed' | 'timeout';
export type DownloadStatus = 'pending' | 'uploading' | 'complete' | 'failed' | 'expired';
export type MaintenanceWindowStatus = 'planned' | 'approved' | 'active' | 'ended' | 'cancelled';

export interface OpsDiagnostic {
  id: string;
  device_sn?: string;
  diag_type: string;
  initiator: string;
  request: unknown;
  result?: unknown;
  status: DiagnosticStatus;
  started_at: string;
  completed_at?: string;
  duration_ms: number;
  operator?: string;
  task_id?: string;
  file_path?: string;
  error_message?: string;
  created_at: string;
}

export interface OpsDownload {
  id: string;
  device_sn: string;
  content_type: string;
  file_path: string;
  file_size: number;
  checksum?: string;
  status: DownloadStatus;
  operator?: string;
  task_id?: string;
  expires_at?: string;
  created_at: string;
}

export interface OpsAuditLog {
  id: string;
  op_type: string;
  target_type: string;
  target_id: string;
  operator_user_id?: string;
  operator_name: string;
  risk_level: RiskLevel;
  input?: unknown;
  output_summary?: string;
  result: string;
  approver_user_id?: string;
  break_glass: boolean;
  client_ip?: string;
  user_agent?: string;
  created_at: string;
}

export interface OpsMaintenanceWindow {
  id: string;
  name: string;
  scope_type: 'device' | 'group' | 'all';
  scope_ids: string[];
  start_at: string;
  end_at: string;
  suppress_alarms: boolean;
  pause_provision: boolean;
  allow_dangerous: boolean;
  reason?: string;
  creator_user_id?: string;
  approver_user_id?: string;
  status: MaintenanceWindowStatus;
  created_at: string;
  updated_at: string;
}

export interface OpsPlaybook {
  id: string;
  alarm_pattern: { alarm_code?: string[]; severity?: string[] };
  recommended_templates: string[];
  title: string;
  docs?: string;
  tags: string[];
  success_rate?: number;
  use_count: number;
  created_at: string;
  updated_at: string;
}

export interface OpsTaskExecution {
  id: string;
  task_id: string;
  device_sn: string;
  step_index: number;
  step_name: string;
  step_type: string;
  status: string;
  started_at?: string;
  completed_at?: string;
  duration_ms: number;
  request?: unknown;
  response?: unknown;
  error_message?: string;
  created_at: string;
}

// ============================================================
// API 服务
// ============================================================

export const opsExtApi = {
  // --- 诊断（T-0104）---
  async ping(params: { device_sn: string; host: string; count?: number; timeout?: number }) {
    const { data } = await http.post<OpsDiagnostic>('/ops/diagnostics/ping', params);
    return data;
  },
  async traceroute(params: { device_sn: string; host: string }) {
    const { data } = await http.post<OpsDiagnostic>('/ops/diagnostics/traceroute', params);
    return data;
  },
  async throughput(params: { device_sn: string; url: string; direction?: 'download' | 'upload' }) {
    const { data } = await http.post<OpsDiagnostic>('/ops/diagnostics/throughput', params);
    return data;
  },
  async listDiagnostics(
    params: { deviceSn?: string; diagType?: string; status?: DiagnosticStatus } & PageRequest,
  ): Promise<PageResponse<OpsDiagnostic>> {
    const { data } = await http.get<{
      items: OpsDiagnostic[];
      total: number;
      page: number;
      page_size: number;
    }>('/ops/diagnostics', {
      params: {
        device_sn: params.deviceSn,
        diag_type: params.diagType,
        status: params.status,
        page: params.page,
        page_size: params.pageSize,
      },
    });
    return { items: data.items || [], total: data.total, page: data.page, pageSize: data.page_size };
  },
  async getDiagnostic(id: string): Promise<OpsDiagnostic> {
    const { data } = await http.get<OpsDiagnostic>(`/ops/diagnostics/${id}`);
    return data;
  },
  async runInspection(scope: string = 'all') {
    const { data } = await http.post<{ status: string; scope: string }>('/ops/diagnostics/inspection', { scope });
    return data;
  },

  // --- 下载（T-0105）---
  async collectDownload(params: { device_sn: string; content_types: string[] }) {
    const { data } = await http.post<{ items: OpsDownload[] }>('/ops/downloads/collect', params);
    return data.items;
  },
  async listDownloads(
    params: { deviceSn?: string; contentType?: string; status?: DownloadStatus } & PageRequest,
  ): Promise<PageResponse<OpsDownload>> {
    const { data } = await http.get<{
      items: OpsDownload[];
      total: number;
      page: number;
      page_size: number;
    }>('/ops/downloads', {
      params: {
        device_sn: params.deviceSn,
        content_type: params.contentType,
        status: params.status,
        page: params.page,
        page_size: params.pageSize,
      },
    });
    return { items: data.items || [], total: data.total, page: data.page, pageSize: data.page_size };
  },
  async getDownload(id: string): Promise<OpsDownload> {
    const { data } = await http.get<OpsDownload>(`/ops/downloads/${id}`);
    return data;
  },

  // --- 审计（T-0109）---
  async listAuditLogs(
    params: {
      opType?: string;
      targetType?: string;
      targetId?: string;
      operatorUserId?: string;
      riskLevel?: RiskLevel;
      breakGlass?: boolean;
      from?: string;
      to?: string;
    } & PageRequest,
  ): Promise<PageResponse<OpsAuditLog>> {
    const { data } = await http.get<{
      items: OpsAuditLog[];
      total: number;
      page: number;
      page_size: number;
    }>('/ops/audit-logs', {
      params: {
        op_type: params.opType,
        target_type: params.targetType,
        target_id: params.targetId,
        operator_user_id: params.operatorUserId,
        risk_level: params.riskLevel,
        break_glass: params.breakGlass ? 'true' : undefined,
        from: params.from,
        to: params.to,
        page: params.page,
        page_size: params.pageSize,
      },
    });
    return { items: data.items || [], total: data.total, page: data.page, pageSize: data.page_size };
  },

  // --- 维护窗口（T-0107）---
  async createMaintenanceWindow(params: {
    name: string;
    scope_type: 'device' | 'group' | 'all';
    scope_ids?: string[];
    start_at: string;
    end_at: string;
    suppress_alarms?: boolean;
    pause_provision?: boolean;
    allow_dangerous?: boolean;
    reason?: string;
  }): Promise<OpsMaintenanceWindow> {
    const { data } = await http.post<OpsMaintenanceWindow>('/ops/maintenance-windows', params);
    return data;
  },
  async listMaintenanceWindows(
    params: { status?: MaintenanceWindowStatus } & PageRequest,
  ): Promise<PageResponse<OpsMaintenanceWindow>> {
    const { data } = await http.get<{
      items: OpsMaintenanceWindow[];
      total: number;
      page: number;
      page_size: number;
    }>('/ops/maintenance-windows', {
      params: { status: params.status, page: params.page, page_size: params.pageSize },
    });
    return { items: data.items || [], total: data.total, page: data.page, pageSize: data.page_size };
  },
  async listActiveMaintenanceWindows(): Promise<OpsMaintenanceWindow[]> {
    const { data } = await http.get<{ items: OpsMaintenanceWindow[]; total: number }>(
      '/ops/maintenance-windows/active',
    );
    return data.items || [];
  },
  async approveMaintenanceWindow(id: string) {
    const { data } = await http.post<{ status: string }>(`/ops/maintenance-windows/${id}/approve`);
    return data;
  },

  // --- 知识库（T-0110）---
  async listPlaybooks(params: { keyword?: string } & PageRequest): Promise<PageResponse<OpsPlaybook>> {
    const { data } = await http.get<{
      items: OpsPlaybook[];
      total: number;
      page: number;
      page_size: number;
    }>('/ops/playbooks', {
      params: { keyword: params.keyword, page: params.page, page_size: params.pageSize },
    });
    return { items: data.items || [], total: data.total, page: data.page, pageSize: data.page_size };
  },
  async createPlaybook(p: Partial<OpsPlaybook>): Promise<OpsPlaybook> {
    const { data } = await http.post<OpsPlaybook>('/ops/playbooks', p);
    return data;
  },
  async matchPlaybook(params: { alarm_code: string; severity?: string }) {
    const { data } = await http.post<{ items: OpsPlaybook[]; total: number }>('/ops/playbooks/match', params);
    return data.items;
  },

  // --- 任务编排扩展（T-0101）---
  async runTask(id: string) {
    const { data } = await http.post<{ status: string }>(`/ops/tasks/${id}/run`);
    return data;
  },
  async listTaskExecutions(
    taskId: string,
    params: { deviceSn?: string; status?: string } & PageRequest,
  ): Promise<PageResponse<OpsTaskExecution>> {
    const { data } = await http.get<{
      items: OpsTaskExecution[];
      total: number;
      page: number;
      page_size: number;
    }>(`/ops/tasks/${taskId}/executions`, {
      params: {
        device_sn: params.deviceSn,
        status: params.status,
        page: params.page,
        page_size: params.pageSize,
      },
    });
    return { items: data.items || [], total: data.total, page: data.page, pageSize: data.page_size };
  },
  async approveTask(id: string, params: { approve: boolean; reason?: string }) {
    const { data } = await http.post<{ status: string }>(`/ops/tasks/${id}/approve`, params);
    return data;
  },

  // --- 即时命令（T-0102）---
  async executeRPC(params: { device_sn: string; action: string; params?: Record<string, unknown> }) {
    const { data } = await http.post<{ task_id: string; status: string; stream_url: string }>(
      '/ops/commands/rpc',
      params,
    );
    return data;
  },

  // --- 紧急响应（T-0111）---
  async activateBreakGlass(params: { ticket_id: string; reason: string }) {
    const { data } = await http.post<{ status: string; expires_in_seconds: number }>(
      '/ops/break-glass/activate',
      params,
    );
    return data;
  },
  async deactivateBreakGlass() {
    const { data } = await http.post<{ status: string }>('/ops/break-glass/deactivate');
    return data;
  },
  async getBreakGlassStatus() {
    const { data } = await http.get<{ active: boolean }>('/ops/break-glass/status');
    return data;
  },
};

// ============================================================
// SSE 订阅工具（T-0102）
// ============================================================

/**
 * 订阅 SSE 通道，返回 EventSource 实例和取消订阅函数。
 *
 * 用法：
 * ```ts
 * const { source, close } = subscribeSSE(`/api/v1/ops/tasks/${taskId}/events`, {
 *   onEvent: (event, data) => console.log(event, data),
 * });
 * // ...
 * close();
 * ```
 */
export function subscribeSSE(
  url: string,
  opts: {
    onEvent?: (event: string, data: unknown) => void;
    onError?: (err: Event) => void;
    onOpen?: () => void;
  } = {},
): { source: EventSource; close: () => void } {
  // 使用 fetch token 注入暂未做，依赖 cookie 鉴权；JWT 模式需 EventSource polyfill
  // 见 PRD §4.2.3 R-O02 nginx/k8s ingress 长连接配置
  const fullURL = url.startsWith('/api') ? url : `/api/v1${url.startsWith('/') ? '' : '/'}${url}`;
  const source = new EventSource(fullURL, { withCredentials: true });

  if (opts.onOpen) {
    source.addEventListener('open', opts.onOpen);
  }
  if (opts.onError) {
    source.addEventListener('error', opts.onError);
  }
  if (opts.onEvent) {
    // 已知事件类型
    const knownEvents = ['open', 'ping', 'command.dispatched', 'task.progress', 'task.completed'];
    knownEvents.forEach((evt) => {
      source.addEventListener(evt, (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          opts.onEvent!(evt, data);
        } catch {
          opts.onEvent!(evt, e.data);
        }
      });
    });
    // 默认 message
    source.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        opts.onEvent!('message', data);
      } catch {
        opts.onEvent!('message', e.data);
      }
    };
  }

  return {
    source,
    close: () => source.close(),
  };
}
