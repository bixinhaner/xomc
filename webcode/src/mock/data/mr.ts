export interface MRIndicator {
  id: string;
  indicatorName: string;
  indicatorCode: string;
  description: string;
  unit: string;
  category: string;
  valueRange: [number, number];
}

export interface MRDeviceMapping {
  id: string;
  deviceSn: string;
  deviceName: string;
  cellId: string;
  cellName: string;
  enabled: boolean;
  samplingInterval: number;
  lastCollectTime?: string;
  totalRecords: number;
}

export interface MRRecord {
  id: string;
  deviceSn: string;
  cellId: string;
  timestamp: string;
  indicators: Record<string, number>;
}

export const mockMRIndicators: MRIndicator[] = [
  {
    id: 'mri-001',
    indicatorName: '参考信号接收功率',
    indicatorCode: 'RSRP',
    description: '用户设备测量的参考信号接收功率，反映覆盖质量',
    unit: 'dBm',
    category: '覆盖指标',
    valueRange: [-140, -44],
  },
  {
    id: 'mri-002',
    indicatorName: '参考信号接收质量',
    indicatorCode: 'RSRQ',
    description: '参考信号接收质量，反映信号质量',
    unit: 'dB',
    category: '覆盖指标',
    valueRange: [-19.5, -3],
  },
  {
    id: 'mri-003',
    indicatorName: '信号与干扰加噪声比',
    indicatorCode: 'SINR',
    description: '信号质量指标，值越高表示质量越好',
    unit: 'dB',
    category: '质量指标',
    valueRange: [-20, 30],
  },
  {
    id: 'mri-004',
    indicatorName: '信道质量指示',
    indicatorCode: 'CQI',
    description: 'UE报告的信道质量，影响调制编码方案选择',
    unit: '',
    category: '质量指标',
    valueRange: [0, 15],
  },
  {
    id: 'mri-005',
    indicatorName: '预编码矩阵指示',
    indicatorCode: 'PMI',
    description: 'UE推荐的预编码矩阵，用于MIMO波束成形',
    unit: '',
    category: 'MIMO指标',
    valueRange: [0, 15],
  },
  {
    id: 'mri-006',
    indicatorName: '秩指示',
    indicatorCode: 'RI',
    description: 'MIMO传输层数指示',
    unit: '',
    category: 'MIMO指标',
    valueRange: [1, 8],
  },
  {
    id: 'mri-007',
    indicatorName: '定时提前量',
    indicatorCode: 'TA',
    description: '基站到UE的传播时延估计，反映距离',
    unit: 'ns',
    category: '覆盖指标',
    valueRange: [0, 100000],
  },
  {
    id: 'mri-008',
    indicatorName: '上行路损',
    indicatorCode: 'PL_UL',
    description: '上行方向路径损耗估计',
    unit: 'dB',
    category: '覆盖指标',
    valueRange: [50, 170],
  },
  {
    id: 'mri-009',
    indicatorName: 'UCI格式',
    indicatorCode: 'UCI_FORMAT',
    description: '上行控制信息格式',
    unit: '',
    category: '接入指标',
    valueRange: [0, 3],
  },
  {
    id: 'mri-010',
    indicatorName: '最优小区RSRP',
    indicatorCode: 'BEST_RSRP',
    description: '测量报告中最优服务小区RSRP',
    unit: 'dBm',
    category: '覆盖指标',
    valueRange: [-140, -44],
  },
];

export const mockMRDeviceMappings: MRDeviceMapping[] = [
  { id: 'mrdm-001', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', cellId: 'CELL-001-1', cellName: '北京-eNB-0001-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 158400 },
  { id: 'mrdm-002', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', cellId: 'CELL-001-2', cellName: '北京-eNB-0001-Cell2', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 156200 },
  { id: 'mrdm-003', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', cellId: 'CELL-001-3', cellName: '北京-eNB-0001-Cell3', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 162000 },
  { id: 'mrdm-004', deviceSn: 'ENB00002', deviceName: '北京-eNB-0002', cellId: 'CELL-002-1', cellName: '北京-eNB-0002-Cell1', enabled: true, samplingInterval: 30, lastCollectTime: '2024-06-14T23:30:00.000Z', totalRecords: 88000 },
  { id: 'mrdm-005', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', cellId: 'CELL-GNB-001-1', cellName: '北京-gNB-0001-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 200000 },
  { id: 'mrdm-006', deviceSn: 'ENB00010', deviceName: '上海-eNB-0010', cellId: 'CELL-010-1', cellName: '上海-eNB-0010-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 145000 },
  { id: 'mrdm-007', deviceSn: 'GNB00010', deviceName: '上海-gNB-0010', cellId: 'CELL-GNB-010-1', cellName: '上海-gNB-0010-Cell1', enabled: false, samplingInterval: 15, totalRecords: 50000 },
  { id: 'mrdm-008', deviceSn: 'ENB00020', deviceName: '广州-eNB-0020', cellId: 'CELL-020-1', cellName: '广州-eNB-0020-Cell1', enabled: true, samplingInterval: 60, lastCollectTime: '2024-06-14T23:00:00.000Z', totalRecords: 35000 },
  { id: 'mrdm-009', deviceSn: 'GNB00020', deviceName: '广州-gNB-0020', cellId: 'CELL-GNB-020-1', cellName: '广州-gNB-0020-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 178000 },
  { id: 'mrdm-010', deviceSn: 'GNB00030', deviceName: '深圳-gNB-0030', cellId: 'CELL-GNB-030-1', cellName: '深圳-gNB-0030-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 210000 },
  { id: 'mrdm-011', deviceSn: 'ENB00030', deviceName: '成都-eNB-0030', cellId: 'CELL-030-1', cellName: '成都-eNB-0030-Cell1', enabled: true, samplingInterval: 30, lastCollectTime: '2024-06-14T23:30:00.000Z', totalRecords: 68000 },
  { id: 'mrdm-012', deviceSn: 'ENB00040', deviceName: '西安-eNB-0040', cellId: 'CELL-040-1', cellName: '西安-eNB-0040-Cell1', enabled: false, samplingInterval: 60, totalRecords: 22000 },
  { id: 'mrdm-013', deviceSn: 'ENB00050', deviceName: '杭州-eNB-0050', cellId: 'CELL-050-1', cellName: '杭州-eNB-0050-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 135000 },
  { id: 'mrdm-014', deviceSn: 'GNB00040', deviceName: '武汉-gNB-0040', cellId: 'CELL-GNB-040-1', cellName: '武汉-gNB-0040-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 95000 },
  { id: 'mrdm-015', deviceSn: 'ENB00060', deviceName: '南京-eNB-0060', cellId: 'CELL-060-1', cellName: '南京-eNB-0060-Cell1', enabled: true, samplingInterval: 30, lastCollectTime: '2024-06-14T23:30:00.000Z', totalRecords: 42000 },
  { id: 'mrdm-016', deviceSn: 'GNB00050', deviceName: '长沙-gNB-0050', cellId: 'CELL-GNB-050-1', cellName: '长沙-gNB-0050-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 78000 },
  { id: 'mrdm-017', deviceSn: 'ENB00070', deviceName: '深圳-eNB-0070', cellId: 'CELL-070-1', cellName: '深圳-eNB-0070-Cell1', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 188000 },
  { id: 'mrdm-018', deviceSn: 'CPE00001', deviceName: '北京-CPE-0001', cellId: 'CELL-CPE-001-1', cellName: '北京-CPE-0001-Cell1', enabled: true, samplingInterval: 60, lastCollectTime: '2024-06-14T23:00:00.000Z', totalRecords: 15000 },
  { id: 'mrdm-019', deviceSn: 'ENB00002', deviceName: '北京-eNB-0002', cellId: 'CELL-002-2', cellName: '北京-eNB-0002-Cell2', enabled: true, samplingInterval: 30, lastCollectTime: '2024-06-14T23:30:00.000Z', totalRecords: 85000 },
  { id: 'mrdm-020', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', cellId: 'CELL-GNB-001-2', cellName: '北京-gNB-0001-Cell2', enabled: true, samplingInterval: 15, lastCollectTime: '2024-06-14T23:45:00.000Z', totalRecords: 195000 },
];

export const mockMRRecords: MRRecord[] = Array.from({ length: 20 }, (_, i) => ({
  id: `mrr-${String(i + 1).padStart(6, '0')}`,
  deviceSn: mockMRDeviceMappings[i % mockMRDeviceMappings.length].deviceSn,
  cellId: mockMRDeviceMappings[i % mockMRDeviceMappings.length].cellId,
  timestamp: new Date(Date.now() - i * 900000).toISOString(),
  indicators: {
    RSRP: parseFloat((-100 + Math.random() * 30).toFixed(1)),
    RSRQ: parseFloat((-15 + Math.random() * 8).toFixed(1)),
    SINR: parseFloat((5 + Math.random() * 15).toFixed(1)),
    CQI: Math.floor(8 + Math.random() * 7),
    PMI: Math.floor(Math.random() * 4),
    RI: Math.floor(1 + Math.random() * 3),
    TA: Math.floor(Math.random() * 5000),
  },
}));
