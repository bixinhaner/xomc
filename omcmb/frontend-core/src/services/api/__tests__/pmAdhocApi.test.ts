/**
 * PM-DASH-DIMFILTER pmAdhocApi 维度子集过滤 + filter-options 端点契约测试。
 *
 * 承重点：results() 的 query 是手动构造 snake_case key（不靠 Axios 自动 camelCase→snake_case），
 * 数组发 CSV（join(',')）而非原始数组——http.ts 无 paramsSerializer，Axios 默认把数组序列化成
 * product_ids[]=a&product_ids[]=b 方括号形态，后端 gin QueryArray("product_ids") 收不到；CSV
 * product_ids=a,b 后端 parseCSVQuery 才能解析。故验 productIds→product_ids='p1,p2' 字符串。
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';

// 用 vi.hoisted 让 mock 工厂能安全引用 getMock（vi.mock 被提升到文件顶部）。
const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }));
vi.mock('../../http', () => ({
  default: { get: getMock, post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

import { pmAdhocApi } from '../pmAdhocApi';

beforeEach(() => {
  getMock.mockReset();
  getMock.mockResolvedValue({ data: { items: [], total: 0 } });
});

describe('pmAdhocApi.results — 维度子集过滤 query（手动 snake_case，CSV 形态）', () => {
  it('productIds → query product_ids（CSV join，后端 QueryArray 可解析）', async () => {
    await pmAdhocApi.results('t1', 100, 0, undefined, undefined, ['p1', 'p2']);
    const [, opts] = getMock.mock.calls[0];
    expect(opts.params.product_ids).toBe('p1,p2');
    expect('object_ldns' in opts.params).toBe(false);
  });

  it('objectLdns → query object_ldns（设备组/频段维度共用，CSV join）', async () => {
    await pmAdhocApi.results('t1', 100, 0, undefined, undefined, undefined, ['Band=42', 'Band=1']);
    const [, opts] = getMock.mock.calls[0];
    expect(opts.params.object_ldns).toBe('Band=42,Band=1');
    expect('product_ids' in opts.params).toBe(false);
  });

  it('两者皆空/未传 → 不带这两个 query（无回归，现行行为）', async () => {
    await pmAdhocApi.results('t1');
    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/pm/adhoc/tasks/t1/results');
    expect('product_ids' in opts.params).toBe(false);
    expect('object_ldns' in opts.params).toBe(false);
  });

  it('空数组视为不过滤（不带 query）', async () => {
    await pmAdhocApi.results('t1', 100, 0, undefined, undefined, [], []);
    const [, opts] = getMock.mock.calls[0];
    expect('product_ids' in opts.params).toBe(false);
    expect('object_ldns' in opts.params).toBe(false);
  });
});

describe('pmAdhocApi.filterOptions — 端点 + 响应透传', () => {
  it('打到 /pm/adhoc/tasks/:id/filter-options 并回 {dimension,options}', async () => {
    getMock.mockResolvedValue({
      data: {
        dimension: 'band',
        options: [{ value: 'Band=42', label: 'Band=42' }],
      },
    });
    const out = await pmAdhocApi.filterOptions('t9');
    const [url] = getMock.mock.calls[0];
    expect(url).toBe('/pm/adhoc/tasks/t9/filter-options');
    expect(out.dimension).toBe('band');
    expect(out.options).toEqual([{ value: 'Band=42', label: 'Band=42' }]);
  });

  it('options 缺失（null）兜底空数组，不崩', async () => {
    getMock.mockResolvedValue({ data: { dimension: 'product', options: null } });
    const out = await pmAdhocApi.filterOptions('t9');
    expect(out.options).toEqual([]);
  });
});
