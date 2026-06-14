import { Badge } from '@/components/ui/badge'
import type {
  TransferTaskStatus,
  TransferTaskResult,
  TransferExecutionMode,
  UnifiedFileTransferDeviceStatus,
  TransferStepId,
} from '@core/types/unifiedFileTransfer'

// ============================================================
// 文件传输模块共享:状态/结果/执行方式 的标签 + Badge variant 映射
// 标签直接采用业务语义中文,数据本身的 categoryLabel / typeDisplayName 走真实 API 字段
// ============================================================

type BadgeVariant =
  | 'default'
  | 'secondary'
  | 'destructive'
  | 'warning'
  | 'success'
  | 'muted'
  | 'outline'

export const TASK_STATUS_META: Record<
  TransferTaskStatus,
  { label: string; variant: BadgeVariant }
> = {
  pending: { label: '待执行', variant: 'muted' },
  in_progress: { label: '执行中', variant: 'default' },
  suspended: { label: '已暂停', variant: 'warning' },
  ended: { label: '已结束', variant: 'success' },
}

export const TASK_RESULT_META: Record<
  TransferTaskResult,
  { label: string; variant: BadgeVariant }
> = {
  success: { label: '成功', variant: 'success' },
  partial: { label: '部分成功', variant: 'warning' },
  failure: { label: '失败', variant: 'destructive' },
  terminated: { label: '已终止', variant: 'muted' },
}

export const EXEC_MODE_LABEL: Record<TransferExecutionMode, string> = {
  immediate: '立即执行',
  scheduled: '定时执行',
  suspended: '挂起',
}

export const DEVICE_STATUS_META: Record<
  UnifiedFileTransferDeviceStatus,
  { label: string; variant: BadgeVariant }
> = {
  pending: { label: '待执行', variant: 'muted' },
  downloading: { label: '下载中', variant: 'default' },
  uploading: { label: '上传中', variant: 'default' },
  awaiting_tc: { label: '等待回传', variant: 'warning' },
  verifying: { label: '校验中', variant: 'default' },
  suspended: { label: '已暂停', variant: 'warning' },
  ended: { label: '已结束', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
}

export const STEP_LABEL: Record<TransferStepId, string> = {
  CHECK_PERMISSION: '权限校验',
  CHECK_ONLINE: '在线检查',
  CHECK_CONFLICT: '冲突检查',
  PRE_VALIDATE: '预校验',
  SEND_RPC: '下发指令',
  WAIT_RPC_RESPONSE: '等待响应',
  WAIT_FILE_TRANSFER: '文件传输',
  WAIT_TRANSFER_COMPLETE: '传输完成',
  WAIT_INFORM_EVENT: '事件上报',
  WAIT_REBOOT_COMPLETE: '重启完成',
}

export const RPC_TYPE_LABEL: Record<string, string> = {
  DOWNLOAD: '下载 (Download)',
  UPLOAD: '上传 (Upload)',
  SET_PARAM_VALUES: '参数下发 (SetParameterValues)',
}

export function TaskStatusBadge({ status }: { status: TransferTaskStatus }) {
  const meta = TASK_STATUS_META[status] ?? { label: status, variant: 'muted' as const }
  return <Badge variant={meta.variant}>{meta.label}</Badge>
}

export function TaskResultBadge({ result }: { result?: TransferTaskResult }) {
  if (!result) return <span className="text-xs text-muted-foreground">—</span>
  const meta = TASK_RESULT_META[result] ?? { label: result, variant: 'muted' as const }
  return <Badge variant={meta.variant}>{meta.label}</Badge>
}

export function DeviceStatusBadge({ status }: { status: UnifiedFileTransferDeviceStatus }) {
  const meta = DEVICE_STATUS_META[status] ?? { label: status, variant: 'muted' as const }
  return <Badge variant={meta.variant}>{meta.label}</Badge>
}

/** 进度条 — 纯 CSS 近似(无外部图表依赖) */
export function ProgressBar({ percent }: { percent: number }) {
  const pct = Math.max(0, Math.min(100, Math.round(percent)))
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full bg-primary transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs tabular-nums text-muted-foreground">{pct}%</span>
    </div>
  )
}
