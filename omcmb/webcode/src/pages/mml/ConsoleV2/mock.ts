// MML 控制台 V2 —— mock 数据。
//
// 当前阶段（设计 §5 P0）为纯前端布局重排，后端 results-schema / export 端点尚未就绪，
// 因此设备列表、命令树、执行结果全部由本文件提供。待 P1 接入真实端点后，本文件可删除，
// index.tsx 改为消费 useMML* hooks。

import type {
  CommandItem,
  DeviceItem,
  DeviceStatus,
  ExecRecord,
  ExecStatus,
  PathTask,
  ResultColumn,
  ResultRow,
  UnverifiedReason,
  VerifyItem,
} from './types';
import { isReadOp } from './constants';

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

/** 确定性伪 UUID（mock 用；真实由后端 device_tasks.id / mml_tasks.id 提供）。 */
function mockUuid(seed: string): string {
  let h = 2166136261;
  for (let i = 0; i < seed.length; i++) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  const next = (): number => {
    h ^= h << 13;
    h ^= h >>> 17;
    h ^= h << 5;
    return h >>> 0;
  };
  let hex = '';
  while (hex.length < 32) hex += next().toString(16).padStart(8, '0');
  hex = hex.slice(0, 32);
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20, 32)}`;
}

/** 确定性时钟 HH:mm:ss（基线 09:12:00 + 设备/阶段偏移），模拟下发/响应时间。 */
function mockClock(deviceIdx: number, phase: 0 | 1, extra = 0): string {
  const base = 9 * 3600 + 12 * 60; // 09:12:00
  const t = base + deviceIdx * 2 + phase * (1 + (deviceIdx % 3)) + extra;
  const hh = Math.floor(t / 3600) % 24;
  const mm = Math.floor((t % 3600) / 60);
  const ss = t % 60;
  const p = (n: number) => String(n).padStart(2, '0');
  return `${p(hh)}:${p(mm)}:${p(ss)}`;
}

/** 写类命令的 mock 场景（按设备序号确定性分桶，演示四态）。 */
type WriteScenario = 'success' | 'mismatch' | 'reboot' | 'write-only';
function writeScenario(deviceIdx: number): WriteScenario {
  // deviceIdx % 6 === 5 已在 buildResultRows 判为 RPC failed，这里覆盖 0..4
  const m = deviceIdx % 6;
  if (m === 2) return 'mismatch';
  if (m === 3) return 'reboot';
  if (m === 4) return 'write-only';
  return 'success';
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

/** 生成逐 PATH 子任务（详情页展示；父任务 = deviceTaskId，设计 §3.11.3）。 */
function buildPathTasks(
  columns: ResultColumn[],
  sn: string,
  deviceIdx: number,
  rowStatus: ExecStatus,
  cells: Record<string, string>,
): PathTask[] {
  return columns.map((col, colIdx) => ({
    pathIndex: colIdx,
    path: col.path,
    subTaskId: mockUuid(`st-${sn}-${colIdx}`),
    status: rowStatus,
    dispatchedAt: mockClock(deviceIdx, 0, colIdx),
    respondedAt: mockClock(deviceIdx, 1, colIdx),
    value: cells[col.path] ?? '',
  }));
}

/**
 * 生成一次执行的 mock 结果行（列驱动，标准/裸路径两模式通用；设计 §3.11.2 读后核实）。
 *
 * 状态分布（按设备序号确定性分桶）：约 1/6 设备 RPC 失败(failed)；其余写类按 §3.11.2 演示
 * success(已核实) / mismatch(未生效) / unverified(只写 · 重启生效)。读类仅 success/failed。
 * 每行带 deviceTaskId、下发/响应时间、逐 PATH 子任务；写类带核实对比 verify。
 * `raw` 为格式化前的 CWMP SOAP XML（详情页用 XmlViewer 美化）。
 *
 * @param columns       结果表格动态列（标准模式来自命令勾选项，裸路径模式来自手输路径）
 * @param deviceSns     目标设备
 * @param read          是否「读」类操作（true 填参数值矩阵；false 写类走读后核实）
 * @param operationType 操作类型（决定写结果 XML 形态）
 */
export function buildResultRows(
  columns: ResultColumn[],
  deviceSns: string[],
  read: boolean,
  operationType: string,
): ResultRow[] {
  return deviceSns.map((sn, deviceIdx) => {
    const deviceTaskId = mockUuid(`dt-${sn}-${deviceIdx}`);
    const dispatchedAt = mockClock(deviceIdx, 0);
    const respondedAt = mockClock(deviceIdx, 1);
    const elapsedMs = 200 + Math.floor(pseudo(deviceIdx + 1) * 1800);
    const rpcFailed = deviceIdx % 6 === 5;

    // 共有：RPC 本身失败（读写通用）
    if (rpcFailed) {
      const faultCode = FAULT_CODES[deviceIdx % FAULT_CODES.length];
      return {
        deviceSn: sn,
        deviceTaskId,
        status: 'failed',
        cells: {},
        faultCode,
        dispatchedAt,
        respondedAt,
        pathTasks: buildPathTasks(columns, sn, deviceIdx, 'failed', {}),
        raw: faultXml(faultCode, sn),
        elapsedMs,
      };
    }

    // 读类：参数值矩阵
    if (read) {
      const cells: Record<string, string> = {};
      columns.forEach((col, colIdx) => {
        cells[col.path] = mockValue(col.path, deviceIdx, colIdx);
      });
      return {
        deviceSn: sn,
        deviceTaskId,
        status: 'success',
        cells,
        dispatchedAt,
        respondedAt,
        pathTasks: buildPathTasks(columns, sn, deviceIdx, 'success', cells),
        raw: gpvXml(columns, cells, sn),
        elapsedMs,
      };
    }

    // 写类：读后核实（§3.11.2）—— 写 RPC 成功后比对读回值派生四态
    const scen = writeScenario(deviceIdx);
    const verify: VerifyItem[] = columns.map((col, colIdx) => {
      const expected = mockValue(col.path, deviceIdx, colIdx);
      if (scen === 'mismatch') {
        return { path: col.path, label: col.label, expected, actual: `${expected}（旧值）`, matched: false };
      }
      if (scen === 'reboot' || scen === 'write-only') {
        return { path: col.path, label: col.label, expected, actual: '', matched: false };
      }
      return { path: col.path, label: col.label, expected, actual: expected, matched: true };
    });

    const status: ExecStatus = scen === 'success' ? 'success' : scen === 'mismatch' ? 'mismatch' : 'unverified';
    const unverifiedReason: UnverifiedReason | undefined =
      scen === 'reboot' ? 'reboot-required' : scen === 'write-only' ? 'write-only' : undefined;

    // 可读场景（success/mismatch）单元格回填读回值；不可读（unverified）留空
    const cells: Record<string, string> = {};
    if (scen === 'success' || scen === 'mismatch') {
      verify.forEach((v) => {
        cells[v.path] = v.actual;
      });
    }

    return {
      deviceSn: sn,
      deviceTaskId,
      status,
      cells,
      unverifiedReason,
      verify,
      dispatchedAt,
      respondedAt,
      pathTasks: buildPathTasks(columns, sn, deviceIdx, status, cells),
      raw: writeXml(operationType, sn),
      elapsedMs,
    };
  });
}

// ── mock 历史命令记录种子（设计 §3.10.4-5，决策 2026-06-04=方案 A） ──────────────
// 模拟「已持久化的历史 MML 任务」，使命令记录面板在首次进入时非空。接后端时由
// useMMLTasks(GET /mml/tasks) 列表 + useMMLTaskResults 结果替换（见 useConsoleHistory）。
function seedRecord(id: string, time: string, cmd: CommandItem, deviceCount: number): ExecRecord {
  const sns = MOCK_DEVICES.slice(0, deviceCount).map((d) => d.sn);
  const columns = buildColumns(cmd);
  const read = isReadOp(cmd.operationType);
  return {
    id,
    commandId: mockUuid(`cmd-${id}`),
    time,
    commandName: cmd.commandName,
    operationType: cmd.operationType,
    deviceCount,
    execMeta: {
      operationType: cmd.operationType,
      read,
      label: cmd.commandCode,
      commandName: cmd.commandName,
    },
    columns,
    rows: buildResultRows(columns, sns, read, cmd.operationType),
  };
}

export const MOCK_HISTORY: ExecRecord[] = [
  seedRecord('seed-2', '09:48:12', MOCK_COMMANDS[6], 8), // LST 查询活动告警
  seedRecord('seed-1', '09:12:30', MOCK_COMMANDS[0], 12), // LST 查询设备基本信息
];
