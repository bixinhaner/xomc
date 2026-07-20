import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock, postMock, putMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
  deleteMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, put: putMock, patch: vi.fn(), delete: deleteMock },
}));

import { adminApi } from '../adminApi';

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  putMock.mockReset();
  deleteMock.mockReset();
});

describe('adminApi dictionary i18n mapping', () => {
  it('maps label_i18n from dictionary detail list responses', async () => {
    getMock.mockResolvedValue({
      data: {
        list: [
          {
            id: 1,
            label: '激活',
            label_i18n: { 'zh-CN': '激活', 'en-US': 'Active' },
            value: '1',
            extend: '',
            status: true,
            sort: 0,
            sysDictionaryId: 9,
            parent_id: null,
            level: 0,
            origin: 'manual',
          },
        ],
        total: 1,
      },
    });

    const out = await adminApi.getDictionaryDetailList({ sysDictionaryId: 9 });

    expect(out.list[0].labelI18n).toEqual({ 'zh-CN': '激活', 'en-US': 'Active' });
  });

  it('maps name_i18n and nested detail label_i18n from findDictionaryByType', async () => {
    getMock.mockResolvedValue({
      data: {
        id: 9,
        name: '设备激活状态',
        name_i18n: { 'zh-CN': '设备激活状态', 'en-US': 'Device Activation State' },
        description_i18n: { 'en-US': 'Controls activation label display' },
        type: 'op_state',
        status: true,
        desc: '设备激活状态字典',
        sysDictionaryDetails: [
          {
            id: 1,
            label: '激活',
            label_i18n: { 'zh-CN': '激活', 'en-US': 'Active' },
            value: '1',
            extend: '',
            status: true,
            sort: 0,
            sysDictionaryId: 9,
            parent_id: null,
            level: 0,
            origin: 'manual',
          },
        ],
      },
    });

    const out = await adminApi.findDictionaryByType('op_state');

    expect(out?.nameI18n).toEqual({ 'zh-CN': '设备激活状态', 'en-US': 'Device Activation State' });
    expect(out?.descriptionI18n).toEqual({ 'en-US': 'Controls activation label display' });
    expect(out?.sysDictionaryDetails?.[0].labelI18n).toEqual({ 'zh-CN': '激活', 'en-US': 'Active' });
  });
});

describe('adminApi system config secret mapping', () => {
  it('maps backend secret metadata to camelCase fields', async () => {
    getMock.mockResolvedValue({
      data: [
        {
          id: 'config-1',
          category: 'security',
          key: 'defaultPasswd',
          value: '',
          value_type: 'string',
          is_public: false,
          is_secret: true,
          is_configured: true,
        },
      ],
    });

    const [item] = await adminApi.getSysConfigsByCategory('security');

    expect(item).toMatchObject({
      key: 'defaultPasswd',
      value: '',
      isPublic: false,
      isSecret: true,
      isConfigured: true,
    });
  });
});
