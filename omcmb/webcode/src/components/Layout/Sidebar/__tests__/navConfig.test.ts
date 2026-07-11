import { describe, expect, it } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import enUS from '@core/i18n/en-US';
import { NAV_CONFIG } from '../navConfig';

describe('NAV_CONFIG MML menu', () => {
  it('keeps the v1 visible MML order and script management label', () => {
    const mmlGroup = NAV_CONFIG.find((group) => group.key === 'mml');

    expect(mmlGroup?.children.map((child) => child.path)).toEqual([
      '/mml/console',
      '/mml/script',
      '/mml/task-records',
    ]);
    expect(mmlGroup?.children.find((child) => child.path === '/mml/script')?.label)
      .toBe('nav.mml.script');
    expect(zhCN['nav.mml.console']).toBe('MML控制台');
    expect(zhCN['nav.mml.script']).toBe('脚本管理');
    expect(enUS['nav.mml.console']).toBe('MML Console');
    expect(enUS['nav.mml.script']).toBe('Script Management');
  });
});
