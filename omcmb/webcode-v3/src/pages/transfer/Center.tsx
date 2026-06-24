import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Send,
  PauseCircle,
  StopCircle,
  RotateCcw,
  Trash2,
  ChevronRight,
  PackageSearch,
  Activity,
  Layers3,
  CheckCircle2,
  X,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatTime } from '@/lib/format'
import {
  useUnifiedFileTransferOverview,
  useUnifiedFileTransferTaskTypes,
  useUnifiedFileTransferTasks,
  useUnifiedFileTransferDevices,
  useStartUfteTask,
  useSuspendUfteTask,
  useTerminateUfteTask,
  useRetryUfteTask,
  useDeleteUfteTask,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  UnifiedFileTransferTask,
  TransferTaskStatus,
  TransferTaskResult,
  UnifiedFileTransferDeviceStatus,
} from '@core/types/unifiedFileTransfer'
import {
  DEVICE_UPGRADE_CATEGORY,
  aggregateCategoryOptions,
  resolveBackendCategoryParam,
} from '@core/utils/ufteCategory'

const PAGE_SIZE = 20

// 任务状态 → StatusBadge 状态色 + 中文标签
const STATUS_BADGE: Record<TransferTaskStatus, { status: string; label: string }> = {
  pending: { status: 'unknown', label: '待执行' },
  in_progress: { status: 'active', label: '执行中' },
  suspended: { status: 'warning', label: '已挂起' },
  ended: { status: 'offline', label: '已结束' },
}

const RESULT_BADGE: Record<TransferTaskResult, { status: string; label: string }> = {
  success: { status: 'ok', label: '成功' },
  partial: { status: 'warning', label: '部分成功' },
  failure: { status: 'critical', label: '失败' },
  terminated: { status: 'offline', label: '已终止' },
}

const EXEC_MODE_LABEL: Record<UnifiedFileTransferTask['executionMode'], string> = {
  immediate: '立即',
  scheduled: '定时',
  suspended: '挂起',
}

export default function TransferCenterPage() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<TransferTaskStatus | ''>('')
  const [category, setCategory] = useState<string>('')
  const [opError, setOpError] = useState<string | null>(null)
  // #615 任务详情抽屉：任务名与「详情」按钮打开抽屉展示任务+已选设备（原 跳转 /transfer/center/${id} 为死链）。
  const [viewingTask, setViewingTask] = useState<UnifiedFileTransferTask | null>(null)

  const { data: overview, isFetching: overviewFetching } = useUnifiedFileTransferOverview()
  const { data: taskTypes } = useUnifiedFileTransferTaskTypes()

  // 分类下拉来自任务类型表（去重）—— 真实后端 category/categoryLabel。
  // qa-614 c6 #368：4G(enb_upgrade)+5G(gnb_upgrade) 折叠为单条『设备升级』(device_upgrade)，
  // 与 v1/v2 口径一致（聚合逻辑共享自 @core/utils/ufteCategory）。v3 无 typeCode 选择器，
  // 选中『设备升级』chip 即按成员超集筛出 4G+5G 全部升级任务。
  const categories = useMemo(() => {
    return aggregateCategoryOptions(taskTypes ?? []).map((opt) =>
      opt.value === DEVICE_UPGRADE_CATEGORY ? { value: opt.value, label: '设备升级' } : opt,
    )
  }, [taskTypes])

  // qa-614 c6 #368：device_upgrade 展开为后端 category 查询参数（无 typeCode → 成员超集）。
  const backendCategory = category ? resolveBackendCategoryParam(category) : undefined

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(status ? { status } : {}),
      ...(backendCategory ? { category: backendCategory } : {}),
    }),
    [page, keyword, status, backendCategory],
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useUnifiedFileTransferTasks(params)
  const rows = useMemo<UnifiedFileTransferTask[]>(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const startTask = useStartUfteTask()
  const suspendTask = useSuspendUfteTask()
  const terminateTask = useTerminateUfteTask()
  const retryTask = useRetryUfteTask()
  const deleteTask = useDeleteUfteTask()

  const runOp = async (fn: () => Promise<unknown>) => {
    setOpError(null)
    try {
      await fn()
      await refetch()
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '操作失败')
    }
  }

  const submitting =
    startTask.isPending ||
    suspendTask.isPending ||
    terminateTask.isPending ||
    retryTask.isPending ||
    deleteTask.isPending

  return (
    <PageShell
      code="F06"
      title="TRANSFER CENTER · 文件传输中心"
      subtitle="UNIFIED FILE TRANSFER ENGINE · 10s AUTO-REFRESH"
      isFetching={isFetching || overviewFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="任务名 / 类型"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 概览仪表 */}
      <div className="mb-3 grid grid-cols-12 gap-3">
        <GlassPanel
          title="ENGINE OVERVIEW · 引擎概览"
          meta="LAST 30D"
          className="col-span-12 lg:col-span-5"
        >
          <div className="flex items-center justify-around gap-2 px-3 py-4">
            <div className="flex flex-col items-center gap-1">
              <RadialGauge
                value={overview?.successRate30d ?? 0}
                label="成功率"
                size={108}
                color="#00ff88"
              />
            </div>
            <div className="flex flex-col gap-3">
              <MiniStat
                icon={<Activity className="size-3.5" />}
                label="运行中任务"
                value={overview?.runningTaskCount ?? 0}
                color="#00f0ff"
              />
              <MiniStat
                icon={<Layers3 className="size-3.5" />}
                label="启用类型"
                value={overview?.enabledTypeCount ?? 0}
                color="#5b9eff"
              />
              <MiniStat
                icon={<CheckCircle2 className="size-3.5" />}
                label="自定义类型"
                value={overview?.customTypeCount ?? 0}
                color="#a855f7"
              />
            </div>
          </div>
        </GlassPanel>

        {/* 分类筛选 */}
        <GlassPanel
          title="CATEGORY FILTER · 业务分类"
          meta={`${categories.length} CATEGORIES`}
          className="col-span-12 lg:col-span-7"
        >
          <div className="flex flex-wrap gap-2 p-3.5">
            <button
              type="button"
              onClick={() => {
                setCategory('')
                setPage(1)
              }}
              className={`chip transition-all ${
                category === '' ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/45 hover:text-cyan-300/80'
              }`}
            >
              ALL · 全部
            </button>
            {categories.map((c) => (
              <button
                key={c.value}
                type="button"
                onClick={() => {
                  setCategory(c.value)
                  setPage(1)
                }}
                className={`chip transition-all ${
                  category === c.value
                    ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                    : 'text-cyan-300/45 hover:text-cyan-300/80'
                }`}
              >
                {c.label}
              </button>
            ))}
          </div>
          <div className="flex flex-wrap items-center gap-2 border-t border-cyan-500/10 px-3.5 py-2.5">
            <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
              STATUS
            </span>
            {(['', 'pending', 'in_progress', 'suspended', 'ended'] as const).map((s) => (
              <button
                key={s || 'all'}
                type="button"
                onClick={() => {
                  setStatus(s)
                  setPage(1)
                }}
                className={`chip transition-all ${
                  status === s ? 'text-cyan-200 shadow-[0_0_10px_currentColor]' : 'text-cyan-300/45 hover:text-cyan-300/80'
                }`}
              >
                {s ? STATUS_BADGE[s].label : '全部'}
              </button>
            ))}
          </div>
        </GlassPanel>
      </div>

      {opError && (
        <div className="mb-2 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
          OP FAILED · {opError}
          {submitting && <Loader2 className="ml-2 inline size-3 animate-spin" />}
        </div>
      )}

      {/* 列表头 */}
      {rows.length > 0 && (
        <div className="mb-1 grid grid-cols-[2fr_1.2fr_120px_1.4fr_1fr_150px_180px] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
          <span>TASK · 任务</span>
          <span>TYPE · 类型</span>
          <span>STATUS</span>
          <span>PROGRESS · 进度</span>
          <span>EXEC · 方式</span>
          <span>CREATED · 创建</span>
          <span className="text-right">OPS · 操作</span>
        </div>
      )}

      {/* 列表 */}
      <div className="space-y-1.5">
        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
          </div>
        ) : isError ? (
          <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
            FAILURE · {error instanceof Error ? error.message : '未知错误'}
          </div>
        ) : rows.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <PackageSearch className="size-10 text-cyan-400/50" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
              NO TASKS · 暂无传输任务
            </div>
          </div>
        ) : (
          rows.map((task) => {
            const resultBadge = task.result ? RESULT_BADGE[task.result] : undefined
            const statusBadge = STATUS_BADGE[task.status]
            const canStart = task.status === 'pending' || task.status === 'suspended'
            const canSuspend = task.status === 'in_progress'
            const canTerminate = task.status === 'in_progress' || task.status === 'pending'
            const canRetry = task.status === 'ended' && (task.result === 'failure' || task.result === 'partial')
            return (
              <div
                key={task.id}
                className="fleet-row grid grid-cols-[2fr_1.2fr_120px_1.4fr_1fr_150px_180px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: '#00f0ff' }}
              >
                <button
                  type="button"
                  className="min-w-0 text-left"
                  onClick={() => setViewingTask(task)}
                >
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {task.taskName}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {task.categoryLabel} · {task.createUser || '—'}
                  </div>
                </button>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/85">{task.typeDisplayName}</div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">{task.typeCode}</div>
                </div>
                <div className="flex flex-col items-start gap-1">
                  <StatusBadge status={statusBadge.status} label={statusBadge.label} className="scale-90" />
                  {resultBadge && (
                    <StatusBadge status={resultBadge.status} label={resultBadge.label} className="scale-90" />
                  )}
                </div>
                <ProgressCell task={task} />
                <div className="font-mono text-[11px] text-cyan-300/75">
                  {EXEC_MODE_LABEL[task.executionMode]}
                </div>
                <div className="flex flex-col gap-0.5 font-mono text-[11px] text-cyan-300/75">
                  <span title="创建时间">{formatTime(task.createdAt)}</span>
                  <span className="text-cyan-300/55" title="结束时间">
                    {task.endedAt ? `END · ${formatTime(task.endedAt)}` : 'END · —'}
                  </span>
                </div>
                <div className="flex items-center justify-end gap-1.5">
                  {canStart && (
                    <RowAction
                      title="启动"
                      icon={<Send className="size-3.5" />}
                      disabled={submitting}
                      onClick={() => void runOp(() => startTask.mutateAsync(task.id))}
                    />
                  )}
                  {canSuspend && (
                    <RowAction
                      title="挂起"
                      icon={<PauseCircle className="size-3.5" />}
                      disabled={submitting}
                      onClick={() => void runOp(() => suspendTask.mutateAsync(task.id))}
                    />
                  )}
                  {canTerminate && (
                    <RowAction
                      title="终止"
                      danger
                      icon={<StopCircle className="size-3.5" />}
                      disabled={submitting}
                      onClick={() => void runOp(() => terminateTask.mutateAsync(task.id))}
                    />
                  )}
                  {canRetry && (
                    <RowAction
                      title="重试"
                      icon={<RotateCcw className="size-3.5" />}
                      disabled={submitting}
                      onClick={() => void runOp(() => retryTask.mutateAsync(task.id))}
                    />
                  )}
                  <RowAction
                    title="删除"
                    danger
                    icon={<Trash2 className="size-3.5" />}
                    disabled={submitting}
                    onClick={() => void runOp(() => deleteTask.mutateAsync(task.id))}
                  />
                  <RowAction
                    title="详情"
                    icon={<ChevronRight className="size-3.5" />}
                    onClick={() => setViewingTask(task)}
                  />
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* 分页 */}
      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>

      {viewingTask ? (
        <TaskDetailDrawer task={viewingTask} onClose={() => setViewingTask(null)} />
      ) : null}
    </PageShell>
  )
}

function ProgressCell({ task }: { task: UnifiedFileTransferTask }) {
  const pct = Math.max(0, Math.min(100, Math.round(task.progress)))
  const color = task.result === 'failure' ? '#ff2d6f' : task.result === 'partial' ? '#ffaa00' : '#00f0ff'
  return (
    <div className="min-w-0">
      <div className="mb-1 flex items-center justify-between font-mono text-[10px] text-cyan-300/55">
        <span style={{ color }}>{pct}%</span>
        <span>
          {task.successCount}/{task.totalCount}
          {task.failCount > 0 && <span className="text-rose-300/80"> · ✗{task.failCount}</span>}
        </span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-cyan-500/10">
        <div
          className="h-full rounded-full transition-all"
          style={{ width: `${pct}%`, background: color, boxShadow: `0 0 6px ${color}` }}
        />
      </div>
    </div>
  )
}

function MiniStat({
  icon,
  label,
  value,
  color,
}: {
  icon: React.ReactNode
  label: string
  value: number
  color: string
}) {
  return (
    <div className="flex items-center gap-2.5">
      <span style={{ color }}>{icon}</span>
      <div>
        <div
          className="font-display text-xl font-bold leading-none"
          style={{ color, textShadow: `0 0 8px ${color}` }}
        >
          {value}
        </div>
        <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-cyan-300/55">
          {label}
        </div>
      </div>
    </div>
  )
}

function RowAction({
  title,
  icon,
  danger,
  disabled,
  onClick,
}: {
  title: string
  icon: React.ReactNode
  danger?: boolean
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className={`flex size-6 items-center justify-center rounded-sm border transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
        danger
          ? 'border-rose-500/30 text-rose-300/80 hover:border-rose-400/70 hover:text-rose-200'
          : 'border-cyan-500/25 text-cyan-300/75 hover:border-cyan-400/60 hover:text-cyan-100'
      }`}
    >
      {icon}
    </button>
  )
}

// #615 任务详情抽屉：v3 HUD 风格右侧 fixed-inset 抽屉（不新增路由，对齐 v1/v2）。
// 上半部分展示任务元 KV，下半部分按 taskId 拉「已选设备 / 执行明细」列表（10s 自动刷新）。
const DEVICE_STATUS_BADGE: Record<UnifiedFileTransferDeviceStatus, { status: string; label: string }> = {
  pending: { status: 'unknown', label: '待执行' },
  downloading: { status: 'active', label: '下载中' },
  uploading: { status: 'active', label: '上传中' },
  awaiting_tc: { status: 'warning', label: '等待 TC' },
  verifying: { status: 'active', label: '校验中' },
  suspended: { status: 'warning', label: '已挂起' },
  ended: { status: 'ok', label: '已完成' },
  failed: { status: 'critical', label: '失败' },
}

function TaskDetailDrawer({
  task,
  onClose,
}: {
  task: UnifiedFileTransferTask
  onClose: () => void
}) {
  const PAGE_SIZE = 20
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useUnifiedFileTransferDevices({
    taskId: task.id,
    page,
    pageSize: PAGE_SIZE,
  })
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const statusBadge = STATUS_BADGE[task.status]
  const resultBadge = task.result ? RESULT_BADGE[task.result] : undefined

  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true">
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden
      />
      <div className="relative flex h-full w-full max-w-[860px] flex-col overflow-hidden border-l border-cyan-500/30 bg-[#020611] shadow-[0_0_40px_rgba(0,240,255,0.15)]">
        <div className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <div className="min-w-0">
            <div className="truncate font-display text-base font-bold text-cyan-100" title={task.taskName}>
              {task.taskName || 'TASK DETAIL'}
            </div>
            <div className="truncate font-mono text-[10px] text-cyan-300/55">
              {task.id}
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="flex size-7 items-center justify-center rounded-sm border border-cyan-500/30 text-cyan-300/75 hover:border-cyan-400/60 hover:text-cyan-100"
            aria-label="关闭"
          >
            <X className="size-3.5" />
          </button>
        </div>

        <div className="grid grid-cols-2 gap-3 border-b border-cyan-500/10 px-4 py-3 sm:grid-cols-4">
          <DrawerField label="CATEGORY" value={task.categoryLabel || task.category || '—'} />
          <DrawerField label="TYPE" value={task.typeDisplayName || task.typeCode} />
          <DrawerField
            label="STATUS"
            value={<StatusBadge status={statusBadge.status} label={statusBadge.label} className="scale-90" />}
          />
          <DrawerField
            label="RESULT"
            value={
              resultBadge ? (
                <StatusBadge status={resultBadge.status} label={resultBadge.label} className="scale-90" />
              ) : (
                '—'
              )
            }
          />
          <DrawerField label="EXEC MODE" value={EXEC_MODE_LABEL[task.executionMode]} />
          <DrawerField label="OPERATOR" value={task.createUser || '—'} />
          <DrawerField label="CREATED" value={formatTime(task.createdAt)} />
          <DrawerField
            label="PROGRESS"
            value={`${task.successCount} / ${task.failCount} / ${task.totalCount}`}
          />
        </div>

        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <div className="mb-2 flex items-center justify-between">
            <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
              DEVICES · 已选设备 / 执行明细
            </div>
            <span className="font-mono text-[10px] text-cyan-300/55">
              TOTAL {total}
            </span>
          </div>

          {isLoading ? (
            <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
              <Loader2 className="size-3.5 animate-spin" />
              <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
            </div>
          ) : isError ? (
            <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-4 font-mono text-xs text-rose-300">
              FAILURE · 设备列表加载失败
            </div>
          ) : rows.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-12 text-cyan-300/55">
              <PackageSearch className="size-8 text-cyan-400/50" />
              <div className="font-mono text-[10px] uppercase tracking-[0.2em]">NO DEVICES · 暂无设备</div>
            </div>
          ) : (
            <div className="space-y-1.5">
              <div className="grid grid-cols-[1.4fr_1fr_1.4fr_120px_1fr_1.2fr] items-center gap-3 px-3 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
                <span>SN</span>
                <span>PRODUCT</span>
                <span>VERSION</span>
                <span>STATUS</span>
                <span>PROGRESS</span>
                <span>FAIL REASON</span>
              </div>
              {rows.map((d) => {
                const sb = DEVICE_STATUS_BADGE[d.status] ?? { status: 'unknown', label: d.status }
                return (
                  <div
                    key={d.id}
                    className="grid grid-cols-[1.4fr_1fr_1.4fr_120px_1fr_1.2fr] items-center gap-3 rounded-sm border border-cyan-500/10 bg-cyan-500/[0.02] px-3 py-2"
                  >
                    <span className="truncate font-mono text-xs text-cyan-100/85" title={d.deviceSn}>
                      {d.deviceSn || '—'}
                    </span>
                    <span className="truncate font-mono text-[11px] text-cyan-300/70">
                      {d.productName || d.productType || '—'}
                    </span>
                    <span className="truncate font-mono text-[11px] text-cyan-300/70">
                      {(d.currentVersion || '—') + (d.targetVersion ? ` → ${d.targetVersion}` : '')}
                    </span>
                    <StatusBadge status={sb.status} label={sb.label} className="scale-90" />
                    <div>
                      <div className="mb-1 flex items-center justify-between font-mono text-[10px] text-cyan-300/55">
                        <span>{Math.round(d.progress)}%</span>
                      </div>
                      <div className="h-1 overflow-hidden rounded-full bg-cyan-500/10">
                        <div
                          className="h-full rounded-full bg-cyan-400 transition-all"
                          style={{
                            width: `${Math.max(0, Math.min(100, Math.round(d.progress)))}%`,
                            boxShadow: '0 0 4px #00f0ff',
                          }}
                        />
                      </div>
                    </div>
                    {d.failureReason ? (
                      <span
                        className="truncate font-mono text-[11px] text-rose-300/85"
                        title={d.failureDetail || d.failureReason}
                      >
                        {d.failureReason}
                      </span>
                    ) : (
                      <span className="font-mono text-[11px] text-cyan-300/40">—</span>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {total > PAGE_SIZE ? (
            <div className="mt-3 flex items-center justify-between">
              <span className="font-mono text-[10px] text-cyan-300/55">
                PAGE {page} / {totalPages}
              </span>
              <div className="flex gap-2">
                <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
                  ◂ PREV
                </NeonButton>
                <NeonButton
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                >
                  NEXT ▸
                </NeonButton>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function DrawerField({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
        {label}
      </div>
      <div className="mt-1 truncate text-sm text-cyan-100/90">{value}</div>
    </div>
  )
}
