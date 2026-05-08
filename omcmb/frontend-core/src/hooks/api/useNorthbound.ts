import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  northboundApi,
  type AddPushTargetRequest,
  type ExportPMRequest,
  type ExportAlarmRequest,
  type NorthboundServer,
  type NorthboundServerRole,
} from '../../services/api/northboundApi';

export function usePushTargets() {
  return useQuery({
    queryKey: ['northbound', 'push-targets'],
    queryFn: () => northboundApi.getPushTargets(),
    staleTime: 30 * 1000,
  });
}

export function useAddPushTarget() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: AddPushTargetRequest) => northboundApi.addPushTarget(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['northbound', 'push-targets'] });
    },
  });
}

export function useRemovePushTarget() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => northboundApi.removePushTarget(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['northbound', 'push-targets'] });
    },
  });
}

export function useFullSync() {
  return useMutation({
    mutationFn: (dataType: string) => northboundApi.fullSync(dataType),
  });
}

export function useIncrementalSync() {
  return useMutation({
    mutationFn: ({ dataType, since }: { dataType: string; since: string }) =>
      northboundApi.incrementalSync(dataType, since),
  });
}

export function useExportPM() {
  return useMutation({
    mutationFn: (req: ExportPMRequest) => northboundApi.exportPM(req),
  });
}

export function useExportAlarms() {
  return useMutation({
    mutationFn: (req: ExportAlarmRequest) => northboundApi.exportAlarms(req),
  });
}

// --- 主备服务器（system/config 北向设置）---

const NORTHBOUND_SERVERS_KEY = ['northbound', 'servers'] as const;

/** 拉主备两组配置；30s staleTime（与现有 push-targets 对齐）。 */
export function useNorthboundServers() {
  return useQuery<NorthboundServer[]>({
    queryKey: [...NORTHBOUND_SERVERS_KEY],
    queryFn: () => northboundApi.getServers(),
    staleTime: 30 * 1000,
  });
}

/** 切换激活组 mutation；成功后 invalidate 列表自动 refetch 让 UI 更新高亮。 */
export function useSwitchActiveNorthboundServer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (role: NorthboundServerRole) => northboundApi.switchActiveServer(role),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [...NORTHBOUND_SERVERS_KEY] });
    },
  });
}
