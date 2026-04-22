import http from '../http';

// --- Backend types (from sync_handler.go) ---

interface ParameterValue {
  name: string;
  value: string;
  type?: string; // string, int, boolean, etc.
}

interface SyncStatusResponse {
  device_id: string;
  pending_count: number;
}

interface SyncCommandResponse {
  message: string;
  device_id: string;
  command_id: string;
}

// --- Exported service ---

export const configSyncApi = {
  /**
   * Push configuration parameters to a device (queues SetParameterValues).
   */
  async pushConfig(deviceId: string, parameters: ParameterValue[]): Promise<SyncCommandResponse> {
    const { data } = await http.post<SyncCommandResponse>(
      `/config/sync/push/${deviceId}`,
      { parameters },
    );
    return data;
  },

  /**
   * Pull configuration parameters from a device (queues GetParameterValues).
   */
  async pullConfig(deviceId: string, parameterNames: string[]): Promise<SyncCommandResponse> {
    const { data } = await http.post<SyncCommandResponse>(
      `/config/sync/pull/${deviceId}`,
      { parameter_names: parameterNames },
    );
    return data;
  },

  /**
   * Get the sync status (pending command count) for a device.
   */
  async getSyncStatus(deviceId: string): Promise<SyncStatusResponse> {
    const { data } = await http.get<SyncStatusResponse>(
      `/config/sync/status/${deviceId}`,
    );
    return data;
  },
};
