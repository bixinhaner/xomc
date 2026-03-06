import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ConfigTemplate, BaselineConfig, ConfigTask, ConfigParam, NeighborParam } from '@/types/config';
import type { PageRequest } from '@/types/pagination';
import { configService } from '@/mock/services/configService';

export function useConfigParams(
  params: { deviceSn?: string; category?: string; keyword?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['config', 'params', params],
    queryFn: () => configService.getParams(params),
  });
}

export function useUpdateConfigParam() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, value }: { id: string; value: string | number | boolean }) =>
      configService.updateParam(id, value),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'params'] });
    },
  });
}

export function useConfigTemplates(params: PageRequest) {
  return useQuery({
    queryKey: ['config', 'templates', params],
    queryFn: () => configService.getTemplates(params),
  });
}

export function useConfigTemplateById(id: string) {
  return useQuery({
    queryKey: ['config', 'templates', 'detail', id],
    queryFn: () => configService.getTemplateById(id),
    enabled: Boolean(id),
  });
}

export function useCreateConfigTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ConfigTemplate, 'id' | 'createTime'>) =>
      configService.createTemplate(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'templates'] });
    },
  });
}

export function useUpdateConfigTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ConfigTemplate> }) =>
      configService.updateTemplate(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'templates'] });
    },
  });
}

export function useDeleteConfigTemplates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => configService.deleteTemplates(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'templates'] });
    },
  });
}

export function useBaselineConfigs(
  params: { deviceType?: string; status?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['config', 'baselines', params],
    queryFn: () => configService.getBaselines(params),
  });
}

export function useBaselineConfigById(id: string) {
  return useQuery({
    queryKey: ['config', 'baselines', 'detail', id],
    queryFn: () => configService.getBaselineById(id),
    enabled: Boolean(id),
  });
}

export function useCreateBaselineConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<BaselineConfig, 'id' | 'createTime' | 'updateTime'>) =>
      configService.createBaseline(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'baselines'] });
    },
  });
}

export function useUpdateBaselineConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<BaselineConfig> }) =>
      configService.updateBaseline(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'baselines'] });
    },
  });
}

export function useDeleteBaselineConfigs() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => configService.deleteBaselines(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'baselines'] });
    },
  });
}

export function useConfigTasks(params: PageRequest) {
  return useQuery({
    queryKey: ['config', 'tasks', params],
    queryFn: () => configService.getTasks(params),
  });
}

export function useCreateConfigTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ConfigTask, 'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'successCount' | 'failCount'>) =>
      configService.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'tasks'] });
    },
  });
}

export function useNeighborParams(params: { sourceCellId?: string } & PageRequest) {
  return useQuery({
    queryKey: ['config', 'neighbors', params],
    queryFn: () => configService.getNeighbors(params),
  });
}

// suppress unused import warning
void (null as unknown as ConfigParam);
void (null as unknown as NeighborParam);
