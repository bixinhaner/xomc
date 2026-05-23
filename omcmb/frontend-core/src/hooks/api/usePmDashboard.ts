/**
 * T-0164-P6 / G6 PM 性能查看仪表盘 React Query hooks。
 *
 * 与现有 useDashboard（运营总览 deviceStats/alarmStats）不同；这里专注用户可定制 PM 仪表盘。
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createApiSwitch } from '../../services/apiSwitch';
import { pmDashboardApi, pmDashboardMock } from '../../services/api/pmDashboardApi';
import type {
  CreateDashboardInput,
  UpdateDashboardInput,
  CreatePanelInput,
  ForkInput,
  ShareInput,
} from '../../types/pmDashboard';

const api = createApiSwitch(pmDashboardMock, pmDashboardApi);

const PM_DASH_KEY = ['pm-dashboards'] as const;
const PM_PREFS_KEY = ['pm-user-preferences'] as const;

// ── Dashboard hooks ────────────────────────────────────────────────────

export function usePmDashboardList() {
  return useQuery({
    queryKey: [...PM_DASH_KEY, 'list'],
    queryFn: () => api.list(),
  });
}

export function usePmDashboardDetail(id: string | undefined) {
  return useQuery({
    queryKey: [...PM_DASH_KEY, 'detail', id],
    queryFn: () => api.get(id as string),
    enabled: Boolean(id),
  });
}

export function useCreatePmDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateDashboardInput) => api.create(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_DASH_KEY });
    },
  });
}

export function useUpdatePmDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateDashboardInput }) =>
      api.update(id, input),
    onSuccess: (_, vars) => {
      void qc.invalidateQueries({ queryKey: PM_DASH_KEY });
      void qc.invalidateQueries({ queryKey: [...PM_DASH_KEY, 'detail', vars.id] });
    },
  });
}

export function useDeletePmDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_DASH_KEY });
    },
  });
}

export function useForkPmDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ForkInput) => api.fork(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_DASH_KEY });
    },
  });
}

export function useSharePmDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ShareInput) => api.share(input),
    onSuccess: (_, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_DASH_KEY, 'detail', vars.dashboardId] });
    },
  });
}

export function useUnsharePmDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ dashboardId, userId }: { dashboardId: string; userId: string }) =>
      api.unshare(dashboardId, userId),
    onSuccess: (_, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_DASH_KEY, 'detail', vars.dashboardId] });
    },
  });
}

// ── Panel hooks ────────────────────────────────────────────────────────

export function useCreatePmPanel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreatePanelInput) => api.createPanel(input),
    onSuccess: (_, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_DASH_KEY, 'detail', vars.dashboardId] });
    },
  });
}

export function useUpdatePmPanel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreatePanelInput & { panelId: string }) => api.updatePanel(input),
    onSuccess: (_, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_DASH_KEY, 'detail', vars.dashboardId] });
    },
  });
}

export function useDeletePmPanel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ dashboardId, panelId }: { dashboardId: string; panelId: string }) =>
      api.deletePanel(dashboardId, panelId),
    onSuccess: (_, vars) => {
      void qc.invalidateQueries({ queryKey: [...PM_DASH_KEY, 'detail', vars.dashboardId] });
    },
  });
}

// ── UserPreferences hooks ──────────────────────────────────────────────

export function usePmUserPreferences() {
  return useQuery({
    queryKey: PM_PREFS_KEY,
    queryFn: () => api.getUserPrefs(),
  });
}

export function useUpdatePmUserPreferences() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (layout: Record<string, unknown>) => api.setUserPrefs(layout),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: PM_PREFS_KEY });
    },
  });
}
