// ---------------------------------------------------------------------------
// 软件版本模块共享显示映射 / 工具（模块内私有，非页面组件，不进路由）
// 对照 v1 webcode/src/pages/software 各页的状态/类型语义
// ---------------------------------------------------------------------------
import type {
  VersionStatus,
  TaskStatusType,
  TaskResultType,
  TaskTypeValue,
  SubTaskStatusType,
  UpgradePlanStatus,
  UpgradeTaskInfo,
} from '@core/mock/data/software'

export type BadgeVariant =
  | 'default'
  | 'secondary'
  | 'destructive'
  | 'warning'
  | 'success'
  | 'muted'
  | 'outline'

export const VERSION_STATUS: Record<VersionStatus, { label: string; variant: BadgeVariant }> = {
  current: { label: '当前', variant: 'success' },
  beta: { label: '测试', variant: 'warning' },
  deprecated: { label: '已废弃', variant: 'muted' },
  archived: { label: '已归档', variant: 'muted' },
}

export const TASK_STATUS: Record<TaskStatusType, { label: string; variant: BadgeVariant }> = {
  pending: { label: '等待中', variant: 'muted' },
  in_progress: { label: '执行中', variant: 'default' },
  suspended: { label: '已暂停', variant: 'warning' },
  ended: { label: '已结束', variant: 'success' },
}

export const TASK_RESULT: Record<TaskResultType, { label: string; variant: BadgeVariant }> = {
  success: { label: '成功', variant: 'success' },
  partial: { label: '部分成功', variant: 'warning' },
  failed: { label: '失败', variant: 'destructive' },
  terminated: { label: '已终止', variant: 'muted' },
}

export const TASK_TYPE: Record<TaskTypeValue, string> = {
  1: '软件升级',
  2: '版本回退',
  4: '补丁升级',
  6: 'FPGA升级',
  8: '其他',
}

export const SUB_TASK_STATUS: Record<SubTaskStatusType, { label: string; variant: BadgeVariant }> = {
  pending: { label: '等待中', variant: 'muted' },
  downloading: { label: '下载中', variant: 'default' },
  rebooting: { label: '重启中', variant: 'default' },
  verifying: { label: '校验中', variant: 'default' },
  completed: { label: '完成', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
  suspended: { label: '已暂停', variant: 'warning' },
  terminated: { label: '已终止', variant: 'muted' },
}

export const PLAN_STATUS: Record<UpgradePlanStatus, { label: string; variant: BadgeVariant }> = {
  pending: { label: '等待中', variant: 'muted' },
  scheduled: { label: '已排期', variant: 'default' },
  running: { label: '执行中', variant: 'default' },
  success: { label: '成功', variant: 'success' },
  partial: { label: '部分成功', variant: 'warning' },
  failed: { label: '失败', variant: 'destructive' },
  cancelled: { label: '已取消', variant: 'muted' },
}

export const FILE_TYPE_LABEL: Record<number, string> = {
  0: '固件',
  1: '补丁',
  5: '配置',
  6: 'FPGA',
}

export function progressPct(t: Pick<UpgradeTaskInfo, 'totalCount' | 'successCount' | 'failCount'>): number {
  if (!t.totalCount) return 0
  return Math.round(((t.successCount + t.failCount) / t.totalCount) * 100)
}
