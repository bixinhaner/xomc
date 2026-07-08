/**
 * KPI 导出 Tab 共享：状态/来源标签、字节格式化、文件名派生。
 * 任务管理 Tab 与文件管理 Tab 共用。
 */
import type { KpiExportSource, KpiExportStatus, KpiExportTask } from '@core/types/kpiExport';

/** 来源标签 i18n key。 */
export const SOURCE_LABEL_KEY: Record<KpiExportSource, string> = {
  dashboard: 'kpiExport.source.dashboard',
  kpi_query: 'kpiExport.source.kpiQuery',
  adhoc: 'kpiExport.source.adhoc',
};

/** 状态 Tag 颜色 + i18n key。 */
export const STATUS_TAG: Record<KpiExportStatus, { color: string; labelKey: string }> = {
  pending: { color: 'default', labelKey: 'kpiExport.status.pending' },
  running: { color: 'processing', labelKey: 'kpiExport.status.running' },
  succeeded: { color: 'success', labelKey: 'kpiExport.status.succeeded' },
  failed: { color: 'error', labelKey: 'kpiExport.status.failed' },
};

/** 状态对应进度百分比（无独立进度字段，按状态映射；running 折中 50）。 */
export function statusProgress(status: KpiExportStatus): number {
  switch (status) {
    case 'succeeded':
      return 100;
    case 'running':
      return 50;
    case 'failed':
    case 'pending':
    default:
      return 0;
  }
}

export function formatBytes(n: number): string {
  if (!n || n < 0) return '—';
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(2)} MB`;
}

/** 派生下载文件名：任务名 + .csv（DTO 不下发 file_path，故用任务名兜底）。 */
export function deriveFileName(task: KpiExportTask): string {
  const base = (task.taskName || `kpi_export_${task.id}`).replace(/[\\/:*?"<>|]+/g, '_');
  return base.toLowerCase().endsWith('.csv') ? base : `${base}.csv`;
}
