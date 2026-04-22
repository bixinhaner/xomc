import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { deviceRulesApi } from '../../services/api/deviceRulesApi';
import type {
  RuleListRequest,
  CreateRuleRequest,
  UpdateRuleRequest,
  BatchSortItem,
} from '../../services/api/deviceRulesApi';

// Query keys
const ruleKeys = {
  all: ['device-rules'] as const,
  lists: () => [...ruleKeys.all, 'list'] as const,
  list: (params: RuleListRequest) => [...ruleKeys.lists(), params] as const,
  details: () => [...ruleKeys.all, 'detail'] as const,
  detail: (id: string) => [...ruleKeys.details(), id] as const,
  tasks: (ruleId: string) => [...ruleKeys.all, 'tasks', ruleId] as const,
  task: (ruleId: string, taskId: string) => [...ruleKeys.all, 'task', ruleId, taskId] as const,
  nextPriority: () => [...ruleKeys.all, 'next-priority'] as const,
};

/**
 * 获取规则列表
 */
export function useRules(params: RuleListRequest) {
  return useQuery({
    queryKey: ruleKeys.list(params),
    queryFn: () => deviceRulesApi.list(params),
    staleTime: 30 * 1000,
  });
}

/**
 * 获取规则详情
 */
export function useRule(id: string) {
  return useQuery({
    queryKey: ruleKeys.detail(id),
    queryFn: () => deviceRulesApi.get(id),
    enabled: Boolean(id),
    staleTime: 30 * 1000,
  });
}

/**
 * 获取下一个可用优先级
 */
export function useNextPriority() {
  return useQuery({
    queryKey: ruleKeys.nextPriority(),
    queryFn: () => deviceRulesApi.getNextPriority(),
    staleTime: 60 * 1000,
  });
}

/**
 * 获取规则任务列表
 */
export function useRuleTasks(ruleId: string, limit?: number) {
  return useQuery({
    queryKey: ruleKeys.tasks(ruleId),
    queryFn: () => deviceRulesApi.listTasks(ruleId, limit),
    enabled: Boolean(ruleId),
    staleTime: 10 * 1000,
  });
}

/**
 * 获取任务详情
 */
export function useRuleTask(ruleId: string, taskId: string) {
  return useQuery({
    queryKey: ruleKeys.task(ruleId, taskId),
    queryFn: () => deviceRulesApi.getTask(ruleId, taskId),
    enabled: Boolean(ruleId) && Boolean(taskId),
    staleTime: 10 * 1000,
  });
}

/**
 * 创建规则
 */
export function useCreateRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateRuleRequest) => deviceRulesApi.create(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ruleKeys.lists() });
    },
  });
}

/**
 * 更新规则
 */
export function useUpdateRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, req }: { id: string; req: UpdateRuleRequest }) =>
      deviceRulesApi.update(id, req),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ruleKeys.lists() });
      void queryClient.invalidateQueries({ queryKey: ruleKeys.detail(variables.id) });
    },
  });
}

/**
 * 删除规则
 */
export function useDeleteRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deviceRulesApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ruleKeys.lists() });
    },
  });
}

/**
 * 切换规则启用状态
 */
export function useToggleRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      deviceRulesApi.toggle(id, enabled),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ruleKeys.lists() });
      void queryClient.invalidateQueries({ queryKey: ruleKeys.detail(variables.id) });
    },
  });
}

/**
 * 批量排序规则
 */
export function useBatchSortRules() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (items: BatchSortItem[]) => deviceRulesApi.batchSort(items),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ruleKeys.lists() });
    },
  });
}

/**
 * 应用规则
 */
export function useApplyRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, dryRun }: { id: string; dryRun?: boolean }) =>
      deviceRulesApi.apply(id, { dryRun }),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ruleKeys.tasks(variables.id) });
    },
  });
}
