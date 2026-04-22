import type { KPI, Counter, Measurement, PerformanceThreshold, KPISeries } from '../../types/performance';
import { generateTimeSeries } from '../utils';

export const mockKPIs: KPI[] = [
  { id: 'kpi-001', kpiName: 'RRC连接成功率', kpiCode: 'RRC_SUCC_RATE', unit: '%', description: 'RRC连接建立成功率', category: '接入性' },
  { id: 'kpi-002', kpiName: 'E-RAB建立成功率', kpiCode: 'ERAB_SUCC_RATE', unit: '%', description: 'E-RAB承载建立成功率', category: '接入性' },
  { id: 'kpi-003', kpiName: '切换成功率', kpiCode: 'HO_SUCC_RATE', unit: '%', description: '小区内切换成功率', category: '移动性' },
  { id: 'kpi-004', kpiName: '下行吞吐率', kpiCode: 'DL_THROUGHPUT', unit: 'Mbps', description: '下行方向平均吞吐量', category: '吞吐量' },
  { id: 'kpi-005', kpiName: '上行吞吐率', kpiCode: 'UL_THROUGHPUT', unit: 'Mbps', description: '上行方向平均吞吐量', category: '吞吐量' },
  { id: 'kpi-006', kpiName: 'RRC重建比', kpiCode: 'RRC_REEST_RATIO', unit: '%', description: 'RRC连接重建占比', category: '保持性' },
  { id: 'kpi-007', kpiName: 'PDCP丢包率', kpiCode: 'PDCP_PKT_LOSS', unit: '%', description: 'PDCP层数据包丢失率', category: '质量' },
  { id: 'kpi-008', kpiName: '无线掉线率', kpiCode: 'RADIO_DROP_RATE', unit: '%', description: '无线承载掉线比例', category: '保持性' },
  { id: 'kpi-009', kpiName: 'PRB利用率', kpiCode: 'PRB_UTIL', unit: '%', description: '物理资源块利用率', category: '资源' },
  { id: 'kpi-010', kpiName: '用户面时延', kpiCode: 'USER_PLANE_DELAY', unit: 'ms', description: '用户面平均传输时延', category: '时延' },
  { id: 'kpi-011', kpiName: 'VOLTE接通率', kpiCode: 'VOLTE_SUCC_RATE', unit: '%', description: 'VoLTE通话接通成功率', category: '语音' },
  { id: 'kpi-012', kpiName: 'VOLTE掉话率', kpiCode: 'VOLTE_DROP_RATE', unit: '%', description: 'VoLTE通话掉线率', category: '语音' },
  { id: 'kpi-013', kpiName: 'CQI平均值', kpiCode: 'AVG_CQI', unit: '', description: '信道质量指示平均值', category: '质量' },
  { id: 'kpi-014', kpiName: 'SINR平均值', kpiCode: 'AVG_SINR', unit: 'dB', description: '信噪比平均值', category: '质量' },
  { id: 'kpi-015', kpiName: '最大在线用户数', kpiCode: 'MAX_UE_COUNT', unit: '个', description: '同时在线用户峰值', category: '容量' },
  { id: 'kpi-016', kpiName: '平均在线用户数', kpiCode: 'AVG_UE_COUNT', unit: '个', description: '平均在线用户数', category: '容量' },
  { id: 'kpi-017', kpiName: 'X2切换成功率', kpiCode: 'X2_HO_SUCC_RATE', unit: '%', description: 'X2接口切换成功率', category: '移动性' },
  { id: 'kpi-018', kpiName: 'S1切换成功率', kpiCode: 'S1_HO_SUCC_RATE', unit: '%', description: 'S1接口切换成功率', category: '移动性' },
  { id: 'kpi-019', kpiName: '上行干扰电平', kpiCode: 'UL_INTERFERENCE', unit: 'dBm', description: '上行接收干扰电平', category: '质量' },
  { id: 'kpi-020', kpiName: '天线校准成功率', kpiCode: 'ANT_CALIB_RATE', unit: '%', description: '天线校准成功率', category: '射频' },
];

export const mockCounters: Counter[] = [
  { id: 'cnt-001', counterName: 'RRC连接建立请求次数', counterCode: 'RRC_CONN_ESTAB_ATT', unit: '次', description: 'RRC连接建立尝试次数', category: '接入性', kpiIds: ['kpi-001'] },
  { id: 'cnt-002', counterName: 'RRC连接建立成功次数', counterCode: 'RRC_CONN_ESTAB_SUCC', unit: '次', description: 'RRC连接建立成功次数', category: '接入性', kpiIds: ['kpi-001'] },
  { id: 'cnt-003', counterName: 'E-RAB建立请求次数', counterCode: 'ERAB_ESTAB_ATT', unit: '次', description: 'E-RAB建立尝试次数', category: '接入性', kpiIds: ['kpi-002'] },
  { id: 'cnt-004', counterName: 'E-RAB建立成功次数', counterCode: 'ERAB_ESTAB_SUCC', unit: '次', description: 'E-RAB建立成功次数', category: '接入性', kpiIds: ['kpi-002'] },
  { id: 'cnt-005', counterName: '切换请求次数', counterCode: 'HO_ATT', unit: '次', description: '切换尝试次数', category: '移动性', kpiIds: ['kpi-003'] },
  { id: 'cnt-006', counterName: '切换成功次数', counterCode: 'HO_SUCC', unit: '次', description: '切换成功次数', category: '移动性', kpiIds: ['kpi-003'] },
  { id: 'cnt-007', counterName: '下行PDCP字节数', counterCode: 'DL_PDCP_BYTES', unit: 'bytes', description: '下行PDCP层传输字节数', category: '吞吐量', kpiIds: ['kpi-004'] },
  { id: 'cnt-008', counterName: '上行PDCP字节数', counterCode: 'UL_PDCP_BYTES', unit: 'bytes', description: '上行PDCP层传输字节数', category: '吞吐量', kpiIds: ['kpi-005'] },
];

const deviceSnList = ['ENB00001', 'ENB00002', 'GNB00001', 'GNB00002', 'CPE00001'];
const granularities: Array<Measurement['granularity']> = ['15min', '30min', '1h', '1d'];

function generateMeasurements(): Measurement[] {
  const measurements: Measurement[] = [];
  let idx = 0;
  for (const kpi of mockKPIs.slice(0, 5)) {
    for (const sn of deviceSnList.slice(0, 3)) {
      for (let i = 0; i < 5; i++) {
        const ts = new Date(Date.now() - i * 3600000).toISOString();
        measurements.push({
          id: `meas-${String(++idx).padStart(5, '0')}`,
          measurementName: `${kpi.kpiName}测量`,
          measurementCode: kpi.kpiCode,
          deviceSn: sn,
          deviceName: `设备-${sn}`,
          kpiCode: kpi.kpiCode,
          value: parseFloat((Math.random() * 40 + 60).toFixed(2)),
          unit: kpi.unit,
          timestamp: ts,
          granularity: granularities[i % granularities.length],
        });
      }
    }
  }
  return measurements;
}

export const mockMeasurements: Measurement[] = generateMeasurements();

export const mockThresholds: PerformanceThreshold[] = [
  {
    id: 'thr-001',
    thresholdName: 'RRC连接成功率阈值',
    kpiCode: 'RRC_SUCC_RATE',
    kpiName: 'RRC连接成功率',
    operator: 'lt',
    warningValue: 95,
    criticalValue: 90,
    unit: '%',
    enabled: true,
    deviceGroups: ['grp-001', 'grp-002'],
    createTime: '2024-01-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
  },
  {
    id: 'thr-002',
    thresholdName: 'E-RAB建立成功率阈值',
    kpiCode: 'ERAB_SUCC_RATE',
    kpiName: 'E-RAB建立成功率',
    operator: 'lt',
    warningValue: 95,
    criticalValue: 90,
    unit: '%',
    enabled: true,
    deviceGroups: ['grp-001'],
    createTime: '2024-01-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
  },
  {
    id: 'thr-003',
    thresholdName: '下行吞吐率阈值',
    kpiCode: 'DL_THROUGHPUT',
    kpiName: '下行吞吐率',
    operator: 'lt',
    warningValue: 50,
    criticalValue: 20,
    unit: 'Mbps',
    enabled: true,
    deviceGroups: ['grp-001', 'grp-002', 'grp-003'],
    createTime: '2024-01-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
  },
  {
    id: 'thr-004',
    thresholdName: '无线掉线率阈值',
    kpiCode: 'RADIO_DROP_RATE',
    kpiName: '无线掉线率',
    operator: 'gt',
    warningValue: 1,
    criticalValue: 3,
    unit: '%',
    enabled: true,
    deviceGroups: ['grp-001'],
    createTime: '2024-01-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
  },
  {
    id: 'thr-005',
    thresholdName: 'PRB利用率阈值',
    kpiCode: 'PRB_UTIL',
    kpiName: 'PRB利用率',
    operator: 'gt',
    warningValue: 80,
    criticalValue: 95,
    unit: '%',
    enabled: false,
    deviceGroups: [],
    createTime: '2024-01-01T00:00:00.000Z',
    updateTime: '2024-06-01T00:00:00.000Z',
  },
];

const kpiSeriesConfig: Array<{ kpi: KPI; base: number; variance: number }> = [
  { kpi: mockKPIs[0], base: 97.5, variance: 2.5 },
  { kpi: mockKPIs[1], base: 96.8, variance: 3.0 },
  { kpi: mockKPIs[2], base: 95.2, variance: 4.0 },
  { kpi: mockKPIs[3], base: 85.0, variance: 20.0 },
  { kpi: mockKPIs[4], base: 35.0, variance: 10.0 },
  { kpi: mockKPIs[5], base: 1.2, variance: 0.8 },
  { kpi: mockKPIs[6], base: 0.5, variance: 0.3 },
  { kpi: mockKPIs[7], base: 0.8, variance: 0.5 },
];

export const mockKPISeries: KPISeries[] = kpiSeriesConfig.map(({ kpi, base, variance }) => {
  const rawSeries = generateTimeSeries(7, 60, base, variance);
  return {
    kpiName: kpi.kpiName,
    unit: kpi.unit,
    data: rawSeries.map(([timestamp, value]) => ({ timestamp, value })),
  };
});
