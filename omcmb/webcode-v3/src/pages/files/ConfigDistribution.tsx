import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  ChevronRight,
  Send,
  Boxes,
  Activity,
  CheckCircle2,
  Plus,
  X,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import {
  useUnifiedFileTransferDevices,
  useUnifiedFileTransferTasks,
  useUnifiedFileTransferDeviceCandidates,
  useCreateUnifiedFileTransferTask,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferDeviceStatus,
} from '@core/types/unifiedFileTransfer'

import { OverviewStat, Pager, Row, StateGate, NEON } from './_shared'

// 配置下发 = UFTE「配置恢复」类任务（ACS → CPE Download RPC 把配置文件推给设备）
const CATEGORY = 'config_restore'
const TYPE_CODE = 'CONFIG_RESTORE'
const PAGE_SIZE = 20

const DEV_STATUS: Record<UnifiedFileTransferDeviceStatus, { token: string; label: string }> = {
  pending: { token: 'unknown', label: '待执行' },
  downloading: { token: 'warning', label: '下发中' },
  uploading: { token: 'warning', label: '上传中' },
  awaiting_tc: { token: 'minor', label: '等待完成' },
  verifying: { token: 'minor', label: '校验中' },
  suspended: { token: 'off', label: '已暂停' },
  ended: { token: 'ok', label: '已完成' },
  failed: { token: 'error', label: '失败' },
}

export default function ConfigDistribution() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [creating, setCreating] = useState(false)
  const [picked, setPicked] = useState<Set<string>>(new Set())

  const params = useMemo(
    () => ({
      category: CATEGORY,
      page,
      pageSize: PAGE_SIZE,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useUnifiedFileTransferDevices(params)
  const tasks = useUnifiedFileTransferTasks({ category: CATEGORY, page: 1, pageSize: 100 })
  const candidates = useUnifiedFileTransferDeviceCandidates({
    category: CATEGORY,
    typeCode: TYPE_CODE,
    page: 1,
    pageSize: 50,
  })
  const createTask = useCreateUnifiedFileTransferTask()

  const rows: UnifiedFileTransferDeviceItem[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const taskItems = tasks.data?.items ?? []
  const runningTasks = taskItems.filter((t) => t.status === 'in_progress').length
  const totalSuccess = taskItems.reduce((s, t) => s + t.successCount, 0)
  const totalAttempt = taskItems.reduce((s, t) => s + t.totalCount, 0)
  const successPct = totalAttempt > 0 ? Math.round((totalSuccess / totalAttempt) * 100) : 0

  const candItems: UnifiedFileTransferDeviceItem[] = candidates.data?.items ?? []
  const togglePick = (id: string) =>
    setPicked((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const submit = () => {
    if (picked.size === 0) return
    createTask.mutate(
      {
        taskName: `配置下发-${new Date().toISOString().slice(0, 16).replace(/[-:T]/g, '')}`,
        typeCode: TYPE_CODE,
        deviceIds: [...picked],
        deviceCount: picked.size,
        executionMode: 'immediate',
      },
      {
        onSuccess: () => {
          setPicked(new Set())
          setCreating(false)
          void refetch()
          void tasks.refetch()
        },
      }
    )
  }

  return (
    <PageShell
      code="F06"
      title="CONFIG PUSH · 配置下发"
      subtitle="UFTE · CONFIG_RESTORE · ACS → CPE DOWNLOAD"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="任务 / 设备名 / SN"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <NeonButton icon={creating ? <X /> : <Plus />} onClick={() => setCreating((v) => !v)}>
            {creating ? '取消下发' : '新建下发'}
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <OverviewStat icon={<Boxes className="size-4" />} label="DEVICES" color={NEON.cyan} value={total.toLocaleString()} hint="参与下发设备数" />
        <OverviewStat icon={<Send className="size-4" />} label="TASKS" color={NEON.violet} value={tasks.data?.total ?? taskItems.length} hint="下发任务总数" />
        <OverviewStat icon={<Activity className="size-4" />} label="RUNNING" color={NEON.gold} value={runningTasks} hint="执行中任务" />
        <OverviewStat icon={<CheckCircle2 className="size-4" />} label="SUCCESS" color={NEON.green} value={`${successPct}%`} hint={`${totalSuccess}/${totalAttempt} 设备成功`} />
      </div>

      {creating ? (
        <GlassPanel strong title="NEW DISTRIBUTION · 选择目标设备" meta={`已选 ${picked.size}`} className="mb-3">
          <div className="p-3">
            <StateGate
              isLoading={candidates.isLoading}
              isError={candidates.isError}
              error={candidates.error}
              isEmpty={candItems.length === 0}
              loadingLabel="LOADING CANDIDATES…"
              emptyLabel="NO CANDIDATES · 暂无可下发设备"
            >
              <div className="grid max-h-56 grid-cols-1 gap-1.5 overflow-auto md:grid-cols-2 lg:grid-cols-3">
                {candItems.map((c: UnifiedFileTransferDeviceItem) => {
                  const on = picked.has(c.id)
                  return (
                    <button
                      key={c.id}
                      type="button"
                      onClick={() => togglePick(c.id)}
                      className={`flex items-center gap-2 rounded-sm border px-3 py-2 text-left transition-all ${
                        on
                          ? 'border-cyan-400/60 bg-cyan-400/10 shadow-[0_0_10px_rgba(0,240,255,0.25)]'
                          : 'border-cyan-500/12 hover:border-cyan-400/30'
                      }`}
                    >
                      <span className={`size-2 rounded-full ${on ? 'bg-cyan-400 shadow-[0_0_8px_currentColor]' : 'bg-cyan-500/30'}`} />
                      <span className="min-w-0">
                        <span className="block truncate text-xs text-cyan-100/90">{c.deviceName || c.deviceSn}</span>
                        <span className="block font-mono text-[10px] text-cyan-300/55">{c.deviceSn} · {c.productType || '—'}</span>
                      </span>
                    </button>
                  )
                })}
              </div>
            </StateGate>
            <div className="mt-3 flex justify-end">
              <NeonButton
                icon={<Send />}
                disabled={picked.size === 0 || createTask.isPending}
                onClick={submit}
              >
                {createTask.isPending ? '下发中…' : `下发到 ${picked.size} 台`}
              </NeonButton>
            </div>
          </div>
        </GlassPanel>
      ) : null}

      <GlassPanel strong title="CONFIG PUSH MANIFEST" meta={`${total} DEVICES`}>
        <div className="p-3">
          <div className="grid grid-cols-[2fr_1.2fr_1fr_1fr_1.4fr_90px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>DEVICE / SN</span>
            <span>TASK</span>
            <span>STATUS</span>
            <span>PROGRESS</span>
            <span>LAST REPORT</span>
            <span className="text-right">DETAIL</span>
          </div>
          <div className="mt-1.5 space-y-1.5">
            <StateGate
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              loadingLabel="SYNCING…"
              emptyLabel="NO CONFIG PUSH TASKS · 暂无配置下发记录"
            >
              {rows.map((d) => {
                const st = DEV_STATUS[d.status]
                return (
                  <Row
                    key={d.id}
                    color={st.token === 'error' ? NEON.rose : NEON.green}
                    className="grid-cols-[2fr_1.2fr_1fr_1fr_1.4fr_90px]"
                  >
                    <div className="min-w-0">
                      <div className="truncate font-display text-sm font-bold text-cyan-100" title={d.deviceName}>{d.deviceName || '—'}</div>
                      <div className="font-mono text-[10px] text-cyan-300/55">{d.deviceSn}</div>
                    </div>
                    <div className="min-w-0">
                      <div className="truncate text-xs text-cyan-100/85" title={d.taskName}>{d.taskName}</div>
                      <div className="font-mono text-[10px] text-cyan-300/45">{d.targetVersion || d.productType || '—'}</div>
                    </div>
                    <div>
                      <StatusBadge status={st.token} label={st.label} />
                      {d.failureReason ? (
                        <div className="mt-0.5 truncate font-mono text-[9px] text-rose-300/70" title={d.failureDetail || d.failureReason}>{d.failureReason}</div>
                      ) : null}
                    </div>
                    <div>
                      <div className="h-1.5 w-full overflow-hidden rounded-full bg-cyan-500/10">
                        <div
                          className="h-full rounded-full transition-all"
                          style={{ width: `${Math.max(0, Math.min(100, d.progress))}%`, background: NEON.green, boxShadow: `0 0 8px ${NEON.green}` }}
                        />
                      </div>
                      <div className="mt-0.5 font-mono text-[10px] text-cyan-300/55">{d.progress}%</div>
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/75">{formatTime(d.lastReportAt)}</div>
                    <div className="flex justify-end">
                      <NeonButton icon={<ChevronRight />} onClick={() => navigate(`/files/config-distribution/${d.taskId}`)}>
                        任务
                      </NeonButton>
                    </div>
                  </Row>
                )
              })}
            </StateGate>
          </div>
          <Pager
            page={page}
            totalPages={totalPages}
            total={total}
            pageSize={PAGE_SIZE}
            onPrev={() => setPage((p) => Math.max(1, p - 1))}
            onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
          />
        </div>
      </GlassPanel>
    </PageShell>
  )
}
