/**
 * useConfigSnapshot (T-0164 / F1) — React Query hooks for config_snapshots.
 *
 * 设计文档：docs/project/config-snapshot-table-plan-20260522.md
 *
 * 后端 API：/api/v1/backup/config-snapshots/*
 * 配套 Mock：暂未提供 mock service —— Mock 模式下这些 hook 走 real API（与后端一起跑）；
 * 后期如需独立 mock 可在 mock/services 下补一个，配合 createApiSwitch 接入。
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  configSnapshotApi,
  type ConfigSnapshot,
  type SnapshotListParams,
  type SnapshotImportResult,
  type BatchGetSnapshotsResult,
} from '../../services/api/configSnapshotApi';
import { backupApi } from '../../services/api/backupApi';
import type { RestoreTask } from '../../mock/data/backup';

export function useConfigSnapshots(params: SnapshotListParams) {
  return useQuery({
    queryKey: ['config-snapshots', 'list', params],
    queryFn: () => configSnapshotApi.list(params),
  });
}

export function useConfigSnapshot(sn: string | undefined) {
  return useQuery({
    queryKey: ['config-snapshots', 'detail', sn],
    queryFn: () => configSnapshotApi.getBySerialNumber(sn!),
    enabled: Boolean(sn),
  });
}

/**
 * 批量查询；常用于恢复创建页"按设备快照" Tab —— 用户选完 SN 列表后，
 * 调一次拿到所有快照 + 缺失 SN。
 */
export function useBatchGetSnapshots(serialNumbers: string[]) {
  return useQuery<BatchGetSnapshotsResult>({
    queryKey: ['config-snapshots', 'batch-get', [...serialNumbers].sort()],
    queryFn: () => configSnapshotApi.batchGet(serialNumbers),
    enabled: serialNumbers.length > 0,
  });
}

export function useImportConfigSnapshots() {
  const qc = useQueryClient();
  return useMutation<SnapshotImportResult, Error, File[]>({
    mutationFn: (files: File[]) => configSnapshotApi.import(files),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['config-snapshots'] });
    },
  });
}

export function useDeleteConfigSnapshot() {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (sn: string) => configSnapshotApi.delete(sn),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['config-snapshots'] });
    },
  });
}

export function useBatchDeleteConfigSnapshots() {
  const qc = useQueryClient();
  return useMutation<
    { succeeded: string[]; failed: string[] },
    Error,
    string[]
  >({
    mutationFn: (sns: string[]) => configSnapshotApi.batchDelete(sns),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['config-snapshots'] });
    },
  });
}

/**
 * useCreateRestoreBySnapshot — T-0164 B5 入口。
 *
 * 整批拒绝场景下后端返 4xx，axios 抛错；调用方应通过 onError 读取
 * `error.response?.data?.missing` 展示缺失 SN 列表。
 */
export function useCreateRestoreBySnapshot() {
  const qc = useQueryClient();
  return useMutation<
    { task: RestoreTask | null; missing: string[] },
    Error,
    { targetDeviceSns: string[] }
  >({
    mutationFn: (req) => backupApi.createRestoreBySnapshot(req),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['backup', 'restore-tasks'] });
    },
  });
}
