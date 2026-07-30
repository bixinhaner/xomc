import http from '../http';

interface BackendSystemInfo {
  version: string;
  build_date: string;
  server_time: string;
  uptime_hours: number;
  db_status: string;
  cache_status: string;
  storage?: BackendStorageMetric[];
}

interface BackendStorageMetric {
  id: string;
  kind: StorageMetricKind;
  label: string;
  source: string;
  mount_path?: string;
  mountpoint?: string;
  instance?: string;
  target_type?: string;
  target_id?: string;
  total_bytes?: number;
  used_bytes?: number;
  available_bytes?: number;
  used_percent?: number;
  total_inodes?: number;
  used_inodes?: number;
  available_inodes?: number;
  used_inode_percent?: number;
  collected_at?: string;
  status: 'available' | 'stale' | 'unavailable';
  error?: string;
}

export type StorageMetricKind = 'app_filesystem' | 'host_filesystem' | 'minio_cluster' | 'database_logical' | string;

export interface StorageMetric {
  id: string;
  kind: StorageMetricKind;
  label: string;
  source: string;
  mountPath?: string;
  mountpoint?: string;
  instance?: string;
  targetType?: string;
  targetId?: string;
  totalBytes?: number;
  usedBytes?: number;
  availableBytes?: number;
  usedPercent?: number;
  totalInodes?: number;
  usedInodes?: number;
  availableInodes?: number;
  usedInodePercent?: number;
  collectedAt?: string;
  status: 'available' | 'stale' | 'unavailable';
  error?: string;
}

export interface SystemInfo {
  version: string;
  buildDate: string;
  serverTime: string;
  uptimeHours: number;
  dbStatus: string;
  cacheStatus: string;
  storage: StorageMetric[];
}

function mapBackendStorageMetric(info: BackendStorageMetric): StorageMetric {
  return {
    id: info.id,
    kind: info.kind,
    label: info.label,
    source: info.source,
    mountPath: info.mount_path,
    mountpoint: info.mountpoint,
    instance: info.instance,
    targetType: info.target_type,
    targetId: info.target_id,
    totalBytes: info.total_bytes,
    usedBytes: info.used_bytes,
    availableBytes: info.available_bytes,
    usedPercent: info.used_percent,
    totalInodes: info.total_inodes,
    usedInodes: info.used_inodes,
    availableInodes: info.available_inodes,
    usedInodePercent: info.used_inode_percent,
    collectedAt: info.collected_at,
    status: info.status,
    error: info.error,
  };
}

function mapBackendSystemInfo(b: BackendSystemInfo): SystemInfo {
  return {
    version: b.version,
    buildDate: b.build_date,
    serverTime: b.server_time,
    uptimeHours: b.uptime_hours,
    dbStatus: b.db_status,
    cacheStatus: b.cache_status,
    storage: (b.storage ?? []).map(mapBackendStorageMetric),
  };
}

export const systemApi = {
  async getSystemInfo() {
    const { data } = await http.get<BackendSystemInfo>('/system/info');
    return mapBackendSystemInfo(data);
  },
};
