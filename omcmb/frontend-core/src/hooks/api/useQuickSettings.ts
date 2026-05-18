import { useQuery } from '@tanstack/react-query';
import { quicksettingsApi } from '../../services/api/quicksettingsApi';

/**
 * 拉取「快速设置」分组元数据(T-0138)。
 *
 * 输入:deviceId(后端走 device→product→paramModel 链路解析)。
 * 输出:`{paramModel, groups}`;**groups 为空数组**表示该 paramModel 未配置 XML,
 * 调用方应不渲染「快速设置」tab(规则:无 XML 不显示)。
 *
 * 分组数据是字典级常量,stale-while-revalidate 长缓存即可。
 */
export function useQuickSettingsGroups(deviceId: string | undefined) {
  return useQuery({
    queryKey: ['quicksettings', 'groups', deviceId],
    queryFn: () => quicksettingsApi.getGroups(deviceId as string),
    enabled: Boolean(deviceId),
    staleTime: 10 * 60 * 1000, // 10 分钟
    retry: false, // 设备无对应 paramModel(404/422)时不重试
  });
}
