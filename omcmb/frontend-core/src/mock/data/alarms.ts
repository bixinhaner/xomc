import type { Alarm } from '../../types/alarm';
import type { AlarmSeverity } from '../../types/common';
import { mockDevices } from './devices';

const alarmDefinitions: Array<{ code: string; name: string; type: string }> = [
  { code: 'A0001', name: '小区不可用', type: '无线告警' },
  { code: 'A0002', name: 'S1链路中断', type: '传输告警' },
  { code: 'A0003', name: '射频单元故障', type: '硬件告警' },
  { code: 'A0004', name: '温度过高告警', type: '环境告警' },
  { code: 'A0005', name: 'VSWR异常', type: '射频告警' },
  { code: 'A0006', name: 'GPS失锁', type: '时钟告警' },
  { code: 'A0007', name: 'X2链路故障', type: '传输告警' },
  { code: 'A0008', name: '传输中断', type: '传输告警' },
  { code: 'A0009', name: '电压异常', type: '电源告警' },
  { code: 'A0010', name: 'CPU过载告警', type: '系统告警' },
  { code: 'A0011', name: 'E-RAB建立失败', type: '无线告警' },
  { code: 'A0012', name: '切换成功率低', type: '无线告警' },
  { code: 'A0013', name: '上行干扰过高', type: '射频告警' },
  { code: 'A0014', name: '下行吞吐量低', type: '性能告警' },
  { code: 'A0015', name: '内存利用率高', type: '系统告警' },
  { code: 'A0016', name: '风扇故障', type: '硬件告警' },
  { code: 'A0017', name: '光模块异常', type: '硬件告警' },
  { code: 'A0018', name: 'NTP同步失败', type: '时钟告警' },
  { code: 'A0019', name: '基带板故障', type: '硬件告警' },
  { code: 'A0020', name: 'RRU掉线', type: '无线告警' },
  { code: 'A0021', name: '小区去激活', type: '无线告警' },
  { code: 'A0022', name: 'PDCP层丢包', type: '无线告警' },
  { code: 'A0023', name: '物理层异常', type: '无线告警' },
  { code: 'A0024', name: '功率放大器故障', type: '射频告警' },
  { code: 'A0025', name: '天线端口断路', type: '射频告警' },
  { code: 'A0026', name: '软件运行异常', type: '系统告警' },
  { code: 'A0027', name: '数据库连接失败', type: '系统告警' },
  { code: 'A0028', name: '磁盘空间不足', type: '系统告警' },
  { code: 'A0029', name: 'S1-MME连接断开', type: '核心网告警' },
  { code: 'A0030', name: 'SGW路径异常', type: '核心网告警' },
  { code: 'A0031', name: '邻区配置错误', type: '配置告警' },
  { code: 'A0032', name: 'PCI冲突', type: '配置告警' },
  { code: 'A0033', name: 'TA配置异常', type: '配置告警' },
  { code: 'A0034', name: '射频校准失败', type: '射频告警' },
  { code: 'A0035', name: '过载保护激活', type: '无线告警' },
  { code: 'A0036', name: '用户面承载丢失', type: '无线告警' },
  { code: 'A0037', name: '时延异常', type: '传输告警' },
  { code: 'A0038', name: '吞吐率持续低值', type: '性能告警' },
  { code: 'A0039', name: '切换失败率高', type: '无线告警' },
  { code: 'A0040', name: '核心网接入拒绝', type: '核心网告警' },
];

const severityDistribution: AlarmSeverity[] = [
  ...Array(8).fill('critical'),
  ...Array(20).fill('major'),
  ...Array(30).fill('minor'),
  ...Array(42).fill('warning'),
];

const deviceSnList = mockDevices.map((d) => ({ sn: d.sn, name: d.name, type: d.productClass }));

function pickRandom<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomDate(daysAgo: number): string {
  return new Date(Date.now() - Math.random() * daysAgo * 86400000).toISOString();
}

function generateAlarmContent(def: { name: string; type: string }, deviceName: string): string {
  const templates = [
    `设备${deviceName}发生${def.name}，${def.type}异常，请及时处理`,
    `${deviceName}: ${def.name}，告警类型：${def.type}，需要维护人员检查`,
    `检测到${deviceName}的${def.name}事件，${def.type}模块报告故障`,
    `${def.type}告警：${deviceName}发生${def.name}，请检查相关配置和硬件状态`,
  ];
  return pickRandom(templates);
}

let alarmCounter = 1;

function generateAlarm(isActive: boolean, index: number, unread: '0' | '1' = '0'): Alarm {
  const def = alarmDefinitions[index % alarmDefinitions.length];
  const device = deviceSnList[index % deviceSnList.length];
  const severity: AlarmSeverity = pickRandom(severityDistribution);
  const alarmTime = isActive ? randomDate(7) : randomDate(30);
  const clearTime = isActive ? undefined : new Date(new Date(alarmTime).getTime() + Math.random() * 86400000 * 3).toISOString();
  const isAcknowledged = Math.random() > 0.4;

  const id = `ALM${String(alarmCounter++).padStart(6, '0')}`;

  // 生成告警状态：活动告警为0或1，历史告警为2或3
  const dealState = isActive
    ? (isAcknowledged ? '1' : '0')
    : (isAcknowledged ? '3' : '2');

  const dealUser = isAcknowledged ? pickRandom(['admin', 'operator01', 'operator02', 'operator03']) : undefined;
  const dealTime = isAcknowledged ? new Date(new Date(alarmTime).getTime() + Math.random() * 3600000).toISOString() : undefined;
  const clearUser = clearTime ? pickRandom(['admin', 'system']) : undefined;

  return {
    id,
    alarmIdentifier: def.code,
    alarmName: def.name,
    severity,
    deviceSn: device.sn,
    deviceName: device.name,
    description: generateAlarmContent(def, device.name),
    neType: device.type,
    equipInfo: `SN=${device.sn};Name=${device.name}`,
    eventType: 'device',
    dealState: dealState as '0' | '1' | '2' | '3',
    alarmType: isActive ? 'active' : 'history',
    eventTime: alarmTime,
    updTime: alarmTime,
    clearTime,
    specificProblem: def.name,
    alarmCount: Math.floor(Math.random() * 10) + 1,
    dealMemo: isAcknowledged ? '已确认，正在处理' : '',
    unread,
    dealUser,
    dealTime,
    clearUser,
    duration: clearTime
      ? Math.floor((new Date(clearTime).getTime() - new Date(alarmTime).getTime()) / 60000)
      : Math.floor((Date.now() - new Date(alarmTime).getTime()) / 60000),
    alarmSource: device.sn,
    technology: device.type,
    isActive,
  };
}

// 前20个告警设置为未读
export const mockActiveAlarms: Alarm[] = Array.from({ length: 150 }, (_, i) =>
  generateAlarm(true, i, i < 20 ? '1' : '0')
);
// 与真实后端 UUID 主键保持一致，供 Dashboard 注意事项精准深链回归使用。
mockActiveAlarms[0].id = '892d12c0-ec1a-4fd1-8070-902d3aaf84e9';
export const mockHistoricalAlarms: Alarm[] = Array.from({ length: 500 }, (_, i) =>
  generateAlarm(false, i + 150, '0')
);
