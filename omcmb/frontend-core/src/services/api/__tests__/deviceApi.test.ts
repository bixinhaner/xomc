/**
 * deviceApi 契约测试（#22 关键 API）：
 *   - getList：前端 filter → 后端 query 映射（CSV 多选、is_online 字符串/布尔双源、
 *     legacy networkType eNB/gNB → lte/nr），BackendDevice → Device 映射 + stats 优先后端。
 *   - getById：成功映射；错误（404/500）吞错返 null（详情页不崩，符合现行 catch 兜底）。
 *   - getStats / getProductClasses：端点透传。
 *   - mapListResponse 在 stats 缺省时用当前页 items 估算（向下兼容兜底）。
 *
 * 用 vi.mock 替换 http 客户端（参照 alarmApi.test.ts 既有模式）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}));
vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

import { deviceApi } from '../deviceApi';

function backendDevice(overrides: Record<string, unknown> = {}) {
  return {
    id: 'd1',
    serial_number: 'SN001',
    oui: 'AABBCC',
    product_class: 'PC100',
    manufacturer: 'Acme',
    model_name: 'M1',
    carrier: 'cmcc',
    technology: 'nr',
    status: 'active',
    is_online: true,
    firmware_version: 'v1.0',
    ip_address: '10.0.0.1',
    connection_request_url: '',
    inform_interval: 300,
    device_name: '基站A',
    site_id: 'S1',
    latitude: 30,
    longitude: 120,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-02T00:00:00Z',
    ...overrides,
  };
}

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  getMock.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, total_pages: 0 } });
});

describe('deviceApi.getList — filter → query 映射', () => {
  it('多选字段序列化为 CSV（避免 axios key[] 形态被 gin 静默丢弃）', async () => {
    await deviceApi.getList({
      page: 1,
      pageSize: 20,
      snList: ['SN1', 'SN2'],
      lifecycleState: ['commissioned', 'maintenance'],
      modelName: 'M1',
    });
    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/devices');
    expect(opts.params.sn_list).toBe('SN1,SN2');
    expect(opts.params.lifecycle_state).toBe('commissioned,maintenance');
    expect(opts.params.model_name).toBe('M1');
  });

  it('legacy networkType eNB/gNB → technology lte/nr 翻译', async () => {
    await deviceApi.getList({ page: 1, pageSize: 20, networkType: 'gNB' });
    expect(getMock.mock.calls[0][1].params.technology).toBe('nr');
  });

  it('isOnline 字符串 "true"（来自字典/表单）也映射为 is_online=true', async () => {
    await deviceApi.getList({ page: 1, pageSize: 20, isOnline: 'true' as unknown as boolean });
    expect(getMock.mock.calls[0][1].params.is_online).toBe('true');
  });

  it('isOnline 布尔 false（程序化调用）映射为 is_online=false', async () => {
    await deviceApi.getList({ page: 1, pageSize: 20, isOnline: false });
    expect(getMock.mock.calls[0][1].params.is_online).toBe('false');
  });

  it('BackendDevice → Device 映射 + 后端 stats 优先', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice()],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
        stats: { total: 1, online_count: 1, offline_count: 0, alarmed: 0 },
      },
    });
    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items).toHaveLength(1);
    const d = out.items[0];
    expect(d.sn).toBe('SN001');
    expect(d.name).toBe('基站A');
    expect(d.networkType).toBe('gNB'); // nr → gNB
    expect(d.connStatus).toBe('online'); // is_online → online
    expect(d.isOnline).toBe(true);
    // 后端 stats 直读（不靠 items.filter 估算）
    expect(out.stats.online_count).toBe(1);
  });

  it('映射后的 Device 不含 platformType（#177：该字段及关联逻辑已移除）', async () => {
    getMock.mockResolvedValue({
      data: {
        // 即便后端仍回 platform_type，映射也不再透传到前端 Device
        items: [backendDevice({ platform_type: 'CPE_436Q' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });
    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    const d = out.items[0];
    expect('platformType' in d).toBe(false);
    // 其它正常字段仍在，证明不是整体映射坏掉
    expect(d.sn).toBe('SN001');
    expect(d.productClass).toBe('PC100');
  });

  it('transmit_power=-1 视为占位值，不映射成列表 Tx Power', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice({ transmit_power: -1, tx_power: '' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items[0].txPower).toBe('');
  });

  it('stats 缺省时用当前页 items 估算（向下兼容兜底）', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice({ is_online: true }), backendDevice({ id: 'd2', is_online: false })],
        total: 2,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });
    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.stats.online_count).toBe(1);
    expect(out.stats.offline_count).toBe(1);
  });

  it('items 缺失（null）不崩，返空列表', async () => {
    getMock.mockResolvedValue({ data: { total: 0, page: 1, page_size: 20, total_pages: 0 } });
    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items).toEqual([]);
  });
});

describe('deviceApi.getById', () => {
  it('成功：映射设备名空时回退 SN（friendlyName）', async () => {
    getMock.mockResolvedValue({ data: backendDevice({ device_name: '' }) });
    const d = await deviceApi.getById('d1');
    expect(d?.name).toBe('SN001'); // device_name 空 → SN
  });

  it('lifecycle_state 缺省时从 legacy status 派生（discovered 直传，active→commissioned）', async () => {
    getMock.mockResolvedValue({ data: backendDevice({ lifecycle_state: undefined, status: 'discovered', is_online: undefined }) });
    const d = await deviceApi.getById('d1');
    expect(d?.lifecycleState).toBe('discovered');
  });

  it('404/500 错误吞错返 null（详情页不崩，现行 catch 兜底）', async () => {
    getMock.mockRejectedValue({ response: { status: 404 } });
    const d = await deviceApi.getById('missing');
    expect(d).toBeNull();
  });
});

describe('deviceApi.getStats / getProductClasses — 端点透传', () => {
  it('getStats 打 /devices/stats', async () => {
    getMock.mockResolvedValue({ data: { total: 5, online: 3, offline: 2, alarm: 0 } });
    const s = await deviceApi.getStats();
    expect(getMock.mock.calls[0][0]).toBe('/devices/stats');
    expect(s.total).toBe(5);
  });

  it('getProductClasses 打 /devices/product-classes', async () => {
    getMock.mockResolvedValue({ data: ['PC100', 'PC200'] });
    const list = await deviceApi.getProductClasses();
    expect(getMock.mock.calls[0][0]).toBe('/devices/product-classes');
    expect(list).toEqual(['PC100', 'PC200']);
  });
});
