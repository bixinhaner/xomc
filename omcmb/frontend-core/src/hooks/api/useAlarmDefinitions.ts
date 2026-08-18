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
    onSuccess: () => {
      // #268: 与 create/delete 对齐失效整个 AD_KEY。此前只失效 detail+list,
      // 一级表 ne-types 聚合(按严重性统计)不刷新 —— 一级/二级是同组件内切换,
      // useAlarmNeTypeStats 常驻挂载不会 remount 触发 refetch。
      void qc.invalidateQueries({ queryKey: AD_KEY });
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

// 自定义 XML 上传(名称取自 XML neType 属性;force=true 确认覆盖):
// 成功后 invalidate 所有 alarm-defs 查询。
export function useAlarmUploadXml() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ file, force }: { file: File; force?: boolean }) => api.uploadXml(file, force),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: AD_KEY });
      // #241：后端导入成功后已按 source_table 刷新 alarm_ne_type 绑定字典;
      // 这里失效字典查询,让「新增产品 → 告警名称」下拉立即重取到新 neType。
      void qc.invalidateQueries({ queryKey: ['dictionary', 'alarm_ne_type'] });
    },
  });
}

// 下载 XML(操作列下载图标)。#268: 两种目标 ——
//   loadedFrom 有值:下载磁盘原文件(builtin / custom 均可);
//   仅 neType(手工新增行,无加载源):后端从 DB 动态生成 alarmModel XML。
export function useAlarmDownloadXml() {
  return useMutation({
    mutationFn: (target: { loadedFrom?: string; neType?: string }) => api.downloadXml(target),
  });
}

// 自定义 XML 删除:成功后 invalidate 所有 alarm-defs 查询。
export function useAlarmDeleteFile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (loadedFrom: string) => api.deleteFile(loadedFrom),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: AD_KEY });
    },
  });
}
