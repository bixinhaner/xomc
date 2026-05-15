import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { OpsTemplate, OpsCommandRecord, OpsTask } from '../../mock/data/opsTools';
import type { PageRequest } from '../../types/pagination';
import { opsToolsService } from '../../mock/services/opsToolsService';
import { opsToolsApi } from '../../services/api/opsToolsApi';
import { useMock } from '../../services/apiSwitch';

type QueryMountOptions = {
  refetchOnMount?: boolean | 'always';
};

export function useOpsTemplates(
  params: { category?: string; keyword?: string; targetDeviceType?: string } & PageRequest,
  options?: QueryMountOptions,
) {
  return useQuery({
    queryKey: ['opsTools', 'templates', params],
    queryFn: () =>
      useMock ? opsToolsService.getTemplates(params) : opsToolsApi.getTemplates(params),
    refetchOnMount: options?.refetchOnMount,
  });
}

export function useOpsTemplateById(id: string) {
  return useQuery({
    queryKey: ['opsTools', 'templates', 'detail', id],
    queryFn: () =>
      useMock ? opsToolsService.getTemplateById(id) : opsToolsApi.getTemplateById(id),
    enabled: Boolean(id),
  });
}

export function useCreateOpsTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<OpsTemplate, 'id' | 'createTime' | 'updateTime' | 'useCount'>) =>
      useMock ? opsToolsService.createTemplate(data) : opsToolsApi.createTemplate(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'templates'] });
    },
  });
}

export function useUpdateOpsTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<OpsTemplate> }) =>
      useMock ? opsToolsService.updateTemplate(id, data) : opsToolsApi.updateTemplate(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'templates'] });
    },
  });
}

export function useDeleteOpsTemplates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      useMock ? opsToolsService.deleteTemplates(ids) : opsToolsApi.deleteTemplates(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'templates'] });
    },
  });
}

export function useOpsCommandRecords(
  params: { deviceSn?: string; operator?: string; success?: boolean } & PageRequest
) {
  return useQuery({
    queryKey: ['opsTools', 'commands', params],
    queryFn: () =>
      useMock ? opsToolsService.getCommandRecords(params) : opsToolsApi.getCommandRecords(params),
  });
}

export function useAddOpsCommandRecord() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<OpsCommandRecord, 'id'>) =>
      useMock ? opsToolsService.addCommandRecord(data) : opsToolsApi.addCommandRecord(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'commands'] });
    },
  });
}

export function useOpsTasks(
  params: { status?: string; templateId?: string; keyword?: string; creator?: string } & PageRequest,
  options?: QueryMountOptions,
) {
  return useQuery({
    queryKey: ['opsTools', 'tasks', params],
    queryFn: () =>
      useMock ? opsToolsService.getTasks(params) : opsToolsApi.getTasks(params),
    refetchOnMount: options?.refetchOnMount,
  });
}

export function useOpsTaskById(id: string) {
  return useQuery({
    queryKey: ['opsTools', 'tasks', 'detail', id],
    queryFn: () =>
      useMock ? opsToolsService.getTaskById(id) : opsToolsApi.getTaskById(id),
    enabled: Boolean(id),
    refetchInterval: 5000,
  });
}

export function useCreateOpsTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<OpsTask, 'id' | 'status' | 'currentStep' | 'progress' | 'successCount' | 'failCount' | 'createdAt'>) =>
      useMock ? opsToolsService.createTask(data) : opsToolsApi.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
    },
  });
}

export function useCancelOpsTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? opsToolsService.cancelTask(id) : opsToolsApi.cancelTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
    },
  });
}

export function usePauseOpsTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? opsToolsService.pauseTask(id) : opsToolsApi.pauseTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
    },
  });
}

export function useResumeOpsTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      useMock ? opsToolsService.resumeTask(id) : opsToolsApi.resumeTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
    },
  });
}
