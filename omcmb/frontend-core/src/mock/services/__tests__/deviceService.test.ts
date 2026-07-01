import { describe, expect, it } from 'vitest';

import { deviceService } from '../deviceService';

describe('mock deviceService group filtering', () => {
  it('按多选 groupId 做 OR 过滤，返回设备都属于选中组', async () => {
    const out = await deviceService.getList({
      page: 1,
      pageSize: 200,
      groupId: ['grp-bj', 'grp-sh'],
    });

    expect(out.items.length).toBeGreaterThan(0);
    expect(out.items.every((item) => item.groupId === 'grp-bj' || item.groupId === 'grp-sh')).toBe(true);
  });
});