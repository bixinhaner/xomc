// qa-614 c6 #368 / #483：UFTE 文件传输中心『设备升级』分类聚合（共享业务层）。
//
// 后端 ufte_task_types 把升级类拆成 enb_upgrade(4G) / gnb_upgrade(5G) / gsm_upgrade(2G) / ups_upgrade(UPS)
// 四条独立 category（数据契约不变）。本工具在**展示层**把这四条折叠为单个虚拟分类
// device_upgrade（『设备升级』），制式由具体模板(typeCode)区分，并提供查询参数展开
// ——页面统一调用，保证分类口径一致（业务逻辑/数据契约
// 改在 frontend-core 一处生效）。
//
// 关键事实：后端同样认识 device_upgrade 虚拟分类，并展开为 4G/5G/2G/UPS 升级成员。
// 选中『设备升级』且未选具体 typeCode 时，直接传 device_upgrade，避免 UPS 软件升级
// 因后端仍按 ups_upgrade 映射而被 enb_upgrade 二次过滤掉。
// 5G 的「102 UPGRADE FINISH」等待由软件引擎按设备制式(Is5G)运行期驱动，与本分类无关。

import type { UnifiedFileTransferTaskType } from '../types/unifiedFileTransfer';

export const DEVICE_UPGRADE_CATEGORY = 'device_upgrade';
export const DEVICE_UPGRADE_MEMBER_CATEGORIES = ['enb_upgrade', 'gnb_upgrade', 'gsm_upgrade', 'ups_upgrade'] as const;
export const UPS_UPGRADE_CATEGORY = 'ups_upgrade';
export const UPS_AP_UPGRADE_TYPE_CODE = 'UPS_AP_UPGRADE';

/** 判断某后端 category 是否属于『设备升级』虚拟分类（成员 4G/5G/2G/UPS，或自定义模板直接存为 device_upgrade 字面值）。 */
export function isDeviceUpgradeMember(category: string | undefined): boolean {
  return category === 'enb_upgrade' || category === 'gnb_upgrade' || category === 'gsm_upgrade' || category === 'ups_upgrade'
    || category === DEVICE_UPGRADE_CATEGORY;
}

export interface UfteUPSScopeLike {
  category?: string;
  typeCode?: string;
  productType?: string;
  productName?: string;
  products?: string[];
}

function isUPSProductName(value: string | undefined): boolean {
  return value?.trim().toUpperCase() === 'UPS';
}

/** UPS 软件升级只在 license 支持 UPS 时进入文件传输 UI；其它升级模板仍保留在『设备升级』下。 */
export function isUPSFileTransferScope(item: UfteUPSScopeLike): boolean {
  return item.category === UPS_UPGRADE_CATEGORY
    || item.typeCode === UPS_AP_UPGRADE_TYPE_CODE
    || isUPSProductName(item.productType)
    || isUPSProductName(item.productName)
    || (item.products ?? []).some(isUPSProductName);
}

export function filterTaskTypesByUPSLicense(
  taskTypes: ReadonlyArray<UnifiedFileTransferTaskType>,
  upsLicensed: boolean,
): UnifiedFileTransferTaskType[] {
  if (upsLicensed) return [...taskTypes];
  return taskTypes.filter((item) => !isUPSFileTransferScope(item));
}

export function filterFileTransferItemsByUPSLicense<T extends UfteUPSScopeLike>(
  items: ReadonlyArray<T>,
  upsLicensed: boolean,
): T[] {
  if (upsLicensed) return [...items];
  return items.filter((item) => !isUPSFileTransferScope(item));
}

export interface AggregatedCategoryOption {
  value: string;
  /** 后端原始 categoryLabel；device_upgrade 这种虚拟项 label 由调用方按 i18n 覆盖。 */
  label: string;
}

/**
 * 从 taskTypes 聚合分类选项（去重），并把 enb_upgrade + gnb_upgrade + gsm_upgrade + ups_upgrade
 * 折叠为单条 device_upgrade。device_upgrade 的 label 用占位 'device_upgrade'（i18n key），
 * 各皮肤渲染时自行翻译；其它分类沿用后端 categoryLabel。
 */
export function aggregateCategoryOptions(
  taskTypes: ReadonlyArray<UnifiedFileTransferTaskType>,
): AggregatedCategoryOption[] {
  const map = new Map<string, string>();
  taskTypes.forEach((tt) => {
    if (isDeviceUpgradeMember(tt.category)) {
      if (!map.has(DEVICE_UPGRADE_CATEGORY)) {
        map.set(DEVICE_UPGRADE_CATEGORY, DEVICE_UPGRADE_CATEGORY);
      }
      return;
    }
    if (!map.has(tt.category)) map.set(tt.category, tt.categoryLabel);
  });
  return Array.from(map.entries()).map(([value, label]) => ({ value, label }));
}

/**
 * 把展示层选中的 category 展开为后端任务/设备列表查询用的 category 参数。
 * 选中 device_upgrade 且有 typeCode → 传 undefined（按 typeCode 精确过滤）；
 * 选中 device_upgrade 且无 typeCode → 传虚拟分类 device_upgrade（含 4G/5G/2G/UPS 升级任务）。
 */
export function resolveBackendCategoryParam(
  selectedCategory: string,
  selectedTypeCode?: string,
): string | undefined {
  if (selectedCategory === DEVICE_UPGRADE_CATEGORY) {
    if (selectedTypeCode) return undefined;
    return DEVICE_UPGRADE_CATEGORY;
  }
  return selectedCategory || undefined;
}

/** device_upgrade 虚拟分类下取 enb_upgrade + gnb_upgrade + gsm_upgrade + ups_upgrade 四类 taskType，
 * 以及自定义模板直接存为 device_upgrade 字面值的记录。 */
export function filterTaskTypesForCategory(
  taskTypes: ReadonlyArray<UnifiedFileTransferTaskType>,
  selectedCategory: string,
): UnifiedFileTransferTaskType[] {
  if (selectedCategory === DEVICE_UPGRADE_CATEGORY) {
    return taskTypes.filter((item) => isDeviceUpgradeMember(item.category));
  }
  return taskTypes.filter((item) => item.category === selectedCategory);
}
