import { describe, expect, it } from 'vitest';

import type { UnifiedFileTransferTaskType } from '../types/unifiedFileTransfer';
import {
  DEVICE_UPGRADE_CATEGORY,
  aggregateCategoryOptions,
  filterFileTransferItemsByUPSLicense,
  filterTaskTypesByUPSLicense,
  filterTaskTypesForCategory,
  isDeviceUpgradeMember,
  isUPSFileTransferScope,
  resolveBackendCategoryParam,
} from './ufteCategory';

// 只读取 category / categoryLabel / typeCode，其余字段对本工具无意义，按需最小构造。
function tt(typeCode: string, category: string, categoryLabel: string): UnifiedFileTransferTaskType {
  return { typeCode, category, categoryLabel } as unknown as UnifiedFileTransferTaskType;
}

// #483/UPS：2G(gsm) / 4G(enb) / 5G(gnb) / UPS 四个真实升级分类，折叠后应只剩单条『设备升级』。
const CATALOG: UnifiedFileTransferTaskType[] = [
  tt('ENB_IMG_UPGRADE', 'enb_upgrade', '4G升级'),
  tt('ENB_PATCH_UPGRADE', 'enb_upgrade', '4G升级'),
  tt('GNB_IMG_UPGRADE', 'gnb_upgrade', '5G升级'),
  tt('GSM_IMG_UPGRADE', 'gsm_upgrade', '2G升级'),
  tt('UPS_AP_UPGRADE', 'ups_upgrade', 'UPS升级'),
  tt('VERSION_ROLLBACK', 'version_rollback', '基站版本回退'),
  tt('RUNTIME_LOG_COLLECT', 'station_log', '日志收集'),
];

describe('isDeviceUpgradeMember', () => {
  it('4G/5G/2G/UPS 升级分类都算成员', () => {
    expect(isDeviceUpgradeMember('enb_upgrade')).toBe(true);
    expect(isDeviceUpgradeMember('gnb_upgrade')).toBe(true);
    expect(isDeviceUpgradeMember('gsm_upgrade')).toBe(true);
    expect(isDeviceUpgradeMember('ups_upgrade')).toBe(true);
  });

  it('虚拟升级分类和后端成员都算成员，非升级分类与空值不算成员', () => {
    expect(isDeviceUpgradeMember('version_rollback')).toBe(false);
    expect(isDeviceUpgradeMember('station_log')).toBe(false);
    expect(isDeviceUpgradeMember('device_upgrade')).toBe(true);
    expect(isDeviceUpgradeMember(undefined)).toBe(false);
  });
});

describe('aggregateCategoryOptions', () => {
  it('把 4G/5G/2G/UPS 折叠为单条 device_upgrade，非升级分类原样保留', () => {
    const options = aggregateCategoryOptions(CATALOG);
    const values = options.map((o) => o.value);
    // 多个升级分类塌缩成一条 device_upgrade，且只出现一次。
    expect(values.filter((v) => v === DEVICE_UPGRADE_CATEGORY)).toHaveLength(1);
    expect(values).not.toContain('enb_upgrade');
    expect(values).not.toContain('gnb_upgrade');
    expect(values).not.toContain('gsm_upgrade');
    expect(values).not.toContain('ups_upgrade');
    // 非升级分类保留。
    expect(values).toContain('version_rollback');
    expect(values).toContain('station_log');
    // 7 条模板 → 升级4类折叠为1 + 回退 + 日志 = 3 个分类选项。
    expect(options).toHaveLength(3);
  });
});

describe('filterTaskTypesForCategory', () => {
  it('device_upgrade 展开为 enb+gnb+gsm+ups 全部升级模板', () => {
    const codes = filterTaskTypesForCategory(CATALOG, DEVICE_UPGRADE_CATEGORY).map((t) => t.typeCode);
    expect(codes).toEqual([
      'ENB_IMG_UPGRADE',
      'ENB_PATCH_UPGRADE',
      'GNB_IMG_UPGRADE',
      'GSM_IMG_UPGRADE',
      'UPS_AP_UPGRADE',
    ]);
  });

  it('真实分类按精确等值过滤', () => {
    expect(filterTaskTypesForCategory(CATALOG, 'station_log').map((t) => t.typeCode)).toEqual([
      'RUNTIME_LOG_COLLECT',
    ]);
  });
});

describe('UPS license filters', () => {
  it('未授权 UPS 时只隐藏 UPS 升级模板，保留其它设备升级模板', () => {
    const codes = filterTaskTypesByUPSLicense(CATALOG, false).map((t) => t.typeCode);
    expect(codes).toEqual([
      'ENB_IMG_UPGRADE',
      'ENB_PATCH_UPGRADE',
      'GNB_IMG_UPGRADE',
      'GSM_IMG_UPGRADE',
      'VERSION_ROLLBACK',
      'RUNTIME_LOG_COLLECT',
    ]);
  });

  it('授权 UPS 时保留 UPS 升级模板', () => {
    expect(filterTaskTypesByUPSLicense(CATALOG, true).map((t) => t.typeCode)).toContain('UPS_AP_UPGRADE');
  });

  it('可按 category/typeCode/product 信息识别 UPS 文件传输记录', () => {
    expect(isUPSFileTransferScope({ category: 'ups_upgrade' })).toBe(true);
    expect(isUPSFileTransferScope({ typeCode: 'UPS_AP_UPGRADE' })).toBe(true);
    expect(isUPSFileTransferScope({ productName: 'UPS' })).toBe(true);
    expect(isUPSFileTransferScope({ products: ['UPS'] })).toBe(true);
    expect(isUPSFileTransferScope({ category: 'enb_upgrade', productName: 'ENB' })).toBe(false);
  });

  it('未授权 UPS 时隐藏已有 UPS 任务/设备记录', () => {
    const items = filterFileTransferItemsByUPSLicense([
      { id: '1', typeCode: 'UPS_AP_UPGRADE' },
      { id: '2', typeCode: 'GNB_IMG_UPGRADE', category: 'gnb_upgrade' },
    ], false);
    expect(items).toEqual([{ id: '2', typeCode: 'GNB_IMG_UPGRADE', category: 'gnb_upgrade' }]);
  });
});

describe('resolveBackendCategoryParam', () => {
  it('device_upgrade 无 typeCode → 虚拟分类 device_upgrade（后端展开为全制式和 UPS 升级）', () => {
    expect(resolveBackendCategoryParam(DEVICE_UPGRADE_CATEGORY)).toBe('device_upgrade');
  });

  it('device_upgrade 选了 typeCode → undefined（交后端按 typeCode 精确过滤）', () => {
    expect(resolveBackendCategoryParam(DEVICE_UPGRADE_CATEGORY, 'GSM_IMG_UPGRADE')).toBeUndefined();
  });

  it('非虚拟分类原样透传', () => {
    expect(resolveBackendCategoryParam('station_log')).toBe('station_log');
  });
});
