import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '@/mock/services/dashboardService';

export function useDashboardData() {
  return useQuery({
    queryKey: ['dashboard', 'all'],
    queryFn: () => dashboardService.getDashboardData(),
    refetchInterval: 30000,
  });
}

export function useDashboardSummary() {
  return useQuery({
    queryKey: ['dashboard', 'summary'],
    queryFn: () => dashboardService.getSummary(),
    refetchInterval: 30000,
  });
}

export function useDashboardChartData() {
  return useQuery({
    queryKey: ['dashboard', 'charts'],
    queryFn: () => dashboardService.getChartData(),
    refetchInterval: 60000,
  });
}

export function useAlarmTrend(days = 7) {
  return useQuery({
    queryKey: ['dashboard', 'alarm-trend', days],
    queryFn: () => dashboardService.getAlarmTrend(days),
    refetchInterval: 60000,
  });
}

export function useDeviceStatusPie() {
  return useQuery({
    queryKey: ['dashboard', 'device-status'],
    queryFn: () => dashboardService.getDeviceStatusPie(),
    refetchInterval: 30000,
  });
}

export function useTopAlarmDevices() {
  return useQuery({
    queryKey: ['dashboard', 'top-alarm-devices'],
    queryFn: () => dashboardService.getTopAlarmDevices(),
    refetchInterval: 60000,
  });
}

export function useKPITrend(kpiCode: string) {
  return useQuery({
    queryKey: ['dashboard', 'kpi-trend', kpiCode],
    queryFn: () => dashboardService.getKPITrend(kpiCode),
    enabled: Boolean(kpiCode),
    refetchInterval: 60000,
  });
}

export function useRegionStats() {
  return useQuery({
    queryKey: ['dashboard', 'region-stats'],
    queryFn: () => dashboardService.getRegionStats(),
    refetchInterval: 60000,
  });
}
