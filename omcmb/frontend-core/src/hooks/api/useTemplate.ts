import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ConfigTemplate } from '../../types/config';
import type { PageRequest } from '../../types/pagination';
import { configService } from '../../mock/services/configService';
import { templateApi } from '../../services/api/templateApi';
import { useMock } from '../../services/apiSwitch';

/**
 * Template hooks — direct, single-purpose wrapper over `templateApi`.
 *
 * `useConfig` exposes the config-page-flavored variants
 * (`useConfigTemplates`, `useConfigTemplateById` …) that are co-located with
 * baseline / sync hooks. This file is a slimmer one-to-one mapping for pages
 * that only need template CRUD (e.g. template-library / picker components),
 * keeping the `services/api/*.ts` ↔ `hooks/api/*.ts` file-level parity intact.
 *
 * Query key namespace: `['templates', ...]` (kept distinct from `['config',
 * 'templates', ...]` to avoid invalidation cross-talk).
 */

export function useTemplateList(params: PageRequest) {
  return useQuery({
    queryKey: ['templates', 'list', params],
    queryFn: () =>
      useMock ? configService.getTemplates(params) : templateApi.getTemplates(params),
  });
}

export function useTemplateById(id: string) {
  return useQuery({
    queryKey: ['templates', 'detail', id],
    queryFn: () =>
      useMock ? configService.getTemplateById(id) : templateApi.getTemplateById(id),
    enabled: Boolean(id),
  });
}

export function useCreateTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ConfigTemplate, 'id' | 'createTime'>) =>
      useMock ? configService.createTemplate(data) : templateApi.createTemplate(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['templates'] });
    },
  });
}

export function useUpdateTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ConfigTemplate> }) =>
      useMock ? configService.updateTemplate(id, data) : templateApi.updateTemplate(id, data),
    onSuccess: (_result, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ['templates', 'detail', id] });
      void queryClient.invalidateQueries({ queryKey: ['templates', 'list'] });
    },
  });
}

export function useDeleteTemplates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? configService.deleteTemplates(ids) : templateApi.deleteTemplates(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['templates'] });
    },
  });
}

/**
 * T-0120 / T-0120-b: 模板显式下发。
 * 调用方传 templateId + 设备 id 数组；mutation 返回 DispatchTemplateResponse
 * （包含 dispatched / failed / totalDevices），UI 可据此渲染结果 Modal。
 *
 * 注意：mock 模式当前不模拟 dispatch（业务路径需要真实 ACS）；
 * `useMock=true` 时直接抛错让 UI 显式提示"mock 模式不支持下发"。
 */
export function useDispatchTemplate() {
  return useMutation({
    mutationFn: ({ templateId, deviceIds }: { templateId: string; deviceIds: string[] }) => {
      if (useMock) {
        return Promise.reject(new Error('mock 模式不支持模板下发，请关闭 VITE_USE_MOCK'));
      }
      return templateApi.dispatchTemplate(templateId, deviceIds);
    },
  });
}
