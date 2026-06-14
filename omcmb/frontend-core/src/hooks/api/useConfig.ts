import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { ConfigTemplate, BaselineConfig, ConfigTask, ConfigParam, NeighborParam } from '../../types/config';
import type { PageRequest } from '../../types/pagination';
import { configService } from '../../mock/services/configService';
import { templateApi } from '../../services/api/templateApi';
import { deviceApi } from '../../services/api/deviceApi';
import { configSyncApi } from '../../services/api/configSyncApi';
import { configBaselineApi } from '../../services/api/configBaselineApi';
import { useMock } from '../../services/apiSwitch';

export function useConfigParams(
  params: { deviceSn?: string; deviceId?: string; category?: string; keyword?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['config', 'params', params],
    // 真实模式下没选设备（deviceId/deviceSn 都空）不发请求，否则会打出
    // `/devices//parameters`（空段）→ 400。mock 模式不依赖设备 id，照常放行。
    enabled: useMock || Boolean(params.deviceId || params.deviceSn),
    queryFn: () =>
      useMock
        ? (configService.getParams(params) as unknown as ReturnType<typeof deviceApi.getParameters>)
        : deviceApi.getParameters(params.deviceId ?? ''),
  });
}

export function useUpdateConfigParam() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      value,
      deviceId,
    }: {
      id: string;
      value: string | number | boolean;
      deviceId?: string;
    }) =>
      useMock
        ? (configService.updateParam(id, value) as unknown as Promise<ConfigParam>)
        : (configSyncApi.pushConfig(deviceId ?? '', [
            { name: id, value: String(value) },
          ]) as unknown as Promise<ConfigParam>),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'params'] });
    },
  });
}

export function useConfigTemplates(params: PageRequest) {
  return useQuery({
    queryKey: ['config', 'templates', params],
    queryFn: () =>
      useMock ? configService.getTemplates(params) : templateApi.getTemplates(params),
  });
}

export function useConfigTemplateById(id: string) {
  return useQuery({
    queryKey: ['config', 'templates', 'detail', id],
    queryFn: () =>
      useMock ? configService.getTemplateById(id) : templateApi.getTemplateById(id),
    enabled: Boolean(id),
  });
}

export function useCreateConfigTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ConfigTemplate, 'id' | 'createTime'>) =>
      useMock ? configService.createTemplate(data) : templateApi.createTemplate(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'templates'] });
    },
  });
}

export function useUpdateConfigTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ConfigTemplate> }) =>
      useMock ? configService.updateTemplate(id, data) : templateApi.updateTemplate(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'templates'] });
    },
  });
}

export function useDeleteConfigTemplates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? configService.deleteTemplates(ids) : templateApi.deleteTemplates(ids),
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
    queryFn: () =>
      useMock ? configService.getBaselines(params) : configBaselineApi.getBaselines(params),
  });
}

export function useBaselineConfigById(id: string) {
  return useQuery({
    queryKey: ['config', 'baselines', 'detail', id],
    queryFn: () =>
      useMock ? configService.getBaselineById(id) : configBaselineApi.getBaselineById(id),
    enabled: Boolean(id),
  });
}

export function useCreateBaselineConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<BaselineConfig, 'id' | 'createTime' | 'updateTime'>) =>
      useMock ? configService.createBaseline(data) : configBaselineApi.createBaseline(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'baselines'] });
    },
  });
}

export function useUpdateBaselineConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<BaselineConfig> }) =>
      useMock ? configService.updateBaseline(id, data) : configBaselineApi.updateBaseline(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'baselines'] });
    },
  });
}

export function useDeleteBaselineConfigs() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? configService.deleteBaselines(ids) : configBaselineApi.deleteBaselines(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'baselines'] });
    },
  });
}

export function useConfigTasks(params: PageRequest) {
  return useQuery({
    queryKey: ['config', 'tasks', params],
    queryFn: () =>
      useMock ? configService.getTasks(params) : configBaselineApi.getTasks(params),
  });
}

export function useCreateConfigTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<ConfigTask, 'id' | 'createdAt' | 'updatedAt' | 'status' | 'progress' | 'successCount' | 'failCount'>) =>
      useMock ? configService.createTask(data) : configBaselineApi.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['config', 'tasks'] });
    },
  });
}

export function useNeighborParams(params: { sourceCellId?: string } & PageRequest) {
  return useQuery({
    queryKey: ['config', 'neighbors', params],
    queryFn: () =>
      useMock ? configService.getNeighbors(params) : configBaselineApi.getNeighbors(params),
  });
}

// suppress unused import warning
void (null as unknown as ConfigParam);
void (null as unknown as NeighborParam);
