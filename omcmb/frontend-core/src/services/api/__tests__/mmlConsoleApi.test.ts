import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock },
}));

import { mmlApi } from '../mmlApi';

beforeEach(() => {
  getMock.mockReset();
});

describe('mmlApi console product filtering', () => {
  it('passes the selected device to the command tree request', async () => {
    getMock.mockResolvedValue({ data: { tree: [] } });

    await mmlApi.buildGroupTree(undefined, 'zh-CN', undefined, 'SN-MLQ-1');

    expect(getMock).toHaveBeenCalledWith('/mml/group-tree', {
      params: { lang: 'zh-CN', device_sn: 'SN-MLQ-1' },
    });
  });

  it('passes the selected device to the command sub-fields request', async () => {
    getMock.mockResolvedValue({ data: { sub_fields: [] } });

    await mmlApi.getCommandSubFields('command-1', 'zh-CN', 'SN-MLQ-1');

    expect(getMock).toHaveBeenCalledWith('/mml/commands/command-1/sub-fields', {
      params: { lang: 'zh-CN', device_sn: 'SN-MLQ-1' },
    });
  });
});
