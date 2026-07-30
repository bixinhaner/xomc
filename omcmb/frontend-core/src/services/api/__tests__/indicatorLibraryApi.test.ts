/**
 * PM 指标库 create 契约测试（issue #63）。
 *
 * 后端创建指标/分组时负责生成真实 ID；即使历史调用方误带 id，真实 API 请求体也不能提交 id。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, put: vi.fn(), delete: vi.fn() },
}));

import { indicatorLibraryApi } from '../indicatorLibraryApi';

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
});

describe('indicatorLibraryApi.list', () => {
  it('启用状态过滤使用后端可识别的 1/0 参数', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [],
        total: 0,
      },
    });

    await indicatorLibraryApi.list('ENB', {
      operatorCode: 'default',
      isEnabled: true,
      page: 1,
      pageSize: 20,
    });

    expect(getMock).toHaveBeenCalledWith('/indicators', {
      params: {
        deviceType: 'ENB',
        operatorCode: 'default',
        isEnabled: '1',
        page: 1,
        pageSize: 20,
      },
    });
  });
});

describe('indicatorLibraryApi.create', () => {
  it('新建指标 payload 不提交历史 id 字段', async () => {
    postMock.mockResolvedValue({
      data: {
        id: 'K900000001',
        name: 'Attach success rate',
        cn_name: '接入成功率',
        en_name: 'Attach success rate',
        group_id: 'gsm-call',
        is_counter: '0',
      },
    });

    await indicatorLibraryApi.create('GSM', {
      id: 'frontend-random-id',
      name: '接入成功率',
      cnName: '接入成功率',
      enName: 'Attach success rate',
      groupId: 'gsm-call',
      isCounter: '0',
      formulas: [{ platformName: 'BSC', formula: '[D000000001]/[D000000002]' }],
    });

    const [url, body, config] = postMock.mock.calls[0];
    expect(url).toBe('/indicators');
    expect(body).not.toHaveProperty('id');
    expect(body).toMatchObject({
      cn_name: '接入成功率',
      en_name: 'Attach success rate',
      group_id: 'gsm-call',
      is_counter: '0',
      formulas: [{ platform_name: 'BSC', formula: '[D000000001]/[D000000002]' }],
    });
    expect(config).toEqual({ params: { deviceType: 'GSM' } });
  });

  it('新建指标失败时原样抛出错误', async () => {
    const err = new Error('create indicator failed');
    postMock.mockRejectedValue(err);

    await expect(indicatorLibraryApi.create('GSM', {
      id: 'frontend-random-id',
      name: '接入成功率',
      cnName: '接入成功率',
      enName: 'Attach success rate',
      groupId: 'gsm-call',
      isCounter: '0',
    })).rejects.toBe(err);

    const [, body] = postMock.mock.calls[0];
    expect(body).not.toHaveProperty('id');
  });
});

describe('indicatorLibraryApi.createGroup', () => {
  it('新建分组 payload 不提交历史 id 字段', async () => {
    postMock.mockResolvedValue({
      data: {
        id: 'backend-group-id',
        cn_name: '自定义分组',
        en_name: '自定义分组',
        parent_id: '0',
      },
    });

    await indicatorLibraryApi.createGroup('GSM', {
      id: 'frontend-random-group-id',
      name: '自定义分组',
      parentId: '0',
      description: '现场自定义',
      operatorCode: 'cmcc',
    });

    const [url, body, config] = postMock.mock.calls[0];
    expect(url).toBe('/indicator-groups');
    expect(body).not.toHaveProperty('id');
    expect(body).toMatchObject({
      cn_name: '自定义分组',
      en_name: '自定义分组',
      parent_id: '0',
      description: '现场自定义',
      operator_code: 'cmcc',
    });
    expect(config).toEqual({ params: { deviceType: 'GSM' } });
  });

  it('新建分组失败时原样抛出错误', async () => {
    const err = new Error('create group failed');
    postMock.mockRejectedValue(err);

    await expect(indicatorLibraryApi.createGroup('GSM', {
      id: 'frontend-random-group-id',
      name: '自定义分组',
      parentId: '0',
    })).rejects.toBe(err);

    const [, body] = postMock.mock.calls[0];
    expect(body).not.toHaveProperty('id');
  });
});
