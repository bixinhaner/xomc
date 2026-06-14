import { useMemo, useState } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Package,
  Star,
  Rocket,
  Activity,
  ChevronRight,
  Pause,
  Play,
  Square,
  RotateCcw,
  X,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useSoftwareVersions,
  useUpgradeTasks,
  useToggleRecommend,
  useSubTasks,
  useSuspendTask,
  useResumeTask,
  useTerminateTask,
  useRetryTask,
  useDeleteTask,
  useAdvanceCanary,
  usePauseCanary,
  useResumeCanary,
  useAbortCanary,
} from '@core/hooks/api/useSoftware'
import type {
  SoftwareVersion,
  VersionStatus,
  UpgradeTaskInfo,
  TaskStatusType,
  SubTaskStatusType,
} from '@core/mock/data/software'

// ---------------------------------------------------------------------------
// 视觉常量
// ---------------------------------------------------------------------------

const NEON = {
  cyan: '#00f0ff',
  green: '#00ff88',
  violet: '#a855f7',
  amber: '#ffaa00',
  gold: '#ffd400',
  rose: '#ff2d6f',
  blue: '#5b9eff',
  dim: '#525a78',
} as const

const VERSION_STATUS: Record<VersionStatus, { label: string; color: string }> = {
  current: { label: '当前', color: NEON.green },
  beta: { label: '测试', color: NEON.blue },
  deprecated: { label: '弃用', color: NEON.amber },
  archived: { label: '归档', color: NEON.dim },
}

const TASK_STATUS: Record<TaskStatusType, { label: string; color: string }> = {
  pending: { label: '等待', color: NEON.dim },
  in_progress: { label: '执行中', color: NEON.cyan },
  suspended: { label: '已暂停', color: NEON.amber },
  ended: { label: '已结束', color: NEON.green },
}

const TASK_TYPE: Record<number, { label: string; color: string }> = {
  1: { label: '软件升级', color: NEON.green },
  2: { label: '版本回退', color: NEON.amber },
  4: { label: '补丁升级', color: NEON.blue },
  6: { label: 'FPGA 升级', color: NEON.violet },
  8: { label: '激活', color: NEON.cyan },
}

const RESULT_COLOR: Record<string, string> = {
  success: NEON.green,
  partial: NEON.amber,
  failed: NEON.rose,
  terminated: NEON.dim,
}

const SUB_STATUS: Record<SubTaskStatusType, { label: string; color: string }> = {
  pending: { label: '等待', color: NEON.dim },
  downloading: { label: '下载中', color: NEON.cyan },
  rebooting: { label: '重启中', color: NEON.violet },
  verifying: { label: '校验中', color: NEON.blue },
  completed: { label: '成功', color: NEON.green },
  failed: { label: '失败', color: NEON.rose },
  suspended: { label: '已暂停', color: NEON.amber },
  terminated: { label: '已终止', color: NEON.dim },
}

const TASK_TERMINAL: TaskStatusType = 'ended'

function progressOf(t: UpgradeTaskInfo): number {
  if (!t.totalCount) return 0
  return Math.round(((t.successCount + t.failCount) / t.totalCount) * 100)
}

// ---------------------------------------------------------------------------
// 主页面
// ---------------------------------------------------------------------------

export function SoftwarePage() {
  const [tab, setTab] = useState<'armory' | 'ota'>('armory')

  return (
    <PageShell
      code="F06"
      title="SOFTWARE · 软件武库"
      subtitle="FIRMWARE REGISTRY · OTA ORCHESTRATION · ROLLBACK"
      bare
      toolbar={
        <>
          <button
            type="button"
            onClick={() => setTab('armory')}
            className={`chip transition-all ${
              tab === 'armory' ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
            }`}
            style={{ color: NEON.cyan }}
          >
            <Package className="size-3.5" /> 版本武库
          </button>
          <button
            type="button"
            onClick={() => setTab('ota')}
            className={`chip transition-all ${
              tab === 'ota' ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
            }`}
            style={{ color: NEON.violet }}
          >
            <Rocket className="size-3.5" /> OTA 编排
          </button>
        </>
      }
    >
      {tab === 'armory' ? <ArmoryTab /> : <OtaTab />}
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 版本武库 Tab —— 固件版本注册表
// ---------------------------------------------------------------------------

function ArmoryTab() {
  const [page, setPage] = useState(1)
  const pageSize = 24
  const [keyword, setKeyword] = useState('')
  const [vendor, setVendor] = useState('')
  const [deviceType, setDeviceType] = useState('')
  const [status, setStatus] = useState<VersionStatus | ''>('')
  const [detail, setDetail] = useState<SoftwareVersion | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(vendor ? { vendor } : {}),
      ...(deviceType ? { deviceType } : {}),
      ...(status ? { status } : {}),
    }),
    [page, vendor, deviceType, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSoftwareVersions(params)
  const toggleRecommend = useToggleRecommend()

  const allItems = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 关键词为客户端过滤（后端按 vendor/deviceType/status 过滤）
  const items = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return allItems
    return allItems.filter(
      (v) =>
        v.versionCode.toLowerCase().includes(k) ||
        v.versionName.toLowerCase().includes(k) ||
        (v.fileName ?? '').toLowerCase().includes(k)
    )
  }, [allItems, keyword])

  // 概览统计（基于当前页）
  const stat = useMemo(() => {
    const recommended = allItems.filter((v) => v.recommend).length
    const current = allItems.filter((v) => v.status === 'current').length
    const beta = allItems.filter((v) => v.status === 'beta').length
    const totalBytes = allItems.reduce((acc, v) => acc + (v.fileSize || 0), 0)
    return { recommended, current, beta, totalBytes }
  }, [allItems])

  return (
    <>
      {/* 顶部统计 */}
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard
          label="版本总数 · REGISTRY"
          value={total.toLocaleString()}
          color={NEON.cyan}
          icon={<Package className="size-4" />}
        />
        <StatCard
          label="推荐版本 · RECOMMENDED"
          value={stat.recommended.toLocaleString()}
          color={NEON.gold}
          icon={<Star className="size-4" />}
        />
        <StatCard
          label="当前在用 · CURRENT"
          value={stat.current.toLocaleString()}
          color={NEON.green}
          icon={<Activity className="size-4" />}
        />
        <StatCard
          label="武库容量 · SIZE"
          value={formatBytes(stat.totalBytes)}
          color={NEON.violet}
          icon={<Package className="size-4" />}
        />
      </div>

      {/* 筛选 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-64 pl-9"
            placeholder="版本号 / 名称 / 文件名"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        <FilterSelect
          value={vendor}
          onChange={(v) => {
            setVendor(v)
            setPage(1)
          }}
          placeholder="厂商"
          options={['华为', '中兴', '爱立信', '诺基亚', 'Baicells']}
        />
        <FilterSelect
          value={deviceType}
          onChange={(v) => {
            setDeviceType(v)
            setPage(1)
          }}
          placeholder="制式"
          options={['eNB', 'gNB', 'RRU', 'AAU']}
        />
        {(['', 'current', 'beta', 'deprecated', 'archived'] as const).map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => {
              setStatus(s)
              setPage(1)
            }}
            className={`chip transition-all ${
              status === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
            }`}
            style={{ color: s ? VERSION_STATUS[s as VersionStatus].color : NEON.cyan }}
          >
            {s ? VERSION_STATUS[s as VersionStatus].label : 'ALL'}
          </button>
        ))}
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
        {isFetching ? (
          <span className="flex items-center gap-1.5 text-[11px] text-cyan-300/60">
            <Loader2 className="size-3 animate-spin" /> SYNC
          </span>
        ) : null}
      </div>

      {/* 版本网格 */}
      {isLoading ? (
        <Syncing label="SCANNING ARMORY…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : items.length === 0 ? (
        <EmptyBlock label="NO FIRMWARE IN ARMORY · 无固件" />
      ) : (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-4">
          {items.map((v) => (
            <VersionCard
              key={v.id}
              v={v}
              onDetail={() => setDetail(v)}
              onToggleRecommend={() => toggleRecommend.mutate(v.id)}
              toggling={toggleRecommend.isPending}
            />
          ))}
        </div>
      )}

      {/* 分页 */}
      <Pager page={page} totalPages={totalPages} pageSize={pageSize} total={total} onPage={setPage} />

      {/* 版本详情抽屉 */}
      {detail ? <VersionDetail v={detail} onClose={() => setDetail(null)} /> : null}
    </>
  )
}

function VersionCard({
  v,
  onDetail,
  onToggleRecommend,
  toggling,
}: {
  v: SoftwareVersion
  onDetail: () => void
  onToggleRecommend: () => void
  toggling: boolean
}) {
  const sc = VERSION_STATUS[v.status]
  return (
    <div
      className="glass group relative cursor-pointer overflow-hidden rounded-sm border-l-2 p-3 transition-all hover:bg-cyan-500/[0.04]"
      style={{ borderLeftColor: sc.color }}
      onClick={onDetail}
    >
      <div className="scanline" />
      <div className="relative">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="truncate font-mono text-sm font-bold text-cyan-100">{v.versionCode}</div>
            <div className="truncate font-mono text-[10px] text-cyan-300/55">{v.versionName}</div>
          </div>
          <button
            type="button"
            disabled={toggling}
            onClick={(e) => {
              e.stopPropagation()
              onToggleRecommend()
            }}
            title={v.recommend ? '取消推荐' : '设为推荐'}
            className="shrink-0 transition-transform hover:scale-110 disabled:opacity-40"
          >
            <Star
              className="size-4"
              style={{
                color: v.recommend ? NEON.gold : NEON.dim,
                fill: v.recommend ? NEON.gold : 'transparent',
                filter: v.recommend ? `drop-shadow(0 0 5px ${NEON.gold})` : undefined,
              }}
            />
          </button>
        </div>

        <div className="mt-2 flex flex-wrap items-center gap-1.5">
          <span className="chip" style={{ color: sc.color }}>
            <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
            {sc.label}
          </span>
          <span className="chip" style={{ color: NEON.blue }}>
            {v.deviceType || '—'}
          </span>
          <span className="chip" style={{ color: NEON.cyan }}>
            {v.vendor || v.manufacturer || '—'}
          </span>
        </div>

        <div className="mt-2 grid grid-cols-2 gap-1 font-mono text-[10px] text-cyan-300/65">
          <div>
            SIZE <span className="text-cyan-100/85">{formatBytes(v.fileSize)}</span>
          </div>
          <div>
            DATE{' '}
            <span className="text-cyan-100/85">
              {v.releaseDate ? formatTime(v.releaseDate).slice(0, 10) : '—'}
            </span>
          </div>
        </div>

        {v.releaseNotes ? (
          <div className="mt-1.5 line-clamp-2 text-[11px] leading-snug text-cyan-100/55">
            {v.releaseNotes}
          </div>
        ) : null}

        <div className="mt-2 flex items-center justify-between border-t border-cyan-500/10 pt-2">
          <span className="truncate font-mono text-[9px] text-cyan-300/40">
            {v.checksum ? `MD5 ${v.checksum.slice(0, 12)}…` : v.fileName || '—'}
          </span>
          <span className="flex items-center gap-0.5 font-mono text-[10px] text-cyan-300/55 opacity-60 transition-opacity group-hover:opacity-100">
            DETAIL <ChevronRight className="size-3" />
          </span>
        </div>
      </div>
    </div>
  )
}

function VersionDetail({ v, onClose }: { v: SoftwareVersion; onClose: () => void }) {
  const sc = VERSION_STATUS[v.status]
  return (
    <Drawer title="FIRMWARE DETAIL · 版本详情" onClose={onClose}>
      <div className="space-y-4 p-4">
        <div>
          <div className="font-mono text-base font-bold text-cyan-100 text-glow">{v.versionCode}</div>
          <div className="font-mono text-[11px] text-cyan-300/55">{v.versionName}</div>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <KV label="状态">
            <span className="chip" style={{ color: sc.color }}>
              {sc.label}
            </span>
          </KV>
          <KV label="推荐">
            <span style={{ color: v.recommend ? NEON.gold : NEON.dim }}>
              {v.recommend ? '是' : '否'}
            </span>
          </KV>
          <KV label="制式">{v.deviceType || '—'}</KV>
          <KV label="厂商">{v.vendor || v.manufacturer || '—'}</KV>
          <KV label="文件大小">{formatBytes(v.fileSize)}</KV>
          <KV label="发布日期">{v.releaseDate ? formatTime(v.releaseDate) : '—'}</KV>
          <KV label="最低硬件">{v.minHardwareVersion || '—'}</KV>
          <KV label="上传者">{v.uploader || '—'}</KV>
        </div>

        <KV label="文件名">
          <span className="font-mono text-[11px] text-cyan-100/80">{v.fileName || '—'}</span>
        </KV>
        <KV label="校验和">
          <span className="break-all font-mono text-[11px] text-cyan-100/80">{v.checksum || '—'}</span>
        </KV>

        {v.releaseNotes ? (
          <div>
            <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">
              RELEASE NOTES
            </div>
            <div className="rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] p-2 text-[12px] leading-relaxed text-cyan-100/80">
              {v.releaseNotes}
            </div>
          </div>
        ) : null}

        {v.features?.length ? (
          <TagGroup title="FEATURES · 特性" color={NEON.blue} items={v.features} />
        ) : null}
        {v.bugFixes?.length ? (
          <TagGroup title="BUGFIXES · 修复" color={NEON.green} items={v.bugFixes} />
        ) : null}
        {v.known_issues?.length ? (
          <TagGroup title="KNOWN ISSUES · 已知问题" color={NEON.amber} items={v.known_issues} />
        ) : null}
      </div>
    </Drawer>
  )
}

// ---------------------------------------------------------------------------
// OTA 编排 Tab —— 升级任务编排板
// ---------------------------------------------------------------------------

function OtaTab() {
  const [page, setPage] = useState(1)
  const pageSize = 12
  const [statusFilter, setStatusFilter] = useState<TaskStatusType | ''>('')
  const [keyword, setKeyword] = useState('')
  const [selectedTask, setSelectedTask] = useState<UpgradeTaskInfo | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(statusFilter ? { status: statusFilter } : {}),
    }),
    [page, statusFilter]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useUpgradeTasks(params)
  const allTasks = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const tasks = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return allTasks
    return allTasks.filter(
      (t) =>
        t.taskName.toLowerCase().includes(k) ||
        (t.fileName ?? '').toLowerCase().includes(k) ||
        t.productClass.toLowerCase().includes(k) ||
        t.createUser.toLowerCase().includes(k)
    )
  }, [allTasks, keyword])

  // 编排概览
  const overview = useMemo(() => {
    const active = allTasks.filter((t) => t.status !== TASK_TERMINAL).length
    const devTotal = allTasks.reduce((a, t) => a + t.totalCount, 0)
    const devSuccess = allTasks.reduce((a, t) => a + t.successCount, 0)
    const devFail = allTasks.reduce((a, t) => a + t.failCount, 0)
    const rate = devTotal > 0 ? (devSuccess / devTotal) * 100 : 0
    return { active, devTotal, devSuccess, devFail, rate }
  }, [allTasks])

  return (
    <>
      {/* 编排概览 */}
      <div className="mb-3 grid grid-cols-12 gap-3">
        <div className="col-span-12 grid grid-cols-4 gap-3 lg:col-span-9">
          <StatCard
            label="任务总数 · TASKS"
            value={total.toLocaleString()}
            color={NEON.cyan}
            icon={<Rocket className="size-4" />}
          />
          <StatCard
            label="执行中 · ACTIVE"
            value={overview.active.toLocaleString()}
            color={overview.active > 0 ? NEON.amber : NEON.dim}
            icon={<Activity className="size-4" />}
          />
          <StatCard
            label="升级成功 · OK"
            value={overview.devSuccess.toLocaleString()}
            color={NEON.green}
            icon={<Activity className="size-4" />}
          />
          <StatCard
            label="升级失败 · FAIL"
            value={overview.devFail.toLocaleString()}
            color={overview.devFail > 0 ? NEON.rose : NEON.dim}
            icon={<Activity className="size-4" />}
          />
        </div>
        <GlassPanel title="GLOBAL SUCCESS" meta="本页聚合" className="col-span-12 lg:col-span-3">
          <div className="flex items-center justify-center py-3">
            <RadialGauge
              value={overview.rate}
              label="SUCCESS"
              size={104}
              color={overview.rate >= 90 ? NEON.green : overview.rate >= 60 ? NEON.amber : NEON.rose}
            />
          </div>
        </GlassPanel>
      </div>

      {/* 筛选 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-64 pl-9"
            placeholder="任务名 / 固件 / 制式 / 操作员"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>
        {(['', 'pending', 'in_progress', 'suspended', 'ended'] as const).map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => {
              setStatusFilter(s)
              setPage(1)
            }}
            className={`chip transition-all ${
              statusFilter === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
            }`}
            style={{ color: s ? TASK_STATUS[s as TaskStatusType].color : NEON.cyan }}
          >
            {s ? TASK_STATUS[s as TaskStatusType].label : 'ALL'}
          </button>
        ))}
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
        {isFetching ? (
          <span className="flex items-center gap-1.5 text-[11px] text-cyan-300/60">
            <Loader2 className="size-3 animate-spin" /> LIVE · 5s
          </span>
        ) : null}
      </div>

      {/* 任务编排板 */}
      {isLoading ? (
        <Syncing label="LOADING OTA QUEUE…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : tasks.length === 0 ? (
        <EmptyBlock label="NO UPGRADE TASKS · 无升级任务" />
      ) : (
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
          {tasks.map((t) => (
            <TaskCard key={t.id} t={t} onInspect={() => setSelectedTask(t)} />
          ))}
        </div>
      )}

      <Pager page={page} totalPages={totalPages} pageSize={pageSize} total={total} onPage={setPage} />

      {/* 子任务下钻抽屉 */}
      {selectedTask ? (
        <SubTaskDrawer task={selectedTask} onClose={() => setSelectedTask(null)} />
      ) : null}
    </>
  )
}

function TaskCard({ t, onInspect }: { t: UpgradeTaskInfo; onInspect: () => void }) {
  const suspend = useSuspendTask()
  const resume = useResumeTask()
  const terminate = useTerminateTask()
  const retry = useRetryTask()
  const del = useDeleteTask()
  const advanceCanary = useAdvanceCanary()
  const pauseCanary = usePauseCanary()
  const resumeCanary = useResumeCanary()
  const abortCanary = useAbortCanary()

  const sc = TASK_STATUS[t.status]
  const tc = TASK_TYPE[t.taskType] ?? { label: `TYPE ${t.taskType}`, color: NEON.cyan }
  const pct = progressOf(t)
  const isCanary = t.strategy === 'canary'
  const stageStatus = t.stageStatus ?? 'pending'
  const canaryRunning = isCanary && stageStatus === 'running'
  const canaryPaused = isCanary && stageStatus === 'paused'
  const canaryDone = isCanary && (stageStatus === 'completed' || stageStatus === 'aborted')
  const totalStages = t.canaryStages?.length ?? 0
  const curStage = t.currentStage ?? 0
  const curPercent = t.canaryStages?.[Math.max(0, curStage - 1)]?.percent

  const ended = t.status === TASK_TERMINAL
  const barColor =
    t.result === 'failed' || t.result === 'terminated'
      ? NEON.rose
      : t.result === 'partial'
        ? NEON.amber
        : ended
          ? NEON.green
          : NEON.cyan

  const confirm = (msg: string, fn: () => void) => {
    if (typeof window === 'undefined' || window.confirm(msg)) fn()
  }

  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 p-3" style={{ borderLeftColor: sc.color }}>
      <div className="scanline" />
      <div className="relative">
        {/* 头部 */}
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="truncate font-display text-sm font-bold text-cyan-100">{t.taskName}</div>
            <div className="truncate font-mono text-[10px] text-cyan-300/55">
              {t.productClass || '—'} · FW {t.fileName || '—'} · BY {t.createUser || '—'}
            </div>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-1">
            <span className="chip" style={{ color: sc.color }}>
              <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
              {sc.label}
            </span>
            <span className="chip" style={{ color: tc.color }}>
              {tc.label}
            </span>
          </div>
        </div>

        {/* 灰度阶段条（仅 canary 任务） */}
        {isCanary && totalStages > 0 ? (
          <div className="mt-2 flex items-center gap-2">
            <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-violet-300/70">
              CANARY
            </span>
            <div className="flex flex-1 items-center gap-1">
              {t.canaryStages?.map((s, i) => {
                const reached = i < curStage
                const active = i === curStage - 1
                return (
                  <div
                    key={i}
                    className="h-1.5 flex-1 rounded-full"
                    style={{
                      background: reached ? NEON.violet : 'rgba(168,85,247,0.15)',
                      boxShadow: active ? `0 0 6px ${NEON.violet}` : undefined,
                    }}
                    title={`阶段 ${i + 1} · ${s.percent}% · 阈值 ${s.failureThreshold}%`}
                  />
                )
              })}
            </div>
            <span className="font-mono text-[10px] text-violet-200/80">
              {curStage}/{totalStages}
              {curPercent != null ? ` · ${curPercent}%` : ''} · {stageStatus}
            </span>
          </div>
        ) : null}

        {/* 进度条 */}
        <div className="mt-2.5">
          <div className="mb-1 flex items-center justify-between font-mono text-[10px] text-cyan-300/60">
            <span>
              PROGRESS{' '}
              <span className="text-cyan-100/85">
                {t.successCount + t.failCount}/{t.totalCount}
              </span>
            </span>
            <span style={{ color: barColor }}>{pct}%</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-cyan-500/10">
            <div
              className="h-full rounded-full transition-all"
              style={{
                width: `${pct}%`,
                background: barColor,
                boxShadow: `0 0 8px ${barColor}`,
              }}
            />
          </div>
          <div className="mt-1.5 flex items-center gap-3 font-mono text-[10px]">
            <span style={{ color: NEON.green }}>✓ {t.successCount} 成功</span>
            <span style={{ color: t.failCount > 0 ? NEON.rose : NEON.dim }}>✗ {t.failCount} 失败</span>
            <span className="text-cyan-300/55">并发 {t.maxConcurrent}</span>
            {t.result ? (
              <span style={{ color: RESULT_COLOR[t.result] ?? NEON.dim }}>
                结果 {t.result}
              </span>
            ) : null}
          </div>
        </div>

        {/* 时间 */}
        <div className="mt-2 grid grid-cols-2 gap-1 font-mono text-[10px] text-cyan-300/55">
          <div>开始 {t.startedAt ? formatTime(t.startedAt) : '—'}</div>
          <div>结束 {t.endedAt ? formatTime(t.endedAt) : '—'}</div>
        </div>

        {/* 控制 */}
        <div className="mt-2.5 flex flex-wrap items-center gap-2 border-t border-cyan-500/10 pt-2.5">
          <NeonButton className="!py-1 !px-2" onClick={onInspect}>
            <ChevronRight className="size-3.5" /> 设备明细
          </NeonButton>

          {/* 灰度专属控制 */}
          {canaryRunning ? (
            <>
              <NeonButton
                className="!py-1 !px-2"
                icon={<Play />}
                disabled={advanceCanary.isPending}
                onClick={() => confirm('推进到下一灰度阶段？', () => advanceCanary.mutate(t.id))}
              >
                推进阶段
              </NeonButton>
              <NeonButton
                className="!py-1 !px-2"
                icon={<Pause />}
                disabled={pauseCanary.isPending}
                onClick={() => pauseCanary.mutate(t.id)}
              >
                暂停灰度
              </NeonButton>
            </>
          ) : null}
          {canaryPaused ? (
            <NeonButton
              className="!py-1 !px-2"
              icon={<Play />}
              disabled={resumeCanary.isPending}
              onClick={() => resumeCanary.mutate(t.id)}
            >
              恢复灰度
            </NeonButton>
          ) : null}
          {isCanary && !canaryDone ? (
            <NeonButton
              tone="danger"
              className="!py-1 !px-2"
              icon={<Square />}
              disabled={abortCanary.isPending}
              onClick={() => confirm('中止灰度发布？已升级设备不回退。', () => abortCanary.mutate(t.id))}
            >
              中止灰度
            </NeonButton>
          ) : null}

          {/* 整任务控制（非灰度或通用） */}
          {!isCanary && t.status === 'in_progress' ? (
            <NeonButton
              className="!py-1 !px-2"
              icon={<Pause />}
              disabled={suspend.isPending}
              onClick={() => suspend.mutate(t.id)}
            >
              暂停
            </NeonButton>
          ) : null}
          {!isCanary && t.status === 'suspended' ? (
            <NeonButton
              className="!py-1 !px-2"
              icon={<Play />}
              disabled={resume.isPending}
              onClick={() => resume.mutate(t.id)}
            >
              恢复
            </NeonButton>
          ) : null}
          {!ended ? (
            <NeonButton
              tone="danger"
              className="!py-1 !px-2"
              icon={<Square />}
              disabled={terminate.isPending}
              onClick={() => confirm('终止整个升级任务？', () => terminate.mutate(t.id))}
            >
              终止
            </NeonButton>
          ) : null}
          {ended && t.failCount > 0 ? (
            <NeonButton
              className="!py-1 !px-2"
              icon={<RotateCcw />}
              disabled={retry.isPending}
              onClick={() => confirm('重试失败设备？', () => retry.mutate(t.id))}
            >
              重试失败
            </NeonButton>
          ) : null}
          {ended ? (
            <NeonButton
              tone="danger"
              className="!py-1 !px-2"
              disabled={del.isPending}
              onClick={() => confirm('删除该任务记录？', () => del.mutate(t.id))}
            >
              删除
            </NeonButton>
          ) : null}
        </div>
      </div>
    </div>
  )
}

function SubTaskDrawer({ task, onClose }: { task: UpgradeTaskInfo; onClose: () => void }) {
  const { data, isLoading, isError, error, isFetching } = useSubTasks(task.id, { page: 1, pageSize: 200 })
  const subs = data?.items ?? []

  const counts = useMemo(() => {
    const c: Record<string, number> = {}
    for (const s of subs) c[s.status] = (c[s.status] ?? 0) + 1
    return c
  }, [subs])

  return (
    <Drawer
      title={`SUB-TASKS · ${task.taskName}`}
      meta={isFetching ? 'LIVE · 3s' : undefined}
      onClose={onClose}
      wide
    >
      <div className="space-y-3 p-4">
        {/* 摘要 */}
        <div className="grid grid-cols-4 gap-2">
          <MiniStat label="TOTAL" value={subs.length} color={NEON.cyan} />
          <MiniStat label="成功" value={counts.completed ?? 0} color={NEON.green} />
          <MiniStat label="进行" value={(counts.downloading ?? 0) + (counts.rebooting ?? 0) + (counts.verifying ?? 0)} color={NEON.amber} />
          <MiniStat label="失败" value={counts.failed ?? 0} color={NEON.rose} />
        </div>

        {isLoading ? (
          <Syncing label="LOADING SUB-TASKS…" />
        ) : isError ? (
          <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
        ) : subs.length === 0 ? (
          <EmptyBlock label="NO SUB-TASKS · 无设备子任务" />
        ) : (
          <div className="space-y-1.5">
            {subs.map((s) => {
              const ss = SUB_STATUS[s.status] ?? { label: s.status, color: NEON.dim }
              return (
                <div
                  key={s.id}
                  className="fleet-row grid grid-cols-[10px_1.6fr_1fr_120px] items-center gap-3 rounded-sm px-3 py-2"
                  style={{ ['--row-color' as never]: ss.color }}
                >
                  <span
                    className="size-2 rounded-full"
                    style={{ background: ss.color, boxShadow: `0 0 6px ${ss.color}` }}
                  />
                  <div className="min-w-0">
                    <div className="truncate font-mono text-xs text-cyan-100">{s.deviceSn || s.deviceId}</div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/55">
                      {s.oriVersion || '—'} → {s.destVersion || '—'}
                      {s.retryCount > 0 ? ` · 重试 ${s.retryCount}/${s.maxRetries}` : ''}
                    </div>
                    {s.failureReason || s.errorMessage ? (
                      <div className="truncate font-mono text-[10px] text-rose-300/70">
                        {s.failureReason || s.errorMessage}
                      </div>
                    ) : null}
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/65">
                    {s.completedAt
                      ? formatTime(s.completedAt)
                      : s.startedAt
                        ? formatTime(s.startedAt)
                        : '—'}
                  </div>
                  <div className="text-right">
                    <span className="chip" style={{ color: ss.color }}>
                      <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
                      {ss.label}
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </Drawer>
  )
}

// ---------------------------------------------------------------------------
// 通用小组件（模块内）
// ---------------------------------------------------------------------------

function StatCard({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: string
  color: string
  icon: React.ReactNode
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="scanline" />
      <div className="relative flex items-center justify-between">
        <div className="min-w-0">
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">{label}</div>
          <div
            className="font-display text-2xl font-bold leading-tight text-glow"
            style={{ color }}
          >
            {value}
          </div>
        </div>
        <span style={{ color }}>{icon}</span>
      </div>
    </div>
  )
}

function MiniStat({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="glass rounded-sm px-3 py-2 text-center">
      <div className="font-mono text-[9px] uppercase tracking-[0.2em] text-cyan-300/55">{label}</div>
      <div className="font-display text-lg font-bold" style={{ color, textShadow: `0 0 6px ${color}` }}>
        {value}
      </div>
    </div>
  )
}

function FilterSelect({
  value,
  onChange,
  placeholder,
  options,
}: {
  value: string
  onChange: (v: string) => void
  placeholder: string
  options: string[]
}) {
  return (
    <select
      className="neon-input w-32 cursor-pointer"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    >
      <option value="">{placeholder} · 全部</option>
      {options.map((o) => (
        <option key={o} value={o}>
          {o}
        </option>
      ))}
    </select>
  )
}

function KV({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/50">{label}</span>
      <span className="text-[13px] text-cyan-100/85">{children}</span>
    </div>
  )
}

function TagGroup({ title, color, items }: { title: string; color: string; items: string[] }) {
  return (
    <div>
      <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/55">{title}</div>
      <div className="flex flex-wrap gap-1.5">
        {items.map((it, i) => (
          <span key={i} className="chip" style={{ color }}>
            {it}
          </span>
        ))}
      </div>
    </div>
  )
}

function Drawer({
  title,
  meta,
  onClose,
  children,
  wide,
}: {
  title: string
  meta?: string
  onClose: () => void
  children: React.ReactNode
  wide?: boolean
}) {
  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/55 backdrop-blur-sm" onClick={onClose} />
      <div
        className={`warp-in glass-strong relative h-full overflow-auto border-l border-cyan-500/25 ${
          wide ? 'w-[640px] max-w-[92vw]' : 'w-[460px] max-w-[92vw]'
        }`}
      >
        <div className="sticky top-0 z-10 flex items-center justify-between border-b border-cyan-500/20 bg-[#03050d]/80 px-4 py-3 backdrop-blur-md">
          <div className="flex items-center gap-2">
            <span className="size-1.5 rounded-full bg-cyan-400 shadow-[0_0_8px_currentColor]" />
            <span className="truncate font-mono text-[12px] uppercase tracking-[0.18em] text-cyan-200">
              {title}
            </span>
          </div>
          <div className="flex items-center gap-3">
            {meta ? <span className="font-mono text-[10px] text-cyan-300/50">{meta}</span> : null}
            <button type="button" onClick={onClose} className="text-cyan-300/60 hover:text-cyan-200">
              <X className="size-4" />
            </button>
          </div>
        </div>
        {children}
      </div>
    </div>
  )
}

function Syncing({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">{label}</span>
    </div>
  )
}

function ErrorBlock({ msg }: { msg: string }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      <div className="font-bold uppercase tracking-[0.2em]">FAILURE</div>
      <div className="mt-1 text-rose-200/80">{msg}</div>
    </div>
  )
}

function EmptyBlock({ label }: { label: string }) {
  return (
    <div className="border border-cyan-500/15 px-4 py-12 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
      {label}
    </div>
  )
}

function Pager({
  page,
  totalPages,
  pageSize,
  total,
  onPage,
}: {
  page: number
  totalPages: number
  pageSize: number
  total: number
  onPage: (fn: (p: number) => number) => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton onClick={() => onPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages}>
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}
