import http from '../http';

// --- Backend response types ---

interface BackendPushTarget {
  id: string;
  url: string;
  auth_type: string;
  auth_token: string;
  data_types: string[];
  format: string;
  batch_size: number;
  retry_count: number;
  enabled: boolean;
}

interface BackendSyncResult {
  data_type: string;
  items: unknown[];
  total: number;
  synced_at: string;
  truncated: boolean;
}

/** 后端 northbound_servers 表行 JSON 形态（与 omcgo 内 model 严格对齐）。 */
interface BackendNorthboundServer {
  id: string;
  role: NorthboundServerRole;
  host: string;
  port: number;
  description: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// --- Frontend types ---

export type NorthboundServerRole = 'primary' | 'standby';

/** 北向 OSS 主备服务器配置（system/config 北向设置页消费）。 */
export interface NorthboundServer {
  id: string;
  role: NorthboundServerRole;
  host: string;
  port: number;
  description: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface PushTarget {
  id: string;
  url: string;
  authType: string;
  authToken: string;
  dataTypes: string[];
  format: string;
  batchSize: number;
  retryCount: number;
  enabled: boolean;
}

export interface SyncResult {
  dataType: string;
  items: unknown[];
  total: number;
  syncedAt: string;
  truncated: boolean;
}

export interface AddPushTargetRequest {
  id: string;
  url: string;
  authType?: string;
  authToken?: string;
  dataTypes: string[];
  format?: string;
  batchSize?: number;
  retryCount?: number;
  enabled: boolean;
}

export interface ExportPMRequest {
  deviceId?: string;
  cellId?: string;
  counterGroup?: string;
  startTime: string;
  endTime: string;
  format?: string;
}

export interface ExportAlarmRequest {
  deviceSn?: string;
  severity?: string;
  status?: string;
  startTime?: string;
  endTime?: string;
  format?: string;
}

// --- Mapping functions ---

function mapBackendPushTarget(t: BackendPushTarget): PushTarget {
  return {
    id: t.id,
    url: t.url,
    authType: t.auth_type || 'none',
    authToken: t.auth_token || '',
    dataTypes: t.data_types || [],
    format: t.format || 'json',
    batchSize: t.batch_size || 100,
    retryCount: t.retry_count || 3,
    enabled: t.enabled,
  };
}

function mapBackendSyncResult(r: BackendSyncResult): SyncResult {
  return {
    dataType: r.data_type,
    items: r.items || [],
    total: r.total,
    syncedAt: r.synced_at,
    truncated: r.truncated || false,
  };
}

function mapBackendNorthboundServer(b: BackendNorthboundServer): NorthboundServer {
  return {
    id: b.id,
    role: b.role,
    host: b.host,
    port: b.port,
    description: b.description,
    isActive: b.is_active,
    createdAt: b.created_at,
    updatedAt: b.updated_at,
  };
}

// --- Exported service ---

export const northboundApi = {
  async getPushTargets(): Promise<PushTarget[]> {
    const { data } = await http.get<{ items: BackendPushTarget[]; total: number }>(
      '/northbound/push/targets'
    );
    return (data.items || []).map(mapBackendPushTarget);
  },

  async addPushTarget(req: AddPushTargetRequest): Promise<void> {
    const body = {
      id: req.id,
      url: req.url,
      auth_type: req.authType || 'none',
      auth_token: req.authToken,
      data_types: req.dataTypes,
      format: req.format || 'json',
      batch_size: req.batchSize || 100,
      retry_count: req.retryCount || 3,
      enabled: req.enabled,
    };
    await http.post('/northbound/push/targets', body);
  },

  async removePushTarget(id: string): Promise<void> {
    await http.delete(`/northbound/push/targets/${id}`);
  },

  async fullSync(dataType: string): Promise<SyncResult> {
    const { data } = await http.get<BackendSyncResult>('/northbound/sync/full', {
      params: { data_type: dataType },
    });
    return mapBackendSyncResult(data);
  },

  async incrementalSync(dataType: string, since: string): Promise<SyncResult> {
    const { data } = await http.get<BackendSyncResult>('/northbound/sync/incremental', {
      params: { data_type: dataType, since },
    });
    return mapBackendSyncResult(data);
  },

  async exportPM(req: ExportPMRequest): Promise<void> {
    await http.post('/northbound/export/pm', {
      device_id: req.deviceId,
      cell_id: req.cellId,
      counter_group: req.counterGroup,
      start_time: req.startTime,
      end_time: req.endTime,
      format: req.format || 'json',
    });
  },

  async exportAlarms(req: ExportAlarmRequest): Promise<void> {
    await http.post('/northbound/export/alarms', {
      device_sn: req.deviceSn,
      severity: req.severity,
      status: req.status,
      start_time: req.startTime,
      end_time: req.endTime,
      format: req.format || 'json',
    });
  },

  async exportConfig(deviceId: string): Promise<unknown> {
    const { data } = await http.get(`/northbound/export/config/${deviceId}`);
    return data;
  },

  // --- 主备服务器（system/config 北向设置 切换功能）---

  /** GET /northbound/servers — 列出主备两组配置。 */
  async getServers(): Promise<NorthboundServer[]> {
    const { data } = await http.get<{ items: BackendNorthboundServer[]; total: number }>(
      '/northbound/servers',
    );
    return (data?.items ?? []).map(mapBackendNorthboundServer);
  },

  /**
   * PUT /northbound/servers/active — 切换激活组。
   *
   * 后端语义：role 已激活时返回 200 noop（不报错）；role 不存在返回 404。
   * 切换成功后主备 is_active 自动互换（事务保证 partial unique index 不冲突）。
   */
  async switchActiveServer(role: NorthboundServerRole): Promise<void> {
    await http.put('/northbound/servers/active', { role });
  },

  /**
   * PUT /northbound/servers/:role — 编辑指定 role 的连接配置。
   *
   * body: { host: string, port: int, description?: string }；不动 is_active。
   * 权限：独立编辑权限点（seed/000070 默认仅授予 admin / operator 角色）；
   *       viewer / 未授权角色调用返回 403。
   */
  async updateServer(
    role: NorthboundServerRole,
    payload: { host: string; port: number; description?: string },
  ): Promise<void> {
    await http.put(`/northbound/servers/${role}`, {
      host: payload.host,
      port: payload.port,
      description: payload.description ?? '',
    });
  },
};
