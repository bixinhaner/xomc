import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '../../mock/services/dashboardService';
import { dashboardApi } from '../../services/api/dashboardApi';
import { useMock } from '../../services/apiSwitch';
import type { DashboardSummary, DashboardChartData } from '../../mock/data/dashboard';

const api = useMock ? dashboardService : dashboardApi;

export interface DashboardDataResponse {
  summary: DashboardSummary;
  chartData: DashboardChartData;
  widgets: Record<string, unknown>;
}

export function useDashboardData() {
  return useQuery<DashboardDataResponse>({
    queryKey: ['dashboard', 'all'],
    queryFn: () => api.getDashboardData() as Promise<DashboardDataResponse>,
    refetchInterval: 30000,
  });
}

export function useDashboardSummary() {
  return useQuery({
    queryKey: ['dashboard', 'summary'],
    queryFn: () => api.getSummary(),
    refetchInterval: 30000,
  });
}

export function useDashboardChartData() {
  return useQuery({
    queryKey: ['dashboard', 'charts'],
    queryFn: () => api.getChartData(),
    refetchInterval: 60000,
  });
}

export function useAlarmTrend(days = 7) {
  return useQuery({
    queryKey: ['dashboard', 'alarm-trend', days],
    queryFn: () => api.getAlarmTrend(days),
    refetchInterval: 60000,
  });
}

export function useDeviceStatusPie() {
  return useQuery({
    queryKey: ['dashboard', 'device-status'],
    queryFn: () => api.getDeviceStatusPie(),
    refetchInterval: 30000,
  });
}

export function useTopAlarmDevices() {
  return useQuery({
    queryKey: ['dashboard', 'top-alarm-devices'],
    queryFn: () => api.getTopAlarmDevices(),
    refetchInterval: 60000,
  });
}

export function useKPITrend(kpiCode: string) {
  return useQuery({
    queryKey: ['dashboard', 'kpi-trend', kpiCode],
    queryFn: () => api.getKPITrend(kpiCode),
    enabled: Boolean(kpiCode),
    refetchInterval: 60000,
  });
}

export function useRegionStats() {
  return useQuery({
    queryKey: ['dashboard', 'region-stats'],
    queryFn: () => api.getRegionStats(),
    refetchInterval: 60000,
  });
}
