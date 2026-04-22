import type { OperationLog, OperationType, OperationResult } from '../../types/system';

const operators = ['admin', 'operator01', 'operator02', 'operator03', 'auditor01'];
const clientIPs = ['192.168.1.100', '192.168.1.101', '10.0.0.50', '10.0.0.51', '172.16.0.10'];
const _modules = ['设备管理', '告警管理', '性能管理', '配置管理', 'MML执行', '用户管理', '软件管理', '备份管理', '报表', '系统设置'];
const _targets = [
  'ENB00001', 'ENB00002', 'GNB00001', 'GNB00010', 'CPE00001',
  '告警规则-001', '配置模板-001', '用户-operator03', '备份任务-001', 'MML脚本-001',
];

const opTypeMappings: Array<{ type: OperationType; modules: string[]; targets: string[]; contents: string[]; results: Array<OperationResult> }> = [
  {
    type: 'login',
    modules: ['系统设置'],
    targets: ['系统'],
    contents: ['用户登录系统', '管理员登录', '运维人员登录'],
    results: ['success', 'success', 'success', 'failure'],
  },
  {
    type: 'logout',
    modules: ['系统设置'],
    targets: ['系统'],
    contents: ['用户注销登录', '会话超时退出'],
    results: ['success'],
  },
  {
    type: 'query',
    modules: ['设备管理', '告警管理', '性能管理'],
    targets: ['ENB00001', 'GNB00001', '告警列表', '性能数据'],
    contents: ['查询设备列表', '查询当前告警', '查询KPI数据', '导出设备信息'],
    results: ['success', 'success'],
  },
  {
    type: 'update',
    modules: ['配置管理', '设备管理'],
    targets: ['ENB00001', 'GNB00001', '配置参数'],
    contents: ['修改设备参数', '更新配置模板', '批量修改设备属性'],
    results: ['success', 'success', 'failure'],
  },
  {
    type: 'create',
    modules: ['告警管理', '用户管理', '备份管理'],
    targets: ['告警规则-新建', '用户-operator04', '备份任务-新建'],
    contents: ['创建告警规则', '创建新用户', '创建备份任务', '新建配置模板'],
    results: ['success'],
  },
  {
    type: 'delete',
    modules: ['文件管理', '用户管理', '告警管理'],
    targets: ['文件-old_firmware.tar.gz', '用户-测试账号', '历史告警'],
    contents: ['删除过期文件', '删除测试用户', '清理历史告警', '删除备份'],
    results: ['success', 'success'],
  },
  {
    type: 'execute',
    modules: ['MML执行', '运维工具'],
    targets: ['ENB00001', 'GNB00001', 'MML脚本-巡检'],
    contents: ['执行MML命令:DSP VERSION', '执行巡检脚本', '批量执行MML命令'],
    results: ['success', 'failure'],
  },
  {
    type: 'deploy',
    modules: ['软件管理', '配置管理'],
    targets: ['ENB00001-升级', 'GNB00001-配置下发'],
    contents: ['部署软件升级包', '下发配置参数', '应用基线配置'],
    results: ['success', 'success', 'partial'],
  },
  {
    type: 'export',
    modules: ['报表', '性能管理', '告警管理'],
    targets: ['性能报表', '告警报表', '设备列表'],
    contents: ['导出性能报表', '导出告警数据', '导出设备清单'],
    results: ['success'],
  },
  {
    type: 'approve',
    modules: ['配置管理', '软件管理'],
    targets: ['配置变更申请-001', '升级计划-001'],
    contents: ['审批配置变更', '审批升级计划', '批准许可证申请'],
    results: ['success'],
  },
];

function pickRandom<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomDate(daysAgo: number): string {
  return new Date(Date.now() - Math.random() * daysAgo * 86400000).toISOString();
}

function generateOperationLog(index: number): OperationLog {
  const opDef = opTypeMappings[index % opTypeMappings.length];
  const result = pickRandom(opDef.results);
  return {
    id: `oplog-${String(index + 1).padStart(6, '0')}`,
    operator: pickRandom(operators),
    clientIp: pickRandom(clientIPs),
    module: pickRandom(opDef.modules),
    operationType: opDef.type,
    target: pickRandom(opDef.targets),
    content: pickRandom(opDef.contents),
    result,
    message: result === 'success' ? '操作成功' : result === 'failure' ? '操作失败：权限不足或参数错误' : '部分成功',
    operationTime: randomDate(30),
  };
}

export const mockOperationLogs: OperationLog[] = Array.from({ length: 100 }, (_, i) => generateOperationLog(i));

export interface SystemLog {
  id: string;
  level: 'INFO' | 'WARN' | 'ERROR' | 'DEBUG';
  source: string;
  message: string;
  details?: string;
  timestamp: string;
}

const systemLogSources = ['OMC-Core', 'AlarmService', 'PerformanceService', 'DeviceConnector', 'WebServer', 'DatabasePool', 'TaskScheduler'];
const systemLogMessages: Array<{ level: SystemLog['level']; message: string; details?: string }> = [
  { level: 'INFO', message: '系统启动完成，所有服务正常运行' },
  { level: 'INFO', message: '数据库连接池初始化成功，连接数: 20' },
  { level: 'INFO', message: '设备心跳检测服务启动' },
  { level: 'INFO', message: '告警同步任务开始执行' },
  { level: 'INFO', message: '性能数据采集任务完成，设备数: 50' },
  { level: 'WARN', message: '数据库连接池使用率超过80%', details: 'ConnectionPool: 18/20 connections in use' },
  { level: 'WARN', message: '设备ENB00021心跳超时，尝试重连', details: 'Last heartbeat: 120s ago' },
  { level: 'WARN', message: '磁盘使用率达到75%，建议清理', details: 'Path: /data, Used: 750GB/1TB' },
  { level: 'WARN', message: '内存使用率达到85%', details: 'Heap: 6.8GB/8GB' },
  { level: 'ERROR', message: '设备GNB00010连接断开，告警已生成', details: 'Connection refused: 10.3.1.10:830' },
  { level: 'ERROR', message: '数据库查询超时，语句执行时间超过30秒', details: 'Query: SELECT * FROM measurements WHERE timestamp > ... LIMIT 100000' },
  { level: 'ERROR', message: '邮件告警发送失败，SMTP连接超时', details: 'SMTP host: mail.company.com:587, timeout: 30s' },
  { level: 'DEBUG', message: '收到设备SNMP Trap消息', details: 'Source: 10.1.1.100, OID: .1.3.6.1.4.1.2011.5.25.29.2.1.3' },
  { level: 'INFO', message: '定时备份任务执行成功' },
  { level: 'INFO', message: '许可证有效性检查通过' },
];

export const mockSystemLogs: SystemLog[] = Array.from({ length: 50 }, (_, i) => ({
  id: `syslog-${String(i + 1).padStart(6, '0')}`,
  level: systemLogMessages[i % systemLogMessages.length].level,
  source: pickRandom(systemLogSources),
  message: systemLogMessages[i % systemLogMessages.length].message,
  details: systemLogMessages[i % systemLogMessages.length].details,
  timestamp: randomDate(7),
}));

export interface NEMessageLog {
  id: string;
  deviceSn: string;
  deviceName: string;
  messageType: 'notification' | 'alarm' | 'heartbeat' | 'config_response' | 'perf_data';
  direction: 'northbound' | 'southbound';
  protocol: 'NETCONF' | 'SNMP' | 'TR-069' | 'REST';
  content: string;
  timestamp: string;
  success: boolean;
}

const deviceList = [
  { sn: 'ENB00001', name: '北京-eNB-0001' },
  { sn: 'GNB00001', name: '北京-gNB-0001' },
  { sn: 'ENB00010', name: '上海-eNB-0010' },
  { sn: 'GNB00010', name: '上海-gNB-0010' },
  { sn: 'CPE00001', name: '北京-CPE-0001' },
];

const messageTypes: Array<NEMessageLog['messageType']> = ['notification', 'alarm', 'heartbeat', 'config_response', 'perf_data'];
const protocols: Array<NEMessageLog['protocol']> = ['NETCONF', 'SNMP', 'TR-069', 'REST'];

export const mockNEMessageLogs: NEMessageLog[] = Array.from({ length: 30 }, (_, i) => {
  const device = deviceList[i % deviceList.length];
  const msgType = messageTypes[i % messageTypes.length];
  return {
    id: `nemsg-${String(i + 1).padStart(6, '0')}`,
    deviceSn: device.sn,
    deviceName: device.name,
    messageType: msgType,
    direction: i % 3 === 0 ? 'southbound' : 'northbound',
    protocol: pickRandom(protocols),
    content: msgType === 'heartbeat'
      ? `{"type":"heartbeat","deviceSn":"${device.sn}","timestamp":"${new Date().toISOString()}"}`
      : msgType === 'alarm'
      ? `{"type":"alarm","alarmCode":"A000${(i % 10) + 1}","severity":"major","deviceSn":"${device.sn}"}`
      : msgType === 'perf_data'
      ? `{"type":"perf","kpiCode":"RRC_SUCC_RATE","value":${(Math.random() * 5 + 95).toFixed(2)},"deviceSn":"${device.sn}"}`
      : `{"type":"${msgType}","deviceSn":"${device.sn}","status":"ok"}`,
    timestamp: randomDate(1),
    success: Math.random() > 0.1,
  };
});
