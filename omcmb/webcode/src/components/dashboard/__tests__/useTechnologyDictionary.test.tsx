/**
 * useTechnologyDictionary 单测。
 *
 * Issue C 决策（最终版）：
 * - 网络制式 Segmented/Tab 从字典 `network_type` 解析，纯字典驱动，不再有 hard-coded fallback；
 * - 显示文案：`labelI18n[locale]` → `label` → `value.toUpperCase()`；
 *   与设备模块字典直接复用（"eNB(LTE)" / "gNB(NR)" 等），两边对齐；
 * - loading / error / 空字典 → 返回 `[]`（UI 显示空 Segmented/Tabs，让运维感知字典缺失）；
 * - 未知 value（不在 TechnologyType 联合）→ 过滤；status=false → 过滤；
 * - 按 sort 升序。
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import type { ReactNode } from 'react';

const useDictionaryMock = vi.fn();

vi.mock('@core/hooks/api/useSystem', () => ({
  useDictionary: (...args: unknown[]) => useDictionaryMock(...args),
}));

import { useTechnologyDictionary } from '../useTechnologyDictionary';

function wrap(locale = 'zh-CN') {
  return ({ children }: { children: ReactNode }) => (
    <IntlProvider locale={locale} defaultLocale="zh-CN" messages={{}}>
      {children}
    </IntlProvider>
  );
}

function ok(details: unknown[]) {
  return {
    data: { sysDictionaryDetails: details },
    isLoading: false,
    isError: false,
  };
}

beforeEach(() => {
  useDictionaryMock.mockReset();
});

describe('useTechnologyDictionary', () => {
  it('loading 时返回空数组（无 fallback）', () => {
    useDictionaryMock.mockReturnValue({ data: null, isLoading: true, isError: false });
    const { result } = renderHook(() => useTechnologyDictionary(), { wrapper: wrap() });
    expect(result.current.isLoading).toBe(true);
    expect(result.current.options).toEqual([]);
  });

  it('字典空列表 → 返回空数组', () => {
    useDictionaryMock.mockReturnValue(ok([]));
    const { result } = renderHook(() => useTechnologyDictionary(), { wrapper: wrap() });
    expect(result.current.options).toEqual([]);
  });

  it('使用字典原 label（与设备模块对齐，例如 eNB(LTE) / gNB(NR)）', () => {
    useDictionaryMock.mockReturnValue(
      ok([
        { id: 1, value: 'lte', label: 'eNB(LTE)', sort: 1, status: true },
        { id: 2, value: 'nr', label: 'gNB(NR)', sort: 2, status: true },
      ]),
    );
    const { result } = renderHook(() => useTechnologyDictionary(), { wrapper: wrap() });
    expect(result.current.options).toEqual([
      { value: 'lte', label: 'eNB(LTE)', sort: 1 },
      { value: 'nr', label: 'gNB(NR)', sort: 2 },
    ]);
  });

  it('启用的 gsm 字典项显示为 GSM 页签', () => {
    useDictionaryMock.mockReturnValue(ok([
      { id: 1, value: 'lte', label: 'eNB(LTE)', sort: 1, status: true },
      { id: 2, value: 'nr', label: 'gNB(NR)', sort: 2, status: true },
      { id: 3, value: 'gsm', label: 'GSM', sort: 3, status: true },
    ]));
    const { result } = renderHook(() => useTechnologyDictionary(), { wrapper: wrap() });
    expect(result.current.options).toEqual([
      { value: 'lte', label: 'eNB(LTE)', sort: 1 },
      { value: 'nr', label: 'gNB(NR)', sort: 2 },
      { value: 'gsm', label: 'GSM', sort: 3 },
    ]);
  });

  it('labelI18n[locale] 优先于 label', () => {
    useDictionaryMock.mockReturnValue(
      ok([
        { id: 1, value: 'lte', label: 'eNB(LTE)', labelI18n: { 'zh-CN': '长期演进' }, sort: 1, status: true },
        { id: 2, value: 'nr', label: 'gNB(NR)', sort: 2, status: true },
      ]),
    );
    const { result } = renderHook(() => useTechnologyDictionary(), { wrapper: wrap('zh-CN') });
    expect(result.current.options.map((o) => o.label)).toEqual(['长期演进', 'gNB(NR)']);
  });

  it('labelI18n / label 全为空 → 兜底到 value 大写', () => {
    useDictionaryMock.mockReturnValue(
      ok([{ id: 1, value: 'lte', label: '', sort: 1, status: true }]),
    );
    const { result } = renderHook(() => useTechnologyDictionary(), { wrapper: wrap() });
    expect(result.current.options[0].label).toBe('LTE');
  });

  it('未知 value（联合外）被过滤', () => {
    useDictionaryMock.mockReturnValue(
      ok([
        { id: 1, value: 'lte', label: 'eNB(LTE)', sort: 1, status: true },
        { id: 9, value: 'cdma', label: 'CDMA', sort: 9, status: true },
      ]),
    );
    const { result } = renderHook(() => useTechnologyDictionary(), { wrapper: wrap() });
    expect(result.current.options.map((o) => o.value)).toEqual(['lte']);
  });

  it('status=false 被过滤；按 sort 升序', () => {
    useDictionaryMock.mockReturnValue(
      ok([
        { id: 2, value: 'nr', label: 'gNB(NR)', sort: 2, status: true },
        { id: 3, value: 'gsm', label: 'GSM', sort: 3, status: false },
        { id: 1, value: 'lte', label: 'eNB(LTE)', sort: 1, status: true },
      ]),
    );
    const { result } = renderHook(() => useTechnologyDictionary(), { wrapper: wrap() });
    expect(result.current.options.map((o) => o.value)).toEqual(['lte', 'nr']);
  });
});
