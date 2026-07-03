/**
 * 首页 KPI 布局映射纯逻辑单测（issue #213 S2）
 *
 * 覆盖「布局 → 汇总指标 → 批量取数入参」这段映射，含多张图指标去重；
 * 以及配置读不到 / 为空 / 出错时回退内置默认（首页永不空白）。
 */

import { describe, it, expect } from 'vitest';
import type { KPILayout, KPILayoutPanel } from '@core/types/dashboard';
import {
  buildDefaultLayout,
  resolveLayout,
  collectMetrics,
  layoutToRows,
} from './layoutMapping';

function panel(p: Partial<KPILayoutPanel> & Pick<KPILayoutPanel, 'metrics'>): KPILayoutPanel {
  return {
    title: p.title ?? 'dashboard.panel.traffic',
    metrics: p.metrics,
    x: p.x ?? 0,
    y: p.y ?? 0,
    w: p.w ?? 6,
    h: p.h ?? 8,
    chartType: p.chartType ?? 'line',
  };
}

describe('collectMetrics（汇总指标 → 批量取数入参）', () => {
  it('把多张图的指标汇总成一份去重列表，保持首次出现顺序', () => {
    const panels: KPILayoutPanel[] = [
      panel({ metrics: ['K900010015', 'K900010016'] }),
      panel({ metrics: ['K900010014', 'K900010013'] }),
    ];
    expect(collectMetrics(panels)).toEqual([
      'K900010015',
      'K900010016',
      'K900010014',
      'K900010013',
    ]);
  });

  it('同一指标被多张图引用只取一次（去重）', () => {
    const panels: KPILayoutPanel[] = [
      panel({ metrics: ['K900010002', 'K900010005'] }),
      panel({ metrics: ['K900010005', 'K900010029'] }), // K900010005 重复
    ];
    expect(collectMetrics(panels)).toEqual([
      'K900010002',
      'K900010005',
      'K900010029',
    ]);
  });

  it('空布局汇总出空列表（批量取数应被禁用）', () => {
    expect(collectMetrics([])).toEqual([]);
  });
});

describe('buildDefaultLayout（内置默认/回退）', () => {
  it('LTE 回退布局含 6 张图，指标与 kpi-config 派生一致', () => {
    const layout = buildDefaultLayout('lte');
    expect(layout.tech).toBe('lte');
    expect(layout.panels).toHaveLength(6);
    const titles = layout.panels.map((p) => p.title);
    expect(titles).toEqual([
      'dashboard.panel.traffic',
      'dashboard.panel.availability',
      'dashboard.panel.utilization',
      'dashboard.panel.accessibility',
      'dashboard.panel.retainability',
      'dashboard.panel.mobility',
    ]);
    // traffic 图含四条业务量/速率指标
    expect(layout.panels[0].metrics).toEqual([
      'K900010015',
      'K900010016',
      'K900010040',
      'K900010041',
    ]);
  });

  it('NR 回退布局含 2 张图', () => {
    expect(buildDefaultLayout('nr').panels).toHaveLength(2);
  });

  it('GSM 回退布局含 3 张图，末图满宽占整行', () => {
    const layout = buildDefaultLayout('gsm');
    expect(layout.panels).toHaveLength(3);
    const last = layout.panels[2];
    expect(last.w).toBe(12); // 满宽
    expect(last.x).toBe(0);
  });
});

describe('resolveLayout（有配置用配置，否则回退）', () => {
  it('全局配置非空时直接用配置', () => {
    const remote: KPILayout = {
      tech: 'lte',
      panels: [panel({ title: 'custom', metrics: ['K900010040'] })],
      updatedAt: '2026-06-13T00:00:00Z',
    };
    expect(resolveLayout('lte', remote)).toBe(remote);
  });

  it('配置 undefined（读不到/出错）时回退内置默认', () => {
    const resolved = resolveLayout('lte', undefined);
    expect(resolved.panels).toHaveLength(6);
  });

  it('配置 panels 为空时回退内置默认（不空白）', () => {
    const empty: KPILayout = { tech: 'gsm', panels: [], updatedAt: '' };
    const resolved = resolveLayout('gsm', empty);
    expect(resolved.panels).toHaveLength(3);
  });
});

describe('layoutToRows（按网格坐标排行，首页只读渲染）', () => {
  it('按 y 升序分行、同行按 x 升序', () => {
    const panels: KPILayoutPanel[] = [
      panel({ title: 'b', x: 6, y: 0 }),
      panel({ title: 'a', x: 0, y: 0 }),
      panel({ title: 'c', x: 0, y: 8 }),
    ].map((p) => ({ ...p, metrics: ['X'] }));
    const rows = layoutToRows(panels);
    expect(rows).toHaveLength(2);
    expect(rows[0].panels.map((p) => p.title)).toEqual(['a', 'b']); // 同行 x 升序
    expect(rows[1].panels.map((p) => p.title)).toEqual(['c']);
  });

  it('GSM 默认布局 → 第一行 2 图、第二行 1 图', () => {
    const rows = layoutToRows(buildDefaultLayout('gsm').panels);
    expect(rows.map((r) => r.panels.length)).toEqual([2, 1]);
  });
});
