import { useQuery } from '@tanstack/react-query';
import { quicksettingsApi } from '../../services/api/quicksettingsApi';
import type { TechCode } from '../../types/quicksettings';

/**
 * 拉取「快速设置」分组元数据（T-0138）。
 *
 * 分组数据是字典级常量，stale-while-revalidate 长缓存即可。
 */
export function useQuickSettingsGroups(tech: TechCode | undefined) {
  return useQuery({
    queryKey: ['quicksettings', 'groups', tech],
    queryFn: () => quicksettingsApi.getGroups(tech as TechCode),
    enabled: Boolean(tech),
    staleTime: 10 * 60 * 1000, // 10 分钟
  });
}
