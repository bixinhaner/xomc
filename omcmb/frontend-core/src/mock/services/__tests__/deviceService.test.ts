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

describe('mock deviceService searchText filtering', () => {
  it('searchText 使用多字段包含搜索，长词结果不会多于短词', async () => {
    const all = await deviceService.getList({ page: 1, pageSize: 200 });
    const target = all.items.find((item) => item.sn.length >= 4);
    expect(target).toBeTruthy();

    const short = target!.sn.slice(0, 3);
    const long = target!.sn.slice(0, 4);
    const shortOut = await deviceService.getList({ page: 1, pageSize: 200, searchText: short });
    const longOut = await deviceService.getList({ page: 1, pageSize: 200, searchText: long });

    expect(shortOut.total).toBeGreaterThanOrEqual(longOut.total);
    expect(longOut.items.every((item) => shortOut.items.some((s) => s.id === item.id))).toBe(true);
  });
});
