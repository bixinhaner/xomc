/**
 * alarmApi 失败路径 + 畸形载荷契约测试（#22 关键 API — alarm 域）：
 *   - 列表/统计接口错误码 401/429/500 原样抛（不吞错，hook 走 React Query 错误态）。
 *   - getById 仅把 404 转为 null；403/500 必须保留权限和故障语义。
 *   - 畸形载荷不崩：severity 越界 → 'warning' 兜底；items 缺失 → 空列表；
 *     by_severity 缺键 → 0；空可能原因不跨字段回退；is_read 缺省 → unread='1'。
 *
 * 与既有 alarmApi.test.ts（severity CSV 序列化）互补，不重叠。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}));
vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, put: vi.fn(), delete: vi.fn() },
}));

import { alarmApi } from '../alarmApi';

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
});

describe('alarmApi — 错误码冒泡（不吞错）', () => {
  it('getCurrentAlarms 401 原样抛', async () => {
    getMock.mockRejectedValue({ response: { status: 401 } });
    await expect(
      alarmApi.getCurrentAlarms({ page: 1, pageSize: 20 }),
    ).rejects.toEqual({ response: { status: 401 } });
  });

  it('getCurrentAlarms 429 原样抛', async () => {
    getMock.mockRejectedValue({ response: { status: 429 } });
    await expect(
      alarmApi.getCurrentAlarms({ page: 1, pageSize: 20 }),
    ).rejects.toEqual({ response: { status: 429 } });
  });

  it('getAlarmCount 500 原样抛', async () => {
    getMock.mockRejectedValue({ response: { status: 500 } });
    await expect(alarmApi.getAlarmCount()).rejects.toEqual({ response: { status: 500 } });
  });

  it('acknowledgeAlarms 500 原样抛（写操作失败必须暴露给用户）', async () => {
    postMock.mockRejectedValue({ response: { status: 500 } });
    await expect(alarmApi.acknowledgeAlarms(['a1'])).rejects.toEqual({
      response: { status: 500 },
    });
  });
});

describe('alarmApi.getById — 精准详情错误语义', () => {
  it('404 → null（详情抽屉不崩）', async () => {
    getMock.mockRejectedValue({ response: { status: 404 } });
    const out = await alarmApi.getById('missing');
    expect(out).toBeNull();
  });

  it('403 原样抛，页面不得退化成未过滤列表', async () => {
    getMock.mockRejectedValue({ response: { status: 403 } });
    await expect(alarmApi.getById('forbidden')).rejects.toEqual({ response: { status: 403 } });
  });

  it('500 原样抛，由页面进入标准错误态', async () => {
    getMock.mockRejectedValue({ response: { status: 500 } });
    await expect(alarmApi.getById('x')).rejects.toEqual({ response: { status: 500 } });
  });
});

describe('alarmApi — 畸形载荷兜底（type drift / field missing）', () => {
  function backendAlarm(overrides: Record<string, unknown> = {}) {
    return {
      id: 'al1',
      device_id: 'dev1',
      device_sn: 'SN001',
      carrier: 'cmcc',
      severity: 1,
      alarm_type: 'communicationsAlarm',
      alarm_identifier: 'LINK_DOWN',
      description: '链路中断',
      status: 'active',
      raised_at: '2026-06-10T00:00:00Z',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z',
      ...overrides,
    };
  }

  it('severity 越界（如 99）→ 兜底 warning（不渲染 undefined 级别）', async () => {
    getMock.mockResolvedValue({
      data: { items: [backendAlarm({ severity: 99 })], total: 1, page: 1, page_size: 20, total_pages: 1 },
    });
    const out = await alarmApi.getCurrentAlarms({ page: 1, pageSize: 20 });
    expect(out.items[0].severity).toBe('warning');
  });

  it('legacy severity 码 31001..31004 映射到正确级别（不再全部兜底 warning）', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          backendAlarm({ id: 'al-critical', severity: 31001 }),
          backendAlarm({ id: 'al-major', severity: 31002 }),
          backendAlarm({ id: 'al-minor', severity: 31003 }),
          backendAlarm({ id: 'al-warning', severity: 31004 }),
        ],
        total: 4,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const out = await alarmApi.getCurrentAlarms({ page: 1, pageSize: 20 });
    expect(out.items.map((item) => item.severity)).toEqual([
      'critical',
      'major',
      'minor',
      'warning',
    ]);
  });

  it('items 缺失（null）→ 空列表（不崩）', async () => {
    getMock.mockResolvedValue({ data: { total: 0, page: 1, page_size: 20, total_pages: 0 } });
    const out = await alarmApi.getCurrentAlarms({ page: 1, pageSize: 20 });
    expect(out.items).toEqual([]);
    expect(out.total).toBe(0);
  });

  it('probable_cause 缺失时可能原因不回退到名称或 identifier', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendAlarm({ probable_cause: '', description: '', alarm_identifier: 'ONLY_ID' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });
    const out = await alarmApi.getCurrentAlarms({ page: 1, pageSize: 20 });
    expect(out.items[0].alarmName).toBe('');
  });

  it('is_read 缺省 → unread="1"（未读）；event_type 未知 → communication 兜底', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendAlarm({ event_type: 'BOGUS_TYPE', alarm_type: 'BOGUS_TYPE' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });
    const out = await alarmApi.getCurrentAlarms({ page: 1, pageSize: 20 });
    expect(out.items[0].unread).toBe('1');
    expect(out.items[0].eventType).toBe('communication');
  });

  it('getAlarmCount：by_severity 缺键各级兜 0，total_active 缺省 0', async () => {
    getMock.mockResolvedValue({
      data: { total_active: 7, by_severity: { '1': 3 }, by_type: {} },
    });
    const out = await alarmApi.getAlarmCount();
    expect(out.total_active).toBe(7);
    expect(out.critical).toBe(3);
    expect(out.major).toBe(0); // 缺键兜 0
    expect(out.minor).toBe(0);
    expect(out.warning).toBe(0);
  });

  it('getAlarmCount：by_severity 整体缺失（undefined）不崩，全部兜 0', async () => {
    getMock.mockResolvedValue({ data: { total_active: 0, by_type: {} } });
    const out = await alarmApi.getAlarmCount();
    expect(out.critical).toBe(0);
    expect(out.warning).toBe(0);
  });
});
