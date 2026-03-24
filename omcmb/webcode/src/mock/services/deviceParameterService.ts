import type {
  DeviceParameter,
  ParameterTreeNode,
  ParameterSyncStatus,
  ParameterFilter,
  ParameterUpdateRequest,
  ParameterSyncOptions,
  ParameterSchemaResponse,
  ParameterUpdateResponse,
  ChildParameter,
  DirectChildrenResponse,
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

  // Enrich FAPService and PLMNList with multi-instance metadata
  const services = root.children?.find((c) => c.name === 'Services');
  if (services) {
    const fapService = services.children?.find((c) => c.name === 'FAPService');
    if (fapService) {
      fapService.multiInstance = true;
      fapService.maxInstances = 4;
      fapService.minInstances = 1;
      fapService.instanceCount = 1;
      fapService.canAdd = true;
      fapService.canDelete = false;
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
  ): Promise<ParameterUpdateResponse> {
    await delay(300, 600);
    for (const update of parameters) {
      const existing = mockParameters.find((p) => p.parameterPath === update.parameterPath);
      if (existing) {
        existing.parameterValue = update.parameterValue;
        existing.lastUpdatedAt = new Date().toISOString();
      }
    }
    void deviceId;
    return {
      message: 'set parameter values command queued',
      parameters: parameters.length,
      rebootRequired: false,
    };
  },

  async syncParameters(
    deviceId: string,
    _options?: ParameterSyncOptions
  ): Promise<void> {
    await delay(100, 200);
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

  async getParameterSchema(
    deviceId: string,
    pathPrefix?: string
  ): Promise<ParameterSchemaResponse> {
    await delay(200, 400);
    let filtered = mockParameters;
    if (pathPrefix) {
      filtered = filtered.filter((p) => p.parameterPath.startsWith(pathPrefix));
    }

    return {
      parameters: filtered.map((p) => ({
        path: p.parameterPath,
        type: p.parameterType,
        writable: p.writable,
        description: `Parameter ${p.parameterPath.split('.').pop()}`,
        currentValue: p.parameterValue,
        lastSyncedAt: p.lastUpdatedAt,
      })),
      objects: [
        {
          path: 'Device.Services.FAPService.',
          access: 'READ_WRITE',
          maxInstances: 4,
          minInstances: 1,
          currentInstances: [1],
          canAdd: true,
          canDeleteAny: false,
          isList: true,
        },
        {
          path: 'Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.',
          access: 'READ_WRITE',
          maxInstances: 6,
          minInstances: 1,
          currentInstances: [1],
          canAdd: true,
          canDeleteAny: false,
          isList: true,
        },
      ],
      total: filtered.length,
    };
    void deviceId;
  },

  async getObjectTree(deviceId: string): Promise<ParameterTreeNode[]> {
    await delay(200, 500);
    void deviceId;
    // Return only object nodes from the mock tree
    function filterObjects(nodes: ParameterTreeNode[]): ParameterTreeNode[] {
      return nodes
        .filter((n) => n.isObject)
        .map((n) => ({
          ...n,
          children: n.children ? filterObjects(n.children) : undefined,
        }));
    }
    return filterObjects(buildMockTree());
  },

  async getDirectChildren(
    deviceId: string,
    pathPrefix: string,
    params?: { page?: number; pageSize?: number }
  ): Promise<DirectChildrenResponse> {
    await delay(100, 300);
    void deviceId;
    const prefix = pathPrefix.endsWith('.') ? pathPrefix : pathPrefix + '.';
    // Find direct leaf children under pathPrefix
    const children: ChildParameter[] = mockParameters
      .filter((p) => {
        if (!p.parameterPath.startsWith(prefix)) return false;
        const remainder = p.parameterPath.slice(prefix.length);
        return !remainder.includes('.');
      })
      .map((p) => ({
        parameterPath: p.parameterPath,
        parameterValue: p.parameterValue,
        parameterType: p.parameterType,
        writable: p.writable,
        lastUpdatedAt: p.lastUpdatedAt,
        description: `Parameter ${p.parameterPath.split('.').pop()}`,
      }));
    const page = params?.page ?? 1;
    const pageSize = params?.pageSize ?? 50;
    const paged = paginate(children, page, pageSize);
    return { ...paged, subObjects: [] };
  },

  async addObject(deviceId: string, objectPath: string): Promise<void> {
    await delay(300, 500);
    void deviceId;
    void objectPath;
  },

  async deleteObject(deviceId: string, objectPath: string): Promise<void> {
    await delay(300, 500);
    void deviceId;
    void objectPath;
  },
};
