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

// Durable parameter-sync requests have a 30-minute backend deadline. Keep the
// watcher slightly longer so a valid slow/offline-device request is not
// abandoned before the deadline reconciler can publish its terminal state.
const staleMonitorTimeoutMs = 31 * 60 * 1000;

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
          let timeoutStatusIsActive = false;
          try {
            const timeoutStatus = await deviceParameterApi.getSyncStatus(deviceId);
            queryClient.setQueryData(['devices', 'sync-status', deviceId], timeoutStatus);
            if (isCurrentSyncFailure(timeoutStatus, sync)) {
              timeoutMsg = timeoutStatus.lastParamSyncError || t('device.detail.deviceFetchFailed');
              timeoutAsError = true;
            } else if (timeoutStatus.status === 'syncing' || timeoutStatus.pendingCommands > 0) {
              timeoutStatusIsActive = true;
              timeoutMsg = t('device.detail.deviceFetchTimeoutQueue', { pending: timeoutStatus.pendingCommands });
            }
          } catch {
            // The backend may still be running. A transient status failure is not
            // evidence that a durable request is stale, so retain the monitor.
            return;
          }
          if (!timeoutAsError && timeoutStatusIsActive) return;
          useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId);
          if (timeoutAsError) {
            message.error(timeoutMsg);
          } else {
            message.warning(timeoutMsg);
          }
          return;
        }
        try {
          let request;
          if (sync.requestId) {
            try {
              request = await deviceParameterApi.getParameterSyncRequest(sync.requestId);
            } catch {
              // Request lookup is advisory; always retain the device-level
              // status fallback when this endpoint has a transient failure.
            }
          }
          const status = await deviceParameterApi.getSyncStatus(deviceId);
          queryClient.setQueryData(['devices', 'sync-status', deviceId], status);

          if (shouldHintCongestion(status, sync)) {
            useQuickSettingsFeedbackStore.getState().patchQuickSettingsSync(deviceId, { congestionHinted: true });
            message.warning(t('device.detail.deviceFetchQueueBusy', { pending: status.pendingCommands }));
          }

          const hasNewSuccess = request
            ? request.status === 'succeeded'
            : isCurrentSyncSuccess(status, sync);
          const hasNewFailure = request
            ? ['failed', 'timed_out', 'cancelled', 'rejected'].includes(request.status)
            : isCurrentSyncFailure(status, sync);
          if (request && ['accepted', 'queued', 'running'].includes(request.status)) return;
          if (status.status === 'syncing' && !hasNewSuccess && !hasNewFailure) return;
          if (!hasNewSuccess && !hasNewFailure) return;

          if (hasNewFailure && !hasNewSuccess) {
            useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId);
            message.error(request?.errorMessage || status.lastParamSyncError || t('device.detail.deviceFetchFailed'));
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
              completedAt: request?.completedAt ?? status.lastParamSyncAt ?? status.lastSyncGpv?.lastCompletedAt,
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
