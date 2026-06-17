import { describe, expect, it } from 'vitest';

import type { UnifiedFileTransferTaskType } from '../types/unifiedFileTransfer';
import {
  DEVICE_UPGRADE_CATEGORY,
  aggregateCategoryOptions,
  filterTaskTypesForCategory,
  isDeviceUpgradeMember,
  resolveBackendCategoryParam,
} from './ufteCategory';

// 只读取 category / categoryLabel / typeCode，其余字段对本工具无意义，按需最小构造。
function tt(typeCode: string, category: string, categoryLabel: string): UnifiedFileTransferTaskType {
  return { typeCode, category, categoryLabel } as unknown as UnifiedFileTransferTaskType;
}

// #483：2G(gsm) / 4G(enb) / 5G(gnb) 三个真实升级分类，折叠后应只剩单条『设备升级』。
const CATALOG: UnifiedFileTransferTaskType[] = [
  tt('ENB_IMG_UPGRADE', 'enb_upgrade', '4G升级'),
  tt('ENB_PATCH_UPGRADE', 'enb_upgrade', '4G升级'),
  tt('GNB_IMG_UPGRADE', 'gnb_upgrade', '5G升级'),
  tt('GSM_IMG_UPGRADE', 'gsm_upgrade', '2G升级'),
  tt('VERSION_ROLLBACK', 'version_rollback', '基站版本回退'),
  tt('RUNTIME_LOG_COLLECT', 'station_log', '日志收集'),
];

describe('isDeviceUpgradeMember', () => {
  it('4G/5G/2G 三制式升级分类都算成员', () => {
    expect(isDeviceUpgradeMember('enb_upgrade')).toBe(true);
    expect(isDeviceUpgradeMember('gnb_upgrade')).toBe(true);
    expect(isDeviceUpgradeMember('gsm_upgrade')).toBe(true);
  });

  it('非升级分类与空值不算成员', () => {
    expect(isDeviceUpgradeMember('version_rollback')).toBe(false);
    expect(isDeviceUpgradeMember('station_log')).toBe(false);
    expect(isDeviceUpgradeMember('device_upgrade')).toBe(false);
    expect(isDeviceUpgradeMember(undefined)).toBe(false);
  });
});

describe('aggregateCategoryOptions', () => {
  it('把 4G/5G/2G 折叠为单条 device_upgrade，非升级分类原样保留', () => {
    const options = aggregateCategoryOptions(CATALOG);
    const values = options.map((o) => o.value);
    // 三个升级分类塌缩成一条 device_upgrade，且只出现一次。
    expect(values.filter((v) => v === DEVICE_UPGRADE_CATEGORY)).toHaveLength(1);
    expect(values).not.toContain('enb_upgrade');
    expect(values).not.toContain('gnb_upgrade');
    expect(values).not.toContain('gsm_upgrade');
    // 非升级分类保留。
    expect(values).toContain('version_rollback');
    expect(values).toContain('station_log');
    // 6 条模板 → 升级3类折叠为1 + 回退 + 日志 = 3 个分类选项。
    expect(options).toHaveLength(3);
  });
});

describe('filterTaskTypesForCategory', () => {
  it('device_upgrade 展开为 enb+gnb+gsm 三制式全部升级模板', () => {
    const codes = filterTaskTypesForCategory(CATALOG, DEVICE_UPGRADE_CATEGORY).map((t) => t.typeCode);
    expect(codes).toEqual([
      'ENB_IMG_UPGRADE',
      'ENB_PATCH_UPGRADE',
      'GNB_IMG_UPGRADE',
      'GSM_IMG_UPGRADE',
    ]);
  });

  it('真实分类按精确等值过滤', () => {
    expect(filterTaskTypesForCategory(CATALOG, 'station_log').map((t) => t.typeCode)).toEqual([
      'RUNTIME_LOG_COLLECT',
    ]);
  });
});

describe('resolveBackendCategoryParam', () => {
  it('device_upgrade 无 typeCode → 成员超集 enb_upgrade（后端按 softwareTaskType 覆盖全制式升级）', () => {
    expect(resolveBackendCategoryParam(DEVICE_UPGRADE_CATEGORY)).toBe('enb_upgrade');
  });

  it('device_upgrade 选了 typeCode → undefined（交后端按 typeCode 精确过滤）', () => {
    expect(resolveBackendCategoryParam(DEVICE_UPGRADE_CATEGORY, 'GSM_IMG_UPGRADE')).toBeUndefined();
  });

  it('非虚拟分类原样透传', () => {
    expect(resolveBackendCategoryParam('station_log')).toBe('station_log');
  });
});
