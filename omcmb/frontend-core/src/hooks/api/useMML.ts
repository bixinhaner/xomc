import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  DeviceResultStatus,
  DeviceTaskResultItem,
  MMLScript,
  MMLCustomCommand,
  MMLImportedScriptCreateInput,
  MMLImportedScriptReplaceInput,
  MMLScriptExecutionInput,
  MMLScriptImportValidation,
  MMLTask,
  MMLTaskResultsStats,
  MMLTaskStatus,
} from '../../types/mml';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { mmlService } from '../../mock/services/mmlService';
import { MMLScriptImportApiError, mmlApi } from '../../services/api/mmlApi';
import { createApiSwitch } from '../../services/apiSwitch';
import {
  MML_CUSTOM_COMMAND_PATHS_QUERY_KEY,
  MML_CUSTOM_COMMANDS_QUERY_KEY,
} from './mmlQueryKeys';
import { useAppStore } from '../../store/appStore';

const api = createApiSwitch(mmlService, mmlApi);

export const MML_TASK_LIST_ACTIVE_REFETCH_INTERVAL_MS = 3000;
export const MML_TASK_RESULTS_ACTIVE_REFETCH_INTERVAL_MS = 3000;
const MML_TASK_LIST_ACTIVE_STATUSES = new Set<MMLTaskStatus>(['pending', 'running', 'paused']);
const MML_TASK_RESULT_ACTIVE_STATUSES = new Set<DeviceResultStatus>(['pending', 'running']);

export function getMMLTasksRefetchInterval(data?: { items?: Array<Pick<MMLTask, 'status'>> }) {
  const hasActiveTask = data?.items?.some((task) => MML_TASK_LIST_ACTIVE_STATUSES.has(task.status)) ?? false;
  return hasActiveTask ? MML_TASK_LIST_ACTIVE_REFETCH_INTERVAL_MS : false;
}

export function getMMLTaskResultsRefetchInterval(
  data?: { items?: Array<Pick<DeviceTaskResultItem, 'status'>> },
  pollWhileTaskActive = false,
) {
  const hasActiveResult = data?.items?.some((row) => (
    row.status ? MML_TASK_RESULT_ACTIVE_STATUSES.has(row.status) : false
  )) ?? false;
  return pollWhileTaskActive || hasActiveResult ? MML_TASK_RESULTS_ACTIVE_REFETCH_INTERVAL_MS : false;
}

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

export function useMMLScripts(
  params: PageRequest & { search?: string; deviceType?: string; creator?: string }
) {
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

export function useMMLTasks(
  params: PageRequest & { status?: string; executeType?: string; result?: string; taskName?: string; scriptName?: string; taskOrigin?: string }
) {
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: ['mml', 'tasks', params, locale],
    queryFn: () => api.getTasks(params),
    refetchInterval: (query) => getMMLTasksRefetchInterval(query.state.data),
    refetchIntervalInBackground: false,
  });
}

export function useMMLTaskById(id: string) {
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: ['mml', 'tasks', 'detail', id, locale],
    queryFn: () => api.getTaskById(id),
    enabled: Boolean(id),
  });
}

// P4 C11：脚本关联的历史执行列表（模板 + periodic 子实例）。
// 仅 Real API 支持（mock 暂缺，调用时会直接走 mmlApi.getScriptRuns）。
export function useMMLScriptRuns(scriptId: string, params: PageRequest) {
  return useQuery({
    queryKey: ['mml', 'scripts', 'runs', scriptId, params],
    queryFn: () => api.getScriptRuns(scriptId, params),
    enabled: Boolean(scriptId),
  });
}

export function useExecuteMMLCommand() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      commandCode,
      deviceSns,
      params,
      payload,
    }: {
      commandCode?: string;
      deviceSns?: string[];
      params?: Record<string, unknown>;
      payload?: Record<string, unknown>;
    }) => (payload ? api.executeCommand(payload) : api.executeCommand(commandCode!, deviceSns!, params)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] }),
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

/** Validate a TXT file against the server-authoritative import parser. */
export function useValidateMMLScriptImport() {
  return useMutation<MMLScriptImportValidation, MMLScriptImportApiError, File>({
    mutationFn: (file: File) => mmlApi.validateScriptImport(file),
  });
}

/** Save only a reviewed validation token and script metadata. */
export function useCreateImportedMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: MMLImportedScriptCreateInput) => mmlApi.createImportedScript(input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] }),
  });
}

/** Replace a script from a new reviewed TXT snapshot. */
export function useReplaceImportedMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: MMLImportedScriptReplaceInput }) =>
      mmlApi.replaceImportedScript(id, input),
    onSuccess: (_, vars) => Promise.all([
      queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] }),
      queryClient.invalidateQueries({ queryKey: ['mml', 'scripts', 'detail', vars.id] }),
    ]),
  });
}

/** Create an execution from a stored script snapshot, never from browser plan data. */
export function useCreateMMLScriptExecution() {
  const queryClient = useQueryClient();
  return useMutation<
    Awaited<ReturnType<typeof mmlApi.createScriptExecution>>,
    MMLScriptImportApiError,
    { id: string; input: MMLScriptExecutionInput }
  >({
    mutationFn: ({ id, input }: { id: string; input: MMLScriptExecutionInput }) =>
      mmlApi.createScriptExecution(id, input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['mml', 'tasks'] }),
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

export function useStartMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.startScript(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function usePauseMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.pauseScript(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function useCancelMMLScript() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cancelScript(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'scripts'] });
    },
  });
}

export function useCreateMMLTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof api.createTask>[0]) =>
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

export function useStartMMLTasks() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.startTasks(ids),
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

export function useCancelMMLTasks() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.cancelTasks(ids),
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

export function useDeleteMMLTasks() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => api.deleteTasks(ids),
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

export function useMMLTaskResults(
  taskId: string | null,
  page = 1,
  pageSize = 50,
  options?: { pollWhileTaskActive?: boolean },
) {
  const locale = useAppStore((s) => s.locale);
  return useQuery({
    queryKey: ['mml', 'tasks', taskId, 'results', page, pageSize, locale],
    queryFn: () => api.getTaskResults(taskId!, page, pageSize),
    enabled: Boolean(taskId),
    refetchInterval: (query) => getMMLTaskResultsRefetchInterval(
      query.state.data,
      options?.pollWhileTaskActive ?? false,
    ),
    refetchIntervalInBackground: false,
  });
}

export function getMMLTaskResultsPage(
  taskId: string,
  page: number,
  pageSize: number,
): Promise<PageResponse<DeviceTaskResultItem, MMLTaskResultsStats>> {
  return api.getTaskResults(taskId, page, pageSize);
}

// --- Template hooks ---

export function useMMLTemplates(
  params?: { commandCode?: string; operationType?: string; templateScope?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['mml', 'templates', params],
    queryFn: () => api.getTemplates(params),
  });
}

export function useCreateMMLTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'>) =>
      api.createTemplate(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'templates'] });
      void queryClient.invalidateQueries({ queryKey: MML_CUSTOM_COMMANDS_QUERY_KEY });
      void queryClient.invalidateQueries({ queryKey: MML_CUSTOM_COMMAND_PATHS_QUERY_KEY });
    },
  });
}

export function useUpdateMMLTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<MMLCustomCommand> }) =>
      api.updateTemplate(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'templates'] });
      void queryClient.invalidateQueries({ queryKey: MML_CUSTOM_COMMANDS_QUERY_KEY });
      void queryClient.invalidateQueries({ queryKey: MML_CUSTOM_COMMAND_PATHS_QUERY_KEY });
    },
  });
}

export function useDeleteMMLTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteTemplate(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'templates'] });
      void queryClient.invalidateQueries({ queryKey: MML_CUSTOM_COMMANDS_QUERY_KEY });
      void queryClient.invalidateQueries({ queryKey: MML_CUSTOM_COMMAND_PATHS_QUERY_KEY });
    },
  });
}

export function useCloneMMLTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cloneTemplate(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['mml', 'templates'] });
      void queryClient.invalidateQueries({ queryKey: MML_CUSTOM_COMMANDS_QUERY_KEY });
      void queryClient.invalidateQueries({ queryKey: MML_CUSTOM_COMMAND_PATHS_QUERY_KEY });
    },
  });
}

// --- Dangerous command check hook ---

export function useDangerousCheck(commandCode: string) {
  return useQuery({
    queryKey: ['mml', 'dangerous-check', commandCode],
    queryFn: () => api.checkDangerous(commandCode),
    enabled: Boolean(commandCode),
    staleTime: 5 * 60 * 1000,
  });
}
