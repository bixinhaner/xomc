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

// --- Frontend types ---

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
};
