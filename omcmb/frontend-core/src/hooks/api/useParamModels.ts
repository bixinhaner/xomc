import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { paramModelApi } from '../../services/api/paramModelApi';
import { paramModelService } from '../../mock/services/paramModelService';
import { createApiSwitch } from '../../services/apiSwitch';
import type {
  CreateMappingInput,
  UpdateMappingInput,
  UpsertStandardInput,
  UpdateParamModelInput,
  StandardParamFilter,
  TranslateRequest,
} from '../../types/paramModel';

const api = createApiSwitch(paramModelService, paramModelApi);

const PM_KEY = ['param-models'] as const;

export function useParamModelList() {
  return useQuery({
    queryKey: [...PM_KEY, 'list'],
    queryFn: () => api.list(),
  });
}

export function useParamModelDetail(name: string | undefined) {
  return useQuery({
    queryKey: [...PM_KEY, 'detail', name],
    queryFn: () => api.get(name as string),
    enabled: Boolean(name),
  });
}

export function useUpdateParamModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ name, input }: { name: string; input: UpdateParamModelInput }) =>
      api.update(name, input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_KEY });
    },
  });
}

export function useDeleteParamModel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => api.delete(name),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_KEY });
    },
  });
}

export function useParamMappings(name: string | undefined) {
  return useQuery({
    queryKey: [...PM_KEY, 'mappings', name],
    queryFn: () => api.listMappings(name as string),
    enabled: Boolean(name),
  });
}

export function useCreateMapping() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ name, input }: { name: string; input: CreateMappingInput }) =>
      api.createMapping(name, input),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_KEY, 'mappings', vars.name] });
    },
  });
}

export function useUpdateMapping() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ name, id, input }: { name: string; id: string; input: UpdateMappingInput }) =>
      api.updateMapping(name, id, input),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_KEY, 'mappings', vars.name] });
    },
  });
}

export function useDeleteMapping() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ name, id }: { name: string; id: string }) => api.deleteMapping(name, id),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_KEY, 'mappings', vars.name] });
    },
  });
}

export function useStandardParams(filter?: StandardParamFilter) {
  return useQuery({
    queryKey: [...PM_KEY, 'standard', filter ?? {}],
    queryFn: () => api.listStandard(filter),
  });
}

export function useStandardParam(path: string | undefined) {
  return useQuery({
    queryKey: [...PM_KEY, 'standard-detail', path],
    queryFn: () => api.getStandard(path as string),
    enabled: Boolean(path),
  });
}

export function useUpsertStandard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ input, path }: { input: UpsertStandardInput; path?: string }) =>
      path ? api.updateStandard(path, input) : api.createStandard(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...PM_KEY, 'standard'] });
    },
  });
}

export function useDeleteStandard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (path: string) => api.deleteStandard(path),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: [...PM_KEY, 'standard'] });
    },
  });
}

export function useTranslate() {
  return useMutation({
    mutationFn: (request: TranslateRequest) => api.translate(request),
  });
}

export function useDiscovered(productId: string | undefined, swVersion?: string) {
  return useQuery({
    queryKey: [...PM_KEY, 'discovered', productId, swVersion ?? ''],
    queryFn: () => api.listDiscovered(productId as string, swVersion),
    enabled: Boolean(productId),
  });
}

export function useDiscoveredVersions(productId: string | undefined) {
  return useQuery({
    queryKey: [...PM_KEY, 'discovered-versions', productId],
    queryFn: () => api.listDiscoveredVersions(productId as string),
    enabled: Boolean(productId),
  });
}

export function useDeleteDiscovered() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ productId, swVersion }: { productId: string; swVersion?: string }) =>
      api.deleteDiscovered(productId, swVersion),
    onSuccess: (_data, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_KEY, 'discovered', vars.productId] });
      void qc.invalidateQueries({ queryKey: [...PM_KEY, 'discovered-versions', vars.productId] });
    },
  });
}

// 下载 XML 原文件(操作列下载图标,builtin / custom 均可)。
export function useDownloadParamModelXML() {
  return useMutation({
    mutationFn: ({ loadedFrom }: { loadedFrom: string }) => api.downloadXml(loadedFrom),
  });
}

// 上传自定义 paramModel XML(名称取自 XML paramModel 属性,2026-06-05 取消手填)。
// 重复允许覆盖:不带 force 重复返 409,前端二次确认后带 force=true 重试。
// 成功后 invalidate 所有 param-model 查询,新模型立即出现在列表。
export function useUploadParamModelXML() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ file, force }: { file: File; force?: boolean }) => api.uploadXML(file, force),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_KEY });
    },
  });
}
