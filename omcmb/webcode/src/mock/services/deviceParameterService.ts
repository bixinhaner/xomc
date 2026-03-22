import type {
  DeviceParameter,
  ParameterTreeNode,
  ParameterSyncStatus,
  ParameterFilter,
  ParameterUpdateRequest,
  ParameterSyncOptions,
} from '@/types/deviceParameter';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { delay, paginate } from '../utils';

// Mock parameter data
const mockParameters: DeviceParameter[] = [
  { id: '1', deviceId: '', parameterPath: 'Device.DeviceInfo.Manufacturer', parameterValue: 'Baicells', parameterType: 'string', writable: false, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '2', deviceId: '', parameterPath: 'Device.DeviceInfo.ModelName', parameterValue: 'Nova436Q', parameterType: 'string', writable: false, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '3', deviceId: '', parameterPath: 'Device.DeviceInfo.SoftwareVersion', parameterValue: 'BaiBS_RTS_3.7.11.16', parameterType: 'string', writable: false, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '4', deviceId: '', parameterPath: 'Device.DeviceInfo.HardwareVersion', parameterValue: 'V2.0', parameterType: 'string', writable: false, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '5', deviceId: '', parameterPath: 'Device.DeviceInfo.SerialNumber', parameterValue: 'BAIC12345678', parameterType: 'string', writable: false, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '6', deviceId: '', parameterPath: 'Device.ManagementServer.URL', parameterValue: 'http://acs.example.com:7547', parameterType: 'string', writable: true, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '7', deviceId: '', parameterPath: 'Device.ManagementServer.PeriodicInformEnable', parameterValue: 'true', parameterType: 'boolean', writable: true, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '8', deviceId: '', parameterPath: 'Device.ManagementServer.PeriodicInformInterval', parameterValue: '300', parameterType: 'unsignedInt', writable: true, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '9', deviceId: '', parameterPath: 'Device.ManagementServer.ConnectionRequestURL', parameterValue: 'http://192.168.1.100:7547/cr', parameterType: 'string', writable: false, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '10', deviceId: '', parameterPath: 'Device.ManagementServer.Username', parameterValue: 'cpe_user', parameterType: 'string', writable: true, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '11', deviceId: '', parameterPath: 'Device.LAN.IPAddress', parameterValue: '192.168.1.1', parameterType: 'string', writable: true, lastUpdatedAt: '2026-03-19T08:00:00Z' },
  { id: '12', deviceId: '', parameterPath: 'Device.LAN.SubnetMask', parameterValue: '255.255.255.0', parameterType: 'string', writable: true, lastUpdatedAt: '2026-03-19T08:00:00Z' },
  { id: '13', deviceId: '', parameterPath: 'Device.LAN.DHCPServerEnable', parameterValue: 'true', parameterType: 'boolean', writable: true, lastUpdatedAt: '2026-03-19T08:00:00Z' },
  { id: '14', deviceId: '', parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNDL', parameterValue: '38400', parameterType: 'unsignedInt', writable: true, lastUpdatedAt: '2026-03-18T14:00:00Z' },
  { id: '15', deviceId: '', parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.EARFCNUL', parameterValue: '38400', parameterType: 'unsignedInt', writable: true, lastUpdatedAt: '2026-03-18T14:00:00Z' },
  { id: '16', deviceId: '', parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.FreqBandIndicator', parameterValue: '41', parameterType: 'unsignedInt', writable: true, lastUpdatedAt: '2026-03-18T14:00:00Z' },
  { id: '17', deviceId: '', parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth', parameterValue: '20MHz', parameterType: 'string', writable: true, lastUpdatedAt: '2026-03-18T14:00:00Z' },
  { id: '18', deviceId: '', parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ReferenceSignalPower', parameterValue: '15', parameterType: 'int', writable: true, lastUpdatedAt: '2026-03-18T14:00:00Z' },
  { id: '19', deviceId: '', parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity', parameterValue: '1', parameterType: 'unsignedInt', writable: true, lastUpdatedAt: '2026-03-18T14:00:00Z' },
  { id: '20', deviceId: '', parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID', parameterValue: '46000', parameterType: 'string', writable: true, lastUpdatedAt: '2026-03-18T14:00:00Z' },
  { id: '21', deviceId: '', parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC', parameterValue: '1', parameterType: 'unsignedInt', writable: true, lastUpdatedAt: '2026-03-18T14:00:00Z' },
  { id: '22', deviceId: '', parameterPath: 'Device.Time.NTPServer1', parameterValue: 'ntp.aliyun.com', parameterType: 'string', writable: true, lastUpdatedAt: '2026-03-17T12:00:00Z' },
  { id: '23', deviceId: '', parameterPath: 'Device.Time.LocalTimeZone', parameterValue: 'CST-8', parameterType: 'string', writable: true, lastUpdatedAt: '2026-03-17T12:00:00Z' },
  { id: '24', deviceId: '', parameterPath: 'Device.IP.Interface.1.IPv4Address.1.IPAddress', parameterValue: '10.20.30.40', parameterType: 'string', writable: false, lastUpdatedAt: '2026-03-20T10:30:00Z' },
  { id: '25', deviceId: '', parameterPath: 'Device.IP.Interface.1.IPv4Address.1.SubnetMask', parameterValue: '255.255.255.0', parameterType: 'string', writable: false, lastUpdatedAt: '2026-03-20T10:30:00Z' },
];

// Build tree structure from flat parameters
function buildMockTree(): ParameterTreeNode[] {
  const root: ParameterTreeNode = {
    name: 'Device',
    fullPath: 'Device.',
    isObject: true,
    children: [],
  };

  for (const param of mockParameters) {
    const parts = param.parameterPath.split('.');
    let current = root;

    for (let i = 1; i < parts.length; i++) {
      const isLeaf = i === parts.length - 1;
      const partPath = parts.slice(0, i + 1).join('.') + (isLeaf ? '' : '.');
      const partName = parts[i];

      if (!current.children) current.children = [];
      let child = current.children.find((c) => c.name === partName);

      if (!child) {
        child = {
          name: partName,
          fullPath: partPath,
          isObject: !isLeaf,
          ...(isLeaf
            ? {
                parameterType: param.parameterType,
                parameterValue: param.parameterValue,
                writable: param.writable,
                lastUpdatedAt: param.lastUpdatedAt,
              }
            : {}),
          children: isLeaf ? undefined : [],
        };
        current.children.push(child);
      }
      current = child;
    }
  }

  return root.children ?? [];
}

// Track sync state per device
const syncStates = new Map<string, ParameterSyncStatus>();

export const deviceParameterService = {
  async getParameters(
    deviceId: string,
    params?: ParameterFilter & PageRequest
  ): Promise<PageResponse<DeviceParameter>> {
    await delay(100, 300);
    let filtered = mockParameters.map((p) => ({ ...p, deviceId }));

    if (params?.search) {
      const keyword = params.search.toLowerCase();
      filtered = filtered.filter(
        (p) =>
          p.parameterPath.toLowerCase().includes(keyword) ||
          p.parameterValue.toLowerCase().includes(keyword)
      );
    }
    if (params?.writable !== undefined) {
      filtered = filtered.filter((p) => p.writable === params.writable);
    }
    if (params?.parameterType) {
      filtered = filtered.filter((p) => p.parameterType === params.parameterType);
    }

    const page = params?.page ?? 1;
    const pageSize = params?.pageSize ?? 50;
    return paginate(filtered, page, pageSize);
  },

  async getParameterTree(deviceId: string): Promise<ParameterTreeNode[]> {
    await delay(200, 500);
    void deviceId;
    return buildMockTree();
  },

  async updateParameters(
    deviceId: string,
    parameters: ParameterUpdateRequest[]
  ): Promise<void> {
    await delay(300, 600);
    // Update mock data in place
    for (const update of parameters) {
      const existing = mockParameters.find((p) => p.parameterPath === update.parameterPath);
      if (existing) {
        existing.parameterValue = update.parameterValue;
        existing.lastUpdatedAt = new Date().toISOString();
      }
    }
    void deviceId;
  },

  async syncParameters(
    deviceId: string,
    _options?: ParameterSyncOptions
  ): Promise<void> {
    await delay(100, 200);
    // Start mock sync process
    syncStates.set(deviceId, {
      deviceId,
      status: 'syncing',
      totalBatches: 5,
      completedBatches: 0,
      totalParameters: mockParameters.length,
      syncedParameters: 0,
      percentage: 0,
      startedAt: new Date().toISOString(),
    });

    // Simulate gradual progress
    let batch = 0;
    const interval = setInterval(() => {
      batch++;
      const state = syncStates.get(deviceId);
      if (!state) {
        clearInterval(interval);
        return;
      }
      const synced = Math.min(
        Math.round((batch / 5) * mockParameters.length),
        mockParameters.length
      );
      syncStates.set(deviceId, {
        ...state,
        completedBatches: batch,
        syncedParameters: synced,
        percentage: Math.round((batch / 5) * 100),
        status: batch >= 5 ? 'completed' : 'syncing',
        completedAt: batch >= 5 ? new Date().toISOString() : undefined,
      });
      if (batch >= 5) clearInterval(interval);
    }, 1500);
  },

  async discoverParameters(deviceId: string): Promise<void> {
    await delay(200, 400);
    // Reuses sync simulation
    return this.syncParameters(deviceId);
  },

  async getSyncStatus(deviceId: string): Promise<ParameterSyncStatus> {
    await delay(50, 100);
    return (
      syncStates.get(deviceId) ?? {
        deviceId,
        status: 'idle',
        totalBatches: 0,
        completedBatches: 0,
        totalParameters: 0,
        syncedParameters: 0,
        percentage: 0,
      }
    );
  },
};
