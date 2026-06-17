import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Search,
  RefreshCcw,
  Loader2,
  FileText,
  Download,
  Play,
  CalendarClock,
  CheckCircle2,
  AlertTriangle,
  FileBarChart,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useReportDefinitions,
  useReportRecords,
  useGenerateReport,
  useDownloadReport,
} from '@core/hooks/api/useReports'
import type {
  ReportType,
  ReportStatus,
  ReportPeriod,
} from '@core/mock/data/reports'

// ---------------------------------------------------------------------------
// /reports/lte-standard · LTE 标准化报表
//   对齐 v1 webcode/report/LTEStandardReport：报表分类（类型）树 + 生成记录流。
//   v1 旧页面把分类树挂在本地 mock；本皮肤用真实 @core/useReports 的报表定义
//   作为「标准报表模板」，按 reportType 归类，点模板生成记录、记录可下载。
//   全部经 @core/hooks/api/useReports（mock/real 自动切换）取数。
// ---------------------------------------------------------------------------

const TYPE_LABEL: Record<ReportType, string> = {
  performance: '性能',
  alarm: '告警',
  device: '设备',
  capacity: '容量',
  security: '安全',
}
const TYPE_COLOR: Record<ReportType, string> = {
  performance: '#00f0ff',
  alarm: '#ff2d6f',
  device: '#a855f7',
  capacity: '#ffaa00',
  security: '#00ff88',
}

const PERIOD_LABEL: Record<ReportPeriod, string> = {
  daily: '日报',
  weekly: '周报',
  monthly: '月报',
  quarterly: '季报',
  custom: '自定义',
}

const DEF_STATUS_LABEL: Record<ReportStatus, string> = {
  published: '已发布',
  draft: '草稿',
  archived: '已归档',
}
const DEF_STATUS_COLOR: Record<ReportStatus, string> = {
  published: '#00ff88',
  draft: '#5b9eff',
  archived: '#525a78',
}

type RecordStatus = 'generating' | 'ready' | 'failed'
const REC_STATUS_LABEL: Record<RecordStatus, string> = {
  ready: '就绪',
  generating: '生成中',
  failed: '失败',
}
const REC_STATUS_COLOR: Record<RecordStatus, string> = {
  ready: '#00ff88',
  generating: '#ffaa00',
  failed: '#ff2d6f',
}

const ALL_TYPES: ReportType[] = ['performance', 'alarm', 'device', 'capacity', 'security']

export default function LteStandardReportPage() {
  const navigate = useNavigate()
  const [typeFilter, setTypeFilter] = useState<ReportType | ''>('')
  const [keyword, setKeyword] = useState('')
  const [selectedDefId, setSelectedDefId] = useState<string | null>(null)
  const [recPage, setRecPage] = useState(1)
  const recPageSize = 20

  const defParams = useMemo(
    () => ({
      page: 1,
      pageSize: 200,
      ...(typeFilter ? { reportType: typeFilter } : {}),
    }),
    [typeFilter],
  )
  const {
    data: defData,
    isLoading: defLoading,
    isError: defError,
    error: defErrObj,
    isFetching,
    refetch: refetchDefs,
  } = useReportDefinitions(defParams)

  const recParams = useMemo(
    () => ({
      page: recPage,
      pageSize: recPageSize,
      ...(selectedDefId ? { reportDefinitionId: selectedDefId } : {}),
    }),
    [recPage, selectedDefId],
  )
  const {
    data: recData,
    isLoading: recLoading,
    isError: recError,
    refetch: refetchRecs,
  } = useReportRecords(recParams)

  const generateReport = useGenerateReport()
  const downloadReport = useDownloadReport()

  const allDefs = defData?.items ?? []
  const defs = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return allDefs
    return allDefs.filter(
      (d) =>
        d.reportName.toLowerCase().includes(kw) ||
        (d.description ?? '').toLowerCase().includes(kw),
    )
  }, [allDefs, keyword])

  // 按报表类型归类（替代 v1 的本地 mock 分类树，数据全真实）
  const grouped = useMemo(() => {
    return ALL_TYPES.map((tp) => ({
      type: tp,
      items: defs.filter((d) => d.reportType === tp),
    })).filter((g) => g.items.length > 0)
  }, [defs])

  const stats = useMemo(() => {
    const total = allDefs.length
    const published = allDefs.filter((d) => d.status === 'published').length
    const auto = allDefs.filter((d) => d.autoGenerate).length
    return { total, published, auto }
  }, [allDefs])

  const records = recData?.items ?? []
  const recTotal = recData?.total ?? 0
  const recTotalPages = Math.max(1, Math.ceil(recTotal / recPageSize))
  const readyRecCount = records.filter((r) => r.status === 'ready').length

  const selectedDef = selectedDefId ? allDefs.find((d) => d.id === selectedDefId) ?? null : null

  const onGenerate = (definitionId: string) => {
    generateReport.mutate(
      { definitionId },
      { onSuccess: () => void refetchRecs() },
    )
  }

  const onDownload = (id: string) => {
    downloadReport.mutate(id, {
      onSuccess: (res) => {
        if (res?.url) window.open(res.url, '_blank', 'noopener,noreferrer')
      },
    })
  }

  return (
    <PageShell
      code="F06"
      title="LTE STANDARD · 标准化报表"
      subtitle="REPORT TEMPLATE TREE · GENERATED ARCHIVE"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <button
            type="button"
            onClick={() => navigate('/report/lte-standard')}
            className="chip text-cyan-300/70 hover:opacity-100"
          >
            ← 简报中心
          </button>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="报表名称 / 描述"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {(['', ...ALL_TYPES] as const).map((tp) => (
            <button
              key={tp || 'all'}
              type="button"
              onClick={() => setTypeFilter(tp)}
              className={`chip transition-all ${
                typeFilter === tp ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: tp ? TYPE_COLOR[tp] : '#00f0ff' }}
            >
              {tp ? TYPE_LABEL[tp] : 'ALL'}
            </button>
          ))}
          <NeonButton
            icon={<RefreshCcw />}
            onClick={() => {
              void refetchDefs()
              void refetchRecs()
            }}
          >
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <BigStat label="TEMPLATES · 报表模板" value={stats.total} color="#00f0ff" icon={<FileText className="size-4" />} />
        <BigStat label="PUBLISHED · 已发布" value={stats.published} color="#00ff88" icon={<CheckCircle2 className="size-4" />} />
        <BigStat label="SCHEDULED · 自动调度" value={stats.auto} color="#a855f7" icon={<CalendarClock className="size-4" />} />
        <BigStat label="READY RECORDS · 就绪记录" value={recError ? '—' : readyRecCount} color="#ffaa00" icon={<FileBarChart className="size-4" />} />
      </div>

      <div className="grid grid-cols-12 gap-3">
        {/* 左：按类型归类的模板树 */}
        <div className="col-span-12 lg:col-span-7">
          <GlassPanel title="REPORT TEMPLATES · 报表模板（按类型）" meta={`${defs.length} / ${stats.total}`}>
            <div className="max-h-[calc(100vh-330px)] min-h-[280px] space-y-4 overflow-auto p-3">
              {defLoading ? (
                <Centered>
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING TEMPLATES…</span>
                </Centered>
              ) : defError ? (
                <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
                  FAILURE · {defErrObj instanceof Error ? defErrObj.message : '加载失败'}
                </div>
              ) : grouped.length === 0 ? (
                <EmptyState text="无匹配报表模板" />
              ) : (
                grouped.map((g) => {
                  const tColor = TYPE_COLOR[g.type]
                  return (
                    <div key={g.type}>
                      <div className="mb-1.5 flex items-center gap-2">
                        <span className="size-1.5 rounded-full" style={{ background: tColor, boxShadow: `0 0 8px ${tColor}` }} />
                        <span className="font-display text-xs font-bold" style={{ color: tColor }}>
                          {TYPE_LABEL[g.type]}报表
                        </span>
                        <span className="font-mono text-[10px] text-cyan-300/45">{g.items.length}</span>
                      </div>
                      <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
                        {g.items.map((d) => {
                          const active = selectedDefId === d.id
                          return (
                            <div
                              key={d.id}
                              role="button"
                              tabIndex={0}
                              onClick={() => {
                                setSelectedDefId(active ? null : d.id)
                                setRecPage(1)
                              }}
                              onKeyDown={(e) => {
                                if (e.key === 'Enter' || e.key === ' ') {
                                  setSelectedDefId(active ? null : d.id)
                                  setRecPage(1)
                                }
                              }}
                              className={`group relative cursor-pointer rounded-sm border-l-2 bg-[#070b16]/55 p-3 transition-all hover:bg-cyan-500/5 ${
                                active ? 'bg-cyan-500/10 shadow-[0_0_12px_rgba(0,240,255,0.15)]' : ''
                              }`}
                              style={{ borderLeftColor: tColor }}
                            >
                              <div className="min-w-0">
                                <div className="truncate font-display text-sm font-bold text-cyan-100">{d.reportName}</div>
                                <div className="mt-0.5 truncate font-mono text-[10px] text-cyan-300/55">{d.description || '—'}</div>
                              </div>
                              <div className="mt-2 flex flex-wrap items-center gap-1.5">
                                <span className="chip text-cyan-300/70">{PERIOD_LABEL[d.period]}</span>
                                <span className="chip" style={{ color: DEF_STATUS_COLOR[d.status] }}>
                                  {DEF_STATUS_LABEL[d.status]}
                                </span>
                                {d.autoGenerate ? (
                                  <span className="chip text-[#a855f7]">
                                    <CalendarClock className="size-2.5" />
                                    {d.cronExpression || 'CRON'}
                                  </span>
                                ) : (
                                  <span className="chip text-cyan-300/40">手动</span>
                                )}
                              </div>
                              <div className="mt-2 flex items-center justify-between border-t border-cyan-500/10 pt-2">
                                <span className="truncate font-mono text-[10px] text-cyan-300/45">
                                  上次 {d.lastGenTime ? formatTime(d.lastGenTime) : '—'} · {d.creator || '—'}
                                </span>
                                <NeonButton
                                  className="px-2 py-0.5"
                                  icon={<Play />}
                                  disabled={generateReport.isPending}
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    onGenerate(d.id)
                                  }}
                                >
                                  生成
                                </NeonButton>
                              </div>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          </GlassPanel>
        </div>

        {/* 右：生成记录流 */}
        <div className="col-span-12 lg:col-span-5">
          <GlassPanel
            title="GENERATED ARCHIVE · 生成记录"
            meta={selectedDef ? `FILTER · ${selectedDef.reportName}` : `TOTAL ${recTotal}`}
          >
            <div className="flex items-center justify-between border-b border-cyan-500/10 px-3 py-2">
              <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
                {selectedDef ? '已按模板过滤' : '全部记录'}
              </span>
              {selectedDef ? (
                <button
                  type="button"
                  className="chip text-cyan-300/70 hover:opacity-100"
                  onClick={() => {
                    setSelectedDefId(null)
                    setRecPage(1)
                  }}
                >
                  清除过滤
                </button>
              ) : null}
            </div>

            <div className="max-h-[calc(100vh-372px)] min-h-[240px] overflow-auto">
              {recLoading ? (
                <Centered>
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING RECORDS…</span>
                </Centered>
              ) : recError ? (
                <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
                  RECORD FEED UNAVAILABLE
                </div>
              ) : records.length === 0 ? (
                <EmptyState text="无生成记录" icon="ok" />
              ) : (
                records.map((r) => {
                  const st = r.status as RecordStatus
                  const color = REC_STATUS_COLOR[st] ?? '#6b86b6'
                  return (
                    <div
                      key={r.id}
                      className="fleet-row grid grid-cols-[1fr_auto] items-center gap-2 px-3 py-2.5"
                      style={{ ['--row-color' as never]: color }}
                    >
                      <div className="min-w-0">
                        <div className="truncate font-display text-xs font-bold text-cyan-100">{r.reportName}</div>
                        <div className="mt-0.5 flex items-center gap-2 font-mono text-[10px] text-cyan-300/55">
                          <span>{r.period || '—'}</span>
                          <span className="text-cyan-300/30">·</span>
                          <span className="uppercase">{r.format}</span>
                          <span className="text-cyan-300/30">·</span>
                          <span>{r.fileSize ? formatBytes(r.fileSize) : '—'}</span>
                        </div>
                        <div className="font-mono text-[10px] text-cyan-300/40">{formatTime(r.generateTime)}</div>
                      </div>
                      <div className="flex flex-col items-end gap-1.5">
                        <span className="chip" style={{ color }}>
                          {st === 'generating' ? <Loader2 className="size-2.5 animate-spin" /> : null}
                          {REC_STATUS_LABEL[st] ?? st}
                        </span>
                        <NeonButton
                          className="px-2 py-0.5"
                          icon={<Download />}
                          disabled={r.status !== 'ready' || downloadReport.isPending}
                          onClick={() => onDownload(r.id)}
                        >
                          下载
                        </NeonButton>
                      </div>
                    </div>
                  )
                })
              )}
            </div>

            <div className="flex items-center justify-between border-t border-cyan-500/15 px-3 py-2">
              <span className="font-mono text-[10px] text-cyan-300/55">
                PAGE {recPage} / {recTotalPages} · {recTotal}
              </span>
              <div className="flex gap-2">
                <NeonButton className="px-2 py-0.5" onClick={() => setRecPage((p) => Math.max(1, p - 1))} disabled={recPage <= 1}>
                  ◂ PREV
                </NeonButton>
                <NeonButton
                  className="px-2 py-0.5"
                  onClick={() => setRecPage((p) => Math.min(recTotalPages, p + 1))}
                  disabled={recPage >= recTotalPages}
                >
                  NEXT ▸
                </NeonButton>
              </div>
            </div>
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

function BigStat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: number | string
  color: string
  icon: React.ReactNode
}) {
  const trend = useMemo(() => pseudoTrend(label, 16), [label])
  return (
    <div className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3" style={{ borderLeftColor: color }}>
      <div className="scanline" />
      <div className="relative flex items-center justify-between">
        <div>
          <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
            <span style={{ color }}>{icon}</span>
            {label}
          </div>
          <div className="font-display text-3xl font-bold leading-tight text-glow" style={{ color }}>
            {value}
          </div>
        </div>
        <Sparkline data={trend} color={color} width={72} height={34} />
      </div>
    </div>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">{children}</div>
}

function EmptyState({ text, icon }: { text: string; icon?: 'ok' }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      {icon === 'ok' ? (
        <CheckCircle2 className="size-9 text-emerald-400/55" />
      ) : (
        <AlertTriangle className="size-9 text-cyan-300/40" />
      )}
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">{text}</div>
    </div>
  )
}

/** 确定性伪随机趋势 —— 同 seed 输出同序列，避免 render 中 Math.random */
function pseudoTrend(seed: string, n: number): number[] {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  const out: number[] = []
  for (let i = 0; i < n; i++) {
    h = (h * 1664525 + 1013904223) >>> 0
    out.push((h % 100) / 100)
  }
  return out
}
