import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock },
}));

import { topologyApi } from '../topologyApi';

beforeEach(() => {
  getMock.mockReset();
});

describe('topologyApi device-group i18n mapping', () => {
  it('递归透传父子设备组的 name_i18n', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            id: 'default-root',
            name: '默认设备组',
            name_i18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' },
            parent_id: null,
            carrier: '',
            description: '',
            sort_order: 0,
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
            children: [
              {
                id: 'default-child',
                name: '默认设备组',
                name_i18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' },
                parent_id: 'default-root',
                carrier: '',
                description: '',
                sort_order: 0,
                created_at: '2026-01-01T00:00:00Z',
                updated_at: '2026-01-01T00:00:00Z',
              },
            ],
          },
        ],
      },
    });

    const tree = await topologyApi.getDomainTree();

    expect(tree[0].nameI18n).toEqual({
      'zh-CN': '默认设备组',
      'en-US': 'Default Group',
    });
    expect(tree[0].children?.[0].nameI18n).toEqual({
      'zh-CN': '默认设备组',
      'en-US': 'Default Group',
    });
  });

  it('透传设备搜索结果的 group_id 以便按当前语言显示设备组', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            id: 'device-1',
            name: 'Device 1',
            sn: 'SN001',
            status: 'online',
            longitude: 116.4,
            latitude: 39.9,
            group_id: 'default-child',
            group_name: '默认设备组',
          },
        ],
      },
    });

    const results = await topologyApi.searchDevices('SN001');

    expect(results[0].groupId).toBe('default-child');
  });
});
