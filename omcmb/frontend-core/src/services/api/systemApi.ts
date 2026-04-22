import http from '../http';

interface BackendSystemInfo {
  version: string;
  build_date: string;
  server_time: string;
  uptime_hours: number;
  db_status: string;
  cache_status: string;
}

function mapBackendSystemInfo(b: BackendSystemInfo) {
  return {
    version: b.version,
    buildDate: b.build_date,
    serverTime: b.server_time,
    uptimeHours: b.uptime_hours,
    dbStatus: b.db_status,
    cacheStatus: b.cache_status,
  };
}

export const systemApi = {
  async getSystemInfo() {
    const { data } = await http.get<BackendSystemInfo>('/system/info');
    return mapBackendSystemInfo(data);
  },
};
