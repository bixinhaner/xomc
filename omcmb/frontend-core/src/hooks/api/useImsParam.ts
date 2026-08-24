/**
 * useImsParam — React Query hooks for 核心网参数文件库（ims_param_files）。
 *
 * 后端 API：/api/v1/imsparam/*（docs/design/imscore-file-transfer.md）
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  imsParamApi,
  type ImsParamImportResult,
  type ImsParamListParams,
  type ImsParamTypeOption,
} from '../../services/api/imsParamApi';

/** 参数类型注册表（P1~P12）。任务创建抽屉 + 文件库上传共用。 */
export function useImsParamTypes() {
  return useQuery<ImsParamTypeOption[]>({
    queryKey: ['ims-param', 'types'],
    queryFn: () => imsParamApi.getParamTypes(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useImsParamFiles(params: ImsParamListParams, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['ims-param', 'files', params],
    queryFn: () => imsParamApi.list(params),
    enabled: options?.enabled ?? true,
  });
}

export function useImportImsParamFiles() {
  const qc = useQueryClient();
  return useMutation<
    ImsParamImportResult,
    Error,
    { paramType: string; files: File[]; description?: string }
  >({
    mutationFn: ({ paramType, files, description }) =>
      imsParamApi.import(paramType, files, description),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['ims-param', 'files'] });
    },
  });
}

export function useDeleteImsParamFile() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (id: string) => imsParamApi.delete(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['ims-param', 'files'] });
    },
  });
}

export function useBatchDeleteImsParamFiles() {
  const qc = useQueryClient();
  return useMutation<{ succeeded: string[]; failed: string[] }, Error, string[]>({
    mutationFn: (ids: string[]) => imsParamApi.batchDelete(ids),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['ims-param', 'files'] });
    },
  });
}
