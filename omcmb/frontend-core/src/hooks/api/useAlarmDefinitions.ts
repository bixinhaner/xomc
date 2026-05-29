import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { alarmDefinitionApi } from '../../services/api/alarmDefinitionApi';
import { alarmDefinitionService } from '../../mock/services/alarmDefinitionService';
import { createApiSwitch } from '../../services/apiSwitch';
import type {
  AlarmDefinitionFilter,
  CreateAlarmDefinitionInput,
  UpdateAlarmDefinitionInput,
  UnknownStatsFilter,
} from '../../types/alarmDefinition';

const api = createApiSwitch(alarmDefinitionService, alarmDefinitionApi);

const AD_KEY = ['alarm-definitions'] as const;

export function useAlarmDefinitionList(filter?: AlarmDefinitionFilter) {
  return useQuery({
    queryKey: [...AD_KEY, 'list', filter ?? {}],
    queryFn: () => api.list(filter),
  });
}

export function useAllAlarmDefinitions(filter?: Omit<AlarmDefinitionFilter, 'page'>) {
  return useQuery({
    queryKey: [...AD_KEY, 'list-all', filter ?? {}],
    queryFn: () => api.listAll(filter),
  });
}

export function useAlarmDefinitionDetail(identifier: string | undefined) {
  return useQuery({
    queryKey: [...AD_KEY, 'detail', identifier],
    queryFn: () => api.get(identifier as string),
    enabled: Boolean(identifier),
  });
}

export function useCreateAlarmDefinition() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateAlarmDefinitionInput) => api.create(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: AD_KEY });
    },
  });
}

export function useUpdateAlarmDefinition() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ identifier, input }: { identifier: string; input: UpdateAlarmDefinitionInput }) =>
      api.update(identifier, input),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...AD_KEY, 'detail', vars.identifier] });
      void qc.invalidateQueries({ queryKey: [...AD_KEY, 'list'] });
    },
  });
}

export function useDeleteAlarmDefinition() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (identifier: string) => api.delete(identifier),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: AD_KEY });
    },
  });
}

export function useUnknownAlarmStats(filter?: UnknownStatsFilter) {
  return useQuery({
    queryKey: [...AD_KEY, 'unknown-stats', filter ?? {}],
    queryFn: () => api.unknownStats(filter),
  });
}

export function useAlarmNeTypeStats() {
  return useQuery({
    queryKey: [...AD_KEY, 'ne-types'],
    queryFn: () => api.listNeTypes(),
  });
}

export function useAlarmSeverityLevels() {
  return useQuery({
    queryKey: [...AD_KEY, 'severity-levels'],
    queryFn: () => api.severityLevels(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useAlarmDefinitionCacheRefresh() {
  return useMutation({
    mutationFn: () => api.cacheRefresh(),
  });
}

export function useAlarmDefinitionImportDirectory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.importDirectory(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: AD_KEY });
    },
  });
}

// 2026-05-29:destructive 全量重载;成功后 invalidate 所有 alarm-defs 查询,
// 让前端立刻看到孤儿告警被删除后的最新清单(与 useParamModelReloadDirectory 同形)。
export function useAlarmDefinitionReloadDirectory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.reloadDirectory(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: AD_KEY });
    },
  });
}
