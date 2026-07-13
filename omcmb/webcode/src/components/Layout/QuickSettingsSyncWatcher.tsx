import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { App } from 'antd';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import type { QuickSettingsSyncMonitor } from '@core/store/quickSettingsFeedbackStore';
import type { ParameterSyncStatus } from '@core/types/deviceParameter';
import { isCurrentSyncFailure, isCurrentSyncSuccess } from '@core/utils/quickSettingsSyncStatus';
import { useT } from '@/hooks/useT';

const congestionHintAfterMs = 15 * 1000;
const congestionPendingThreshold = 8;

// staleMonitorTimeoutMs：sync monitor 启动后超过该时长仍未拿到终态（既无 success 也无 failure）
// 视为悬挂状态自动回收并提示用户。P0 止血阶段收敛到 60s，避免长期 loading。
// 触发场景：API mutate 的 onSuccess/onError 因极端网络情况未回调，导致 monitor 长期占位。
const staleMonitorTimeoutMs = 60 * 1000;

function shouldHintCongestion(status: ParameterSyncStatus, sync: QuickSettingsSyncMonitor) {
  if (sync.congestionHinted) return false;
  if (status.status !== 'syncing') return false;
  if (status.pendingCommands < congestionPendingThreshold) return false;
  return Date.now() - sync.startedAt >= congestionHintAfterMs;
}

export default function QuickSettingsSyncWatcher() {
  const t = useT();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const quickSettingsSyncs = useQuickSettingsFeedbackStore((s) => s.quickSettingsSyncs);
  const pollingRef = useRef(false);

  useEffect(() => {
    let disposed = false;

    const poll = async () => {
      if (disposed) return;
      const entries = Object.entries(useQuickSettingsFeedbackStore.getState().quickSettingsSyncs);
      if (entries.length === 0) return;

      await Promise.all(entries.map(async ([deviceId, sync]) => {
        if (disposed) return;
        // 悬挂 monitor 自动回收：拒绝把过期 sync 当作仍在运行。
        if (Date.now() - sync.startedAt > staleMonitorTimeoutMs) {
          let timeoutMsg = t('device.detail.deviceFetchWaitingPersist');
          let timeoutAsError = false;
          try {
            const timeoutStatus = await deviceParameterApi.getSyncStatus(deviceId);
            queryClient.setQueryData(['devices', 'sync-status', deviceId], timeoutStatus);
            if (isCurrentSyncFailure(timeoutStatus, sync)) {
              timeoutMsg = timeoutStatus.lastParamSyncError || t('device.detail.deviceFetchFailed');
              timeoutAsError = true;
            } else if (timeoutStatus.status === 'syncing' || timeoutStatus.pendingCommands > 0) {
              timeoutMsg = t('device.detail.deviceFetchTimeoutQueue', { pending: timeoutStatus.pendingCommands });
            }
          } catch {
            // Keep default timeout message.
          }
          useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId);
          if (timeoutAsError) {
            message.error(timeoutMsg);
          } else {
            message.warning(timeoutMsg);
          }
          return;
        }
        try {
          const status = await deviceParameterApi.getSyncStatus(deviceId);
          queryClient.setQueryData(['devices', 'sync-status', deviceId], status);

          if (shouldHintCongestion(status, sync)) {
            useQuickSettingsFeedbackStore.getState().patchQuickSettingsSync(deviceId, { congestionHinted: true });
            message.warning(t('device.detail.deviceFetchQueueBusy', { pending: status.pendingCommands }));
          }

          const hasNewSuccess = isCurrentSyncSuccess(status, sync);
          const hasNewFailure = isCurrentSyncFailure(status, sync);
          if (status.status === 'syncing' && !hasNewSuccess && !hasNewFailure) return;
          if (!hasNewSuccess && !hasNewFailure) return;

          if (hasNewFailure && !hasNewSuccess) {
            useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId);
            message.error(status.lastParamSyncError || t('device.detail.deviceFetchFailed'));
            return;
          }

          if (hasNewSuccess) {
            useQuickSettingsFeedbackStore.getState().clearDraftsByDevice(deviceId);
            useQuickSettingsFeedbackStore.getState().bumpRefreshTick(deviceId);
            deviceParameterApi.invalidateParameterSchemaCache(deviceId);
            void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
            void queryClient.invalidateQueries({ queryKey: ['devices', 'detail-composite-v2', deviceId] });
            void queryClient.invalidateQueries({ queryKey: ['quicksettings', 'groups', deviceId] });
            void queryClient.invalidateQueries({ queryKey: ['devices', 'parameters', 'search', deviceId] });
            void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-schema', deviceId] });
          }

          useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId, hasNewSuccess && sync.targetCount > 0
            ? {
              targetCount: sync.targetCount,
              gpvTaskCount: sync.gpvTaskCount || status.lastSyncGpv?.taskCount || 0,
              completedAt: status.lastParamSyncAt ?? status.lastSyncGpv?.lastCompletedAt,
              wallClockSeconds: status.lastSyncGpv?.wallClockSeconds,
            }
            : undefined);
          message.success(sync.targetCount > 0
            ? t('device.detail.deviceFetchLatestScoped', {
              count: sync.targetCount,
              gpvCount: sync.gpvTaskCount || status.lastSyncGpv?.taskCount || 0,
            })
            : t('device.detail.deviceFetchLatest'));
        } catch {
          // Keep the pending marker; the next interval or mounted detail page can retry.
        }
      }));
    };

    const runPoll = async () => {
      if (disposed || pollingRef.current) return;
      pollingRef.current = true;
      try {
        await poll();
      } finally {
        pollingRef.current = false;
      }
    };

    void runPoll();
    const timer = window.setInterval(() => {
      void runPoll();
    }, 2000);
    return () => {
      disposed = true;
      window.clearInterval(timer);
    };
  }, [message, queryClient, quickSettingsSyncs, t]);

  return null;
}
