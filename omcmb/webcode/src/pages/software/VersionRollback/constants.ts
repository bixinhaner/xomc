/**
 * VersionRollback 页面常量。
 * 从 index.tsx 拆出（react-refresh/only-export-components：页面文件只导出组件）。
 * 以下两组配色为 export 占位（T-0136：保留以便未来按 task result / sub-task 状态着色复用）。
 */

// Task result display color
export const TASK_RESULT_COLORS: Record<string, string> = {
  success: 'success',
  partial: 'warning',
  failed: 'error',
  terminated: 'default',
};

// Sub-task status display color
export const SUB_TASK_STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  downloading: 'processing',
  rebooting: 'processing',
  verifying: 'processing',
  completed: 'success',
  failed: 'error',
  suspended: 'warning',
  terminated: 'default',
};
