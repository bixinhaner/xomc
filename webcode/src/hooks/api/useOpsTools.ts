import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { OpsTemplate, OpsCommandRecord, OpsTask } from '@/mock/data/opsTools';
import type { PageRequest } from '@/types/pagination';
import { opsToolsService } from '@/mock/services/opsToolsService';

export function useOpsTemplates(
  params: { category?: string; keyword?: string; targetDeviceType?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['opsTools', 'templates', params],
    queryFn: () => opsToolsService.getTemplates(params),
  });
}

export function useOpsTemplateById(id: string) {
  return useQuery({
    queryKey: ['opsTools', 'templates', 'detail', id],
    queryFn: () => opsToolsService.getTemplateById(id),
    enabled: Boolean(id),
  });
}

export function useCreateOpsTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<OpsTemplate, 'id' | 'createTime' | 'updateTime' | 'useCount'>) =>
      opsToolsService.createTemplate(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'templates'] });
    },
  });
}

export function useUpdateOpsTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<OpsTemplate> }) =>
      opsToolsService.updateTemplate(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'templates'] });
    },
  });
}

export function useDeleteOpsTemplates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => opsToolsService.deleteTemplates(ids),
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
    queryFn: () => opsToolsService.getCommandRecords(params),
  });
}

export function useAddOpsCommandRecord() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<OpsCommandRecord, 'id'>) =>
      opsToolsService.addCommandRecord(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'commands'] });
    },
  });
}

export function useOpsTasks(
  params: { status?: string; templateId?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['opsTools', 'tasks', params],
    queryFn: () => opsToolsService.getTasks(params),
  });
}

export function useOpsTaskById(id: string) {
  return useQuery({
    queryKey: ['opsTools', 'tasks', 'detail', id],
    queryFn: () => opsToolsService.getTaskById(id),
    enabled: Boolean(id),
    refetchInterval: 5000,
  });
}

export function useCreateOpsTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<OpsTask, 'id' | 'status' | 'currentStep' | 'progress' | 'successCount' | 'failCount' | 'createdAt'>) =>
      opsToolsService.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
    },
  });
}

export function useCancelOpsTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => opsToolsService.cancelTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
    },
  });
}

export function usePauseOpsTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => opsToolsService.pauseTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
    },
  });
}

export function useResumeOpsTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => opsToolsService.resumeTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
    },
  });
}
