import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import type { QuickSettingsSyncMonitor } from '@core/store/quickSettingsFeedbackStore';
import type { ParameterSyncStatus } from '@core/types/deviceParameter';

const terminalClockSkewMs = 5000;
const staleMonitorTimeoutMs = 60 * 1000;
const congestionHintAfterMs = 15 * 1000;
const congestionPendingThreshold = 8;

function comparableSourceId(sourceId: string | undefined) {
  return sourceId?.startsWith('manual:') ? sourceId.slice('manual:'.length) : sourceId;
}

function isTerminalAfterSyncStart(timestamp: string | undefined, sync: QuickSettingsSyncMonitor) {
  if (!timestamp) return false;
  const terminalAt = Date.parse(timestamp);
  return Number.isFinite(terminalAt) && terminalAt >= sync.startedAt - terminalClockSkewMs;
}

function isCurrentSyncSuccess(status: ParameterSyncStatus, sync: QuickSettingsSyncMonitor) {
  if (sync.sourceId && status.lastSyncGpv?.sourceId) {
    const sourceMatched = comparableSourceId(status.lastSyncGpv.sourceId) === comparableSourceId(sync.sourceId);
    if (!sourceMatched) return false;
    if (isTerminalAfterSyncStart(status.lastSyncGpv.lastCompletedAt, sync)) return true;
  }
  if (!status.lastParamSyncAt || status.lastParamSyncAt === sync.lastParamSyncAt) return false;
  if (!sync.sourceId && !sync.lastParamSyncAt) return false;
  return isTerminalAfterSyncStart(status.lastParamSyncAt, sync);
}

function isCurrentSyncFailure(status: ParameterSyncStatus, sync: QuickSettingsSyncMonitor) {
  if (!status.lastParamSyncFailedAt || status.lastParamSyncFailedAt === sync.lastParamSyncFailedAt) return false;
  if (!sync.sourceId && !sync.lastParamSyncFailedAt) return false;
  return isTerminalAfterSyncStart(status.lastParamSyncFailedAt, sync);
}

function shouldHintCongestion(status: ParameterSyncStatus, sync: QuickSettingsSyncMonitor) {
  if (sync.congestionHinted) return false;
  if (status.status !== 'syncing') return false;
  if (status.pendingCommands < congestionPendingThreshold) return false;
  return Date.now() - sync.startedAt >= congestionHintAfterMs;
}

export default function QuickSettingsSyncWatcherCore() {
  const queryClient = useQueryClient();
  const quickSettingsSyncs = useQuickSettingsFeedbackStore((s) => s.quickSettingsSyncs);

  useEffect(() => {
    const poll = async () => {
      const entries = Object.entries(useQuickSettingsFeedbackStore.getState().quickSettingsSyncs);
      if (entries.length === 0) return;

      await Promise.all(entries.map(async ([deviceId, sync]) => {
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

          if (status.status === 'syncing') return;

          const hasNewSuccess = isCurrentSyncSuccess(status, sync);
          const hasNewFailure = isCurrentSyncFailure(status, sync);
          if (!hasNewSuccess && !hasNewFailure) return;

          if (hasNewFailure && !hasNewSuccess) {
            useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId);
            return;
          }

          if (hasNewSuccess) {
            useQuickSettingsFeedbackStore.getState().clearDraftsByDevice(deviceId);
            useQuickSettingsFeedbackStore.getState().bumpRefreshTick(deviceId);
            void queryClient.invalidateQueries({ queryKey: ['devices', 'detail-composite-v2', deviceId] });
            void queryClient.invalidateQueries({ queryKey: ['quicksettings', 'groups', deviceId] });
            void queryClient.invalidateQueries({ queryKey: ['devices', 'parameter-schema', deviceId] });
          }

          useQuickSettingsFeedbackStore.getState().finishQuickSettingsSync(deviceId, hasNewSuccess && sync.targetCount > 0
            ? {
              targetCount: sync.targetCount,
              gpvTaskCount: sync.gpvTaskCount || status.lastSyncGpv?.taskCount || 0,
              completedAt: status.lastParamSyncAt,
              wallClockSeconds: status.lastSyncGpv?.wallClockSeconds,
            }
            : undefined);
        } catch {
          // Keep pending marker and retry on next interval.
        }
      }));
    };

    void poll();
    const timer = window.setInterval(() => {
      void poll();
    }, 2000);

    return () => window.clearInterval(timer);
  }, [queryClient, quickSettingsSyncs]);

  return null;
}
