import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Pencil, Plus, RefreshCcw, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  formatTime,
  LoadingRow,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { usePmAdhocList, useCancelPmAdhoc } from '@core/hooks/api/usePmAdhoc'
import type { AdhocStatus, AdhocTask } from '@core/types/pmAdhoc'

// ============================================================
// 自定义聚合任务 — 对齐 v1 /performance/pm-adhoc
//   内置区 + 自建区两张表;行可进编辑(/performance/pm-adhoc/:id/edit);新建跳 /new。
// ============================================================

const STATUS_META: Record<
  AdhocStatus,
  { label: string; variant: 'default' | 'success' | 'warning' | 'destructive' | 'muted' }
> = {
  pending: { label: '等待中', variant: 'muted' },
  running: { label: '执行中', variant: 'warning' },
  scheduled: { label: '已调度', variant: 'default' },
  succeeded: { label: '成功', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
  canceled: { label: '已取消', variant: 'muted' },
}

const DIMENSION_LABEL: Record<string, string> = {
  device: '按设备',
  aggregate_group: '自选组聚合',
  product: '按产品',
  band: '按频段',
  network: '全网汇总',
  device_group: '按设备组',
}

const TECH_LABEL: Record<string, string> = {
  lte: 'LTE',
  nr: 'NR',
  gsm: 'GSM',
}

function statusBadge(status: AdhocStatus) {
  const meta = STATUS_META[status] ?? { label: status, variant: 'outline' as const }
  return <Badge variant={meta.variant}>{meta.label}</Badge>
}

function AdhocTable({
  title,
  rows,
  isLoading,
  isError,
  error,
  onEdit,
  onCancel,
  cancelingId,
  builtin,
}: {
  title: string
  rows: AdhocTask[]
  isLoading: boolean
  isError: boolean
  error: unknown
  onEdit: (id: string) => void
  onCancel: (id: string) => void
  cancelingId: string | null
  builtin: boolean
}) {
  const cols = builtin
    ? ['任务名称', '制式', '维度', '指标数', '粒度', '状态', '更新时间', '操作']
    : ['任务名称', '模式', '维度', '设备数', '指标数', '状态', '进度', '创建时间', '操作']

  return (
    <div>
      <div className="mb-2 text-sm font-medium">{title}</div>
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
              <EmptyRow colSpan={cols.length}>
                {builtin ? '暂无内置任务' : '暂无自建任务'}
              </EmptyRow>
            ) : (
              rows.map((t) => {
                const busy = cancelingId === t.id
                const active = t.status === 'running' || t.status === 'pending' || t.status === 'scheduled'
                return (
                  <TableRow key={t.id} className={cn(busy && 'opacity-50')}>
                    <TableCell className="font-medium">
                      <button
                        type="button"
                        className="text-primary hover:underline"
                        onClick={() => onEdit(t.id)}
                      >
                        {t.name}
                      </button>
                    </TableCell>
                    {builtin ? (
                      <TableCell className="text-xs">
                        {t.technology ? TECH_LABEL[t.technology] ?? t.technology : '不限'}
                      </TableCell>
                    ) : (
                      <TableCell className="text-xs">
                        {t.mode === 'continuous' ? '持续' : '单次'}
                      </TableCell>
                    )}
                    <TableCell className="text-xs">
                      {DIMENSION_LABEL[t.dimension] ?? t.dimension}
                    </TableCell>
                    {builtin ? (
                      <TableCell className="tabular-nums">{t.metricPaths.length}</TableCell>
                    ) : (
                      <TableCell className="tabular-nums">{t.deviceSns.length}</TableCell>
                    )}
                    {builtin ? (
                      <TableCell className="text-xs">
                        {t.granularities.join(', ') || '—'}
                      </TableCell>
                    ) : (
                      <TableCell className="tabular-nums">{t.metricPaths.length}</TableCell>
                    )}
                    <TableCell>{statusBadge(t.status)}</TableCell>
                    {builtin ? (
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(t.updatedAt)}
                      </TableCell>
                    ) : (
                      <TableCell className="w-32">
                        <div className="flex items-center gap-2">
                          <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                            <div
                              className="h-full rounded-full bg-primary transition-all"
                              style={{ width: `${Math.min(100, Math.max(0, t.progress))}%` }}
                            />
                          </div>
                          <span className="w-8 text-right text-xs tabular-nums text-muted-foreground">
                            {Math.round(t.progress)}%
                          </span>
                        </div>
                      </TableCell>
                    )}
                    {!builtin && (
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(t.createdAt)}
                      </TableCell>
                    )}
                    <TableCell>
                      <div className="flex items-center gap-1.5">
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={busy}
                          onClick={() => onEdit(t.id)}
                        >
                          <Pencil className="size-3.5" /> 编辑
                        </Button>
                        {!builtin && active && (
                          <Button
                            size="sm"
                            variant="ghost"
                            disabled={busy}
                            className="text-destructive hover:text-destructive"
                            onClick={() => onCancel(t.id)}
                            aria-label="取消任务"
                          >
                            <XCircle className="size-4" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
    </div>
  )
}

export function PmAdhocPage() {
  const navigate = useNavigate()
  const [cancelingId, setCancelingId] = useState<string | null>(null)

  const builtin = usePmAdhocList({ isBuiltin: true, refetchInterval: 10_000 })
  const custom = usePmAdhocList({ isBuiltin: false, refetchInterval: 10_000 })
  const cancel = useCancelPmAdhoc()

  const builtinRows = useMemo(() => builtin.data ?? [], [builtin.data])
  const customRows = useMemo(() => custom.data ?? [], [custom.data])

  const onEdit = (id: string) => navigate(`/performance/pm-adhoc/${id}/edit`)
  const onCancel = (id: string) => {
    setCancelingId(id)
    cancel.mutate(id, { onSettled: () => setCancelingId(null) })
  }

  const fetching = builtin.isFetching || custom.isFetching || cancel.isPending

  return (
    <PageShell
      title="自定义聚合任务"
      description="按维度(产品/频段/设备组/全网)聚合 PM 指标的内置与自建任务"
      isFetching={fetching}
      toolbar={
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={fetching}
            onClick={() => {
              void builtin.refetch()
              void custom.refetch()
            }}
          >
            <RefreshCcw className="size-4" /> 刷新
          </Button>
          <Button size="sm" onClick={() => navigate('/performance/pm-adhoc/new')}>
            <Plus className="size-4" /> 新建任务
          </Button>
        </div>
      }
    >
      <div className="space-y-6">
        <AdhocTable
          title="内置任务"
          rows={builtinRows}
          isLoading={builtin.isLoading}
          isError={builtin.isError}
          error={builtin.error}
          onEdit={onEdit}
          onCancel={onCancel}
          cancelingId={cancelingId}
          builtin
        />
        <AdhocTable
          title="自建任务"
          rows={customRows}
          isLoading={custom.isLoading}
          isError={custom.isError}
          error={custom.error}
          onEdit={onEdit}
          onCancel={onCancel}
          cancelingId={cancelingId}
          builtin={false}
        />
      </div>
    </PageShell>
  )
}

export default PmAdhocPage
