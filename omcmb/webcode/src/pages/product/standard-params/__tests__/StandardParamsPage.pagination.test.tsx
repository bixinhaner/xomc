/**
 * StandardParamsPage 分页回归测试（issue #190）。
 *
 * 标准参数树是「前端一次性全量加载 + 本地分页」。缺陷：在非第 1 页时改每页条数，
 * 页码不归 1、切片不刷新。
 *
 * 测两层：
 *  1) 纯函数 nextPageOnPaginationChange —— 死判分页页码决策（成功 + 失败两路）。
 *  2) 组件 DOM —— 默认 20/页切片、翻页可见行数，确保 helper 真接到 antd onChange。
 *
 * 注：antd 「每页条数」下拉在 jsdom 里交互不稳定，真实 DOM 行数随 pageSize 变化由
 *     运行栈 agent 用 playwright 在浏览器中验（自判项 pagesize-works）。
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import type { StandardParam } from '@core/types/paramModel';
import { PRODUCT_TABLE_DEFAULT_PAGE_SIZE } from '../../pagination';
import { nextPageOnPaginationChange } from '../pagination';

describe('nextPageOnPaginationChange（issue #190 死判）', () => {
  it('改每页条数 → 回第 1 页', () => {
    // 在第 5 页、原 50/页，改成 100/页：应归 1。
    const r = nextPageOnPaginationChange(5, 100, 50);
    expect(r).toEqual({ page: 1, pageSize: 100, sizeChanged: true });
  });

  it('仅翻页（每页条数不变）→ 按目标页码走', () => {
    const r = nextPageOnPaginationChange(3, 50, 50);
    expect(r).toEqual({ page: 3, pageSize: 50, sizeChanged: false });
  });

  it('每页条数变小同样回第 1 页', () => {
    const r = nextPageOnPaginationChange(4, 10, 50);
    expect(r).toEqual({ page: 1, pageSize: 10, sizeChanged: true });
  });
});

// ---- 组件层 DOM 测 ----

// i18n：直接回 key，避免依赖语料。
vi.mock('@/hooks/useT', () => ({
  useT:
    () =>
    (id: string, values?: Record<string, unknown>) =>
      values ? `${id}|${JSON.stringify(values)}` : id,
}));

// 造 120 条标准参数，默认 20/页 → 6 页。
const ITEMS: StandardParam[] = Array.from({ length: 120 }, (_, i) => ({
  standardPath: `Device.Std.Param${String(i).padStart(3, '0')}`,
  entryType: 'parameter',
  access: 'readWrite',
  dataType: 'string',
  changeApplies: 'reload',
  minValue: '',
  maxValue: '',
  updatedAt: '2026-07-15T00:00:00Z',
  updatedFields: [],
})) as StandardParam[];

vi.mock('@core/hooks/api/useParamModels', () => ({
  useStandardParams: () => ({ data: { items: ITEMS }, isLoading: false }),
  useUpsertStandard: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteStandard: () => ({ mutateAsync: vi.fn() }),
}));

import StandardParamsPage from '../index';

function countBodyRows(): number {
  return document.querySelectorAll('.ant-table-tbody tr.ant-table-row').length;
}

describe('StandardParamsPage 本地分页 DOM（issue #190）', () => {
  it('默认每页 20 条，120 条数据首页显示 20 行', () => {
    render(<StandardParamsPage />);
    expect(countBodyRows()).toBe(PRODUCT_TABLE_DEFAULT_PAGE_SIZE);
    expect(screen.getByText('Device.Std.Param000')).toBeInTheDocument();
  });

  it('翻到第 2 页：仍 20 行，内容切到第二页区间', () => {
    render(<StandardParamsPage />);
    const pager = screen.getByRole('list');
    fireEvent.click(within(pager).getByText('2'));
    expect(countBodyRows()).toBe(PRODUCT_TABLE_DEFAULT_PAGE_SIZE);
    // 第 2 页第一条是第 21 条（index 20）。
    expect(screen.getByText('Device.Std.Param020')).toBeInTheDocument();
  });
});
