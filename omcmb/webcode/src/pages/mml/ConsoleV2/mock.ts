// MML 控制台 V2 —— mock 数据。
//
// 当前阶段（设计 §5 P0）为纯前端布局重排，后端 results-schema / export 端点尚未就绪，
// 因此设备列表、命令树、执行结果全部由本文件提供。待 P1 接入真实端点后，本文件可删除，
// index.tsx 改为消费 useMML* hooks。

import type {
  CommandItem,
  DeviceItem,
  DeviceStatus,
  ResultColumn,
  ResultRow,
} from './types';

const PRODUCTS = ['Baicells Nova-436', 'Baicells Nova-227', 'Comba X1-Pro', 'Baicells pBS3101'];
const PRODUCT_CLASSES = ['ENB', 'CPE', 'GNB'];
const GROUPS = ['默认设备组', '移动设备域', '电信测试域', '联通试点'];
const STATUSES: DeviceStatus[] = ['online', 'online', 'online', 'offline', 'alarm'];

/** 生成 60 台 mock 设备（确定性，便于分页/筛选演示）。 */
export const MOCK_DEVICES: DeviceItem[] = Array.from({ length: 60 }, (_, i) => {
  const seq = String(i + 1).padStart(4, '0');
  return {
    sn: `1202000240194DP${seq}`,
    productName: PRODUCTS[i % PRODUCTS.length],
    productClass: PRODUCT_CLASSES[i % PRODUCT_CLASSES.length],
    status: STATUSES[i % STATUSES.length],
    groupName: GROUPS[i % GROUPS.length],
  };
});

/** mock 命令树（两层：分组 → 命令）。 */
export const MOCK_COMMANDS: CommandItem[] = [
  {
    id: 'cmd-lst-devinfo',
    groupName: '设备信息参数管理',
    commandCode: 'LST DEVINFO',
    commandName: '查询设备基本信息',
    operationType: 'LST',
    description: '查询设备厂商、型号、软硬件版本、运行时长等基础信息。',
    paramPaths: [
      { path: 'Device.DeviceInfo.Manufacturer', label: '厂商', writable: false },
      { path: 'Device.DeviceInfo.ModelName', label: '型号', writable: false },
      { path: 'Device.DeviceInfo.SoftwareVersion', label: '软件版本', writable: false },
      { path: 'Device.DeviceInfo.UpTime', label: '运行时长(s)', writable: false },
    ],
  },
  {
    id: 'cmd-mod-devinfo',
    groupName: '设备信息参数管理',
    commandCode: 'MOD DEVINFO',
    commandName: '修改设备基本信息',
    operationType: 'MOD',
    description: '修改设备友好名称、位置描述等可写信息。',
    paramPaths: [
      { path: 'Device.DeviceInfo.FriendlyName', label: '友好名称', writable: true },
      { path: 'Device.DeviceInfo.Location', label: '位置描述', writable: true },
    ],
  },
  {
    id: 'cmd-lst-cell',
    groupName: '小区参数管理',
    commandCode: 'LST CELL',
    commandName: '查询小区配置',
    operationType: 'LST',
    description: '查询小区 PCI、频点、带宽、发射功率等无线参数。',
    paramPaths: [
      { path: 'Device.Cellular.Cell.1.PCI', label: 'PCI', writable: false },
      { path: 'Device.Cellular.Cell.1.EARFCN', label: '频点', writable: false },
      { path: 'Device.Cellular.Cell.1.Bandwidth', label: '带宽(MHz)', writable: false },
      { path: 'Device.Cellular.Cell.1.TxPower', label: '发射功率(dBm)', writable: false },
    ],
  },
  {
    id: 'cmd-mod-cell',
    groupName: '小区参数管理',
    commandCode: 'MOD CELL',
    commandName: '修改小区功率',
    operationType: 'MOD',
    description: '修改小区参考信号发射功率（需谨慎，影响覆盖）。',
    paramPaths: [
      { path: 'Device.Cellular.Cell.1.TxPower', label: '发射功率(dBm)', writable: true },
    ],
  },
  {
    id: 'cmd-add-neighbor',
    groupName: '邻区参数管理',
    commandCode: 'ADD NCELL',
    commandName: '新增邻区',
    operationType: 'ADD',
    description: '为指定小区添加一条邻区关系。',
    paramPaths: [
      { path: 'Device.Cellular.Cell.1.Neighbor.{i}.PCI', label: '邻区PCI', writable: true },
      { path: 'Device.Cellular.Cell.1.Neighbor.{i}.EARFCN', label: '邻区频点', writable: true },
    ],
  },
  {
    id: 'cmd-rmv-neighbor',
    groupName: '邻区参数管理',
    commandCode: 'RMV NCELL',
    commandName: '删除邻区',
    operationType: 'RMV',
    description: '删除指定的邻区关系实例。',
    paramPaths: [
      { path: 'Device.Cellular.Cell.1.Neighbor.{i}.', label: '邻区实例', writable: true },
    ],
  },
  {
    id: 'cmd-lst-alarm',
    groupName: '告警参数管理',
    commandCode: 'LST ALARM',
    commandName: '查询活动告警',
    operationType: 'LST',
    description: '查询设备当前活动告警条目与等级。',
    paramPaths: [
      { path: 'Device.FaultMgmt.ActiveAlarm.1.AlarmType', label: '告警类型', writable: false },
      { path: 'Device.FaultMgmt.ActiveAlarm.1.Severity', label: '等级', writable: false },
      { path: 'Device.FaultMgmt.ActiveAlarm.1.Timestamp', label: '发生时间', writable: false },
    ],
  },
];

/** 由命令的 paramPaths 派生结果表格动态列；可选 checkedPaths 仅保留勾选的列。 */
export function buildColumns(command: CommandItem, checkedPaths?: string[]): ResultColumn[] {
  const checked = checkedPaths ? new Set(checkedPaths) : null;
  return command.paramPaths
    .filter((p) => !checked || checked.has(p.path))
    .map((p, idx) => ({ key: `c${idx}`, label: p.label, path: p.path }));
}

/** 由裸路径列表派生结果表格动态列（参数路径指定模式）。列标签取路径叶子名。 */
export function buildColumnsFromRawPaths(paths: string[]): ResultColumn[] {
  return paths
    .map((p) => p.trim())
    .filter(Boolean)
    .map((path, idx) => {
      const leaf = path.split('.').filter(Boolean).pop() || path;
      return { key: `c${idx}`, label: leaf, path };
    });
}

// ── 确定性 mock 取值（避免 Math.random，保证可复现） ─────────────────────────
function pseudo(seed: number): number {
  // 简易确定性散列：基于序号生成 [0,1)
  const x = Math.sin(seed * 12.9898) * 43758.5453;
  return x - Math.floor(x);
}

function mockValue(path: string, deviceIdx: number, colIdx: number): string {
  const seed = deviceIdx * 7 + colIdx * 13;
  if (path.includes('Manufacturer')) return 'Baicells';
  if (path.includes('ModelName')) return PRODUCTS[deviceIdx % PRODUCTS.length];
  if (path.includes('SoftwareVersion')) return `BaiOMC_${2 + (deviceIdx % 3)}.6.${deviceIdx % 10}`;
  if (path.includes('UpTime')) return String(100000 + Math.floor(pseudo(seed) * 900000));
  if (path.includes('PCI')) return String(Math.floor(pseudo(seed) * 503));
  if (path.includes('EARFCN')) return String(38950 + Math.floor(pseudo(seed) * 50));
  if (path.includes('Bandwidth')) return ['5', '10', '15', '20'][deviceIdx % 4];
  if (path.includes('TxPower')) return String(15 + Math.floor(pseudo(seed) * 30));
  if (path.includes('FriendlyName')) return `eNB-${deviceIdx + 1}`;
  if (path.includes('Location')) return `机房-${(deviceIdx % 12) + 1}F`;
  if (path.includes('AlarmType')) return ['链路中断', '温度过高', '电源异常'][deviceIdx % 3];
  if (path.includes('Severity')) return ['Critical', 'Major', 'Minor'][deviceIdx % 3];
  if (path.includes('Timestamp')) return '2026-06-04 0' + (deviceIdx % 9) + ':12:30';
  return `v${seed}`;
}

const FAULT_CODES = ['9005 参数越界', '9013 设备无响应', '9021 不支持的参数'];

const SOAP_NS =
  'xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" ' +
  'xmlns:cwmp="urn:dslforum-org:cwmp-1-0" ' +
  'xmlns:xsd="http://www.w3.org/2001/XMLSchema" ' +
  'xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"';

function xmlEscape(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function envelope(id: string, body: string): string {
  return (
    `<soap:Envelope ${SOAP_NS}>` +
    `<soap:Header><cwmp:ID soap:mustUnderstand="1">${id}</cwmp:ID></soap:Header>` +
    `<soap:Body>${body}</soap:Body>` +
    `</soap:Envelope>`
  );
}

/** GetParameterValuesResponse（LST/DSP 读结果，含 path/value/type）。 */
function gpvXml(columns: ResultColumn[], cells: Record<string, string>, sn: string): string {
  const structs = columns
    .map((c) => {
      const v = cells[c.path] ?? '';
      const type = /^\d+$/.test(v) ? 'xsd:unsignedInt' : 'xsd:string';
      return (
        `<ParameterValueStruct>` +
        `<Name>${c.path}</Name>` +
        `<Value xsi:type="${type}">${xmlEscape(v)}</Value>` +
        `</ParameterValueStruct>`
      );
    })
    .join('');
  return envelope(
    sn,
    `<cwmp:GetParameterValuesResponse>` +
      `<ParameterList soap:arrayType="cwmp:ParameterValueStruct[${columns.length}]">${structs}</ParameterList>` +
      `</cwmp:GetParameterValuesResponse>`,
  );
}

/** 写类操作响应（MOD/ADD/RMV）。 */
function writeXml(operationType: string, sn: string): string {
  let body: string;
  if (operationType === 'ADD') {
    body = `<cwmp:AddObjectResponse><InstanceNumber>1</InstanceNumber><Status>0</Status></cwmp:AddObjectResponse>`;
  } else if (operationType === 'RMV') {
    body = `<cwmp:DeleteObjectResponse><Status>0</Status></cwmp:DeleteObjectResponse>`;
  } else {
    body = `<cwmp:SetParameterValuesResponse><Status>0</Status></cwmp:SetParameterValuesResponse>`;
  }
  return envelope(sn, body);
}

/** CWMP SOAP Fault（执行失败）。 */
function faultXml(faultCode: string, sn: string): string {
  const [code, ...rest] = faultCode.split(' ');
  return envelope(
    sn,
    `<soap:Fault>` +
      `<faultcode>Client</faultcode><faultstring>CWMP fault</faultstring>` +
      `<detail><cwmp:Fault>` +
      `<FaultCode>${code}</FaultCode>` +
      `<FaultString>${xmlEscape(rest.join(' '))}</FaultString>` +
      `</cwmp:Fault></detail>` +
      `</soap:Fault>`,
  );
}

/**
 * 生成一次执行的 mock 结果行（列驱动，标准/裸路径两模式通用）。约 1/6 设备失败，
 * 用于演示失败行展开 + 状态过滤。`raw` 为格式化前的 CWMP SOAP XML（详情页用 XmlViewer 美化）。
 *
 * @param columns       结果表格动态列（标准模式来自命令勾选项，裸路径模式来自手输路径）
 * @param deviceSns     目标设备
 * @param read          是否「读」类操作（true 填参数值矩阵，false 填 ✓ 状态）
 * @param operationType 操作类型（决定写结果 XML 形态）
 */
export function buildResultRows(
  columns: ResultColumn[],
  deviceSns: string[],
  read: boolean,
  operationType: string,
): ResultRow[] {
  return deviceSns.map((sn, deviceIdx) => {
    const failed = deviceIdx % 6 === 5;
    const cells: Record<string, string> = {};
    if (!failed) {
      columns.forEach((col, colIdx) => {
        cells[col.path] = read ? mockValue(col.path, deviceIdx, colIdx) : '✓';
      });
    }
    const elapsedMs = 200 + Math.floor(pseudo(deviceIdx + 1) * 1800);
    const faultCode = failed ? FAULT_CODES[deviceIdx % FAULT_CODES.length] : undefined;
    return {
      deviceSn: sn,
      status: failed ? 'failed' : 'success',
      cells,
      faultCode,
      raw: failed
        ? faultXml(faultCode as string, sn)
        : read
          ? gpvXml(columns, cells, sn)
          : writeXml(operationType, sn),
      elapsedMs,
    };
  });
}
