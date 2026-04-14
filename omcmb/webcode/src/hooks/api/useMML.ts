import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { MMLScript, MMLTask } from '@/types/mml';
import type { PageRequest } from '@/types/pagination';
import { mmlService } from '@/mock/services/mmlService';
import { mmlApi } from '@/services/api/mmlApi';
import { createApiSwitch } from '@/services/apiSwitch';

const api = createApiSwitch(mmlService, mmlApi);

export function useMMLCommands(params: { keyword?: string; category?: string } & PageRequest) {
  return useQuery({
    queryKey: ['mml', 'commands', params],
    queryFn: () => api.getCommands(params),
    staleTime: 10 * 60 * 1000,
  });
}

export function useAllMMLCommands() {
  return useQuery({
    queryKey: ['mml', 'commands', 'all'],
    queryFn: () => api.getAllCommands(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useMMLScripts(params: PageRequest) {
  return useQuery({
    queryKey: ['mml', 'scripts', params],
    queryFn: () => api.getScripts(params),
  });
}

export function useMMLScriptById(id: string) {
  return useQuery({
    queryKey: ['mml', 'scripts', 'detail', id],
    queryFn: () => api.getScriptById(id),
    enabled: Boolean(id),
  });
}

export function useMMLTasks(params: PageRequest) {
  return useQuery({
    queryKey: ['mml', 'tasks', params],
    queryFn: () => api.getTasks(params),
  });
}

export function useMMLTaskById(id: string) {
  return useQuery({
    queryKey: ['mml', 'tasks', 'detail', id],
    queryFn: () => api.getTaskById(id),
    enabled: Boolean(id),
  });
}

export function useExecuteMMLCommand() {
  return useMutation({
    mutationFn: ({
      commandCode,
      deviceSns,
      params,
    }: {
      commandCode: string;
      deviceSns: string[];
      params?: Record<string, string | number | boolean>;
    }) => api.executeCommand(commandCode, deviceSns, params),
  });
}

export function useCreateMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<MMLScript, 'id' | 'createTime' | 'updateTime'>) =>
      api.createScript(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function useUpdateMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<MMLScript> }) =>
      api.updateScript(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function useDeleteMMLScripts() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.deleteScripts(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function useCreateMMLTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<MMLTask, 'id' | 'status' | 'results' | 'createdAt' | 'updatedAt'>) =>
      api.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] });
    },
  });
}

export function useExecuteMMLScript() {
  return useMutation({
    mutationFn: ({ scriptId, deviceSns }: { scriptId: string; deviceSns: string[] }) =>
      api.executeScript(scriptId, deviceSns),
  });
}

// --- Task control hooks ---

export function useStartMMLTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.startTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] });
    },
  });
}

export function usePauseMMLTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.pauseTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] });
    },
  });
}

export function useCancelMMLTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cancelTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] });
    },
  });
}

export function useDeleteMMLTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] });
    },
  });
}

// --- Task polling hook ---

const TERMINAL_STATES: string[] = ['completed', 'failed', 'cancelled'];

export function useMMLTaskPolling(taskId: string | null, enabled: boolean) {
  return useQuery({
    queryKey: ['mml', 'tasks', 'poll', taskId],
    queryFn: () => api.getTaskById(taskId!),
    enabled: Boolean(taskId) && enabled,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data && TERMINAL_STATES.includes(data.status)) return false;
      return 2000;
    },
  });
}
