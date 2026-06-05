import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { indicatorLibraryApi } from '../../services/api/indicatorLibraryApi';
import { indicatorLibraryService } from '../../mock/services/indicatorLibraryService';
import { createApiSwitch } from '../../services/apiSwitch';
import type {
  DeviceType,
  IndicatorListFilter,
  CreateIndicatorInput,
  UpdateIndicatorInput,
  CreateGroupInput,
  UpdateGroupInput,
  EnabledIndicatorsRequest,
  TechLower,
} from '../../types/indicatorLibrary';

const api = createApiSwitch(indicatorLibraryService, indicatorLibraryApi);

const IL_KEY = ['indicator-library'] as const;

export function useIndicatorList(deviceType: DeviceType, filter?: IndicatorListFilter) {
  return useQuery({
    queryKey: [...IL_KEY, 'list', deviceType, filter ?? {}],
    queryFn: () => api.list(deviceType, filter),
  });
}

export function usePlatformList(deviceType: DeviceType) {
  return useQuery({
    queryKey: [...IL_KEY, 'platforms', deviceType],
    queryFn: () => api.listPlatforms(deviceType),
    staleTime: 5 * 60 * 1000,
  });
}

export function useIndicatorDetail(deviceType: DeviceType, id: string | undefined) {
  return useQuery({
    queryKey: [...IL_KEY, 'detail', deviceType, id],
    queryFn: () => api.get(deviceType, id as string),
    enabled: Boolean(id),
  });
}

export function useCreateIndicator() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceType, input }: { deviceType: DeviceType; input: CreateIndicatorInput }) =>
      api.create(deviceType, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'list', vars.deviceType] });
    },
  });
}

export function useUpdateIndicator() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      id,
      input,
    }: {
      deviceType: DeviceType;
      id: string;
      input: UpdateIndicatorInput;
    }) => api.update(deviceType, id, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'detail', vars.deviceType, vars.id] });
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'list', vars.deviceType] });
    },
  });
}

export function useDeleteIndicator() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceType, id }: { deviceType: DeviceType; id: string }) =>
      api.delete(deviceType, id),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'list', vars.deviceType] });
    },
  });
}

export function useFormulas(deviceType: DeviceType, indicatorId: string | undefined) {
  return useQuery({
    queryKey: [...IL_KEY, 'formulas', deviceType, indicatorId],
    queryFn: () => api.listFormulas(deviceType, indicatorId as string),
    enabled: Boolean(indicatorId),
  });
}

export function useUpsertFormula() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      indicatorId,
      platform,
      formula,
    }: {
      deviceType: DeviceType;
      indicatorId: string;
      platform: string;
      formula: string;
    }) => api.upsertFormula(deviceType, indicatorId, platform, formula),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({
        queryKey: [...IL_KEY, 'formulas', vars.deviceType, vars.indicatorId],
      });
    },
  });
}

export function useDeleteFormula() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      indicatorId,
      platform,
    }: {
      deviceType: DeviceType;
      indicatorId: string;
      platform: string;
    }) => api.deleteFormula(deviceType, indicatorId, platform),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({
        queryKey: [...IL_KEY, 'formulas', vars.deviceType, vars.indicatorId],
      });
    },
  });
}

/** 2026-05-29:platform 进入 queryKey,平台切换时自动重新拉数据;不传 platform
 *  时等价于全量(后端 OR 短路) — 列表态全量 + 详情态按平台都可以共用此 hook。 */
export function useIndicatorGroups(
  deviceType: DeviceType,
  operatorCode?: string,
  platform?: string,
) {
  return useQuery({
    queryKey: [...IL_KEY, 'groups', deviceType, operatorCode ?? '', platform ?? ''],
    queryFn: () => api.listGroups(deviceType, operatorCode, platform),
  });
}

export function useCreateGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceType, input }: { deviceType: DeviceType; input: CreateGroupInput }) =>
      api.createGroup(deviceType, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'groups', vars.deviceType] });
    },
  });
}

export function useUpdateGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceType,
      id,
      input,
    }: {
      deviceType: DeviceType;
      id: string;
      input: UpdateGroupInput;
    }) => api.updateGroup(deviceType, id, input),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'groups', vars.deviceType] });
    },
  });
}

export function useDeleteGroup() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceType, id }: { deviceType: DeviceType; id: string }) =>
      api.deleteGroup(deviceType, id),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'groups', vars.deviceType] });
    },
  });
}

export function useEnabledIndicators(deviceType: DeviceType, operatorCode: string) {
  return useQuery({
    queryKey: [...IL_KEY, 'enabled', deviceType, operatorCode],
    queryFn: () => api.listEnabled(deviceType, operatorCode),
    enabled: Boolean(operatorCode),
  });
}

export function useSetEnabledIndicators() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (request: EnabledIndicatorsRequest) => api.setEnabled(request),
    onSuccess: (_d, vars) => {
      void qc.invalidateQueries({
        queryKey: [...IL_KEY, 'enabled', vars.deviceType, vars.operatorCode],
      });
      void qc.invalidateQueries({ queryKey: [...IL_KEY, 'list', vars.deviceType] });
    },
  });
}

// T-0180 P4: drill-down 一级 SummaryTab 数据源
export function useIndicatorSummary() {
  return useQuery({
    queryKey: [...IL_KEY, 'summary'],
    queryFn: () => api.summary(),
  });
}

// T-0180 P4: "管理 XML 文件" Modal 数据源(per tech)
export function useIndicatorFiles(tech: TechLower | undefined) {
  return useQuery({
    queryKey: [...IL_KEY, 'files', tech],
    queryFn: () => api.listFiles(tech as TechLower),
    enabled: Boolean(tech),
  });
}

// 上传自定义 XML(三库 XML 导入重构:name 必填,去 force);成功后 invalidate summary + files + indicators
export function useIndicatorUploadXml() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ tech, file, name }: { tech: TechLower; file: File; name: string }) =>
      api.uploadXml(tech, file, name),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: IL_KEY });
    },
  });
}

// T-0180 P4: 删除自定义 XML 文件
export function useIndicatorDeleteFile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (loadedFrom: string) => api.deleteFile(loadedFrom),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: IL_KEY });
    },
  });
}

// 2026-06-02:按 (tech, platform) 编辑描述,成功后刷新 summary。
export function useUpdateIndicatorFileDescription() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { tech: TechLower; platform: string; description: string }) =>
      api.updateFileDescription(vars.tech, vars.platform, vars.description),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: IL_KEY });
    },
  });
}
