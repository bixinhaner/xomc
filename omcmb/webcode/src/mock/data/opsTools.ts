export interface OpsTemplate {
  id: string;
  templateName: string;
  description: string;
  category: string;
  targetDeviceTypes: string[];
  steps: OpsStep[];
  estimatedDuration: number;
  creator: string;
  createTime: string;
  updateTime: string;
  useCount: number;
  tags: string[];
}

export interface OpsStep {
  stepNo: number;
  stepName: string;
  stepType: 'mml' | 'check' | 'wait' | 'notify' | 'script';
  command?: string;
  condition?: string;
  waitSeconds?: number;
  notifyTarget?: string;
  description: string;
  rollbackCommand?: string;
}

export interface OpsCommandRecord {
  id: string;
  commandText: string;
  deviceSn: string;
  deviceName: string;
  operator: string;
  executeTime: string;
  duration: number;
  success: boolean;
  output: string;
  errorMessage?: string;
}

export interface OpsTask {
  id: string;
  taskName: string;
  templateId?: string;
  deviceSns: string[];
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled' | 'paused';
  currentStep: number;
  totalSteps: number;
  progress: number;
  successCount: number;
  failCount: number;
  totalCount: number;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  creator: string;
  message?: string;
}

export const mockOpsTemplates: OpsTemplate[] = [
  {
    id: 'opst-001',
    templateName: '基站日常巡检流程',
    description: '标准化的基站日常巡检操作流程，包含关键状态检查',
    category: '巡检运维',
    targetDeviceTypes: ['eNB', 'gNB'],
    steps: [
      { stepNo: 1, stepName: '检查软件版本', stepType: 'mml', command: 'DSP VERSION', description: '获取当前软件版本信息' },
      { stepNo: 2, stepName: '检查单板状态', stepType: 'mml', command: 'DSP BOARDSTATUS', description: '查看所有单板运行状态' },
      { stepNo: 3, stepName: '检查系统资源', stepType: 'mml', command: 'DSP SYSRESOURCE', description: '查看CPU、内存使用情况' },
      { stepNo: 4, stepName: '检查时钟状态', stepType: 'mml', command: 'DSP CLOCKSTATUS', description: '检查GPS时钟同步状态' },
      { stepNo: 5, stepName: '检查传输链路', stepType: 'mml', command: 'DSP LINKSTATUS', description: '查看S1/X2链路状态' },
      { stepNo: 6, stepName: '检查当前告警', stepType: 'mml', command: 'LST ALMAF', description: '列出当前活动告警' },
      { stepNo: 7, stepName: '检查小区状态', stepType: 'mml', command: 'LST CELL', description: '查看所有小区运行状态' },
      { stepNo: 8, stepName: '等待结果汇总', stepType: 'wait', waitSeconds: 5, description: '等待所有查询结果返回' },
      { stepNo: 9, stepName: '发送巡检报告', stepType: 'notify', notifyTarget: '运维组', description: '将巡检结果通知运维团队' },
    ],
    estimatedDuration: 300,
    creator: 'admin',
    createTime: '2024-01-10T00:00:00.000Z',
    updateTime: '2024-05-15T00:00:00.000Z',
    useCount: 256,
    tags: ['巡检', '日常运维', '自动化'],
  },
  {
    id: 'opst-002',
    templateName: '小区故障快速处置',
    description: '小区无法使用时的标准化快速处置流程',
    category: '故障处置',
    targetDeviceTypes: ['eNB', 'gNB'],
    steps: [
      { stepNo: 1, stepName: '查看小区状态', stepType: 'mml', command: 'LST CELL', description: '确认小区故障状态' },
      { stepNo: 2, stepName: '检查硬件状态', stepType: 'mml', command: 'DSP BOARDSTATUS', description: '排查硬件故障' },
      { stepNo: 3, stepName: '检查RRU状态', stepType: 'mml', command: 'DSP RRUINFO', description: '检查射频单元状态' },
      { stepNo: 4, stepName: '检查告警信息', stepType: 'mml', command: 'LST ALMAF', description: '查看相关告警' },
      { stepNo: 5, stepName: '尝试重置小区', stepType: 'mml', command: 'RST CELL CELLID:0 MODE:GRACEFUL', description: '优雅重置故障小区', rollbackCommand: 'ACT CELL CELLID:0' },
      { stepNo: 6, stepName: '等待重置完成', stepType: 'wait', waitSeconds: 60, description: '等待小区重置完成' },
      { stepNo: 7, stepName: '验证小区恢复', stepType: 'check', condition: 'CELL_STATUS == ACTIVE', description: '确认小区是否恢复正常' },
      { stepNo: 8, stepName: '通知处置结果', stepType: 'notify', notifyTarget: '告警组', description: '通知故障处置结果' },
    ],
    estimatedDuration: 180,
    creator: 'operator01',
    createTime: '2024-02-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
    useCount: 89,
    tags: ['故障处置', '小区', '紧急'],
  },
  {
    id: 'opst-003',
    templateName: '切换参数优化流程',
    description: '切换成功率低时的参数调整标准化操作',
    category: '性能优化',
    targetDeviceTypes: ['eNB'],
    steps: [
      { stepNo: 1, stepName: '查询切换参数', stepType: 'mml', command: 'LST CELL', description: '获取当前切换参数' },
      { stepNo: 2, stepName: '查询邻区关系', stepType: 'mml', command: 'LST NRCELL CELLID:0', description: '检查邻区配置' },
      { stepNo: 3, stepName: '调整A3偏置', stepType: 'mml', command: 'SET CELLPARAM CELLID:0', description: '优化切换触发参数', rollbackCommand: 'SET CELLPARAM CELLID:0' },
      { stepNo: 4, stepName: '观察等待', stepType: 'wait', waitSeconds: 300, description: '等待5分钟观察效果' },
      { stepNo: 5, stepName: '验证优化效果', stepType: 'check', condition: 'HO_SUCC_RATE > 95', description: '验证切换成功率是否提升' },
    ],
    estimatedDuration: 600,
    creator: 'operator02',
    createTime: '2024-03-01T00:00:00.000Z',
    updateTime: '2024-03-01T00:00:00.000Z',
    useCount: 34,
    tags: ['优化', '切换', '参数调整'],
  },
  {
    id: 'opst-004',
    templateName: '软件升级前预检',
    description: '软件升级前的标准化预检流程，确保升级环境就绪',
    category: '软件管理',
    targetDeviceTypes: ['eNB', 'gNB', 'CPE', 'eGW'],
    steps: [
      { stepNo: 1, stepName: '检查当前版本', stepType: 'mml', command: 'DSP VERSION', description: '记录当前软件版本' },
      { stepNo: 2, stepName: '检查磁盘空间', stepType: 'check', condition: 'DISK_FREE > 2048', description: '确认磁盘空间充足（>2GB）' },
      { stepNo: 3, stepName: '检查设备状态', stepType: 'mml', command: 'DSP BOARDSTATUS', description: '确认硬件正常' },
      { stepNo: 4, stepName: '备份当前配置', stepType: 'script', description: '自动备份当前配置文件' },
      { stepNo: 5, stepName: '通知维护窗口', stepType: 'notify', notifyTarget: '相关业务方', description: '通知升级维护窗口开始' },
    ],
    estimatedDuration: 120,
    creator: 'admin',
    createTime: '2024-04-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
    useCount: 45,
    tags: ['升级', '预检', '维护'],
  },
  {
    id: 'opst-005',
    templateName: 'GPS时钟同步检查恢复',
    description: 'GPS失锁告警的标准处置流程',
    category: '故障处置',
    targetDeviceTypes: ['eNB', 'gNB'],
    steps: [
      { stepNo: 1, stepName: '查询时钟状态', stepType: 'mml', command: 'DSP CLOCKSTATUS', description: '确认GPS失锁状态' },
      { stepNo: 2, stepName: '查询同步源', stepType: 'mml', command: 'LST IPADDR', description: '检查备用时钟源' },
      { stepNo: 3, stepName: '等待GPS自恢复', stepType: 'wait', waitSeconds: 120, description: '等待GPS信号恢复' },
      { stepNo: 4, stepName: '验证时钟同步', stepType: 'check', condition: 'CLOCK_STATUS == LOCKED', description: '确认GPS重新锁定' },
      { stepNo: 5, stepName: '上报处置结果', stepType: 'notify', notifyTarget: '网管中心', description: '上报时钟故障处置结果' },
    ],
    estimatedDuration: 240,
    creator: 'operator01',
    createTime: '2024-05-10T00:00:00.000Z',
    updateTime: '2024-05-10T00:00:00.000Z',
    useCount: 18,
    tags: ['GPS', '时钟', '故障处置'],
  },
  {
    id: 'opst-006',
    templateName: 'eNB容量扩展配置',
    description: '增加小区最大用户数和PRB配置的标准操作',
    category: '性能优化',
    targetDeviceTypes: ['eNB'],
    steps: [
      { stepNo: 1, stepName: '查询当前容量', stepType: 'mml', command: 'DSP PERF COUNTER:MAX_UE_COUNT', description: '查询当前最大用户数' },
      { stepNo: 2, stepName: '修改最大用户数', stepType: 'mml', command: 'SET CELLPARAM', description: '增加每小区最大用户数', rollbackCommand: 'SET CELLPARAM' },
      { stepNo: 3, stepName: '检查配置生效', stepType: 'mml', command: 'LST CELL', description: '确认配置已生效' },
      { stepNo: 4, stepName: '观察性能变化', stepType: 'wait', waitSeconds: 180, description: '观察容量变化效果' },
    ],
    estimatedDuration: 300,
    creator: 'operator02',
    createTime: '2024-05-20T00:00:00.000Z',
    updateTime: '2024-05-20T00:00:00.000Z',
    useCount: 12,
    tags: ['容量', '优化', 'eNB'],
  },
  {
    id: 'opst-007',
    templateName: '5G NSA添加锚点小区',
    description: '5G NSA组网添加LTE锚点小区的操作流程',
    category: '网络配置',
    targetDeviceTypes: ['eNB', 'gNB'],
    steps: [
      { stepNo: 1, stepName: '检查eNB版本支持', stepType: 'mml', command: 'DSP VERSION', description: '确认eNB支持NSA功能' },
      { stepNo: 2, stepName: '配置X2接口', stepType: 'mml', command: 'DSP LINKSTATUS LINKTYPE:X2', description: '建立eNB-gNB X2连接' },
      { stepNo: 3, stepName: '配置锚点参数', stepType: 'script', description: '批量配置NSA锚点小区参数' },
      { stepNo: 4, stepName: '激活NSA特性', stepType: 'mml', command: 'SET CELLPARAM', description: '启用NSA功能', rollbackCommand: 'SET CELLPARAM' },
      { stepNo: 5, stepName: '验证NSA接入', stepType: 'check', condition: 'NSA_UE_COUNT > 0', description: '验证5G用户成功接入' },
    ],
    estimatedDuration: 600,
    creator: 'admin',
    createTime: '2024-06-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
    useCount: 5,
    tags: ['5G', 'NSA', '配置'],
  },
  {
    id: 'opst-008',
    templateName: '传输故障快速恢复',
    description: 'S1/X2链路中断的快速诊断和恢复流程',
    category: '故障处置',
    targetDeviceTypes: ['eNB', 'gNB'],
    steps: [
      { stepNo: 1, stepName: '检查链路状态', stepType: 'mml', command: 'DSP LINKSTATUS', description: '确认链路中断情况' },
      { stepNo: 2, stepName: '检查SCTP状态', stepType: 'mml', command: 'DSP SCTP', description: '查看SCTP链路详情' },
      { stepNo: 3, stepName: '确认IP配置', stepType: 'mml', command: 'LST IPADDR', description: '检查IP地址配置正确性' },
      { stepNo: 4, stepName: '重建SCTP链路', stepType: 'script', description: '尝试重建SCTP传输链路', },
      { stepNo: 5, stepName: '等待链路恢复', stepType: 'wait', waitSeconds: 90, description: '等待链路重建完成' },
      { stepNo: 6, stepName: '验证链路状态', stepType: 'check', condition: 'S1_STATUS == ACTIVE', description: '确认链路已恢复' },
    ],
    estimatedDuration: 300,
    creator: 'operator01',
    createTime: '2024-04-15T00:00:00.000Z',
    updateTime: '2024-04-15T00:00:00.000Z',
    useCount: 28,
    tags: ['传输', 'S1', 'X2', '故障恢复'],
  },
  {
    id: 'opst-009',
    templateName: '天线参数调整流程',
    description: '覆盖优化时的天线参数调整标准化操作',
    category: '性能优化',
    targetDeviceTypes: ['eNB', 'gNB'],
    steps: [
      { stepNo: 1, stepName: '查询当前天线配置', stepType: 'mml', command: 'LST CELL', description: '记录当前天线参数' },
      { stepNo: 2, stepName: '调整下倾角', stepType: 'mml', command: 'SET CELLPARAM', description: '修改电调下倾角', rollbackCommand: 'SET CELLPARAM' },
      { stepNo: 3, stepName: '等待生效', stepType: 'wait', waitSeconds: 60, description: '等待参数生效' },
      { stepNo: 4, stepName: '检查覆盖效果', stepType: 'mml', command: 'DSP PERF', description: '检查覆盖相关KPI' },
      { stepNo: 5, stepName: '记录优化结果', stepType: 'notify', notifyTarget: '射频优化组', description: '上报优化结果' },
    ],
    estimatedDuration: 240,
    creator: 'operator02',
    createTime: '2024-05-05T00:00:00.000Z',
    updateTime: '2024-05-05T00:00:00.000Z',
    useCount: 21,
    tags: ['天线', '覆盖', '优化'],
  },
  {
    id: 'opst-010',
    templateName: '设备重启流程',
    description: '设备需要重启时的标准化安全操作流程',
    category: '维护操作',
    targetDeviceTypes: ['eNB', 'gNB', 'eGW'],
    steps: [
      { stepNo: 1, stepName: '确认维护窗口', stepType: 'check', condition: 'MAINTENANCE_WINDOW == true', description: '确认在维护窗口内操作' },
      { stepNo: 2, stepName: '通知相关方', stepType: 'notify', notifyTarget: '业务保障组', description: '通知即将重启，预计中断时间' },
      { stepNo: 3, stepName: '备份当前配置', stepType: 'script', description: '重启前备份配置' },
      { stepNo: 4, stepName: '检查用户连接', stepType: 'mml', command: 'DSP UE', description: '记录当前在线用户数' },
      { stepNo: 5, stepName: '执行重启', stepType: 'script', description: '发起设备重启指令' },
      { stepNo: 6, stepName: '等待设备启动', stepType: 'wait', waitSeconds: 300, description: '等待设备完成启动' },
      { stepNo: 7, stepName: '验证设备正常', stepType: 'mml', command: 'DSP BOARDSTATUS', description: '确认所有单板正常运行' },
      { stepNo: 8, stepName: '验证业务恢复', stepType: 'check', condition: 'CELL_COUNT > 0', description: '确认业务已恢复' },
      { stepNo: 9, stepName: '解除维护告知', stepType: 'notify', notifyTarget: '业务保障组', description: '通知重启完成，业务已恢复' },
    ],
    estimatedDuration: 600,
    creator: 'admin',
    createTime: '2024-01-20T00:00:00.000Z',
    updateTime: '2024-03-10T00:00:00.000Z',
    useCount: 15,
    tags: ['重启', '维护', '安全操作'],
  },
];

export const mockOpsCommandRecords: OpsCommandRecord[] = [
  { id: 'ocmd-001', commandText: 'DSP VERSION', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', operator: 'admin', executeTime: '2024-06-14T10:00:00.000Z', duration: 1230, success: true, output: 'VERSION: V100R011C10SPC200\nBUILD: 20240315\nSTATUS: Normal' },
  { id: 'ocmd-002', commandText: 'DSP BOARDSTATUS', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', operator: 'admin', executeTime: '2024-06-14T10:00:01.000Z', duration: 1560, success: true, output: 'SRN=0 SN=0 TYPE=BBU STATUS=NORMAL\nSRN=0 SN=1 TYPE=UMPT STATUS=NORMAL\nSRN=0 SN=2 TYPE=UBBP STATUS=NORMAL' },
  { id: 'ocmd-003', commandText: 'DSP SYSRESOURCE', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', operator: 'operator01', executeTime: '2024-06-14T11:00:00.000Z', duration: 980, success: true, output: 'CPU_USAGE=45%\nMEM_USAGE=68%\nDISK_FREE=12GB' },
  { id: 'ocmd-004', commandText: 'LST CELL', deviceSn: 'ENB00010', deviceName: '上海-eNB-0010', operator: 'operator01', executeTime: '2024-06-14T11:30:00.000Z', duration: 1100, success: true, output: 'CELLID=0 STATUS=ACTIVE ADMINSTATE=UNLOCKED\nCELLID=1 STATUS=ACTIVE ADMINSTATE=UNLOCKED\nCELLID=2 STATUS=ACTIVE ADMINSTATE=UNLOCKED' },
  { id: 'ocmd-005', commandText: 'DSP LINKSTATUS', deviceSn: 'ENB00010', deviceName: '上海-eNB-0010', operator: 'operator01', executeTime: '2024-06-14T11:30:05.000Z', duration: 890, success: true, output: 'LINKTYPE=S1 STATUS=ACTIVE\nLINKTYPE=X2 PEER=ENB00011 STATUS=ACTIVE' },
  { id: 'ocmd-006', commandText: 'RST CELL CELLID:1 MODE:GRACEFUL', deviceSn: 'ENB00020', deviceName: '广州-eNB-0020', operator: 'operator02', executeTime: '2024-06-13T14:00:00.000Z', duration: 2300, success: false, output: '', errorMessage: 'ERROR: Cell reset failed - Cell is currently serving 45 active users, use FORCE mode or wait' },
  { id: 'ocmd-007', commandText: 'LST ALMAF', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', operator: 'admin', executeTime: '2024-06-14T09:00:00.000Z', duration: 760, success: true, output: 'No active alarm filters configured' },
  { id: 'ocmd-008', commandText: 'DSP CLOCKSTATUS', deviceSn: 'GNB00010', deviceName: '上海-gNB-0010', operator: 'operator01', executeTime: '2024-06-12T15:00:00.000Z', duration: 1050, success: true, output: 'CLOCK_SOURCE=GPS\nSTATUS=LOCKED\nACCURACY=<10ns\nGPS_SATELLITES=8' },
  { id: 'ocmd-009', commandText: 'DSP PERF COUNTER:RRC_CONN_ESTAB_ATT CELLID:0', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', operator: 'operator01', executeTime: '2024-06-14T08:00:00.000Z', duration: 1200, success: true, output: 'COUNTER=RRC_CONN_ESTAB_ATT CELLID=0 VALUE=15420 PERIOD=15MIN' },
  { id: 'ocmd-010', commandText: 'SET CELLPARAM CELLID:0 TXPOWER:43', deviceSn: 'ENB00030', deviceName: '成都-eNB-0030', operator: 'operator02', executeTime: '2024-06-11T10:00:00.000Z', duration: 1800, success: true, output: 'RETCODE=0 SUCCESS\nCELLID=0 TXPOWER changed from 40 to 43 dBm' },
  { id: 'ocmd-011', commandText: 'DSP RRUINFO', deviceSn: 'GNB00020', deviceName: '广州-gNB-0020', operator: 'operator02', executeTime: '2024-06-10T16:00:00.000Z', duration: 1400, success: true, output: 'RRUID=0 TYPE=AAU5239 STATUS=NORMAL TEMP=35C VSWR=1.2\nRRUID=1 TYPE=AAU5239 STATUS=NORMAL TEMP=36C VSWR=1.3' },
  { id: 'ocmd-012', commandText: 'DSP SCTP', deviceSn: 'ENB00040', deviceName: '西安-eNB-0040', operator: 'operator01', executeTime: '2024-06-09T13:00:00.000Z', duration: 950, success: true, output: 'LNKID=0 LOCALIP=10.11.1.100 PEERIP=192.168.100.1 STATUS=ACTIVE STREAMS=5' },
  { id: 'ocmd-013', commandText: 'LST NRCELL CELLID:0', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', operator: 'admin', executeTime: '2024-06-08T11:00:00.000Z', duration: 1100, success: true, output: 'CELLID=0 NRCELLID=1 NRFREQ=2585 NRTYPE=INTRAFREQ STATUS=ACTIVE\nCELLID=0 NRCELLID=2 NRFREQ=2585 NRTYPE=INTRAFREQ STATUS=ACTIVE' },
  { id: 'ocmd-014', commandText: 'RUN SELFCHECK CHECKTYPE:QUICK', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', operator: 'admin', executeTime: '2024-06-07T22:00:00.000Z', duration: 15000, success: true, output: 'SELFCHECK RESULT: PASS\nHARDWARE: OK\nSOFTWARE: OK\nNETWORK: OK\nDURATION: 15s' },
  { id: 'ocmd-015', commandText: 'DSP UE CELLID:0', deviceSn: 'ENB00010', deviceName: '上海-eNB-0010', operator: 'operator01', executeTime: '2024-06-14T12:00:00.000Z', duration: 1300, success: true, output: 'CELLID=0 UE_COUNT=128\nUEID=1001 RNTI=0x1A2B IMSI=460011234567890 STATUS=ACTIVE\n... (128 users total)' },
  { id: 'ocmd-016', commandText: 'CLR ALM ALMID:100001', deviceSn: 'ENB00002', deviceName: '北京-eNB-0002', operator: 'operator01', executeTime: '2024-06-13T09:30:00.000Z', duration: 800, success: false, output: '', errorMessage: 'ERROR: Cannot clear alarm - alarm condition still active' },
  { id: 'ocmd-017', commandText: 'LST IPADDR', deviceSn: 'ENB00050', deviceName: '杭州-eNB-0050', operator: 'operator02', executeTime: '2024-06-12T14:00:00.000Z', duration: 900, success: true, output: 'IFNAME=ETH0 IPADDR=10.13.1.100 MASK=255.255.0.0 GW=10.13.0.1\nIFNAME=ETH1 IPADDR=192.168.200.50 MASK=255.255.255.0' },
  { id: 'ocmd-018', commandText: 'SET ALMTHD ALMCODE:A0004 THRESHOLD:70 DIRECTION:UPPER', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', operator: 'admin', executeTime: '2024-06-11T16:00:00.000Z', duration: 1050, success: true, output: 'RETCODE=0 SUCCESS\nALMCODE=A0004 THRESHOLD set to 70°C' },
  { id: 'ocmd-019', commandText: 'DEACT CELL CELLID:2 REASON:维护窗口', deviceSn: 'GNB00030', deviceName: '深圳-gNB-0030', operator: 'operator02', executeTime: '2024-06-10T02:00:00.000Z', duration: 2100, success: true, output: 'RETCODE=0 SUCCESS\nCELLID=2 ADMINSTATE changed to LOCKED\nActive users gracefully handed over' },
  { id: 'ocmd-020', commandText: 'ACT CELL CELLID:2', deviceSn: 'GNB00030', deviceName: '深圳-gNB-0030', operator: 'operator02', executeTime: '2024-06-10T04:30:00.000Z', duration: 1800, success: true, output: 'RETCODE=0 SUCCESS\nCELLID=2 ADMINSTATE changed to UNLOCKED\nCell now ACTIVE and serving users' },
];

export const mockOpsTasks: OpsTask[] = [
  {
    id: 'opstask-001',
    taskName: '华北区批量巡检-2024-06-14',
    templateId: 'opst-001',
    deviceSns: ['ENB00001', 'ENB00002', 'ENB00003', 'GNB00001'],
    status: 'success',
    currentStep: 9,
    totalSteps: 9,
    progress: 100,
    successCount: 4,
    failCount: 0,
    totalCount: 4,
    createdAt: '2024-06-14T08:00:00.000Z',
    startedAt: '2024-06-14T08:01:00.000Z',
    completedAt: '2024-06-14T08:25:00.000Z',
    creator: 'admin',
    message: '巡检完成，所有设备状态正常',
  },
  {
    id: 'opstask-002',
    taskName: '广州eNB小区故障处置',
    templateId: 'opst-002',
    deviceSns: ['ENB00020'],
    status: 'failed',
    currentStep: 5,
    totalSteps: 8,
    progress: 62,
    successCount: 0,
    failCount: 1,
    totalCount: 1,
    createdAt: '2024-06-13T14:00:00.000Z',
    startedAt: '2024-06-13T14:00:30.000Z',
    completedAt: '2024-06-13T14:08:00.000Z',
    creator: 'operator02',
    message: '小区重置失败：当前有活跃用户，需等待业务低峰期',
  },
  {
    id: 'opstask-003',
    taskName: '深圳gNB批量巡检',
    templateId: 'opst-001',
    deviceSns: ['GNB00030', 'GNB00031'],
    status: 'running',
    currentStep: 4,
    totalSteps: 9,
    progress: 44,
    successCount: 0,
    failCount: 0,
    totalCount: 2,
    createdAt: new Date(Date.now() - 600000).toISOString(),
    startedAt: new Date(Date.now() - 590000).toISOString(),
    creator: 'operator02',
  },
  {
    id: 'opstask-004',
    taskName: '全网软件升级预检',
    templateId: 'opst-004',
    deviceSns: ['ENB00001', 'ENB00002', 'ENB00010', 'GNB00001', 'GNB00010'],
    status: 'pending',
    currentStep: 0,
    totalSteps: 5,
    progress: 0,
    successCount: 0,
    failCount: 0,
    totalCount: 5,
    createdAt: new Date(Date.now() - 300000).toISOString(),
    creator: 'admin',
  },
  {
    id: 'opstask-005',
    taskName: '切换优化-上海区域',
    templateId: 'opst-003',
    deviceSns: ['ENB00010', 'ENB00011'],
    status: 'paused',
    currentStep: 3,
    totalSteps: 5,
    progress: 60,
    successCount: 0,
    failCount: 0,
    totalCount: 2,
    createdAt: '2024-06-12T15:00:00.000Z',
    startedAt: '2024-06-12T15:01:00.000Z',
    creator: 'operator01',
    message: '用户手动暂停，等待确认后继续',
  },
  {
    id: 'opstask-006',
    taskName: '传输故障恢复-西安',
    templateId: 'opst-008',
    deviceSns: ['ENB00040'],
    status: 'success',
    currentStep: 6,
    totalSteps: 6,
    progress: 100,
    successCount: 1,
    failCount: 0,
    totalCount: 1,
    createdAt: '2024-06-09T13:00:00.000Z',
    startedAt: '2024-06-09T13:00:30.000Z',
    completedAt: '2024-06-09T13:18:00.000Z',
    creator: 'operator01',
    message: 'S1链路已成功恢复，业务正常',
  },
  {
    id: 'opstask-007',
    taskName: '设备重启-深圳维护窗口',
    templateId: 'opst-010',
    deviceSns: ['GNB00030'],
    status: 'success',
    currentStep: 9,
    totalSteps: 9,
    progress: 100,
    successCount: 1,
    failCount: 0,
    totalCount: 1,
    createdAt: '2024-06-10T01:30:00.000Z',
    startedAt: '2024-06-10T02:00:00.000Z',
    completedAt: '2024-06-10T02:50:00.000Z',
    creator: 'admin',
    message: '设备重启完成，所有业务已恢复正常',
  },
  {
    id: 'opstask-008',
    taskName: 'GPS故障处置-成都',
    templateId: 'opst-005',
    deviceSns: ['ENB00030'],
    status: 'success',
    currentStep: 5,
    totalSteps: 5,
    progress: 100,
    successCount: 1,
    failCount: 0,
    totalCount: 1,
    createdAt: '2024-06-08T09:00:00.000Z',
    startedAt: '2024-06-08T09:01:00.000Z',
    completedAt: '2024-06-08T09:15:00.000Z',
    creator: 'operator02',
    message: 'GPS时钟已重新锁定，告警自动清除',
  },
  {
    id: 'opstask-009',
    taskName: '天线优化-南京',
    templateId: 'opst-009',
    deviceSns: ['ENB00060'],
    status: 'cancelled',
    currentStep: 2,
    totalSteps: 5,
    progress: 40,
    successCount: 0,
    failCount: 0,
    totalCount: 1,
    createdAt: '2024-06-07T14:00:00.000Z',
    startedAt: '2024-06-07T14:01:00.000Z',
    completedAt: '2024-06-07T14:15:00.000Z',
    creator: 'operator01',
    message: '用户取消：协调到更优维护时间',
  },
  {
    id: 'opstask-010',
    taskName: '华东全区日常巡检',
    templateId: 'opst-001',
    deviceSns: ['ENB00010', 'ENB00011', 'ENB00050', 'ENB00060', 'GNB00010'],
    status: 'success',
    currentStep: 9,
    totalSteps: 9,
    progress: 100,
    successCount: 4,
    failCount: 1,
    totalCount: 5,
    createdAt: '2024-06-14T07:00:00.000Z',
    startedAt: '2024-06-14T07:00:30.000Z',
    completedAt: '2024-06-14T07:45:00.000Z',
    creator: 'operator01',
    message: '巡检完成：4台正常，1台(ENB00011)发现异常已记录',
  },
];
