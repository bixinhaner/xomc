/**
 * useDeviceNameSyncMode
 *
 * 读取系统配置中设备名称同步策略（nameSyncMode），用于快设「小区名称」字段的
 * 策略联动禁用：auto_lmt_to_omc 时输入框禁用 + 灰字引导。
 *
 * 读取路径：GET /admin/sys-configs?category=device → 找 key=nameSyncMode。
 * 使用 useSysConfigsByCategory 共享缓存（staleTime=30s），多处调用不重复请求。
 */
import { useSysConfigsByCategory } from './useSystem';

export type NameSyncMode = 'auto_lmt_to_omc' | 'auto_omc_to_lmt' | 'prompt';

export function useDeviceNameSyncMode(): NameSyncMode {
  const { data } = useSysConfigsByCategory('device');
  const item = data?.find((c) => c.key === 'nameSyncMode');
  const value = item?.value;
  if (value === 'auto_lmt_to_omc' || value === 'auto_omc_to_lmt' || value === 'prompt') {
    return value;
  }
  return 'prompt'; // 默认最保守策略
}
