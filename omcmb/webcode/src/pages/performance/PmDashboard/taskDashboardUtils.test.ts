import { describe, it, expect } from 'vitest';
import type { AdhocResultRow } from '@core/types/pmAdhoc';
import {
  seriesKeyOf,
  seriesLabelOf,
  buildMetricCharts,
  buildTrustedSeriesIdentities,
  formatMetricChartDisplayName,
  ensureConfiguredMetricCharts,
  filterChartsByMetricPaths,
  filterRowsByMetricPaths,
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
    unit: p.unit,
    metricType: 'counter',
    metricValue: p.metricValue === undefined ? 0 : p.metricValue,
    statisType: 'sum',
    granularity: p.granularity ?? 'hourly',
    time: p.startTime ?? '2026-05-30T00:00:00Z',
    startTime: p.startTime ?? '2026-05-30T00:00:00Z',
    endTime: p.endTime ?? '2026-05-30T01:00:00Z',
    productId: p.productId,
    objectLdn: p.objectLdn,
    productName: p.productName,
    deviceGroupName: p.deviceGroupName,
    periodComplete: p.periodComplete,
    partial: p.partial,
    receivedSlots: p.receivedSlots,
    expectedSlots: p.expectedSlots,
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
  it('英文环境下使用英文系统前缀', () => {
    expect(seriesLabelOf('__network__', 'network', undefined, 'en-US')).toBe('Network');
    expect(seriesLabelOf('Band=42', 'band', undefined, 'en-US')).toBe('Band 42');
    expect(seriesLabelOf('DeviceGroup=abcd1234-ef', 'device_group', '华东一区', 'en-US')).toBe(
      'Device Group 华东一区',
    );
    expect(seriesLabelOf('9259a43e-301f-49b5', 'product', 'CMCC 皮基站 LTE', 'en-US')).toBe(
      'Product CMCC 皮基站 LTE',
    );
    expect(seriesLabelOf('Cellid=111172245,PLMN=46068', 'aggregate_group', undefined, 'en-US')).toBe(
      'Aggregate Group Cellid=111172245,PLMN=46068',
    );
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
  it('图标题同时展示指标名称和指标 ID', () => {
    const rows = [row({ metricPath: 'K001', displayName: 'RRC 成功率', startTime: 't0', metricValue: 1 })];
    const charts = buildMetricCharts(rows, 'network', 'hourly');
    expect(charts[0].displayName).toBe('RRC 成功率（K001）');
  });

  it('结果行名称等于指标 ID 时，用指标库名称兜底生成图标题', () => {
    const rows = [row({ metricPath: 'C001', displayName: 'C001', startTime: 't0', metricValue: 1 })];
    const charts = buildMetricCharts(
      rows,
      'network',
      'hourly',
      'zh-CN',
      new Map([['C001', '上行流量原始计数']]),
    );
    expect(charts[0].displayName).toBe('上行流量原始计数（C001）');
  });

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

  it("NULL 缺值结果保留桶和系列，单元补 '-' 断点", () => {
    const rows = [
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't0', metricValue: null }),
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't1', metricValue: 2 }),
    ];
    const charts = buildMetricCharts(rows, 'device', 'hourly');
    expect(charts[0].buckets).toEqual(['t0', 't1']);
    expect(charts[0].series[0].values).toEqual(['-', 2]);
  });

  it("NaN/Infinity 缺值结果保留桶和系列，单元补 '-' 断点", () => {
    const rows = [
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't0', metricValue: Number.NaN }),
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't1', metricValue: Number.POSITIVE_INFINITY }),
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't2', metricValue: 2 }),
    ];
    const charts = buildMetricCharts(rows, 'device', 'hourly');
    expect(charts[0].buckets).toEqual(['t0', 't1', 't2']);
    expect(charts[0].series[0].values).toEqual(['-', '-', 2]);
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

  it('daily/weekly 进行中行与首页一致进入图表点、图例和 tooltip 桶', () => {
    const rows = [
      row({
        metricPath: 'M1',
        granularity: 'daily',
        startTime: '2026-08-03T08:00:00+08:00',
        endTime: '2026-08-04T08:00:00+08:00',
        metricValue: 99,
        partial: true,
        periodComplete: false,
      }),
      row({
        metricPath: 'M1',
        granularity: 'weekly',
        startTime: '2026-08-03T08:00:00+08:00',
        endTime: '2026-08-10T08:00:00+08:00',
        metricValue: 88,
        partial: true,
        periodComplete: false,
      }),
    ];

    const daily = buildMetricCharts(rows, 'network', 'daily');
    const weekly = buildMetricCharts(rows, 'network', 'weekly');

    expect(daily).toHaveLength(1);
    expect(daily[0].series[0].values).toEqual([99]);
    expect(weekly).toHaveLength(1);
    expect(weekly[0].series[0].values).toEqual([88]);
  });

  it('正式 daily rows 可以正常出图', () => {
    const charts = buildMetricCharts([
      row({
        metricPath: 'M1',
        granularity: 'daily',
        startTime: '2026-08-02T00:00:00+08:00',
        endTime: '2026-08-03T00:00:00+08:00',
        metricValue: 42,
        partial: false,
        periodComplete: true,
      }),
    ], 'network', 'daily');

    expect(charts).toHaveLength(1);
    expect(charts[0].buckets).toEqual(['2026-08-02T00:00:00+08:00']);
    expect(charts[0].bucketEnds).toEqual(['2026-08-03T00:00:00+08:00']);
    expect(charts[0].series[0].values).toEqual([42]);
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
    expect(charts[0].displayName).toBe('RRC 成功率（K001）');
    expect(charts[0].series[0].name).toBe('全网');
  });
  it('同一指标多行单位一致时，图表保留该单位', () => {
    const rows = [
      row({ metricPath: 'K001', unit: '%', startTime: 't0', metricValue: 98.1 }),
      row({ metricPath: 'K001', unit: '%', startTime: 't1', metricValue: 98.2 }),
    ];
    const charts = buildMetricCharts(rows, 'network', 'hourly');
    expect(charts[0].unit).toBe('%');
  });

  it('单位为空时保持 undefined；同一指标单位冲突时不展示单位', () => {
    expect(
      buildMetricCharts([row({ metricPath: 'K001', unit: '   ' })], 'network', 'hourly')[0].unit,
    ).toBeUndefined();
    const charts = buildMetricCharts(
      [
        row({ metricPath: 'K001', unit: '%', startTime: 't0' }),
        row({ metricPath: 'K001', unit: 'Mbps', startTime: 't1' }),
      ],
      'network',
      'hourly',
    );
    expect(charts[0].unit).toBeUndefined();
  });

  it('英文环境下图表系列名使用英文系统前缀', () => {
    const rows = [row({ metricPath: 'M1', objectLdn: 'Band=42', startTime: 't0', metricValue: 1 })];
    const charts = buildMetricCharts(rows, 'band', 'hourly', 'en-US');
    expect(charts[0].series[0].name).toBe('Band 42');
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

describe('buildMetricCharts — #194 设备组 legend 固定全集（缺指标留断点不丢组）', () => {
  // 任务覆盖两个设备组：上行流量(M_TRAFFIC)两组都有；E-RAB掉线率(M_ERAB)只有 GA 有（GB 分母 counter 缺失被后端跳过）。
  // 期望：M_ERAB 图也要列出 GA、GB 两条 legend，GB 全为 '-'（断点），不能整组消失。
  const groupRows: AdhocResultRow[] = [
    row({ metricPath: 'M_TRAFFIC', objectLdn: 'DeviceGroup=GA', deviceGroupName: '华东', startTime: 't0', metricValue: 10 }),
    row({ metricPath: 'M_TRAFFIC', objectLdn: 'DeviceGroup=GB', deviceGroupName: '华南', startTime: 't0', metricValue: 20 }),
    row({ metricPath: 'M_ERAB', objectLdn: 'DeviceGroup=GA', deviceGroupName: '华东', startTime: 't0', metricValue: 0.5 }),
    // M_ERAB 没有 GB 行（GB 该指标被后端跳过）
  ];

  it('device_group：某指标缺某组 → 该图仍含全集 legend，缺组留 \'-\' 断点', () => {
    const charts = buildMetricCharts(groupRows, 'device_group', 'hourly');
    const erab = charts.find((c) => c.metricPath === 'M_ERAB')!;
    const keys = erab.series.map((s) => s.key).sort();
    expect(keys).toEqual(['DeviceGroup=GA', 'DeviceGroup=GB']);
    const byKey = Object.fromEntries(erab.series.map((s) => [s.key, s]));
    expect(byKey['DeviceGroup=GA'].values).toEqual([0.5]);
    // GB 在 M_ERAB 缺数据 → 全 '-'（断点），不是 0 假点
    expect(byKey['DeviceGroup=GB'].values).toEqual(['-']);
    // 名字沿用别的指标里解析到的组名（华南），不退化成 uuid
    expect(byKey['DeviceGroup=GB'].name).toBe('设备组 华南');
  });

  it('device 维度不套全集：缺设备的指标 legend 不补该设备（设备真无数据有意义）', () => {
    const rows = [
      row({ metricPath: 'M1', deviceSn: 'SN-A', startTime: 't0', metricValue: 1 }),
      row({ metricPath: 'M2', deviceSn: 'SN-B', startTime: 't0', metricValue: 2 }),
    ];
    const charts = buildMetricCharts(rows, 'device', 'hourly');
    const m1 = charts.find((c) => c.metricPath === 'M1')!;
    expect(m1.series.map((s) => s.key)).toEqual(['SN-A']); // 不含 SN-B
  });
});

describe('filterChartsByMetricPaths — T-0194 按任务已选指标过滤出图', () => {
  const rows: AdhocResultRow[] = [
    row({ metricPath: 'K1', deviceSn: 'SN-A' }),
    row({ metricPath: 'K2', deviceSn: 'SN-A' }),
    row({ metricPath: 'K3', deviceSn: 'SN-A' }),
  ];
  const charts = buildMetricCharts(rows, 'device', 'hourly');

  it('只保留清单内的图，并按任务配置顺序返回', () => {
    const out = filterChartsByMetricPaths(charts, ['K1', 'K3']);
    expect(out.map((c) => c.metricPath)).toEqual(['K1', 'K3']);

    const reordered = filterChartsByMetricPaths(charts, ['K3', 'K1']);
    expect(reordered.map((c) => c.metricPath)).toEqual(['K3', 'K1']);
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

  it('内置-产品-eNB 14 指标验收：额外结果指标不会让右侧图表超过 14 个', () => {
    const taskMetrics = Array.from({ length: 14 }, (_, index) => `K-ENB-${String(index + 1).padStart(2, '0')}`);
    const rows = [
      ...taskMetrics.map((metricPath) => row({ metricPath, productId: 'prod-1', startTime: 't0', metricValue: 1 })),
      row({ metricPath: 'K-EXTRA-1', productId: 'prod-1', startTime: 't0', metricValue: 9 }),
      row({ metricPath: 'C-EXTRA-2', productId: 'prod-1', startTime: 't0', metricValue: 10 }),
    ];

    const out = filterChartsByMetricPaths(
      buildMetricCharts(filterRowsByMetricPaths(rows, taskMetrics), 'product', 'hourly'),
      taskMetrics,
    );

    expect(out).toHaveLength(14);
    expect(out.map((c) => c.metricPath)).toEqual(taskMetrics);
  });
});

describe('ensureConfiguredMetricCharts — #198 任务指标空图骨架', () => {
  it('混合有/无结果时仍按 metric_paths 生成全部图卡，空指标不补 0', () => {
    const charts = buildMetricCharts(
      [row({ metricPath: 'K1', deviceSn: 'SN-A', startTime: 't0', metricValue: 7 })],
      'device',
      'hourly',
    );

    const out = ensureConfiguredMetricCharts(
      charts,
      ['K1', 'K2', 'K1'],
      [{ key: 'SN-A', name: 'SN-A' }],
      new Map([['K2', 'VoLTE 接通率']]),
    );

    expect(out.map((chart) => chart.metricPath)).toEqual(['K1', 'K2']);
    expect(out[0].series[0].values).toEqual([7]);
    expect(out[1]).toMatchObject({
      metricPath: 'K2',
      displayName: 'VoLTE 接通率（K2）',
      buckets: [],
      bucketEnds: [],
      series: [{ key: 'SN-A', name: 'SN-A', values: [] }],
    });
    expect(out[1].series.flatMap((series) => series.values)).not.toContain(0);
  });
});

describe('formatMetricChartDisplayName — 指标标题统一格式', () => {
  it('有名称时显示 名称（ID），名称缺失或等于 ID 时只显示 ID', () => {
    expect(formatMetricChartDisplayName('K001', 'RRC 成功率')).toBe('RRC 成功率（K001）');
    expect(formatMetricChartDisplayName('K001', 'K001')).toBe('K001');
    expect(formatMetricChartDisplayName('K001', '  ')).toBe('K001');
    expect(formatMetricChartDisplayName('K001')).toBe('K001');
  });
});

describe('buildTrustedSeriesIdentities — #198 空图系列只取可信骨架', () => {
  const labels = {
    network: '全网',
    band: '频段',
    deviceGroup: '设备组',
    product: '产品',
    aggregateGroup: '聚合组',
  };

  it('按维度选择任务配置或 filter-options，未知来源不造 __unknown__ 假线', () => {
    expect(buildTrustedSeriesIdentities('network', {}, labels)).toEqual([
      { key: '__network__', name: '全网' },
    ]);
    expect(
      buildTrustedSeriesIdentities('device', { deviceSns: ['SN-A', 'SN-A', 'SN-B'] }, labels),
    ).toEqual([
      { key: 'SN-A', name: 'SN-A' },
      { key: 'SN-B', name: 'SN-B' },
    ]);
    expect(
      buildTrustedSeriesIdentities('aggregate_group', {
        objectLdns: ['Cell=1', 'Cell=2'],
      }, labels),
    ).toEqual([
      { key: 'Cell=1', name: '聚合组 Cell=1' },
      { key: 'Cell=2', name: '聚合组 Cell=2' },
    ]);
    expect(
      buildTrustedSeriesIdentities('product', {
        filterOptions: [
          { value: 'prod-a', label: '产品 A' },
          { value: 'prod-b', label: '产品 B' },
        ],
        selectedKeys: ['prod-b'],
      }, labels),
    ).toEqual([{ key: 'prod-b', name: '产品 B' }]);
    expect(
      buildTrustedSeriesIdentities('device_group', {
        filterOptions: [{ value: 'DeviceGroup=group-a', label: '华东' }],
      }, labels),
    ).toEqual([{ key: 'DeviceGroup=group-a', name: '设备组 华东' }]);
    expect(
      buildTrustedSeriesIdentities('band', {
        filterOptions: [{ value: 'Band=78', label: 'Band=78' }],
      }, labels),
    ).toEqual([{ key: 'Band=78', name: '频段 78' }]);
    expect(buildTrustedSeriesIdentities('product', {}, labels)).toEqual([]);
  });

  it('系列前缀由调用方国际化资源注入，不在纯函数内写死语言', () => {
    const englishLabels = {
      network: 'Network',
      band: 'Band',
      deviceGroup: 'Device Group',
      product: 'Product',
      aggregateGroup: 'Aggregate Group',
    };

    expect(buildTrustedSeriesIdentities('network', {}, englishLabels)).toEqual([
      { key: '__network__', name: 'Network' },
    ]);
    expect(
      buildTrustedSeriesIdentities('product', {
        filterOptions: [{ value: 'prod-a', label: 'Product A' }],
      }, englishLabels),
    ).toEqual([{ key: 'prod-a', name: 'Product A' }]);
  });
});

describe('filterRowsByMetricPaths — #192 按任务已选指标过滤原始结果行', () => {
  it('配置外结果行不会进入出图数据源', () => {
    const rows = [
      row({ metricPath: 'K1', deviceSn: 'SN-A' }),
      row({ metricPath: 'K2', deviceSn: 'SN-A' }),
      row({ metricPath: 'K-EXTRA', deviceSn: 'SN-A' }),
    ];

    const out = filterRowsByMetricPaths(rows, ['K1', 'K2']);

    expect(out.map((r) => r.metricPath)).toEqual(['K1', 'K2']);
  });

  it('先滤原始行可防止额外指标污染固定 legend 全集', () => {
    const rows = [
      row({ metricPath: 'K1', objectLdn: 'DeviceGroup=GA', deviceGroupName: '华东', startTime: 't0', metricValue: 1 }),
      row({ metricPath: 'K-EXTRA', objectLdn: 'DeviceGroup=GB', deviceGroupName: '华南', startTime: 't0', metricValue: 9 }),
    ];

    const charts = buildMetricCharts(
      filterRowsByMetricPaths(rows, ['K1']),
      'device_group',
      'hourly',
    );

    expect(charts.map((c) => c.metricPath)).toEqual(['K1']);
    expect(charts[0].series.map((s) => s.key)).toEqual(['DeviceGroup=GA']);
  });

  it('清单为空或 undefined 时保留原始行，兼容历史边界任务', () => {
    const rows = [row({ metricPath: 'K1' }), row({ metricPath: 'K2' })];

    expect(filterRowsByMetricPaths(rows, [])).toBe(rows);
    expect(filterRowsByMetricPaths(rows, undefined)).toBe(rows);
  });
});
