/**
 * 设备列表批量操作的任务类型映射 / 详情判定（纯逻辑，便于单测）。
 *
 * #179：TR069 按需抓包入口（batch-tr069-collect）已从设备列表移除，统一走
 * 「运维管理 - TR069 报文跟踪」菜单。此处不再识别该 action，作为回归守门。
 */

export type BatchTaskTypeLabel = (key: string) => string;

/** 批量操作 actionKey → 任务类型展示名。不含已移除的 batch-tr069-collect。 */
export function buildBatchTaskTypeMap(t: BatchTaskTypeLabel): Record<string, string> {
  return {
    'batch-reboot': t('common.batchReboot'),
    'batch-log-collect': t('device.action.logCollect'),
    'batch-alarm-sync': t('device.action.alarmSync'),
    'batch-param-sync': t('device.action.paramSync'),
  };
}

/** 仅「日志采集」批量操作带任务详情（抓包入口移除后不再有 tr069-collect）。 */
export function batchActionHasDetail(actionKey?: string): boolean {
  return actionKey === 'batch-log-collect';
}

export function removeParamSyncOptimisticDeviceId(prev: Set<string>, deviceId: string): Set<string> {
  if (!prev.has(deviceId)) return prev;
  const next = new Set(prev);
  next.delete(deviceId);
  return next;
}
