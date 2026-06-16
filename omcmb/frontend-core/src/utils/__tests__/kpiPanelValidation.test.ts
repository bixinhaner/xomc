import { describe, it, expect } from 'vitest';

import {
  validateKpiPanels,
  formatKpiPanelViolations,
  type KpiPanelForValidation,
} from '../kpiPanelValidation';

describe('validateKpiPanels', () => {
  it('全部合规（有标题 + 至少 1 指标）通过', () => {
    const panels: KpiPanelForValidation[] = [
      { title: '流量', metrics: ['k1'] },
      { title: '可用性', metrics: ['k2', 'k3'] },
    ];
    const result = validateKpiPanels(panels);
    expect(result.valid).toBe(true);
    expect(result.violations).toHaveLength(0);
  });

  it('空图列表视为合规（无图可违规）', () => {
    const result = validateKpiPanels([]);
    expect(result.valid).toBe(true);
    expect(result.violations).toHaveLength(0);
  });

  it('空标题拦截（含纯空白标题）', () => {
    const result = validateKpiPanels([{ title: '   ', metrics: ['k1'] }]);
    expect(result.valid).toBe(false);
    expect(result.violations).toHaveLength(1);
    expect(result.violations[0]).toMatchObject({
      index: 0,
      titleEmpty: true,
      noMetrics: false,
    });
  });

  it('0 指标拦截', () => {
    const result = validateKpiPanels([{ title: '流量', metrics: [] }]);
    expect(result.valid).toBe(false);
    expect(result.violations).toHaveLength(1);
    expect(result.violations[0]).toMatchObject({
      index: 0,
      titleEmpty: false,
      noMetrics: true,
    });
  });

  it('空标题 + 0 指标：两条原因都标记', () => {
    const result = validateKpiPanels([{ title: '', metrics: [] }]);
    expect(result.valid).toBe(false);
    expect(result.violations[0]).toMatchObject({
      index: 0,
      titleEmpty: true,
      noMetrics: true,
    });
  });

  it('多图含一张不合规：拦截并指出是哪张（序号正确）', () => {
    const panels: KpiPanelForValidation[] = [
      { title: '流量', metrics: ['k1'] },
      { title: '', metrics: [] },
      { title: '可用性', metrics: ['k2'] },
    ];
    const result = validateKpiPanels(panels);
    expect(result.valid).toBe(false);
    expect(result.violations).toHaveLength(1);
    expect(result.violations[0].index).toBe(1);
  });

  it('多图多张不合规：全部列出', () => {
    const panels: KpiPanelForValidation[] = [
      { title: '', metrics: ['k1'] },
      { title: '可用性', metrics: ['k2'] },
      { title: '利用率', metrics: [] },
    ];
    const result = validateKpiPanels(panels);
    expect(result.valid).toBe(false);
    expect(result.violations.map((v) => v.index)).toEqual([0, 2]);
  });
});

describe('formatKpiPanelViolations', () => {
  it('无违规返回空串', () => {
    expect(formatKpiPanelViolations([])).toBe('');
  });

  it('空标题违规提示含序号与原因', () => {
    const msg = formatKpiPanelViolations([
      { index: 0, title: '', titleEmpty: true, noMetrics: false },
    ]);
    expect(msg).toContain('第 1 张图');
    expect(msg).toContain('标题为空');
  });

  it('0 指标违规提示含标题与原因', () => {
    const msg = formatKpiPanelViolations([
      { index: 2, title: '利用率', titleEmpty: false, noMetrics: true },
    ]);
    expect(msg).toContain('第 3 张图');
    expect(msg).toContain('「利用率」');
    expect(msg).toContain('未选指标');
  });

  it('多张违规以分号连接', () => {
    const msg = formatKpiPanelViolations([
      { index: 0, title: '', titleEmpty: true, noMetrics: true },
      { index: 1, title: '可用性', titleEmpty: false, noMetrics: true },
    ]);
    expect(msg).toContain('；');
    expect(msg).toContain('第 1 张图');
    expect(msg).toContain('第 2 张图');
  });
});
