import { describe, expect, it } from 'vitest';
import type { Domain } from '@core/types/topology';
import type { DeviceGroupNode } from '@core/types/map';
import {
  buildGroupDisplayNameById,
  domainToGroupNode,
  filterGroupTreeBySearch,
  getGroupNodeDisplayName,
} from './groupDisplay';

function group(overrides: Partial<DeviceGroupNode> = {}): DeviceGroupNode {
  return {
    id: 'default-root',
    name: '默认设备组',
    nameI18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' },
    parentId: null,
    level: 1,
    ...overrides,
  };
}

describe('GIS device-group display i18n', () => {
  it('按当前语言显示系统内置设备组名称', () => {
    const node = group();

    expect(getGroupNodeDisplayName(node, 'en-US')).toBe('Default Group');
    expect(getGroupNodeDisplayName(node, 'zh-CN')).toBe('默认设备组');
  });

  it('自定义设备组没有对应译文时保留用户名称', () => {
    const node = group({
      id: 'custom',
      name: '5G group716',
      nameI18n: undefined,
    });

    expect(getGroupNodeDisplayName(node, 'en-US')).toBe('5G group716');
  });

  it('Domain 转换时递归保留父子节点的 nameI18n', () => {
    const domain: Domain = {
      id: 'default-root',
      name: '默认设备组',
      nameI18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' },
      parentId: null,
      level: 1,
      deviceCount: 0,
      children: [
        {
          id: 'default-child',
          name: '默认设备组',
          nameI18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' },
          parentId: 'default-root',
          level: 2,
          deviceCount: 0,
        },
      ],
    };

    const node = domainToGroupNode(domain);

    expect(node.nameI18n?.['en-US']).toBe('Default Group');
    expect(node.children?.[0].nameI18n?.['en-US']).toBe('Default Group');
  });

  it('英文界面按英文显示名称搜索，并为地图弹窗生成同一名称映射', () => {
    const tree = [
      group({
        children: [
          group({
            id: 'default-child',
            parentId: 'default-root',
            level: 2,
          }),
        ],
      }),
    ];

    expect(filterGroupTreeBySearch(tree, 'Default', 'en-US')).toHaveLength(1);
    expect(buildGroupDisplayNameById(tree, 'en-US').get('default-child')).toBe('Default Group');
  });
});
