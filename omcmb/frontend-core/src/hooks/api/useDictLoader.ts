import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  dictLoaderApi,
  type DictLoaderName,
  type DictLoaderReloadResult,
} from '../../services/api/dictLoaderApi';

/**
 * T-0183 — dictload Loader 热重载 mutation hook。
 *
 * 用法:
 *   const { mutateAsync, isPending } = useReloadDictLoader();
 *   await mutateAsync('param-model').then(res => { ... });
 *
 * 成功后自动失效 mml-related queries(因 mml-standard / param-model 重载会
 * 改变 console 命令树 + sub_field 元数据)。其他 loader 影响小,不强制失效。
 */
export function useReloadDictLoader() {
  const queryClient = useQueryClient();

  return useMutation<DictLoaderReloadResult, Error, DictLoaderName>({
    mutationFn: (name) => dictLoaderApi.reload(name),
    onSuccess: (_result, name) => {
      // mml-standard / param-model 影响 MML console:命令树 + sub_field 列表都失效重读
      if (name === 'mml-standard' || name === 'param-model') {
        void queryClient.invalidateQueries({ queryKey: ['mml'] });
      }
      // product loader 影响 ProductRegistry,理论上后端 reload 会自动失效,但前端
      // 缓存层(devices/products 列表)也应跟着失效
      if (name === 'product') {
        void queryClient.invalidateQueries({ queryKey: ['products'] });
      }
    },
  });
}
