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
      { key: 'LTE_PDCP_VOLUME_DL', k_code: 'K1', cn_name: '下行流量', unit: 'MByte', panel: 'traffic', needs_review: false, available: true },
      { key: 'LTE_CELL_AVAILABLE', k_code: '', cn_name: '小区可用率', unit: '%', panel: 'availability', needs_review: true, available: false },
    ],
  },
  {
    tech: 'nr',
    items: [
      { key: 'NR_PDCP_VOLUME_DL', k_code: 'KGNB1', cn_name: 'NR下行流量', unit: 'MByte', panel: 'traffic', needs_review: false, available: true },
    ],
  },
  {
    tech: 'gsm',
    items: [
      { key: 'GSM_CALL_SETUP_SR', k_code: 'KGSM1', cn_name: '电话成功率', unit: '%', panel: 'accessibility', needs_review: false, available: true },
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
    expect(keys).toEqual(['LTE_PDCP_VOLUME_DL', 'LTE_CELL_AVAILABLE']);
    expect(keys.some((k) => k.startsWith('NR_'))).toBe(false);
    expect(keys.some((k) => k.startsWith('GSM_'))).toBe(false);
  });

  it('NR / GSM tab 各只返回自己制式指标', () => {
    expect(definitionsForTech(DEFINITIONS, 'nr').map((i) => i.key)).toEqual(['NR_PDCP_VOLUME_DL']);
    expect(definitionsForTech(DEFINITIONS, 'gsm').map((i) => i.key)).toEqual(['GSM_CALL_SETUP_SR']);
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
      { value: 'LTE_PDCP_VOLUME_DL', label: '下行流量', unit: 'MByte', disabled: false },
      { value: 'LTE_CELL_AVAILABLE', label: '小区可用率', unit: '%', disabled: true },
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

  it('addPanel 追加空图：半宽、无指标、line、落到现有图下方', () => {
    const next = addPanel(base, '新图');
    expect(next.length).toBe(3);
    const added = next[2];
    expect(added.title).toBe('新图');
    expect(added.metrics).toEqual([]);
    expect(added.w).toBe(HALF_WIDTH);
    expect(added.h).toBe(DEFAULT_HEIGHT);
    expect(added.chartType).toBe('line');
    // 落到最大底边（base 两图 y=0 h=8 → bottom 8）
    expect(added.y).toBe(8);
    // 不可变：原数组未变
    expect(base.length).toBe(2);
  });

  it('removePanel 按 id 删除', () => {
    const next = removePanel(base, base[0].id);
    expect(next.map((p) => p.title)).toEqual(['B']);
    expect(base.length).toBe(2);
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
    expect(moved[0]).toMatchObject({ x: 3, y: 5, w: 12, h: 10, title: 'A' });
    expect(moved[0].metrics).toEqual(['M1']);
    expect(moved[1]).toMatchObject({ x: 6, y: 0, w: 6, h: 8 });
  });

  it('applyGridLayout 对 layout 里缺失的 id 保持原样（竞态边界）', () => {
    const result = applyGridLayout(working, [{ i: working[0].id, x: 1, y: 1, w: 6, h: 8 }]);
    expect(result[1]).toMatchObject({ x: 6, y: 0 }); // B 未在 layout 中 → 不变
  });
});
