import { describe, it, expect } from 'vitest';
import type { AdhocResultRow } from '@core/types/pmAdhoc';
import {
  seriesKeyOf,
  seriesLabelOf,
  buildMetricCharts,
  filterChartsByMetricPaths,
} from './taskDashboardUtils';

// 构造结果行的小工厂——只填测试关心的字段，其余给确定默认值。
function row(p: Partial<AdhocResultRow>): AdhocResultRow {
  return {
    id: p.id ?? Math.random().toString(36).slice(2),
    taskId: 't1',
    deviceOui: '',
    deviceSn: p.deviceSn ?? 'AGGREGATED',
    metricPath: p.metricPath ?? 'M1',
    displayName: p.displayName,
    metricType: 'counter',
    metricValue: p.metricValue ?? 0,
    statisType: 'sum',
    granularity: p.granularity ?? 'hourly',
    time: p.startTime ?? '2026-05-30T00:00:00Z',
    startTime: p.startTime ?? '2026-05-30T00:00:00Z',
    endTime: p.endTime ?? '2026-05-30T01:00:00Z',
    productId: p.productId,
    objectLdn: p.objectLdn,
    productName: p.productName,
    deviceGroupName: p.deviceGroupName,
  };
}

describe('seriesKeyOf — 6 维度系列键派生', () => {
  it('network → 固定单线键', () => {
    expect(seriesKeyOf(row({ deviceSn: 'AGGREGATED' }), 'network')).toBe('__network__');
  });
  it('device → deviceSn', () => {
    expect(seriesKeyOf(row({ deviceSn: 'SN-A' }), 'device')).toBe('SN-A');
  });
  it('band → objectLdn(Band=)', () => {
    expect(seriesKeyOf(row({ objectLdn: 'Band=42' }), 'band')).toBe('Band=42');
  });
  it('device_group → objectLdn(DeviceGroup=)', () => {
    expect(
      seriesKeyOf(row({ objectLdn: 'DeviceGroup=abcd1234-ef' }), 'device_group'),
    ).toBe('DeviceGroup=abcd1234-ef');
  });
  it('product → productId', () => {
    expect(seriesKeyOf(row({ productId: 'prod-9259a43e' }), 'product')).toBe('prod-9259a43e');
  });
  it('aggregate_group → objectLdn', () => {
    expect(
      seriesKeyOf(row({ objectLdn: 'Cellid=111172245,PLMN=46068' }), 'aggregate_group'),
    ).toBe('Cellid=111172245,PLMN=46068');
  });
});

describe('seriesLabelOf — 系列标签', () => {
  it('network → 全网', () => {
    expect(seriesLabelOf('__network__', 'network')).toBe('全网');
  });
  it('device → SN 原样', () => {
    expect(seriesLabelOf('SN-A', 'device')).toBe('SN-A');
  });
  it('band → 剥前缀', () => {
    expect(seriesLabelOf('Band=42', 'band')).toBe('频段 42');
  });
  it('device_group → 剥前缀 + uuid 前 8', () => {
    expect(seriesLabelOf('DeviceGroup=abcd1234- effff', 'device_group')).toBe('设备组 abcd1234');
  });
  it('product → id 前 8', () => {
    expect(seriesLabelOf('9259a43e-301f-49b5', 'product')).toBe('产品 9259a43e');
  });

  // PM-线名解析：传入后端解析名时显示可读名，缺失回退 id 前 8。
  it('product → 命中解析名显示产品名', () => {
    expect(seriesLabelOf('9259a43e-301f-49b5', 'product', 'CMCC 皮基站 LTE')).toBe('产品 CMCC 皮基站 LTE');
  });
  it('product → 解析名缺失回退 id 前 8', () => {
    expect(seriesLabelOf('9259a43e-301f-49b5', 'product', undefined)).toBe('产品 9259a43e');
    expect(seriesLabelOf('9259a43e-301f-49b5', 'product', '')).toBe('产品 9259a43e');
  });
  it('device_group → 命中解析名显示组名', () => {
    expect(seriesLabelOf('DeviceGroup=abcd1234-ef', 'device_group', '华东一区')).toBe('设备组 华东一区');
  });
  it('device_group → 解析名缺失回退 uuid 前 8', () => {
    expect(seriesLabelOf('DeviceGroup=abcd1234-ef', 'device_group', undefined)).toBe('设备组 abcd1234');
    expect(seriesLabelOf('DeviceGroup=abcd1234-ef', 'device_group', '')).toBe('设备组 abcd1234');
  });
});

describe('buildMetricCharts — 转置', () => {
  it('按指标分图：N 指标 → N 张图', () => {
    const rows = [
      row({ metricPath: 'M1', startTime: 't0', metricValue: 1 }),
      row({ metricPath: 'M2', startTime: 't0', metricValue: 2 }),
    ];
    const charts = buildMetricCharts(rows, 'network', 'hourly');
    expect(charts.map((c) => c.metricPath).sort()).toEqual(['M1', 'M2']);
    expect(charts).toHaveLength(2);
  });

  it('只取选中粒度的行', () => {
    const rows = [
      row({ metricPath: 'M1', granularity: 'hourly', startTime: 't0', metricValue: 1 }),
      row({ metricPath: 'M1', granularity: 'daily', startTime: 't0', metricValue: 9 }),
    ];
    const charts = buildMetricCharts(rows, 'network', 'hourly');
    expect(charts).toHaveLength(1);
    expect(charts[0].series[0].values).toEqual([1]);
  });

  it('多设备不互相覆盖：同图两条线', () => {
    const rows = [
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't0', metricValue: 10 }),
      row({ metricPath: 'M1', deviceSn: 'SN-B', startTime: 't0', metricValue: 20 }),
    ];
    const charts = buildMetricCharts(rows, 'device', 'hourly');
    expect(charts).toHaveLength(1);
    expect(charts[0].series).toHaveLength(2);
    const byKey = Object.fromEntries(charts[0].series.map((s) => [s.key, s.values]));
    expect(byKey['SN-A']).toEqual([10]);
    expect(byKey['SN-B']).toEqual([20]);
  });

  it('多组(band)不互相覆盖：两条线各自对齐', () => {
    const rows = [
      row({ metricPath: 'M1', objectLdn: 'Band=42', startTime: 't0', metricValue: 1 }),
      row({ metricPath: 'M1', objectLdn: 'Band=42', startTime: 't1', metricValue: 2 }),
      row({ metricPath: 'M1', objectLdn: 'Band=78', startTime: 't0', metricValue: 100 }),
      row({ metricPath: 'M1', objectLdn: 'Band=78', startTime: 't1', metricValue: 200 }),
    ];
    const charts = buildMetricCharts(rows, 'band', 'hourly');
    expect(charts[0].buckets).toEqual(['t0', 't1']);
    const byKey = Object.fromEntries(charts[0].series.map((s) => [s.key, s.values]));
    expect(byKey['Band=42']).toEqual([1, 2]);
    expect(byKey['Band=78']).toEqual([100, 200]);
  });

  it("缺桶补 '-'：某线缺一个桶则该位为 '-'", () => {
    const rows = [
      // SN-A 有 t0、t1 两桶；SN-B 只有 t0 → SN-B 在 t1 应补 '-'
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't0', metricValue: 1 }),
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't1', metricValue: 2 }),
      row({ metricPath: 'M1', deviceSn: 'SN-B', startTime: 't0', metricValue: 5 }),
    ];
    const charts = buildMetricCharts(rows, 'device', 'hourly');
    expect(charts[0].buckets).toEqual(['t0', 't1']);
    const byKey = Object.fromEntries(charts[0].series.map((s) => [s.key, s.values]));
    expect(byKey['SN-A']).toEqual([1, 2]);
    expect(byKey['SN-B']).toEqual([5, '-']);
  });

  it('桶按 startTime 升序去重', () => {
    const rows = [
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't2', metricValue: 3 }),
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't0', metricValue: 1 }),
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't1', metricValue: 2 }),
    ];
    const charts = buildMetricCharts(rows, 'device', 'hourly');
    expect(charts[0].buckets).toEqual(['t0', 't1', 't2']);
    expect(charts[0].series[0].values).toEqual([1, 2, 3]);
  });

  it('空数据 → 空图列表', () => {
    expect(buildMetricCharts([], 'network', 'hourly')).toEqual([]);
  });

  it('bucketEnds 与 buckets 一一对应记录每桶结束时间', () => {
    const rows = [
      row({ metricPath: 'M1', startTime: 't0', endTime: 't0-end', metricValue: 1 }),
      row({ metricPath: 'M1', startTime: 't1', endTime: 't1-end', metricValue: 2 }),
    ];
    const charts = buildMetricCharts(rows, 'network', 'hourly');
    expect(charts[0].buckets).toEqual(['t0', 't1']);
    expect(charts[0].bucketEnds).toEqual(['t0-end', 't1-end']);
  });

  it('displayName 回填系列名（network 仍用维度标签）', () => {
    const rows = [row({ metricPath: 'K001', displayName: 'RRC 成功率', startTime: 't0', metricValue: 1 })];
    const charts = buildMetricCharts(rows, 'network', 'hourly');
    expect(charts[0].displayName).toBe('RRC 成功率');
    expect(charts[0].series[0].name).toBe('全网');
  });

  // PM-线名解析：product 维度系列名优先用后端解析名；缺失回退 id 前 8。
  it('product 维度：解析名命中显示产品名', () => {
    const rows = [
      row({ metricPath: 'M1', productId: '9259a43e-301f', productName: 'CMCC 皮基站', startTime: 't0', metricValue: 1 }),
    ];
    const charts = buildMetricCharts(rows, 'product', 'hourly');
    expect(charts[0].series[0].name).toBe('产品 CMCC 皮基站');
  });
  it('product 维度：解析名缺失回退 id 前 8', () => {
    const rows = [
      row({ metricPath: 'M1', productId: '9259a43e-301f', productName: undefined, startTime: 't0', metricValue: 1 }),
    ];
    const charts = buildMetricCharts(rows, 'product', 'hourly');
    expect(charts[0].series[0].name).toBe('产品 9259a43e');
  });

  // PM-线名解析：device_group 维度系列名优先用后端解析名；缺失回退 uuid 前 8。
  it('device_group 维度：解析名命中显示组名', () => {
    const rows = [
      row({ metricPath: 'M1', objectLdn: 'DeviceGroup=abcd1234-ef', deviceGroupName: '华东一区', startTime: 't0', metricValue: 1 }),
    ];
    const charts = buildMetricCharts(rows, 'device_group', 'hourly');
    expect(charts[0].series[0].name).toBe('设备组 华东一区');
  });
  it('device_group 维度：解析名缺失回退 uuid 前 8', () => {
    const rows = [
      row({ metricPath: 'M1', objectLdn: 'DeviceGroup=abcd1234-ef', deviceGroupName: undefined, startTime: 't0', metricValue: 1 }),
    ];
    const charts = buildMetricCharts(rows, 'device_group', 'hourly');
    expect(charts[0].series[0].name).toBe('设备组 abcd1234');
  });
});

describe('filterChartsByMetricPaths — T-0194 按任务已选指标过滤出图', () => {
  const rows: AdhocResultRow[] = [
    row({ metricPath: 'K1', deviceSn: 'SN-A' }),
    row({ metricPath: 'K2', deviceSn: 'SN-A' }),
    row({ metricPath: 'K3', deviceSn: 'SN-A' }),
  ];
  const charts = buildMetricCharts(rows, 'device', 'hourly');

  it('只保留清单内的图（数量与清单一致）', () => {
    const out = filterChartsByMetricPaths(charts, ['K1', 'K3']);
    expect(out.map((c) => c.metricPath)).toEqual(['K1', 'K3']);
  });

  it('清单为空 → 不过滤（兜底全画）', () => {
    expect(filterChartsByMetricPaths(charts, [])).toHaveLength(3);
  });

  it('清单为 undefined → 不过滤（兜底全画）', () => {
    expect(filterChartsByMetricPaths(charts, undefined)).toHaveLength(3);
  });

  it('清单含不存在的指标 → 只画命中的、不报错', () => {
    const out = filterChartsByMetricPaths(charts, ['K2', 'K999']);
    expect(out.map((c) => c.metricPath)).toEqual(['K2']);
  });
});
