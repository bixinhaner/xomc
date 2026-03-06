import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { MMLScript, MMLTask } from '@/types/mml';
import type { PageRequest } from '@/types/pagination';
import { mmlService } from '@/mock/services/mmlService';

export function useMMLCommands(params: { keyword?: string; category?: string } & PageRequest) {
  return useQuery({
    queryKey: ['mml', 'commands', params],
    queryFn: () => mmlService.getCommands(params),
    staleTime: 10 * 60 * 1000,
  });
}

export function useAllMMLCommands() {
  return useQuery({
    queryKey: ['mml', 'commands', 'all'],
    queryFn: () => mmlService.getAllCommands(),
    staleTime: 10 * 60 * 1000,
  });
}

export function useMMLScripts(params: PageRequest) {
  return useQuery({
    queryKey: ['mml', 'scripts', params],
    queryFn: () => mmlService.getScripts(params),
  });
}

export function useMMLScriptById(id: string) {
  return useQuery({
    queryKey: ['mml', 'scripts', 'detail', id],
    queryFn: () => mmlService.getScriptById(id),
    enabled: Boolean(id),
  });
}

export function useMMLTasks(params: PageRequest) {
  return useQuery({
    queryKey: ['mml', 'tasks', params],
    queryFn: () => mmlService.getTasks(params),
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
    }) => mmlService.executeCommand(commandCode, deviceSns, params),
  });
}

export function useCreateMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<MMLScript, 'id' | 'createTime' | 'updateTime'>) =>
      mmlService.createScript(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function useUpdateMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<MMLScript> }) =>
      mmlService.updateScript(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function useDeleteMMLScripts() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => mmlService.deleteScripts(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function useCreateMMLTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<MMLTask, 'id' | 'status' | 'results' | 'createdAt' | 'updatedAt'>) =>
      mmlService.createTask(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] });
    },
  });
}

export function useExecuteMMLScript() {
  return useMutation({
    mutationFn: ({ scriptId, deviceSns }: { scriptId: string; deviceSns: string[] }) =>
      mmlService.executeScript(scriptId, deviceSns),
  });
}
