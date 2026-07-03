// qa-614 c6 #368 / #483：UFTE 文件传输中心『设备升级』分类聚合（三皮肤共享）。
//
// 后端 ufte_task_types 把升级类拆成 enb_upgrade(4G) / gnb_upgrade(5G) / gsm_upgrade(2G)
// 三条独立 category（数据契约不变）。本工具在**展示层**把这三条折叠为单个虚拟分类
// device_upgrade（『设备升级』），制式由具体模板(typeCode)区分，并提供查询参数展开
// ——三皮肤（v1/v2/v3）统一调用，保证分类口径一致（三皮肤铁律：业务逻辑/数据契约
// 改在 frontend-core 一处生效）。
//
// 关键事实：后端任务/设备列表的 category 过滤实际是 `category → softwareTaskType 集`
// 映射（ufte/model.go filterTaskTypeSet），不区分制式——enb_upgrade 的 softwareTaskType
// 集 {Upgrade,Patch,FPGA} 是 gnb_upgrade/gsm_upgrade 集 {Upgrade} 的超集；且 2G/4G/5G
// 升级在后端同归 software.TaskTypeUpgrade。故选中『设备升级』且未选具体 typeCode 时，
// 传成员超集 'enb_upgrade' 即可一次列出 4G/5G/2G 全部升级任务，无需改后端。
// 5G 的「102 UPGRADE FINISH」等待由软件引擎按设备制式(Is5G)运行期驱动，与本分类无关。

import type { UnifiedFileTransferTaskType } from '../types/unifiedFileTransfer';

export const DEVICE_UPGRADE_CATEGORY = 'device_upgrade';
export const DEVICE_UPGRADE_MEMBER_CATEGORIES = ['enb_upgrade', 'gnb_upgrade', 'gsm_upgrade'] as const;

/** 判断某后端 category 是否属于『设备升级』虚拟分类（成员 4G/5G/2G，或自定义模板直接存为 device_upgrade 字面值）。 */
export function isDeviceUpgradeMember(category: string | undefined): boolean {
  return category === 'enb_upgrade' || category === 'gnb_upgrade' || category === 'gsm_upgrade'
    || category === DEVICE_UPGRADE_CATEGORY;
}

export interface AggregatedCategoryOption {
  value: string;
  /** 后端原始 categoryLabel；device_upgrade 这种虚拟项 label 由调用方按 i18n 覆盖。 */
  label: string;
}

/**
 * 从 taskTypes 聚合分类选项（去重），并把 enb_upgrade + gnb_upgrade + gsm_upgrade
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
 * 选中 device_upgrade 且无 typeCode → 传成员超集 'enb_upgrade'（含 4G/5G/2G 升级任务）。
 */
export function resolveBackendCategoryParam(
  selectedCategory: string,
  selectedTypeCode?: string,
): string | undefined {
  if (selectedCategory === DEVICE_UPGRADE_CATEGORY) {
    if (selectedTypeCode) return undefined;
    return DEVICE_UPGRADE_MEMBER_CATEGORIES[0];
  }
  return selectedCategory || undefined;
}

/** device_upgrade 虚拟分类下取 enb_upgrade + gnb_upgrade + gsm_upgrade 三类 taskType，
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
