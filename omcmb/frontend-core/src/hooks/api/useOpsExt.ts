// F06 运维管理扩展 React Query Hooks（包装 opsExtApi）

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { PageRequest } from '../../types/pagination';
import {
  opsExtApi,
  type DiagnosticStatus,
  type DownloadStatus,
  type MaintenanceWindowStatus,
  type RiskLevel,
  type OpsPlaybook,
} from '../../services/api/opsExtApi';

// ---- Diagnostics ----

export function useOpsDiagnostics(
  params: { deviceSn?: string; diagType?: string; status?: DiagnosticStatus } & PageRequest,
) {
  return useQuery({
    queryKey: ['ops-ext', 'diagnostics', 'list', params],
    queryFn: () => opsExtApi.listDiagnostics(params),
  });
}

export function useOpsDiagnostic(id: string | undefined) {
  return useQuery({
    queryKey: ['ops-ext', 'diagnostics', 'detail', id],
    queryFn: () => opsExtApi.getDiagnostic(id!),
    enabled: Boolean(id),
  });
}

export function useDiagnosticPing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: opsExtApi.ping,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'diagnostics'] }),
  });
}

export function useDiagnosticTraceroute() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: opsExtApi.traceroute,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'diagnostics'] }),
  });
}

export function useDiagnosticThroughput() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: opsExtApi.throughput,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'diagnostics'] }),
  });
}

export function useRunInspection() {
  return useMutation({
    mutationFn: (scope: string = 'all') => opsExtApi.runInspection(scope),
  });
}

// ---- Downloads ----

export function useOpsDownloads(
  params: { deviceSn?: string; contentType?: string; status?: DownloadStatus } & PageRequest,
) {
  return useQuery({
    queryKey: ['ops-ext', 'downloads', 'list', params],
    queryFn: () => opsExtApi.listDownloads(params),
  });
}

export function useOpsDownload(id: string | undefined) {
  return useQuery({
    queryKey: ['ops-ext', 'downloads', 'detail', id],
    queryFn: () => opsExtApi.getDownload(id!),
    enabled: Boolean(id),
  });
}

export function useCollectDownload() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: opsExtApi.collectDownload,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'downloads'] }),
  });
}

// ---- Audit Logs ----

export function useOpsAuditLogs(
  params: {
    opType?: string;
    targetType?: string;
    targetId?: string;
    operatorUserId?: string;
    riskLevel?: RiskLevel;
    breakGlass?: boolean;
    from?: string;
    to?: string;
  } & PageRequest,
) {
  return useQuery({
    queryKey: ['ops-ext', 'audit-logs', params],
    queryFn: () => opsExtApi.listAuditLogs(params),
  });
}

// ---- Maintenance Windows ----

export function useMaintenanceWindows(params: { status?: MaintenanceWindowStatus } & PageRequest) {
  return useQuery({
    queryKey: ['ops-ext', 'maintenance-windows', 'list', params],
    queryFn: () => opsExtApi.listMaintenanceWindows(params),
  });
}

export function useActiveMaintenanceWindows() {
  return useQuery({
    queryKey: ['ops-ext', 'maintenance-windows', 'active'],
    queryFn: () => opsExtApi.listActiveMaintenanceWindows(),
    refetchInterval: 60_000, // 每分钟刷一次
  });
}

export function useCreateMaintenanceWindow() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: opsExtApi.createMaintenanceWindow,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'maintenance-windows'] }),
  });
}

export function useApproveMaintenanceWindow() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => opsExtApi.approveMaintenanceWindow(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'maintenance-windows'] }),
  });
}

// ---- Playbooks ----

export function usePlaybooks(params: { keyword?: string } & PageRequest) {
  return useQuery({
    queryKey: ['ops-ext', 'playbooks', 'list', params],
    queryFn: () => opsExtApi.listPlaybooks(params),
  });
}

export function useCreatePlaybook() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (p: Partial<OpsPlaybook>) => opsExtApi.createPlaybook(p),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'playbooks'] }),
  });
}

export function useMatchPlaybook() {
  return useMutation({
    mutationFn: opsExtApi.matchPlaybook,
  });
}

// ---- Task executor ----

export function useRunOpsTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => opsExtApi.runTask(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ['opsTools', 'tasks'] });
      qc.invalidateQueries({ queryKey: ['ops-ext', 'task-executions', id] });
    },
  });
}

export function useTaskExecutions(taskId: string | undefined, params: { deviceSn?: string; status?: string } & PageRequest) {
  return useQuery({
    queryKey: ['ops-ext', 'task-executions', taskId, params],
    queryFn: () => opsExtApi.listTaskExecutions(taskId!, params),
    enabled: Boolean(taskId),
  });
}

export function useApproveTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, approve, reason }: { id: string; approve: boolean; reason?: string }) =>
      opsExtApi.approveTask(id, { approve, reason }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['opsTools', 'tasks'] }),
  });
}

// ---- Commands ----

export function useExecuteRPC() {
  return useMutation({
    mutationFn: opsExtApi.executeRPC,
  });
}

// ---- Break-glass ----

export function useBreakGlassStatus() {
  return useQuery({
    queryKey: ['ops-ext', 'break-glass', 'status'],
    queryFn: () => opsExtApi.getBreakGlassStatus(),
    refetchInterval: 30_000, // 30s 刷新（30 分钟自动失效）
  });
}

export function useActivateBreakGlass() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: opsExtApi.activateBreakGlass,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'break-glass'] }),
  });
}

export function useDeactivateBreakGlass() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => opsExtApi.deactivateBreakGlass(),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['ops-ext', 'break-glass'] }),
  });
}
