import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  LoadingRow,
  PageShell,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import {
  useUnifiedFileTransferTasks,
  useUnifiedFileTransferDevices,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type { UnifiedFileTransferTask } from '@core/types/unifiedFileTransfer'

import {
  DeviceStatusBadge,
  EXEC_MODE_LABEL,
  ProgressBar,
  TaskResultBadge,
  TaskStatusBadge,
} from './_shared'

// ============================================================
// 任务详情 — 从文件传输中心任务列表点进
// 经任务列表查询定位单条任务(无单任务端点),并列出该任务下设备执行项
// ============================================================

export default function TaskDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // 任务列表不提供单条端点:拉一页较大量(200)再按 id 定位真实任务数据
  const tasksQuery = useUnifiedFileTransferTasks({ page: 1, pageSize: 200 })
  const task = useMemo(
    () => (tasksQuery.data?.items ?? []).find((t) => t.id === id),
    [tasksQuery.data, id]
  )

  // 设备执行项:用任务的 typeCode 缩小范围,再按 taskId 客户端过滤
  const devicesQuery = useUnifiedFileTransferDevices({
    page: 1,
    pageSize: 200,
    ...(task?.typeCode ? { typeCode: task.typeCode } : {}),
  })
  const devices = useMemo(
    () => (devicesQuery.data?.items ?? []).filter((d) => d.taskId === id),
    [devicesQuery.data, id]
  )

  const loadingTask = tasksQuery.isLoading

  return (
    <PageShell
      title={task ? task.taskName || '任务详情' : '任务详情'}
      description={task ? `${task.categoryLabel || task.category} · ${task.typeDisplayName || task.typeCode}` : undefined}
      isFetching={tasksQuery.isFetching || devicesQuery.isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate('/transfer/center')}>
            <ArrowLeft className="size-4" /> 返回传输中心
          </Button>
        </div>
      }
    >
      {loadingTask ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : tasksQuery.isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{tasksQuery.error instanceof Error ? tasksQuery.error.message : '未知错误'}
        </Card>
      ) : !task ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到任务 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/transfer/center')}>
            返回传输中心
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <TaskSummary task={task} />
          <DeviceItems
            isLoading={devicesQuery.isLoading}
            rows={devices}
          />
        </div>
      )}
    </PageShell>
  )
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-1.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm">{value ?? '—'}</span>
    </div>
  )
}

function TaskSummary({ task }: { task: UnifiedFileTransferTask }) {
  return (
    <Card>
      <CardHeader className="p-4 pb-2">
        <CardTitle className="text-base font-medium">任务概览</CardTitle>
      </CardHeader>
      <CardContent className="grid grid-cols-1 gap-x-8 p-4 pt-2 sm:grid-cols-2 lg:grid-cols-3">
        <InfoRow label="任务名称" value={task.taskName} />
        <InfoRow label="业务分类" value={task.categoryLabel || task.category} />
        <InfoRow label="任务类型" value={task.typeDisplayName || task.typeCode} />
        <InfoRow label="状态" value={<TaskStatusBadge status={task.status} />} />
        <InfoRow label="结果" value={<TaskResultBadge result={task.result} />} />
        <InfoRow label="进度" value={<ProgressBar percent={task.progress} />} />
        <InfoRow
          label="成功 / 失败 / 总数"
          value={
            <span className="tabular-nums">
              {task.successCount} / {task.failCount} / {task.totalCount}
            </span>
          }
        />
        <InfoRow label="执行方式" value={EXEC_MODE_LABEL[task.executionMode] ?? task.executionMode} />
        <InfoRow label="创建人" value={task.createUser} />
        <InfoRow label="创建时间" value={formatTime(task.createdAt)} />
        {task.scheduledAt && <InfoRow label="计划执行时间" value={formatTime(task.scheduledAt)} />}
        {task.targetVersion && <InfoRow label="目标版本" value={task.targetVersion} />}
        {task.productType && <InfoRow label="产品型号" value={task.productType} />}
        <InfoRow label="操作员范围" value={task.operatorScope || '—'} />
      </CardContent>
    </Card>
  )
}

function DeviceItems({
  isLoading,
  rows,
}: {
  isLoading: boolean
  rows: import('@core/types/unifiedFileTransfer').UnifiedFileTransferDeviceItem[]
}) {
  const cols = 6
  return (
    <div>
      <h2 className="mb-2 text-sm font-medium text-muted-foreground">
        设备执行项 ({rows.length})
      </h2>
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>设备 SN</TableHead>
              <TableHead>设备名称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>进度</TableHead>
              <TableHead>失败原因</TableHead>
              <TableHead>最近上报</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols}>该任务暂无设备执行项</EmptyRow>
            ) : (
              rows.map((d) => (
                <TableRow key={d.id}>
                  <TableCell>
                    <span className="font-mono text-xs">{d.deviceSn || '—'}</span>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm">{d.deviceName || '—'}</span>
                  </TableCell>
                  <TableCell>
                    <DeviceStatusBadge status={d.status} />
                  </TableCell>
                  <TableCell>
                    <ProgressBar percent={d.progress} />
                  </TableCell>
                  <TableCell>
                    <span
                      className="block max-w-[260px] truncate text-xs text-muted-foreground"
                      title={d.failureDetail || d.failureReason || ''}
                    >
                      {d.failureReason || '—'}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs text-muted-foreground">
                      {formatTime(d.lastReportAt)}
                    </span>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </div>
  )
}
