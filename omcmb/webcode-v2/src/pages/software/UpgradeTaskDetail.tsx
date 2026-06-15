import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
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
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'

import { useUpgradeTaskById, useSubTasks } from '@core/hooks/api/useSoftware'
import type { UpgradeSubTaskInfo, UpgradeTaskInfo } from '@core/mock/data/software'

import { SUB_TASK_STATUS, TASK_RESULT, TASK_STATUS, TASK_TYPE, progressPct } from './_shared'
import { InfoRow, ProgressBar } from './_components'

// ===========================================================================
// 升级任务详情 — 对照 v1 UpgradePlan 的子任务抽屉（Drawer + 子任务列表）
// 按 id 经 useUpgradeTaskById 加载任务头，useSubTasks 加载真实子任务分页
// 升级计划 /software/upgrade-plan 与版本回退 /software/rollback 共用此详情页
// ===========================================================================

export default function UpgradeTaskDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: task, isLoading, isError, error, isFetching } = useUpgradeTaskById(id)

  const backTo = task?.taskType === 2 ? '/software/rollback' : '/software/upgrade-plan'

  return (
    <PageShell
      title={task ? task.taskName : '任务详情'}
      description={task ? `${TASK_TYPE[task.taskType] ?? `类型${task.taskType}`} · ${task.productClass || '—'}` : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" onClick={() => navigate(backTo)}>
            <ArrowLeft className="size-4" /> 返回任务列表
          </Button>
          {task && (
            <Badge variant={TASK_STATUS[task.status].variant} className="ml-2">
              {TASK_STATUS[task.status].label}
            </Badge>
          )}
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : !task ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到任务 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate(backTo)}>
            返回任务列表
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <TaskHeader task={task} />
          <SubTasks taskId={task.id} />
        </div>
      )}
    </PageShell>
  )
}

function TaskHeader({ task: t }: { task: UpgradeTaskInfo }) {
  return (
    <Card>
      <CardHeader className="p-4 pb-2">
        <CardTitle className="text-base font-medium">任务概要</CardTitle>
      </CardHeader>
      <CardContent className="grid grid-cols-1 gap-x-8 p-4 pt-2 sm:grid-cols-2">
        <InfoRow label="任务名称" value={t.taskName} />
        <InfoRow label="任务类型" value={TASK_TYPE[t.taskType] ?? `类型${t.taskType}`} />
        <InfoRow
          label="状态"
          value={<Badge variant={TASK_STATUS[t.status].variant}>{TASK_STATUS[t.status].label}</Badge>}
        />
        <InfoRow
          label="结果"
          value={t.result ? <Badge variant={TASK_RESULT[t.result].variant}>{TASK_RESULT[t.result].label}</Badge> : '—'}
        />
        <InfoRow label="设备型号" value={t.productClass} />
        <InfoRow label="固件文件" value={t.fileName} mono />
        <InfoRow label="保留配置" value={t.isKeepConfig ? '是' : '否'} />
        <InfoRow label="最大并发" value={t.maxConcurrent} />
        <InfoRow label="创建人" value={t.createUser} />
        <InfoRow label="创建时间" value={formatTime(t.createdAt)} />
        <InfoRow label="开始时间" value={formatTime(t.startedAt)} />
        <InfoRow label="结束时间" value={formatTime(t.endedAt)} />
        <div className="flex items-center justify-between gap-4 border-b py-2.5 text-sm">
          <span className="shrink-0 text-muted-foreground">进度</span>
          <div className="min-w-0">
            <ProgressBar pct={progressPct(t)} />
          </div>
        </div>
        <div className="flex items-center justify-between gap-4 border-b py-2.5 text-sm">
          <span className="shrink-0 text-muted-foreground">成功 / 失败 / 总数</span>
          <span className="font-mono text-xs">
            <span className="text-emerald-600 dark:text-emerald-400">{t.successCount}</span>
            {' / '}
            <span className="text-destructive">{t.failCount}</span>
            {' / '}
            <span className="text-muted-foreground">{t.totalCount}</span>
          </span>
        </div>
      </CardContent>
    </Card>
  )
}

function SubTasks({ taskId }: { taskId: string }) {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const { data, isLoading, isError, error } = useSubTasks(taskId, { page, pageSize })

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // qa-614 #371：补"起始时间"列以呼应起止时间（5G 回退起始时间现已记录）。
  const cols = ['设备SN', '原版本', '目标版本', '状态', '重试', '失败原因', '起始时间', '完成时间']

  return (
    <div>
      <h2 className="mb-3 text-base font-medium">子任务（设备级）</h2>
      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无子任务</EmptyRow>
            ) : (
              rows.map((s: UpgradeSubTaskInfo) => {
                const st = SUB_TASK_STATUS[s.status]
                return (
                  <TableRow key={s.id}>
                    <TableCell className="font-mono text-xs">{s.deviceSn || s.deviceId}</TableCell>
                    <TableCell className="text-xs">{s.oriVersion || '—'}</TableCell>
                    {/* qa-614 #371：目标版本完成后由后端回填，未回报时显示"待上报"。 */}
                    <TableCell className="text-xs">{s.destVersion || '待上报'}</TableCell>
                    <TableCell>
                      <Badge variant={st.variant}>{st.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {s.retryCount}/{s.maxRetries}
                    </TableCell>
                    <TableCell
                      className="max-w-[220px] truncate text-xs text-destructive"
                      title={s.failureReason || s.errorMessage || ''}
                    >
                      {s.failureReason || s.errorMessage || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatTime(s.startedAt)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatTime(s.completedAt)}</TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />
    </div>
  )
}
