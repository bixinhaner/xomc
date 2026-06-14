import { useMemo, useState } from 'react'
import {
  Activity,
  Boxes,
  Database,
  FileStack,
  Loader2,
  Power,
  RefreshCcw,
  Search,
  SignalHigh,
  Trash2,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import {
  useMRFileDevices,
  useBatchDeleteMRFiles,
  useMRIndicators,
  useMRMappings,
  useToggleMRMapping,
} from '@core/hooks/api/useMR'
import {
  useMRTasks,
  useStopMRTask,
  useDeleteMRTask,
} from '@core/hooks/api/useMrTasks'
import type { MRTask, MRTaskStatus } from '@core/types/mrTask'
import type { MRFileDeviceItem } from '@core/services/api/mrApi'

// ───────────────────────────── 视图切换 ─────────────────────────────

type View = 'files' | 'tasks' | 'indicators' | 'mappings'

const VIEWS: { key: View; label: string; en: string }[] = [
  { key: 'files', label: '文件', en: 'FILES' },
  { key: 'tasks', label: '任务', en: 'TASKS' },
  { key: 'indicators', label: '指标', en: 'INDICATORS' },
  { key: 'mappings', label: '采集映射', en: 'MAPPINGS' },
]

const PAGE_SIZE = 20

const MR_TYPE_COLOR: Record<string, string> = {
  MRO: '#00f0ff',
  MRE: '#00ff88',
  MRS: '#ffaa00',
}

const TASK_STATUS_META: Record<MRTaskStatus, { label: string; badge: string }> = {
  waitting: { label: '待开启', badge: 'unknown' },
  on: { label: '上报中', badge: 'active' },
  off: { label: '已结束', badge: 'off' },
  suspend: { label: '已暂停', badge: 'warning' },
  termination: { label: '已终止', badge: 'major' },
}

// ───────────────────────────── 主页面 ─────────────────────────────

export function MRPage() {
  const [view, setView] = useState<View>('files')

  // 顶部概览用文件设备聚合 + 任务列表两路真实数据派生统计
  const overviewFiles = useMRFileDevices({ page: 1, pageSize: 1 })
  const overviewTasks = useMRTasks({ page: 1, pageSize: 100 })

  const taskItems = overviewTasks.data?.items ?? []
  const activeTaskCount = taskItems.filter((t) => t.taskStatus === 'on').length
  const totalDeviceCount = overviewFiles.data?.total ?? 0
  const totalTaskCount = overviewTasks.data?.total ?? 0
  const reportingNow = useMemo(
    () => taskItems.filter((t) => t.taskStatus === 'on' || t.taskStatus === 'waitting').length,
    [taskItems]
  )

  const isFetching = overviewFiles.isFetching || overviewTasks.isFetching

  return (
    <PageShell
      code="F05"
      title="MR ARRAY · 测量报告"
      subtitle="MRO / MRS / MRE FILE INTAKE · 10s AUTO-REFRESH"
      isFetching={isFetching}
      bare
      toolbar={VIEWS.map((v) => (
        <button
          key={v.key}
          type="button"
          onClick={() => setView(v.key)}
          className={`chip transition-all ${
            view === v.key
              ? 'shadow-[0_0_10px_currentColor]'
              : 'opacity-55 hover:opacity-100'
          }`}
          style={{ color: view === v.key ? '#00f0ff' : '#6b86b6' }}
        >
          {v.en}
        </button>
      ))}
    >
      {/* 概览统计带 —— 全视图共享 */}
      <div className="mb-3 grid grid-cols-4 gap-3">
        <OverviewStat
          label="DEVICES · 上报设备"
          value={totalDeviceCount.toLocaleString()}
          color="#00f0ff"
          icon={<Boxes className="size-4" />}
          trend={[12, 18, 16, 24, 30, 28, 36, 40, 44, 52]}
        />
        <OverviewStat
          label="TASKS · 测量任务"
          value={totalTaskCount.toLocaleString()}
          color="#a855f7"
          icon={<FileStack className="size-4" />}
          trend={[2, 3, 3, 5, 4, 6, 7, 7, 8, 9]}
        />
        <OverviewStat
          label="ACTIVE · 上报中"
          value={activeTaskCount.toLocaleString()}
          color="#00ff88"
          icon={<SignalHigh className="size-4" />}
          trend={[1, 1, 2, 2, 3, 3, 4, 4, 5, 5]}
        />
        <OverviewStat
          label="PENDING+ON · 调度中"
          value={reportingNow.toLocaleString()}
          color={reportingNow > 0 ? '#ffaa00' : '#525a78'}
          icon={<Activity className="size-4" />}
          trend={[3, 2, 4, 3, 5, 4, 6, 5, 7, 6]}
        />
      </div>

      {view === 'files' && <FilesView />}
      {view === 'tasks' && <TasksView />}
      {view === 'indicators' && <IndicatorsView />}
      {view === 'mappings' && <MappingsView />}
    </PageShell>
  )
}

// ───────────────────────────── 文件视图 ─────────────────────────────

function FilesView() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [siteName, setSiteName] = useState('')
  const [productClass, setProductClass] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      keyword: keyword.trim() || undefined,
      siteName: siteName.trim() || undefined,
      productClass: productClass.trim() || undefined,
    }),
    [page, keyword, siteName, productClass]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMRFileDevices(params)
  const batchDelete = useBatchDeleteMRFiles()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const toggleRow = (sn: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(sn)) next.delete(sn)
      else next.add(sn)
      return next
    })
  }

  const handleDelete = () => {
    if (selected.size === 0) return
    const sns = [...selected]
    if (!window.confirm(`确认删除 ${sns.length} 台设备的全部 MR 文件？该操作不可恢复。`)) return
    batchDelete.mutate(sns, {
      onSuccess: () => {
        setSelected(new Set())
        void refetch()
      },
    })
  }

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <SearchInput placeholder="设备 SN" value={keyword} onChange={(v) => { setKeyword(v); setPage(1) }} />
        <SearchInput placeholder="基站名称" value={siteName} onChange={(v) => { setSiteName(v); setPage(1) }} />
        <SearchInput placeholder="产品类" value={productClass} onChange={(v) => { setProductClass(v); setPage(1) }} />
        <NeonButton
          tone="danger"
          icon={<Trash2 />}
          disabled={selected.size === 0 || batchDelete.isPending}
          onClick={handleDelete}
        >
          DELETE ({selected.size})
        </NeonButton>
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
        {isFetching ? <SyncTag /> : null}
      </div>

      <StateWrap
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={rows.length === 0}
        emptyIcon={<Database className="size-10 text-cyan-400/50" />}
        emptyText="NO MR FILES · 暂无文件"
      >
        <div className="space-y-1.5">
          {rows.map((d: MRFileDeviceItem) => {
            const checked = selected.has(d.deviceSn)
            return (
              <div
                key={d.deviceSn}
                className="fleet-row grid grid-cols-[28px_1.6fr_1.4fr_1fr_1.4fr_100px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: d.reporting ? '#00ff88' : '#00f0ff' }}
              >
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() => toggleRow(d.deviceSn)}
                  className="size-3.5 accent-cyan-400"
                />
                <div className="min-w-0">
                  <div className="truncate font-mono text-sm text-cyan-100">{d.deviceSn}</div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {d.siteName || '—'}
                  </div>
                </div>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/80">{d.productClass || '—'}</div>
                  <div className="font-mono text-[10px] text-cyan-300/55">PRODUCT CLASS</div>
                </div>
                <div>
                  <div
                    className="font-display text-lg font-bold leading-none"
                    style={{ color: '#00f0ff', textShadow: '0 0 6px #00f0ff' }}
                  >
                    {Number(d.fileCount ?? 0).toLocaleString()}
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/55">FILES</div>
                </div>
                <div className="font-mono text-[10px] leading-relaxed text-cyan-300/70">
                  <div>FIRST · {formatTime(d.firstCollectTime)}</div>
                  <div>LAST · {formatTime(d.lastCollectTime)}</div>
                </div>
                <div className="text-right">
                  <StatusBadge
                    status={d.reporting ? 'active' : 'off'}
                    label={d.reporting ? '上报中' : '已停止'}
                  />
                </div>
              </div>
            )
          })}
        </div>
      </StateWrap>

      <Pager page={page} totalPages={totalPages} total={total} onChange={setPage} />
    </>
  )
}

// ───────────────────────────── 任务视图 ─────────────────────────────

function TasksView() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<MRTaskStatus | ''>('')

  const filter = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      keyword: keyword.trim() || undefined,
      status: status || undefined,
    }),
    [page, keyword, status]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMRTasks(filter)
  const stopTask = useStopMRTask()
  const deleteTask = useDeleteMRTask()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const handleStop = (task: MRTask) => {
    if (!window.confirm(`确认停止任务「${task.taskName}」？停止后将下发关闭指令。`)) return
    stopTask.mutate(task.taskId)
  }
  const handleDelete = (task: MRTask) => {
    if (!window.confirm(`确认删除任务「${task.taskName}」？`)) return
    deleteTask.mutate(task.taskId)
  }

  const STATUS_FILTERS: (MRTaskStatus | '')[] = ['', 'waitting', 'on', 'off', 'termination']

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <SearchInput placeholder="任务名" value={keyword} onChange={(v) => { setKeyword(v); setPage(1) }} />
        {STATUS_FILTERS.map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => { setStatus(s); setPage(1) }}
            className={`chip transition-all ${
              status === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
            }`}
            style={{ color: s ? '#00f0ff' : '#6b86b6' }}
          >
            {s ? TASK_STATUS_META[s].label : 'ALL'}
          </button>
        ))}
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
        {isFetching ? <SyncTag /> : null}
      </div>

      <StateWrap
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={rows.length === 0}
        emptyIcon={<FileStack className="size-10 text-cyan-400/50" />}
        emptyText="NO TASKS · 暂无测量任务"
      >
        <div className="space-y-1.5">
          {rows.map((task: MRTask) => {
            const meta = TASK_STATUS_META[task.taskStatus] ?? { label: task.taskStatus, badge: 'unknown' }
            const stoppable = task.taskStatus === 'waitting' || task.taskStatus === 'on'
            const deletable = task.taskStatus === 'off' || task.taskStatus === 'termination'
            return (
              <div
                key={task.taskId}
                className="fleet-row grid grid-cols-[2fr_1.2fr_1fr_1.6fr_160px] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: task.taskStatus === 'on' ? '#00ff88' : '#00f0ff' }}
              >
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {task.taskName}
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/55">
                    BY {task.creator || '—'} · {task.targetDeviceSns?.length ?? 0} TARGETS
                  </div>
                </div>
                <div className="flex flex-wrap gap-1">
                  {String(task.mrType || '')
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean)
                    .map((tp) => (
                      <span
                        key={tp}
                        className="chip"
                        style={{ color: MR_TYPE_COLOR[tp] ?? '#6b86b6' }}
                      >
                        {tp}
                      </span>
                    ))}
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">
                  <div>{task.reportPeriod} min</div>
                  <div className="text-cyan-300/45">P{task.statisPeriod}</div>
                </div>
                <div className="font-mono text-[10px] leading-relaxed text-cyan-300/70">
                  <div>START · {formatTime(task.startTime)}</div>
                  <div>END · {task.endTime ? formatTime(task.endTime) : '不限'}</div>
                </div>
                <div className="flex items-center justify-end gap-2">
                  <StatusBadge status={meta.badge} label={meta.label} />
                  <button
                    type="button"
                    disabled={!stoppable || stopTask.isPending}
                    onClick={() => handleStop(task)}
                    title="停止"
                    className="text-cyan-300/70 transition-colors enabled:hover:text-amber-300 disabled:opacity-25"
                  >
                    <Power className="size-3.5" />
                  </button>
                  <button
                    type="button"
                    disabled={!deletable || deleteTask.isPending}
                    onClick={() => handleDelete(task)}
                    title="删除"
                    className="text-cyan-300/70 transition-colors enabled:hover:text-rose-400 disabled:opacity-25"
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      </StateWrap>

      <Pager page={page} totalPages={totalPages} total={total} onChange={setPage} />
    </>
  )
}

// ───────────────────────────── 指标视图 ─────────────────────────────

function IndicatorsView() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

  const { data, isLoading, isError, error, isFetching, refetch } = useMRIndicators({
    page,
    pageSize: PAGE_SIZE,
  })

  const all = data?.items ?? []
  const rows = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return all
    return all.filter(
      (i) =>
        i.indicatorName.toLowerCase().includes(kw) ||
        i.indicatorCode.toLowerCase().includes(kw)
    )
  }, [all, keyword])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <SearchInput
          placeholder="指标名 / 编码（本页过滤）"
          value={keyword}
          onChange={(v) => setKeyword(v)}
        />
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
        {isFetching ? <SyncTag /> : null}
      </div>

      <StateWrap
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={rows.length === 0}
        emptyIcon={<SignalHigh className="size-10 text-cyan-400/50" />}
        emptyText="NO INDICATORS · 暂无指标"
      >
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
          {rows.map((ind) => (
            <GlassPanel key={ind.id} className="p-0">
              <div className="flex flex-col gap-2 p-3.5">
                <div className="flex items-baseline justify-between gap-2">
                  <span
                    className="font-mono text-sm font-bold text-glow"
                    style={{ color: '#00f0ff' }}
                  >
                    {ind.indicatorCode}
                  </span>
                  {ind.unit ? (
                    <span className="font-mono text-[10px] text-cyan-300/55">{ind.unit}</span>
                  ) : null}
                </div>
                <div className="text-xs text-cyan-100/85">{ind.indicatorName}</div>
                {ind.description ? (
                  <div className="line-clamp-2 text-[11px] leading-relaxed text-cyan-300/55">
                    {ind.description}
                  </div>
                ) : null}
                <div className="flex items-center justify-between border-t border-cyan-500/12 pt-2">
                  {ind.category ? (
                    <span className="chip" style={{ color: '#a855f7' }}>
                      {ind.category}
                    </span>
                  ) : (
                    <span />
                  )}
                  <span className="font-mono text-[10px] text-cyan-300/55">
                    {ind.valueRange[0]} ~ {ind.valueRange[1]}
                  </span>
                </div>
              </div>
            </GlassPanel>
          ))}
        </div>
      </StateWrap>

      <Pager page={page} totalPages={totalPages} total={total} onChange={setPage} />
    </>
  )
}

// ───────────────────────────── 采集映射视图 ─────────────────────────────

function MappingsView() {
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [enabled, setEnabled] = useState<'' | 'true' | 'false'>('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      deviceSn: deviceSn.trim() || undefined,
      enabled: enabled === '' ? undefined : enabled === 'true',
    }),
    [page, deviceSn, enabled]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMRMappings(params)
  const toggle = useToggleMRMapping()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <SearchInput placeholder="设备 SN" value={deviceSn} onChange={(v) => { setDeviceSn(v); setPage(1) }} />
        {(['', 'true', 'false'] as const).map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => { setEnabled(s); setPage(1) }}
            className={`chip transition-all ${
              enabled === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
            }`}
            style={{ color: s ? (s === 'true' ? '#00ff88' : '#525a78') : '#6b86b6' }}
          >
            {s === '' ? 'ALL' : s === 'true' ? '已启用' : '已禁用'}
          </button>
        ))}
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
        {isFetching ? <SyncTag /> : null}
      </div>

      <StateWrap
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={rows.length === 0}
        emptyIcon={<Boxes className="size-10 text-cyan-400/50" />}
        emptyText="NO MAPPINGS · 暂无采集映射"
      >
        <div className="space-y-1.5">
          {rows.map((m) => (
            <div
              key={m.id}
              className="fleet-row grid grid-cols-[1.6fr_1.6fr_1fr_1.4fr_90px] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{ ['--row-color' as never]: m.enabled ? '#00ff88' : '#525a78' }}
            >
              <div className="min-w-0">
                <div className="truncate font-mono text-sm text-cyan-100">{m.deviceSn}</div>
                <div className="truncate font-mono text-[10px] text-cyan-300/55">
                  {m.deviceName || '—'}
                </div>
              </div>
              <div className="min-w-0">
                <div className="truncate text-xs text-cyan-100/80">{m.cellName || m.cellId}</div>
                <div className="truncate font-mono text-[10px] text-cyan-300/55">CELL {m.cellId}</div>
              </div>
              <div className="font-mono text-[11px] text-cyan-300/75">
                <div>{m.samplingInterval} min</div>
                <div className="text-cyan-300/45">{Number(m.totalRecords ?? 0).toLocaleString()} rec</div>
              </div>
              <div className="font-mono text-[10px] text-cyan-300/70">
                LAST · {m.lastCollectTime ? formatTime(m.lastCollectTime) : '从未采集'}
              </div>
              <div className="flex items-center justify-end">
                <button
                  type="button"
                  disabled={toggle.isPending}
                  onClick={() => toggle.mutate({ id: m.id, enabled: !m.enabled })}
                  className={`chip transition-all disabled:opacity-40 ${
                    m.enabled ? 'shadow-[0_0_8px_currentColor]' : 'opacity-70'
                  }`}
                  style={{ color: m.enabled ? '#00ff88' : '#525a78' }}
                >
                  {m.enabled ? 'ON' : 'OFF'}
                </button>
              </div>
            </div>
          ))}
        </div>
      </StateWrap>

      <Pager page={page} totalPages={totalPages} total={total} onChange={setPage} />
    </>
  )
}

// ───────────────────────────── 共享子组件 ─────────────────────────────

function OverviewStat({
  label,
  value,
  color,
  icon,
  trend,
}: {
  label: string
  value: string
  color: string
  icon: React.ReactNode
  trend: number[]
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative flex items-stretch gap-3 p-4">
        <div className="flex flex-col items-center justify-center border-r border-cyan-500/15 pr-3">
          <span style={{ color }}>{icon}</span>
        </div>
        <div className="flex flex-1 flex-col">
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
            {label}
          </div>
          <div
            className="font-display text-3xl font-bold leading-tight text-glow"
            style={{ color }}
          >
            {value}
          </div>
        </div>
        <Sparkline data={trend} color={color} width={84} height={34} />
      </div>
    </div>
  )
}

function SearchInput({
  placeholder,
  value,
  onChange,
}: {
  placeholder: string
  value: string
  onChange: (v: string) => void
}) {
  return (
    <div className="relative">
      <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
      <input
        className="neon-input w-52 pl-9"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  )
}

function SyncTag() {
  return (
    <span className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
      <Loader2 className="size-3 animate-spin" />
      SYNC
    </span>
  )
}

function StateWrap({
  isLoading,
  isError,
  error,
  isEmpty,
  emptyIcon,
  emptyText,
  children,
}: {
  isLoading: boolean
  isError: boolean
  error: unknown
  isEmpty: boolean
  emptyIcon: React.ReactNode
  emptyText: string
  children: React.ReactNode
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
        <Loader2 className="size-4 animate-spin" />
        <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
      </div>
    )
  }
  if (isError) {
    return (
      <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
        FAILURE · {error instanceof Error ? error.message : '未知错误'}
      </div>
    )
  }
  if (isEmpty) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        {emptyIcon}
        <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
          {emptyText}
        </div>
      </div>
    )
  }
  return <>{children}</>
}

function Pager({
  page,
  totalPages,
  total,
  onChange,
}: {
  page: number
  totalPages: number
  total: number
  onChange: (p: number) => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onChange(Math.max(1, page - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onChange(Math.min(totalPages, page + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}
