import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import type { QuickSettingsSyncMonitor } from '@core/store/quickSettingsFeedbackStore';
import type { ParameterSyncStatus } from '@core/types/deviceParameter';
import { isCurrentSyncFailure, isCurrentSyncSuccess } from '@core/utils/quickSettingsSyncStatus';

const staleMonitorTimeoutMs = 60 * 1000;
const congestionHintAfterMs = 15 * 1000;
const congestionPendingThreshold = 8;

function shouldHintCongestion(status: ParameterSyncStatus, sync: QuickSettingsSyncMonitor) {
  if (sync.congestionHinted) return false;
  if (status.status !== 'syncing') return false;
  if (status.pendingCommands < congestionPendingThreshold) return false;
  return Date.now() - sync.startedAt >= congestionHintAfterMs;
}

export default function QuickSettingsSyncWatcherCore() {
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
        if (Date.now() - sync.startedAt > staleMonitorTimeoutMs) {
          // Core watcher only maintains sync state and cache; UI notification is skin-owned.
          useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId);
          return;
        }

        try {
          const status = await deviceParameterApi.getSyncStatus(deviceId);
          queryClient.setQueryData(['devices', 'sync-status', deviceId], status);

          if (shouldHintCongestion(status, sync)) {
            useQuickSettingsFeedbackStore.getState().patchQuickSettingsSync(deviceId, { congestionHinted: true });
          }

          const hasNewSuccess = isCurrentSyncSuccess(status, sync);
          const hasNewFailure = isCurrentSyncFailure(status, sync);
          if (status.status === 'syncing' && !hasNewSuccess && !hasNewFailure) return;
          if (!hasNewSuccess && !hasNewFailure) return;

          if (hasNewFailure && !hasNewSuccess) {
            useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId);
            return;
          }

          if (hasNewSuccess) {
            if (sync.scope !== 'license') {
              useQuickSettingsFeedbackStore.getState().clearDraftsByDevice(deviceId);
              useQuickSettingsFeedbackStore.getState().bumpRefreshTick(deviceId);
            }
            deviceParameterApi.invalidateParameterSchemaCache(deviceId);
            void queryClient.invalidateQueries({ queryKey: ['devices', 'list'] });
            void queryClient.invalidateQueries({ queryKey: ['devices', 'detail-composite-v2', deviceId] });
            if (sync.scope !== 'license') {
              void queryClient.invalidateQueries({ queryKey: ['quicksettings', 'groups', deviceId] });
            }
            void queryClient.invalidateQueries({ queryKey: ['deviceLicenseParams', deviceId] });
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
        } catch {
          // Keep pending marker and retry on next interval.
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
  }, [queryClient, quickSettingsSyncs]);

  return null;
}
