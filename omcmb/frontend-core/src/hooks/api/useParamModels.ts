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

export function useParamModelCacheRefresh() {
  return useMutation({
    mutationFn: () => api.cacheRefresh(),
  });
}

export function useParamModelImportDirectory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.importDirectory(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_KEY });
    },
  });
}

// 2026-05-28: destructive 全量重载;成功后 invalidate 所有 param-model 查询,
// 让前端立刻看到孤儿模型被删除后的最新清单。
export function useParamModelReloadDirectory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.reloadDirectory(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_KEY });
    },
  });
}

// T-0178: 上传自定义 paramModel XML。force=true 同名强制覆盖(后端备份 .bak.<ts>),
// 默认 false 同名返 409,前端可弹 Modal 二次确认后重试 force=true。
// 成功后 invalidate 所有 param-model 查询,新模型立即出现在列表。
export function useUploadParamModelXML() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ file, force }: { file: File; force?: boolean }) =>
      api.uploadXML(file, force ?? false),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_KEY });
    },
  });
}
