/**
 * 首页 KPI 配置页编辑逻辑单测（issue #213 S3，纯逻辑）
 *
 * 覆盖：按制式过滤指标库（LTE 不混入 NR/GSM）、选项置灰、增删图、改标题/指标、
 * 网格坐标回写、存盘形状剥 id —— 含成功路径与边界/失败路径。
 */

import { describe, it, expect } from 'vitest';
import type { KPITechDefinitions, KPILayoutPanel } from '@core/types/dashboard';
import {
  addPanel,
  applyGridLayout,
  definitionsForTech,
  removePanel,
  toGridLayout,
  toMetricOptions,
  toSavePanels,
  toWorkingPanels,
  updatePanelMetrics,
  updatePanelTitle,
  HALF_WIDTH,
  DEFAULT_HEIGHT,
  type WorkingPanel,
} from './kpiConfigLogic';

const DEFINITIONS: KPITechDefinitions[] = [
  {
    tech: 'lte',
    items: [
      { key: 'K900010015', k_code: 'K900010015', cn_name: '下行流量', unit: 'MByte', panel: 'traffic', needs_review: false, available: true },
      { key: 'K900010076', k_code: 'K900010076', cn_name: '小区可用率', unit: '%', panel: 'availability', needs_review: false, available: true },
    ],
  },
  {
    tech: 'nr',
    items: [
      { key: 'KGNB0511', k_code: 'KGNB0511', cn_name: 'NR下行流量', unit: 'MByte', panel: 'traffic', needs_review: false, available: true },
    ],
  },
  {
    tech: 'gsm',
    items: [
      { key: 'KGSM0102', k_code: 'KGSM0102', cn_name: '电话成功率', unit: '%', panel: 'accessibility', needs_review: false, available: true },
    ],
  },
];

function panel(overrides: Partial<KPILayoutPanel> = {}): KPILayoutPanel {
  return {
    title: 't',
    metrics: ['M1'],
    x: 0,
    y: 0,
    w: 6,
    h: 8,
    chartType: 'line',
    ...overrides,
  };
}

describe('definitionsForTech — 按制式过滤指标库', () => {
  it('LTE tab 只返回 4G/ENB 指标，不混入 NR/GSM', () => {
    const items = definitionsForTech(DEFINITIONS, 'lte');
    const keys = items.map((i) => i.key);
    expect(keys).toEqual(['K900010015', 'K900010076']);
    expect(keys.some((k) => k.startsWith('KGNB'))).toBe(false);
    expect(keys.some((k) => k.startsWith('KGSM'))).toBe(false);
  });

  it('NR / GSM tab 各只返回自己制式指标', () => {
    expect(definitionsForTech(DEFINITIONS, 'nr').map((i) => i.key)).toEqual(['KGNB0511']);
    expect(definitionsForTech(DEFINITIONS, 'gsm').map((i) => i.key)).toEqual(['KGSM0102']);
  });

  it('技术分组缺失 / 未加载 → 返回空数组（失败路径）', () => {
    expect(definitionsForTech(undefined, 'lte')).toEqual([]);
    expect(definitionsForTech([], 'lte')).toEqual([]);
  });
});

describe('toMetricOptions — 选项与置灰', () => {
  it('不可用（available=false）的指标置灰 disabled，可用项不置灰', () => {
    const opts = toMetricOptions(definitionsForTech(DEFINITIONS, 'lte'));
    expect(opts).toEqual([
      { value: 'K900010015', label: '下行流量', unit: 'MByte', disabled: false },
      { value: 'K900010076', label: '小区可用率', unit: '%', disabled: false },
    ]);
  });

  it('中文名缺失时 label 回退 key', () => {
    const opts = toMetricOptions([
      { key: 'X', k_code: '', cn_name: '', unit: '', panel: 'traffic', needs_review: false, available: true },
    ]);
    expect(opts[0].label).toBe('X');
  });
});

describe('working panels 与存盘形状互转', () => {
  it('toWorkingPanels 补 id 且 metrics 深拷贝；toSavePanels 剥 id', () => {
    const src = [panel({ metrics: ['A', 'B'] })];
    const working = toWorkingPanels(src);
    expect(working[0].id).toBeTruthy();
    expect(working[0].metrics).toEqual(['A', 'B']);
    // 深拷贝：改工作态 metrics 不影响源
    working[0].metrics.push('C');
    expect(src[0].metrics).toEqual(['A', 'B']);

    const saved = toSavePanels(working);
    expect(saved[0]).not.toHaveProperty('id');
    expect(saved[0].title).toBe('t');
    expect(saved[0].chartType).toBe('line');
  });
});

describe('panel 增删改（不可变更新）', () => {
  const base: WorkingPanel[] = toWorkingPanels([panel({ title: 'A' }), panel({ title: 'B', x: 6 })]);

  it('addPanel 把新图置首，旧图按视觉顺序依次后移一个展示位', () => {
    const next = addPanel(base, '新图');
    expect(next.length).toBe(3);
    const added = next[0];
    expect(added.title).toBe('新图');
    expect(added.metrics).toEqual([]);
    expect(added.x).toBe(0);
    expect(added.y).toBe(0);
    expect(added.w).toBe(HALF_WIDTH);
    expect(added.h).toBe(DEFAULT_HEIGHT);
    expect(added.chartType).toBe('line');

    // 新图占左上角，原第一张图 A 移到右上，B 再移到下一行左侧。
    expect(next.slice(1).map((p) => p.title)).toEqual(['A', 'B']);
    expect(next[1]).toMatchObject({ x: HALF_WIDTH, y: 0, w: 6, h: 8 });
    expect(next[2]).toMatchObject({ x: 0, y: DEFAULT_HEIGHT, w: 6, h: 8 });
    // 不可变：原数组未变
    expect(base.length).toBe(2);
    expect(base.map((p) => p.y)).toEqual([0, 0]);
  });

  it('addPanel 向空布局新增时直接放在左上角', () => {
    const next = addPanel([], '第一张图');
    expect(next).toHaveLength(1);
    expect(next[0]).toMatchObject({ title: '第一张图', x: 0, y: 0 });
  });

  it('addPanel 连续新增时最后新增的图始终排在第一', () => {
    const once = addPanel(base, '新图 1');
    const twice = addPanel(once, '新图 2');
    expect(twice.map((p) => p.title)).toEqual(['新图 2', '新图 1', 'A', 'B']);
    expect(twice.map((p) => [p.x, p.y])).toEqual([
      [0, 0],
      [HALF_WIDTH, 0],
      [0, DEFAULT_HEIGHT],
      [HALF_WIDTH, DEFAULT_HEIGHT],
    ]);
  });

  it('addPanel 以旧布局的 y/x 视觉顺序为准，不受数组存储顺序影响', () => {
    const existing = toWorkingPanels([
      panel({ title: '可用性', x: HALF_WIDTH, y: 0 }),
      panel({ title: '业务量', x: 0, y: 0 }),
      panel({ title: '利用率', x: 0, y: DEFAULT_HEIGHT }),
    ]);

    const next = addPanel(existing, '新图');

    expect(next.map((p) => p.title)).toEqual(['新图', '业务量', '可用性', '利用率']);
    expect(next.map((p) => [p.x, p.y])).toEqual([
      [0, 0],
      [HALF_WIDTH, 0],
      [0, DEFAULT_HEIGHT],
      [HALF_WIDTH, DEFAULT_HEIGHT],
    ]);
  });

  it('applyGridLayout 忽略缺少当前卡片的过期布局回调', () => {
    const current = addPanel(base, '新图');
    const staleLayout = toGridLayout(base);

    const next = applyGridLayout(current, staleLayout);

    expect(next).toEqual(current);
    expect(next.map((p) => [p.title, p.x, p.y])).toEqual([
      ['新图', 0, 0],
      ['A', HALF_WIDTH, 0],
      ['B', 0, DEFAULT_HEIGHT],
    ]);
  });

  it('applyGridLayout 忽略仍包含已删除卡片的过期布局回调', () => {
    const withNewPanel = addPanel(base, '新图');
    const current = removePanel(withNewPanel, withNewPanel[0].id);
    const staleLayout = toGridLayout(withNewPanel);

    const next = applyGridLayout(current, staleLayout);

    expect(next).toEqual(current);
    expect(next.map((p) => [p.title, p.x, p.y])).toEqual([
      ['A', 0, 0],
      ['B', HALF_WIDTH, 0],
    ]);
  });

  it('removePanel 按 id 删除', () => {
    const next = removePanel(base, base[0].id);
    expect(next.map((p) => p.title)).toEqual(['B']);
    expect(base.length).toBe(2);
  });

  it('removePanel 删除左上图后按剩余数组顺序重新从左上补位', () => {
    const withNewPanel = addPanel(base, '新图');

    const next = removePanel(withNewPanel, withNewPanel[0].id);

    expect(next.map((p) => p.title)).toEqual(['A', 'B']);
    expect(next.map((p) => [p.x, p.y])).toEqual([
      [0, 0],
      [HALF_WIDTH, 0],
    ]);
  });

  it('removePanel 删不存在 id → 原样（边界）', () => {
    expect(removePanel(base, 'nope').length).toBe(2);
  });

  it('updatePanelTitle 只改目标图标题', () => {
    const next = updatePanelTitle(base, base[0].id, 'AA');
    expect(next[0].title).toBe('AA');
    expect(next[1].title).toBe('B');
  });

  it('updatePanelMetrics 改指标并深拷贝', () => {
    const metrics = ['X', 'Y'];
    const next = updatePanelMetrics(base, base[1].id, metrics);
    expect(next[1].metrics).toEqual(['X', 'Y']);
    metrics.push('Z');
    expect(next[1].metrics).toEqual(['X', 'Y']);
  });
});

describe('网格坐标回写', () => {
  const working: WorkingPanel[] = toWorkingPanels([panel({ title: 'A' }), panel({ title: 'B', x: 6 })]);

  it('toGridLayout 由 panels 生成 RGL layout（i=id）', () => {
    const layout = toGridLayout(working);
    expect(layout).toEqual([
      { i: working[0].id, x: 0, y: 0, w: 6, h: 8 },
      { i: working[1].id, x: 6, y: 0, w: 6, h: 8 },
    ]);
  });

  it('applyGridLayout 把拖拽后坐标写回（只动 x/y/w/h，标题指标不变）', () => {
    const moved = applyGridLayout(working, [
      { i: working[0].id, x: 3, y: 5, w: 12, h: 10 },
      { i: working[1].id, x: 6, y: 0, w: 6, h: 8 },
    ]);
    expect(moved[0]).toMatchObject({ x: 6, y: 0, w: 6, h: 8, title: 'B' });
    expect(moved[1]).toMatchObject({ x: 3, y: 5, w: 12, h: 10, title: 'A' });
    expect(moved[1].metrics).toEqual(['M1']);
  });

  it('applyGridLayout 对 layout 里缺失的 id 整次忽略（竞态边界）', () => {
    const result = applyGridLayout(working, [{ i: working[0].id, x: 1, y: 1, w: 6, h: 8 }]);
    expect(result).toEqual(working);
  });

  it('applyGridLayout 按回写后的 y/x 同步数组顺序', () => {
    const moved = applyGridLayout(working, [
      { i: working[0].id, x: 6, y: 0, w: 6, h: 8 },
      { i: working[1].id, x: 0, y: 0, w: 6, h: 8 },
    ]);

    expect(moved.map((p) => p.title)).toEqual(['B', 'A']);
  });
});
