/**
 * PM-DASH-DIMFILTER pmAdhocApi 维度子集过滤 + filter-options 端点契约测试。
 *
 * 承重点：results() 的 query 是手动构造 snake_case key（不靠 Axios 自动 camelCase→snake_case）。
 * 两个维度参数传法不同（issue #401 修复后定型）：
 *   - product_ids：纯 UUID 不含逗号 → CSV join（product_ids='p1,p2'），后端 parseCSVQuery 逗号切分。
 *   - object_ldns：设备组值合法含逗号（'DeviceGroup=<uuid>,Tech=lte'）→ 不能 CSV，发原始数组 +
 *     paramsSerializer 序列化成重复键 object_ldns=a&object_ldns=b（无方括号），后端 parseRepeatedQuery
 *     用 gin QueryArray 逐值取回、整值不拆。故验 objectLdns→params.object_ldns 是原数组、且 serializer
 *     产出无方括号重复键（含逗号的值整体 encode 不被切碎）。
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';

// 用 vi.hoisted 让 mock 工厂能安全引用 getMock/deleteMock（vi.mock 被提升到文件顶部）。
const { getMock, deleteMock } = vi.hoisted(() => ({ getMock: vi.fn(), deleteMock: vi.fn() }));
vi.mock('../../http', () => ({
  default: { get: getMock, post: vi.fn(), patch: vi.fn(), delete: deleteMock },
}));

import { pmAdhocApi } from '../pmAdhocApi';

beforeEach(() => {
  getMock.mockReset();
  getMock.mockResolvedValue({ data: { items: [], total: 0 } });
  deleteMock.mockReset();
  deleteMock.mockResolvedValue({ data: { deleted: true } });
});

describe('pmAdhocApi.results — 维度子集过滤 query（手动 snake_case，CSV 形态）', () => {
  it('productIds → query product_ids（CSV join，后端 QueryArray 可解析）', async () => {
    await pmAdhocApi.results('t1', 100, 0, undefined, undefined, ['p1', 'p2']);
    const [, opts] = getMock.mock.calls[0];
    expect(opts.params.product_ids).toBe('p1,p2');
    expect('object_ldns' in opts.params).toBe(false);
  });

  it('objectLdns → query object_ldns 发原始数组 + 重复键序列化（频段多选不被逗号切碎）', async () => {
    await pmAdhocApi.results('t1', 100, 0, undefined, undefined, undefined, ['Band=42', 'Band=1']);
    const [, opts] = getMock.mock.calls[0];
    // 透传原始数组（非 CSV join）：后端 QueryArray 逐值取回。
    expect(opts.params.object_ldns).toEqual(['Band=42', 'Band=1']);
    expect('product_ids' in opts.params).toBe(false);
    // paramsSerializer 把数组序列化成无方括号重复键，每个值整体 encode。
    expect(typeof opts.paramsSerializer).toBe('function');
    expect(opts.paramsSerializer(opts.params)).toBe(
      'limit=100&offset=0&object_ldns=Band%3D42&object_ldns=Band%3D1',
    );
  });

  it('objectLdns 设备组含逗号值 → 重复键序列化整值不拆（issue #401 根因回归）', async () => {
    const groupVal = 'DeviceGroup=11111111-2222-3333-4444-555555555555,Tech=lte';
    await pmAdhocApi.results('t1', 100, 0, undefined, undefined, undefined, [groupVal]);
    const [, opts] = getMock.mock.calls[0];
    expect(opts.params.object_ldns).toEqual([groupVal]);
    // 含逗号的完整设备组值整体作为单个 object_ldns 值（逗号被 encode，不切成两段）。
    expect(opts.paramsSerializer(opts.params)).toBe(
      `limit=100&offset=0&object_ldns=${encodeURIComponent(groupVal)}`,
    );
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

describe('pmAdhocApi.deleteTask / cancel — 删除 vs 取消打到不同端点（issue #392）', () => {
  it('deleteTask 打到硬删子路径 /pm/adhoc/tasks/:id/definition（成功路径）', async () => {
    await pmAdhocApi.deleteTask('task-终态-自建');
    expect(deleteMock).toHaveBeenCalledTimes(1);
    const [url] = deleteMock.mock.calls[0];
    expect(url).toBe('/pm/adhoc/tasks/task-终态-自建/definition');
  });

  it('cancel 打到软删/取消端点 /pm/adhoc/tasks/:id（无 /definition 后缀，二者不混淆）', async () => {
    await pmAdhocApi.cancel('task-运行中');
    const [url] = deleteMock.mock.calls[0];
    expect(url).toBe('/pm/adhoc/tasks/task-运行中');
    expect(url).not.toContain('/definition');
  });

  it('deleteTask 失败路径（后端 409/403）向上抛错，不静默吞', async () => {
    const err = new Error('task not in terminal state');
    deleteMock.mockRejectedValueOnce(err);
    await expect(pmAdhocApi.deleteTask('task-非终态')).rejects.toThrow(
      'task not in terminal state',
    );
  });
});
