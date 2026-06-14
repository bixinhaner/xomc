import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  ChevronRight,
  Download,
  Archive,
  Boxes,
  Activity,
  CheckCircle2,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import {
  useUnifiedFileTransferDevices,
  useUnifiedFileTransferTasks,
} from '@core/hooks/api/useUnifiedFileTransfer'
import type {
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferDeviceStatus,
} from '@core/types/unifiedFileTransfer'

import { OverviewStat, Pager, Row, StateGate, NEON } from './_shared'

// 配置文件拉取 = UFTE「配置备份」类任务（CPE Upload RPC 上传运行配置到 ACS）
const CATEGORY = 'config_backup'
const PAGE_SIZE = 20

// 设备级传输状态 → StatusBadge token + 中文
const DEV_STATUS: Record<UnifiedFileTransferDeviceStatus, { token: string; label: string }> = {
  pending: { token: 'unknown', label: '待执行' },
  downloading: { token: 'warning', label: '下载中' },
  uploading: { token: 'warning', label: '上传中' },
  awaiting_tc: { token: 'minor', label: '等待完成' },
  verifying: { token: 'minor', label: '校验中' },
  suspended: { token: 'off', label: '已暂停' },
  ended: { token: 'ok', label: '已完成' },
  failed: { token: 'error', label: '失败' },
}

export default function ConfigRetrieval() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')

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

  const rows: UnifiedFileTransferDeviceItem[] = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const taskItems = tasks.data?.items ?? []
  const runningTasks = taskItems.filter((t) => t.status === 'in_progress').length
  const totalSuccess = taskItems.reduce((s, t) => s + t.successCount, 0)
  const totalAttempt = taskItems.reduce((s, t) => s + t.totalCount, 0)
  const successPct = totalAttempt > 0 ? Math.round((totalSuccess / totalAttempt) * 100) : 0

  return (
    <PageShell
      code="F06"
      title="CONFIG PULL · 配置拉取"
      subtitle="UFTE · CONFIG_BACKUP · CPE → ACS UPLOAD"
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
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <OverviewStat
          icon={<Boxes className="size-4" />}
          label="DEVICES"
          color={NEON.cyan}
          value={total.toLocaleString()}
          hint="参与拉取设备数"
        />
        <OverviewStat
          icon={<Archive className="size-4" />}
          label="TASKS"
          color={NEON.violet}
          value={tasks.data?.total ?? taskItems.length}
          hint="备份任务总数"
        />
        <OverviewStat
          icon={<Activity className="size-4" />}
          label="RUNNING"
          color={NEON.gold}
          value={runningTasks}
          hint="执行中任务"
        />
        <OverviewStat
          icon={<CheckCircle2 className="size-4" />}
          label="SUCCESS"
          color={NEON.green}
          value={`${successPct}%`}
          hint={`${totalSuccess}/${totalAttempt} 设备成功`}
        />
      </div>

      <GlassPanel strong title="CONFIG PULL MANIFEST" meta={`${total} DEVICES`}>
        <div className="p-3">
          <div className="grid grid-cols-[2fr_1.2fr_1fr_1fr_1.4fr_110px] items-center gap-3 border-b border-cyan-500/15 px-3 pb-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            <span>DEVICE / SN</span>
            <span>TASK</span>
            <span>STATUS</span>
            <span>PROGRESS</span>
            <span>LAST REPORT</span>
            <span className="text-right">ACTIONS</span>
          </div>

          <div className="mt-1.5 space-y-1.5">
            <StateGate
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              loadingLabel="SYNCING…"
              emptyLabel="NO CONFIG PULL TASKS · 暂无配置拉取记录"
            >
              {rows.map((d) => {
                const st = DEV_STATUS[d.status]
                return (
                  <Row
                    key={d.id}
                    color={st.token === 'error' ? NEON.rose : NEON.cyan}
                    className="grid-cols-[2fr_1.2fr_1fr_1fr_1.4fr_110px]"
                  >
                    <div className="min-w-0">
                      <div className="truncate font-display text-sm font-bold text-cyan-100" title={d.deviceName}>
                        {d.deviceName || '—'}
                      </div>
                      <div className="font-mono text-[10px] text-cyan-300/55">{d.deviceSn}</div>
                    </div>
                    <div className="min-w-0">
                      <div className="truncate text-xs text-cyan-100/85" title={d.taskName}>
                        {d.taskName}
                      </div>
                      <div className="font-mono text-[10px] text-cyan-300/45">{d.productType || '—'}</div>
                    </div>
                    <div>
                      <StatusBadge status={st.token} label={st.label} />
                      {d.failureReason ? (
                        <div className="mt-0.5 truncate font-mono text-[9px] text-rose-300/70" title={d.failureDetail || d.failureReason}>
                          {d.failureReason}
                        </div>
                      ) : null}
                    </div>
                    <div>
                      <div className="h-1.5 w-full overflow-hidden rounded-full bg-cyan-500/10">
                        <div
                          className="h-full rounded-full transition-all"
                          style={{
                            width: `${Math.max(0, Math.min(100, d.progress))}%`,
                            background: NEON.cyan,
                            boxShadow: `0 0 8px ${NEON.cyan}`,
                          }}
                        />
                      </div>
                      <div className="mt-0.5 font-mono text-[10px] text-cyan-300/55">{d.progress}%</div>
                    </div>
                    <div className="font-mono text-[11px] text-cyan-300/75">
                      {formatTime(d.lastReportAt)}
                    </div>
                    <div className="flex justify-end gap-1.5">
                      {d.downloadUrl ? (
                        <a
                          href={d.downloadUrl}
                          target="_blank"
                          rel="noreferrer"
                          className="neon-btn"
                          title={d.targetFile}
                        >
                          <span className="[&_svg]:size-3.5">
                            <Download />
                          </span>
                          DL
                        </a>
                      ) : (
                        <NeonButton disabled title="文件尚未就绪">
                          <Download className="size-3.5" />
                        </NeonButton>
                      )}
                      <NeonButton
                        icon={<ChevronRight />}
                        onClick={() => navigate(`/file/config-retrieval/${d.taskId}`)}
                      >
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
