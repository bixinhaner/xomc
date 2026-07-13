import type { QuickSettingsSyncMonitor } from '@core/store/quickSettingsFeedbackStore';
import type { ParameterSyncStatus } from '@core/types/deviceParameter';

const terminalClockSkewMs = 5000;

function comparableSourceId(sourceId: string | undefined) {
  return sourceId?.startsWith('manual:') ? sourceId.slice('manual:'.length) : sourceId;
}

export function isTerminalAfterSyncStart(timestamp: string | undefined, sync: QuickSettingsSyncMonitor) {
  if (!timestamp) return false;
  const terminalAt = Date.parse(timestamp);
  return Number.isFinite(terminalAt) && terminalAt >= sync.startedAt - terminalClockSkewMs;
}

export function isCurrentSyncSuccess(status: ParameterSyncStatus, sync: QuickSettingsSyncMonitor) {
  if (sync.sourceId && status.lastSyncGpv?.sourceId) {
    const sourceMatched = comparableSourceId(status.lastSyncGpv.sourceId) === comparableSourceId(sync.sourceId);
    if (!sourceMatched) return false;
    if (isTerminalAfterSyncStart(status.lastSyncGpv.lastCompletedAt, sync)) return true;
  }
  if (!status.lastParamSyncAt || status.lastParamSyncAt === sync.lastParamSyncAt) return false;
  if (!sync.sourceId && !sync.lastParamSyncAt) return false;
  return isTerminalAfterSyncStart(status.lastParamSyncAt, sync);
}

export function isCurrentSyncFailure(status: ParameterSyncStatus, sync: QuickSettingsSyncMonitor) {
  // Failure timestamps are device-wide and carry no source ID. While commands are
  // still running, the failure may belong to another sync and cannot be attributed
  // safely to this monitor.
  if (status.status === 'syncing') return false;
  if (!status.lastParamSyncFailedAt || status.lastParamSyncFailedAt === sync.lastParamSyncFailedAt) return false;
  if (!sync.sourceId && !sync.lastParamSyncFailedAt) return false;
  return isTerminalAfterSyncStart(status.lastParamSyncFailedAt, sync);
}
