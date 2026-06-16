import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Ban, Loader2, Pencil, Plus, RefreshCcw, Trash2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import { usePmAdhocList, useCancelPmAdhoc, useDeletePmAdhoc } from '@core/hooks/api/usePmAdhoc'
import type { AdhocStatus, AdhocTask } from '@core/types/pmAdhoc'

/**
 * F03 · 自定义聚合任务（performance/pm-adhoc）
 * 内置区 + 自建区两张表（真实 usePmAdhocList isBuiltin）。
 * 取消运行中任务走真实 useCancelPmAdhoc；新建/编辑跳向导路由。
 */

const STATUS_TOKEN: Record<AdhocStatus, string> = {
  pending: 'minor',
  running: 'ok',
  scheduled: 'warning',
  succeeded: 'ok',
  failed: 'error',
  canceled: 'off',
}
const STATUS_LABEL: Record<AdhocStatus, string> = {
  pending: '排队',
  running: '运行中',
  scheduled: '已排程',
  succeeded: '成功',
  failed: '失败',
  canceled: '已取消',
}

const DIM_LABEL: Record<string, string> = {
  device: '按设备',
  aggregate_group: '自选设备组',
  product: '按产品',
  band: '按频段',
  network: '全网汇总',
  device_group: '按设备组',
}

const TECH_LABEL: Record<string, string> = { lte: 'LTE', nr: 'NR', gsm: 'GSM' }

export default function PmAdhoc() {
  const navigate = useNavigate()
  const { data: builtin = [], isLoading: builtinLoading, isError: builtinError, refetch: refetchBuiltin } =
    usePmAdhocList({ refetchInterval: 5000, isBuiltin: true })
  const { data: custom = [], isLoading: customLoading, isError: customError, refetch: refetchCustom } =
    usePmAdhocList({ refetchInterval: 5000, isBuiltin: false })
  const cancelMut = useCancelPmAdhoc()
  const deleteMut = useDeletePmAdhoc()

  // issue #392：删除终态自建任务 —— 二次确认 → 删除 → 列表自动刷新（任务消失）。
  const onDelete = (id: string) => {
    if (!window.confirm('删除后任务从列表移除且不可恢复；已聚合的结果数据由保留期自动清理。确认删除？')) {
      return
    }
    deleteMut.mutate(id)
  }

  const isFetching = builtinLoading || customLoading

  const refetchAll = () => {
    void refetchBuiltin()
    void refetchCustom()
  }

  return (
    <PageShell
      code="F03"
      title="ADHOC AGGREGATION · 自定义聚合"
      subtitle="BUILTIN + CUSTOM ADHOC TASKS · MULTI-DIMENSION"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<Plus />} onClick={() => navigate('/performance/pm-adhoc/new')}>
            新建任务
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={refetchAll}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="space-y-3">
        <TaskTable
          title="BUILTIN · 内置任务"
          meta="SYSTEM PRESET"
          tasks={builtin}
          loading={builtinLoading}
          isError={builtinError}
          builtinArea
          onCancel={(id) => cancelMut.mutate(id)}
          onEdit={(t) => navigate(`/performance/pm-adhoc/${t.id}/edit`)}
          cancelling={cancelMut.isPending}
        />
        <TaskTable
          title="CUSTOM · 自建任务"
          meta="USER DEFINED"
          tasks={custom}
          loading={customLoading}
          isError={customError}
          builtinArea={false}
          onCancel={(id) => cancelMut.mutate(id)}
          onEdit={(t) => navigate(`/performance/pm-adhoc/${t.id}/edit`)}
          onDelete={onDelete}
          cancelling={cancelMut.isPending}
          deleting={deleteMut.isPending}
        />
      </div>
    </PageShell>
  )
}

function TaskTable({
  title,
  meta,
  tasks,
  loading,
  isError,
  builtinArea,
  onCancel,
  onEdit,
  onDelete,
  cancelling,
  deleting,
}: {
  title: string
  meta: string
  tasks: AdhocTask[]
  loading: boolean
  isError: boolean
  builtinArea: boolean
  onCancel: (id: string) => void
  onEdit: (t: AdhocTask) => void
  // issue #392：删除终态自建任务（仅自建区传入；内置区不传，按钮恒不渲染）。
  onDelete?: (id: string) => void
  cancelling: boolean
  deleting?: boolean
}) {
  const sorted = useMemo(
    () => [...tasks].sort((a, b) => (b.createdAt || '').localeCompare(a.createdAt || '')),
    [tasks],
  )

  return (
    <GlassPanel strong title={title} meta={`${meta} · ${tasks.length}`}>
      <div className="p-3">
        <div className="grid grid-cols-[2fr_90px_1fr_90px_1fr_1.2fr_140px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
          <span>NAME · 名称</span>
          <span>MODE</span>
          <span>DIM · 维度</span>
          <span>TECH</span>
          <span>METRICS · 指标</span>
          <span>STATUS · 进度</span>
          <span className="text-right">ACTIONS</span>
        </div>

        <div className="mt-1.5 space-y-1">
          {loading ? (
            <Loading text="SYNCING TASKS…" />
          ) : isError ? (
            <ErrorBox text="LOAD FAILED" />
          ) : sorted.length === 0 ? (
            <EmptyBox text={builtinArea ? 'NO BUILTIN TASK' : 'NO CUSTOM TASK · 点右上新建'} />
          ) : (
            sorted.map((t) => {
              const cancellable =
                t.status === 'pending' || t.status === 'running' || t.status === 'scheduled'
              // issue #392：终态(成功/失败/已取消)自建任务可删除。
              const terminal =
                t.status === 'succeeded' || t.status === 'failed' || t.status === 'canceled'
              return (
                <div
                  key={t.id}
                  className="grid grid-cols-[2fr_90px_1fr_90px_1fr_1.2fr_140px] items-center gap-3 rounded-sm px-3 py-2 transition-colors hover:bg-cyan-500/5"
                >
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100" title={t.name}>
                      {t.name}
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/45">
                      创建 {t.createdAt ? formatTime(t.createdAt) : '—'}
                    </div>
                  </div>
                  <span className="chip w-fit" style={{ color: t.mode === 'continuous' ? '#a855f7' : '#5b9eff' }}>
                    {t.mode === 'continuous' ? '持续' : '单次'}
                  </span>
                  <span className="truncate font-mono text-[11px] text-cyan-300/70">
                    {DIM_LABEL[t.dimension] ?? t.dimension}
                  </span>
                  <span className="font-mono text-[11px] text-cyan-300/70">
                    {t.technology ? TECH_LABEL[t.technology] ?? t.technology : '不限'}
                  </span>
                  <span className="font-mono text-[11px] text-cyan-300/70">{t.metricPaths.length} 项</span>
                  <div className="flex items-center gap-2">
                    <StatusBadge status={STATUS_TOKEN[t.status] ?? 'unknown'} label={STATUS_LABEL[t.status] ?? t.status} />
                    {t.status === 'running' || t.status === 'succeeded' ? (
                      <div className="h-1.5 w-12 overflow-hidden rounded-full bg-cyan-500/10">
                        <div
                          className="h-full rounded-full bg-cyan-400"
                          style={{ width: `${Math.max(0, Math.min(100, t.progress))}%`, boxShadow: '0 0 8px #00f0ff' }}
                        />
                      </div>
                    ) : null}
                  </div>
                  <div className="flex justify-end gap-1.5">
                    <NeonButton icon={<Pencil />} onClick={() => onEdit(t)}>
                      {builtinArea ? '指标' : '编辑'}
                    </NeonButton>
                    {cancellable ? (
                      <NeonButton tone="danger" icon={<Ban />} disabled={cancelling} onClick={() => onCancel(t.id)}>
                        取消
                      </NeonButton>
                    ) : null}
                    {/* issue #392：终态自建任务给「删除」（onDelete 仅自建区传入） */}
                    {!builtinArea && terminal && onDelete ? (
                      <NeonButton tone="danger" icon={<Trash2 />} disabled={deleting} onClick={() => onDelete(t.id)}>
                        删除
                      </NeonButton>
                    ) : null}
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>
    </GlassPanel>
  )
}

/* ───────── 复用小件 ───────── */

function Loading({ text }: { text: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-10 text-cyan-300/60">
      <Loader2 className="size-5 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{text}</span>
    </div>
  )
}

function ErrorBox({ text }: { text: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">{text}</div>
  )
}

function EmptyBox({ text }: { text: string }) {
  return (
    <div className={cn('flex items-center justify-center py-10 font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40')}>
      {text}
    </div>
  )
}
