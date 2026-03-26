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

// ============================================================
// 性能测试：生成 30000 条参数数据
// ============================================================
const PERF_TEST_MODE = true;
const PERF_PARAM_COUNT = 30000;
const PERF_OBJECT_COUNT = 30000; // 对象树节点数量

// 基础参数模板（用于生成大量数据）
const baseParamTemplates = [
  // DeviceInfo
  { path: 'Device.DeviceInfo.Manufacturer', value: 'Baicells', type: 'string', writable: false },
  { path: 'Device.DeviceInfo.ModelName', value: 'Nova436Q', type: 'string', writable: false },
  { path: 'Device.DeviceInfo.SoftwareVersion', value: 'BaiBS_RTS_3.7.11.16', type: 'string', writable: false },
  { path: 'Device.DeviceInfo.HardwareVersion', value: 'V2.0', type: 'string', writable: false },
  { path: 'Device.DeviceInfo.SerialNumber', value: 'BAIC12345678', type: 'string', writable: false },
  { path: 'Device.DeviceInfo.UpTime', value: '1234567', type: 'unsignedInt', writable: false },
  { path: 'Device.DeviceInfo.MemoryStatus.Total', value: '524288', type: 'unsignedInt', writable: false },
  { path: 'Device.DeviceInfo.MemoryStatus.Free', value: '262144', type: 'unsignedInt', writable: false },
  { path: 'Device.DeviceInfo.ProcessStatus.CPUUsage', value: '45', type: 'unsignedInt', writable: false },
  // ManagementServer
  { path: 'Device.ManagementServer.URL', value: 'http://acs.example.com:7547', type: 'string', writable: true },
  { path: 'Device.ManagementServer.PeriodicInformEnable', value: 'true', type: 'boolean', writable: true },
  { path: 'Device.ManagementServer.PeriodicInformInterval', value: '300', type: 'unsignedInt', writable: true },
  { path: 'Device.ManagementServer.ConnectionRequestURL', value: 'http://192.168.1.100:7547/cr', type: 'string', writable: false },
  { path: 'Device.ManagementServer.Username', value: 'cpe_user', type: 'string', writable: true },
  // Time
  { path: 'Device.Time.NTPServer1', value: 'ntp.aliyun.com', type: 'string', writable: true },
  { path: 'Device.Time.LocalTimeZone', value: 'CST-8', type: 'string', writable: true },
  // LAN
  { path: 'Device.LAN.IPAddress', value: '192.168.1.1', type: 'string', writable: true },
  { path: 'Device.LAN.SubnetMask', value: '255.255.255.0', type: 'string', writable: true },
  { path: 'Device.LAN.DHCPServerEnable', value: 'true', type: 'boolean', writable: true },
];

// FAPService 参数模板（可多实例）
const fapServiceParamTemplates = [
  { path: 'CellConfig.LTE.RAN.RF.EARFCNDL', value: '38400', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.RAN.RF.EARFCNUL', value: '38400', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.RAN.RF.FreqBandIndicator', value: '41', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.RAN.RF.DLBandwidth', value: '20MHz', type: 'string', writable: true },
  { path: 'CellConfig.LTE.RAN.RF.ULBandwidth', value: '20MHz', type: 'string', writable: true },
  { path: 'CellConfig.LTE.RAN.RF.ReferenceSignalPower', value: '15', type: 'int', writable: true },
  { path: 'CellConfig.LTE.RAN.RF.PAState', value: 'ON', type: 'string', writable: true },
  { path: 'CellConfig.LTE.RAN.Common.CellIdentity', value: '1', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.RAN.Common.TAC', value: '1', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.RAN.Common.PCI', value: '100', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.RAN.Common.Bandwidth', value: '20', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.EPC.PLMNList.PLMNID', value: '46000', type: 'string', writable: true },
  { path: 'CellConfig.LTE.EPC.PLMNList.MCC', value: '460', type: 'string', writable: true },
  { path: 'CellConfig.LTE.EPC.PLMNList.MNC', value: '00', type: 'string', writable: true },
  { path: 'CellConfig.LTE.EPC.TAC', value: '1', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.EPC.CellIdentity', value: '12345', type: 'unsignedInt', writable: true },
  { path: 'CellConfig.LTE.EPC.SFN', value: '0', type: 'unsignedInt', writable: false },
  { path: 'Status.LTE.RF.TxPower', value: '23', type: 'int', writable: false },
  { path: 'Status.LTE.RF.RxPower', value: '-65', type: 'int', writable: false },
  { path: 'Status.LTE.RAN.NumUEs', value: '5', type: 'unsignedInt', writable: false },
  { path: 'Status.LTE.RAN.NumActiveUEs', value: '2', type: 'unsignedInt', writable: false },
  { path: 'Status.LTE.EPC.S1APState', value: 'CONNECTED', type: 'string', writable: false },
  { path: 'Status.LTE.EPC.MMECount', value: '1', type: 'unsignedInt', writable: false },
];

// 5G NR 参数模板
const nrParamTemplates = [
  { path: 'RAN.RF.ARFNLDL', value: '520000', type: 'unsignedInt', writable: true },
  { path: 'RAN.RF.ARFNUL', value: '520000', type: 'unsignedInt', writable: true },
  { path: 'RAN.RF.Band', value: 'n78', type: 'string', writable: true },
  { path: 'RAN.RF.Bandwidth', value: '100', type: 'unsignedInt', writable: true },
  { path: 'RAN.RF.TxPower', value: '23', type: 'int', writable: true },
  { path: 'RAN.RF.PCI', value: '200', type: 'unsignedInt', writable: true },
  { path: 'RAN.Common.NGCI', value: '4600012345678901', type: 'string', writable: true },
  { path: 'RAN.Common.TAC', value: '10001', type: 'unsignedInt', writable: true },
  { path: 'EPC.PLMNList.MCC', value: '460', type: 'string', writable: true },
  { path: 'EPC.PLMNList.MNC', value: '11', type: 'string', writable: true },
  { path: 'EPC.NGAPState', value: 'CONNECTED', type: 'string', writable: false },
  { path: 'EPC.AMFCount', value: '1', type: 'unsignedInt', writable: false },
  { path: 'Status.NumUEs', value: '10', type: 'unsignedInt', writable: false },
  { path: 'Status.NumActiveUEs', value: '5', type: 'unsignedInt', writable: false },
];

// IP Interface 参数模板
const ipInterfaceTemplates = [
  { path: 'Enable', value: 'true', type: 'boolean', writable: true },
  { path: 'IPv4Address.IPAddress', value: '10.20.30.40', type: 'string', writable: true },
  { path: 'IPv4Address.SubnetMask', value: '255.255.255.0', type: 'string', writable: true },
  { path: 'IPv4Address.AddressingType', value: 'DHCP', type: 'string', writable: true },
  { path: 'IPv6Enable', value: 'false', type: 'boolean', writable: true },
  { path: 'IPv6Address.IPAddress', value: 'fe80::1', type: 'string', writable: true },
  { path: 'IPv6Address.PrefixLength', value: '64', type: 'unsignedInt', writable: true },
  { path: 'Stats.BytesSent', value: '1234567890', type: 'unsignedInt', writable: false },
  { path: 'Stats.BytesReceived', value: '9876543210', type: 'unsignedInt', writable: false },
  { path: 'Stats.PacketsSent', value: '1000000', type: 'unsignedInt', writable: false },
  { path: 'Stats.PacketsReceived', value: '2000000', type: 'unsignedInt', writable: false },
];

// WiFi 参数模板
const wifiTemplates = [
  { path: 'Enable', value: 'true', type: 'boolean', writable: true },
  { path: 'SSID', value: 'Baicells_5G', type: 'string', writable: true },
  { path: 'BeaconType', value: 'WPA2', type: 'string', writable: true },
  { path: 'Channel', value: '36', type: 'unsignedInt', writable: true },
  { path: 'AutoChannelEnable', value: 'true', type: 'boolean', writable: true },
  { path: 'TransmitPower', value: '20', type: 'int', writable: true },
  { path: 'Stats.BytesSent', value: '500000000', type: 'unsignedInt', writable: false },
  { path: 'Stats.BytesReceived', value: '800000000', type: 'unsignedInt', writable: false },
  { path: 'Stats.AssociatedDevices', value: '3', type: 'unsignedInt', writable: false },
];

// 生成随机值
function randomValue(type: string): string {
  switch (type) {
    case 'boolean': return Math.random() > 0.5 ? 'true' : 'false';
    case 'unsignedInt': return Math.floor(Math.random() * 1000000).toString();
    case 'int': return Math.floor(Math.random() * 200 - 100).toString();
    case 'string':
    default: return `value_${Math.floor(Math.random() * 10000)}`;
  }
}

// 生成所有 mock 参数
let mockParameters: DeviceParameter[];
let cachedTree: ParameterTreeNode[] | null = null;

function generateMockParameters(): DeviceParameter[] {
  if (mockParameters) return mockParameters;

  const params: DeviceParameter[] = [];
  let id = 1;
  const now = new Date().toISOString();

  if (!PERF_TEST_MODE) {
    // 非性能测试模式：使用基础参数
    for (const t of baseParamTemplates) {
      params.push({
        id: String(id++),
        deviceId: '',
        parameterPath: t.path,
        parameterValue: t.value,
        parameterType: t.type,
        writable: t.writable,
        lastUpdatedAt: now,
      });
    }
  } else {
    // 性能测试模式：生成 30000 条参数
    // 1. DeviceInfo (50 条)
    for (let i = 0; i < 50; i++) {
      params.push({
        id: String(id++),
        deviceId: '',
        parameterPath: `Device.DeviceInfo.Param${i}`,
        parameterValue: randomValue('string'),
        parameterType: 'string',
        writable: i % 3 === 0,
        lastUpdatedAt: now,
      });
    }

    // 2. ManagementServer (30 条)
    for (let i = 0; i < 30; i++) {
      params.push({
        id: String(id++),
        deviceId: '',
        parameterPath: `Device.ManagementServer.Param${i}`,
        parameterValue: randomValue('string'),
        parameterType: i % 2 === 0 ? 'string' : 'unsignedInt',
        writable: true,
        lastUpdatedAt: now,
      });
    }

    // 3. IP.Interfaces (2000 条 = 4 interfaces × 500 params)
    for (let iface = 1; iface <= 4; iface++) {
      for (const t of ipInterfaceTemplates) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.IP.Interface.${iface}.${t.path}`,
          parameterValue: t.value,
          parameterType: t.type,
          writable: t.writable,
          lastUpdatedAt: now,
        });
      }
      // 额外参数
      for (let i = 0; i < 240; i++) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.IP.Interface.${iface}.Stats.Counter${i}`,
          parameterValue: randomValue('unsignedInt'),
          parameterType: 'unsignedInt',
          writable: false,
          lastUpdatedAt: now,
        });
      }
    }

    // 4. WiFi (2000 条 = 4 SSIDs × 500 params)
    for (let ssid = 1; ssid <= 4; ssid++) {
      for (const t of wifiTemplates) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.WiFi.SSID.${ssid}.${t.path}`,
          parameterValue: t.value,
          parameterType: t.type,
          writable: t.writable,
          lastUpdatedAt: now,
        });
      }
      // AssociatedDevice 参数
      for (let dev = 1; dev <= 240; dev++) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.WiFi.SSID.${ssid}.AssociatedDevice.${dev}.MACAddress`,
          parameterValue: `00:11:22:33:44:${String(dev).padStart(2, '0')}`,
          parameterType: 'string',
          writable: false,
          lastUpdatedAt: now,
        });
      }
    }

    // 5. FAPService LTE (10000 条 = 2 FAP × 5000 params)
    for (let fap = 1; fap <= 2; fap++) {
      for (const t of fapServiceParamTemplates) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.Services.FAPService.${fap}.${t.path}`,
          parameterValue: t.value,
          parameterType: t.type,
          writable: t.writable,
          lastUpdatedAt: now,
        });
      }
      // PLMNList (6 PLMNs × 20 params)
      for (let plmn = 1; plmn <= 6; plmn++) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.EPC.PLMNList.${plmn}.PLMNID`,
          parameterValue: `4600${plmn}`,
          parameterType: 'string',
          writable: true,
          lastUpdatedAt: now,
        });
        for (let i = 0; i < 18; i++) {
          params.push({
            id: String(id++),
            deviceId: '',
            parameterPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.EPC.PLMNList.${plmn}.Param${i}`,
            parameterValue: randomValue('string'),
            parameterType: 'string',
            writable: true,
            lastUpdatedAt: now,
          });
        }
      }
      // 额外 Cell 参数
      for (let i = 0; i < 2400; i++) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.CellParam.${i}`,
          parameterValue: randomValue(i % 2 === 0 ? 'unsignedInt' : 'string'),
          parameterType: i % 2 === 0 ? 'unsignedInt' : 'string',
          writable: i % 3 !== 0,
          lastUpdatedAt: now,
        });
      }
      // Status 参数
      for (let i = 0; i < 2000; i++) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.Services.FAPService.${fap}.Status.Counter${i}`,
          parameterValue: randomValue('unsignedInt'),
          parameterType: 'unsignedInt',
          writable: false,
          lastUpdatedAt: now,
        });
      }
    }

    // 6. 5G NR FAPService (10000 条 = 2 NR × 5000 params)
    for (let nr = 1; nr <= 2; nr++) {
      for (const t of nrParamTemplates) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.Services.FAPService.${2 + nr}.CellConfig.NR.${t.path}`,
          parameterValue: t.value,
          parameterType: t.type,
          writable: t.writable,
          lastUpdatedAt: now,
        });
      }
      // PLMNList
      for (let plmn = 1; plmn <= 6; plmn++) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.Services.FAPService.${2 + nr}.CellConfig.NR.EPC.PLMNList.${plmn}.PLMNID`,
          parameterValue: `4601${plmn}`,
          parameterType: 'string',
          writable: true,
          lastUpdatedAt: now,
        });
        for (let i = 0; i < 18; i++) {
          params.push({
            id: String(id++),
            deviceId: '',
            parameterPath: `Device.Services.FAPService.${2 + nr}.CellConfig.NR.EPC.PLMNList.${plmn}.Param${i}`,
            parameterValue: randomValue('string'),
            parameterType: 'string',
            writable: true,
            lastUpdatedAt: now,
          });
        }
      }
      // 额外 NR 参数
      for (let i = 0; i < 2400; i++) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.Services.FAPService.${2 + nr}.CellConfig.NR.CellParam.${i}`,
          parameterValue: randomValue(i % 2 === 0 ? 'unsignedInt' : 'string'),
          parameterType: i % 2 === 0 ? 'unsignedInt' : 'string',
          writable: i % 3 !== 0,
          lastUpdatedAt: now,
        });
      }
      // Status 参数
      for (let i = 0; i < 2000; i++) {
        params.push({
          id: String(id++),
          deviceId: '',
          parameterPath: `Device.Services.FAPService.${2 + nr}.Status.NRCounter${i}`,
          parameterValue: randomValue('unsignedInt'),
          parameterType: 'unsignedInt',
          writable: false,
          lastUpdatedAt: now,
        });
      }
    }

    // 7. 其他参数填充到 30000
    const remaining = PERF_PARAM_COUNT - params.length;
    for (let i = 0; i < remaining; i++) {
      const category = i % 5;
      let path: string;
      switch (category) {
        case 0: path = `Device.GatewayInfo.Param${i}`; break;
        case 1: path = `Device.RouterInfo.Param${i}`; break;
        case 2: path = `Device.Firewall.Param${i}`; break;
        case 3: path = `Device.DHCP.Param${i}`; break;
        default: path = `Device.Custom.Param${i}`; break;
      }
      params.push({
        id: String(id++),
        deviceId: '',
        parameterPath: path,
        parameterValue: randomValue('string'),
        parameterType: 'string',
        writable: i % 4 === 0,
        lastUpdatedAt: now,
      });
    }
  }

  mockParameters = params;
  console.log(`[Mock] Generated ${params.length} parameters for performance test`);
  return params;
}

// ============================================================
// 性能测试：生成 30000 个对象树节点
// ============================================================
let cachedObjectTree: ParameterTreeNode[] | null = null;

function generateObjectTree(): ParameterTreeNode[] {
  if (cachedObjectTree) return cachedObjectTree;

  if (!PERF_TEST_MODE) {
    // 非性能测试模式：从参数构建树
    return buildMockTreeFromParams();
  }

  console.log('[Mock] Generating 30000 object tree nodes for performance test...');

  // 创建根节点的子节点
  const rootChildren: ParameterTreeNode[] = [];
  let objectCount = 0;

  // 1. DeviceInfo (1 个对象)
  rootChildren.push({
    name: 'DeviceInfo',
    fullPath: 'Device.DeviceInfo.',
    isObject: true,
    children: [],
  });
  objectCount++;

  // 2. ManagementServer (1 个对象)
  rootChildren.push({
    name: 'ManagementServer',
    fullPath: 'Device.ManagementServer.',
    isObject: true,
    children: [],
  });
  objectCount++;

  // 3. Time (1 个对象)
  rootChildren.push({
    name: 'Time',
    fullPath: 'Device.Time.',
    isObject: true,
    children: [],
  });
  objectCount++;

  // 4. IP.Interfaces (100 个接口对象)
  const ipInterfaces: ParameterTreeNode = {
    name: 'IP',
    fullPath: 'Device.IP.',
    isObject: true,
    children: [],
  };
  objectCount++;

  const interfacesNode: ParameterTreeNode = {
    name: 'Interface',
    fullPath: 'Device.IP.Interface.',
    isObject: true,
    multiInstance: true,
    maxInstances: 100,
    minInstances: 1,
    instanceCount: 100,
    canAdd: true,
    canDelete: true,
    children: [],
  };
  objectCount++;

  for (let i = 1; i <= 100; i++) {
    const ifaceNode: ParameterTreeNode = {
      name: String(i),
      fullPath: `Device.IP.Interface.${i}.`,
      isObject: true,
      children: [],
    };
    objectCount++;

    // 每个接口下有 IPv4Address 和 Stats 对象
    ifaceNode.children = [
      {
        name: 'IPv4Address',
        fullPath: `Device.IP.Interface.${i}.IPv4Address.`,
        isObject: true,
        multiInstance: true,
        maxInstances: 5,
        instanceCount: 2,
        canAdd: true,
        canDelete: true,
        children: [
          {
            name: '1',
            fullPath: `Device.IP.Interface.${i}.IPv4Address.1.`,
            isObject: true,
            children: [],
          },
          {
            name: '2',
            fullPath: `Device.IP.Interface.${i}.IPv4Address.2.`,
            isObject: true,
            children: [],
          },
        ],
      },
      objectCount += 4,
      {
        name: 'Stats',
        fullPath: `Device.IP.Interface.${i}.Stats.`,
        isObject: true,
        children: [],
      },
      objectCount++,
    ];

    interfacesNode.children!.push(ifaceNode);
  }
  ipInterfaces.children = [interfacesNode];
  rootChildren.push(ipInterfaces);

  // 5. WiFi (50 个 SSID 对象)
  const wifiNode: ParameterTreeNode = {
    name: 'WiFi',
    fullPath: 'Device.WiFi.',
    isObject: true,
    children: [],
  };
  objectCount++;

  const ssidNode: ParameterTreeNode = {
    name: 'SSID',
    fullPath: 'Device.WiFi.SSID.',
    isObject: true,
    multiInstance: true,
    maxInstances: 50,
    minInstances: 1,
    instanceCount: 50,
    canAdd: true,
    canDelete: true,
    children: [],
  };
  objectCount++;

  for (let i = 1; i <= 50; i++) {
    const ssid: ParameterTreeNode = {
      name: String(i),
      fullPath: `Device.WiFi.SSID.${i}.`,
      isObject: true,
      children: [
        {
          name: 'Stats',
          fullPath: `Device.WiFi.SSID.${i}.Stats.`,
          isObject: true,
          children: [],
        },
        {
          name: 'AssociatedDevice',
          fullPath: `Device.WiFi.SSID.${i}.AssociatedDevice.`,
          isObject: true,
          multiInstance: true,
          maxInstances: 50,
          instanceCount: 20,
          canAdd: true,
          canDelete: true,
          children: [],
        },
      ],
    };
    objectCount += 3;

    // 每个 SSID 下有 20 个关联设备
    for (let d = 1; d <= 20; d++) {
      ssid.children![1].children!.push({
        name: String(d),
        fullPath: `Device.WiFi.SSID.${i}.AssociatedDevice.${d}.`,
        isObject: true,
        children: [],
      });
      objectCount++;
    }

    ssidNode.children!.push(ssid);
  }
  wifiNode.children = [ssidNode];
  rootChildren.push(wifiNode);

  // 6. Services.FAPService - 主要的大数据源 (25000+ 对象)
  const servicesNode: ParameterTreeNode = {
    name: 'Services',
    fullPath: 'Device.Services.',
    isObject: true,
    children: [],
  };
  objectCount++;

  const fapServiceNode: ParameterTreeNode = {
    name: 'FAPService',
    fullPath: 'Device.Services.FAPService.',
    isObject: true,
    multiInstance: true,
    maxInstances: 100,
    minInstances: 1,
    instanceCount: 100,
    canAdd: true,
    canDelete: false,
    children: [],
  };
  objectCount++;

  // 生成 100 个 FAPService 实例
  for (let fap = 1; fap <= 100; fap++) {
    const fapNode: ParameterTreeNode = {
      name: String(fap),
      fullPath: `Device.Services.FAPService.${fap}.`,
      isObject: true,
      children: [],
    };
    objectCount++;

    // CellConfig.LTE 层级
    const cellConfigNode: ParameterTreeNode = {
      name: 'CellConfig',
      fullPath: `Device.Services.FAPService.${fap}.CellConfig.`,
      isObject: true,
      children: [],
    };
    objectCount++;

    const lteNode: ParameterTreeNode = {
      name: 'LTE',
      fullPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.`,
      isObject: true,
      children: [],
    };
    objectCount++;

    // RAN 层级
    const ranNode: ParameterTreeNode = {
      name: 'RAN',
      fullPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.RAN.`,
      isObject: true,
      children: [
        { name: 'RF', fullPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.RAN.RF.`, isObject: true, children: [] },
        { name: 'Common', fullPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.RAN.Common.`, isObject: true, children: [] },
      ],
    };
    objectCount += 3;

    // EPC 层级
    const epcNode: ParameterTreeNode = {
      name: 'EPC',
      fullPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.EPC.`,
      isObject: true,
      children: [],
    };
    objectCount++;

    // PLMNList - 每个 FAP 有 6 个 PLMN
    const plmnListNode: ParameterTreeNode = {
      name: 'PLMNList',
      fullPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.EPC.PLMNList.`,
      isObject: true,
      multiInstance: true,
      maxInstances: 6,
      minInstances: 1,
      instanceCount: 6,
      canAdd: true,
      canDelete: true,
      children: [],
    };
    objectCount++;

    for (let plmn = 1; plmn <= 6; plmn++) {
      plmnListNode.children!.push({
        name: String(plmn),
        fullPath: `Device.Services.FAPService.${fap}.CellConfig.LTE.EPC.PLMNList.${plmn}.`,
        isObject: true,
        children: [],
      });
      objectCount++;
    }
    epcNode.children = [plmnListNode];
    lteNode.children = [ranNode, epcNode];
    cellConfigNode.children = [lteNode];

    // Status 层级 - 包含大量子对象
    const statusNode: ParameterTreeNode = {
      name: 'Status',
      fullPath: `Device.Services.FAPService.${fap}.Status.`,
      isObject: true,
      children: [],
    };
    objectCount++;

    // LTE Status 子对象
    const lteStatusNode: ParameterTreeNode = {
      name: 'LTE',
      fullPath: `Device.Services.FAPService.${fap}.Status.LTE.`,
      isObject: true,
      children: [
        { name: 'RF', fullPath: `Device.Services.FAPService.${fap}.Status.LTE.RF.`, isObject: true, children: [] },
        { name: 'RAN', fullPath: `Device.Services.FAPService.${fap}.Status.LTE.RAN.`, isObject: true, children: [] },
        { name: 'EPC', fullPath: `Device.Services.FAPService.${fap}.Status.LTE.EPC.`, isObject: true, children: [] },
      ],
    };
    objectCount += 4;

    // UEList - 每个 FAP 有 50 个 UE 对象
    const ueListNode: ParameterTreeNode = {
      name: 'UEList',
      fullPath: `Device.Services.FAPService.${fap}.Status.LTE.UEList.`,
      isObject: true,
      multiInstance: true,
      maxInstances: 200,
      instanceCount: 50,
      canAdd: true,
      canDelete: true,
      children: [],
    };
    objectCount++;

    for (let ue = 1; ue <= 50; ue++) {
      ueListNode.children!.push({
        name: String(ue),
        fullPath: `Device.Services.FAPService.${fap}.Status.LTE.UEList.${ue}.`,
        isObject: true,
        children: [
          { name: 'Stats', fullPath: `Device.Services.FAPService.${fap}.Status.LTE.UEList.${ue}.Stats.`, isObject: true, children: [] },
          { name: 'Bearers', fullPath: `Device.Services.FAPService.${fap}.Status.LTE.UEList.${ue}.Bearers.`, isObject: true, children: [] },
        ],
      });
      objectCount += 3;
    }

    lteStatusNode.children!.push(ueListNode);
    statusNode.children = [lteStatusNode];

    // CounterGroups - 统计计数器组 (每个 FAP 生成大量对象)
    const counterGroupsNode: ParameterTreeNode = {
      name: 'CounterGroups',
      fullPath: `Device.Services.FAPService.${fap}.Status.CounterGroups.`,
      isObject: true,
      children: [],
    };
    objectCount++;

    // 生成 20 个计数器组，每组 10 个计数器对象
    for (let g = 1; g <= 20; g++) {
      const groupNode: ParameterTreeNode = {
        name: `Group${g}`,
        fullPath: `Device.Services.FAPService.${fap}.Status.CounterGroups.Group${g}.`,
        isObject: true,
        children: [],
      };
      objectCount++;

      for (let c = 1; c <= 10; c++) {
        groupNode.children!.push({
          name: `Counter${c}`,
          fullPath: `Device.Services.FAPService.${fap}.Status.CounterGroups.Group${g}.Counter${c}.`,
          isObject: true,
          children: [],
        });
        objectCount++;
      }
      counterGroupsNode.children!.push(groupNode);
    }
    statusNode.children!.push(counterGroupsNode);

    fapNode.children = [cellConfigNode, statusNode];
    fapServiceNode.children!.push(fapNode);
  }

  servicesNode.children = [fapServiceNode];
  rootChildren.push(servicesNode);

  // 7. GatewayInfo, RouterInfo, Firewall, DHCP 等 (填充剩余对象)
  const miscCategories = ['GatewayInfo', 'RouterInfo', 'Firewall', 'DHCP', 'DNS', 'NAT', 'QoS', 'Users'];
  for (const cat of miscCategories) {
    const catNode: ParameterTreeNode = {
      name: cat,
      fullPath: `Device.${cat}.`,
      isObject: true,
      children: [],
    };
    objectCount++;

    // 每个类别下添加 10 个子对象
    for (let i = 1; i <= 10; i++) {
      catNode.children!.push({
        name: `Entry${i}`,
        fullPath: `Device.${cat}.Entry${i}.`,
        isObject: true,
        children: [
          { name: 'Config', fullPath: `Device.${cat}.Entry${i}.Config.`, isObject: true, children: [] },
          { name: 'Stats', fullPath: `Device.${cat}.Entry${i}.Stats.`, isObject: true, children: [] },
        ],
      });
      objectCount += 3;
    }
    rootChildren.push(catNode);
  }

  console.log(`[Mock] Generated ${objectCount} object tree nodes`);

  cachedObjectTree = rootChildren;
  return cachedObjectTree;
}

// 从扁平参数构建树结构（非性能测试模式）
function buildMockTreeFromParams(): ParameterTreeNode[] {
  const params = generateMockParameters();
  const root: ParameterTreeNode = {
    name: 'Device',
    fullPath: 'Device.',
    isObject: true,
    children: [],
  };

  for (const param of params) {
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

  // Enrich FAPService with multi-instance metadata
  const services = root.children?.find((c) => c.name === 'Services');
  if (services?.children) {
    for (const fap of services.children) {
      if (fap.name === 'FAPService') {
        fap.multiInstance = true;
        fap.maxInstances = 4;
        fap.minInstances = 1;
        fap.instanceCount = 4;
        fap.canAdd = true;
        fap.canDelete = false;
      }
    }
  }

  return root.children ?? [];
}

// 从扁平参数构建树结构
function buildMockTree(): ParameterTreeNode[] {
  if (cachedTree) return cachedTree;
  cachedTree = generateObjectTree();
  return cachedTree;
}

// Track sync state per device
const syncStates = new Map<string, ParameterSyncStatus>();

export const deviceParameterService = {
  async getParameters(
    deviceId: string,
    params?: ParameterFilter & PageRequest
  ): Promise<PageResponse<DeviceParameter>> {
    await delay(100, 300);
    const allParams = generateMockParameters();
    let filtered = allParams.map((p) => ({ ...p, deviceId }));

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
    const allParams = generateMockParameters();
    for (const update of parameters) {
      const existing = allParams.find((p) => p.parameterPath === update.parameterPath);
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
    const allParams = generateMockParameters();
    syncStates.set(deviceId, {
      deviceId,
      status: 'syncing',
      totalBatches: 5,
      completedBatches: 0,
      totalParameters: allParams.length,
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
        Math.round((batch / 5) * allParams.length),
        allParams.length
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
    const allParams = generateMockParameters();
    let filtered = allParams;
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
    // 直接返回性能测试对象树（已全部是对象节点）
    return generateObjectTree();
  },

  async getDirectChildren(
    deviceId: string,
    pathPrefix: string,
    params?: { page?: number; pageSize?: number }
  ): Promise<DirectChildrenResponse> {
    await delay(100, 300);
    void deviceId;
    const allParams = generateMockParameters();
    const prefix = pathPrefix.endsWith('.') ? pathPrefix : pathPrefix + '.';
    // Find direct leaf children under pathPrefix
    const children: ChildParameter[] = allParams
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
