/**
 * #177：UE 详情统一为单载波列展示，移除 platformType 驱动的 436Q 双载波
 * P/S 列变体。断言列头为单载波固定列、且不出现 436Q / P-S 双份指标列。
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { IntlProvider } from 'react-intl';
import { zhCN } from '@core/i18n';
import UeDetail from './index';

function renderUeDetail(path = '/device/ue-detail/SN001') {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <MemoryRouter initialEntries={[path]}>
        <UeDetail />
      </MemoryRouter>
    </IntlProvider>,
  );
}

describe('UeDetail 单载波列展示 (#177)', () => {
  it('渲染单载波固定列', () => {
    renderUeDetail();
    const headers = screen.getAllByRole('columnheader').map((h) => (h.textContent ?? '').trim());
    expect(headers).toContain('UEID');
    expect(headers).toContain('下行CQI');
    expect(headers).toContain('MME_S1AP_ID');
  });

  it('不出现 436Q 双载波 P/S 变体列（下行CQI 仅一列）', () => {
    renderUeDetail();
    const headers = screen.getAllByRole('columnheader').map((h) => (h.textContent ?? '').trim());
    expect(headers.filter((h) => h.includes('下行CQI'))).toHaveLength(1);
    expect(headers.some((h) => /436Q/i.test(h))).toBe(false);
  });
});
