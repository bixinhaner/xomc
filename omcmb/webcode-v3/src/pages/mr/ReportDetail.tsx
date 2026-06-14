import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  CalendarClock,
  Download,
  FileText,
  Loader2,
  Play,
  RefreshCcw,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatBytes, formatTime } from '@/lib/format'
import {
  useReportDefinitionById,
  useReportRecords,
  useGenerateReport,
  useDownloadReport,
} from '@core/hooks/api/useReports'
import type { ReportType, ReportStatus, ReportPeriod } from '@core/mock/data/reports'

const TYPE_LABEL: Record<ReportType, string> = {
  performance: '性能',
  alarm: '告警',
  device: '设备',
  capacity: '容量',
  security: '安全',
}
const PERIOD_LABEL: Record<ReportPeriod, string> = {
  daily: '日报',
  weekly: '周报',
  monthly: '月报',
  quarterly: '季报',
  custom: '自定义',
}
const DEF_STATUS_BADGE: Record<ReportStatus, string> = {
  published: 'active',
  draft: 'unknown',
  archived: 'off',
}
const DEF_STATUS_LABEL: Record<ReportStatus, string> = {
  published: '已发布',
  draft: '草稿',
  archived: '已归档',
}
type RecStatus = 'generating' | 'ready' | 'failed'
const REC_BADGE: Record<RecStatus, string> = {
  ready: 'active',
  generating: 'warning',
  failed: 'critical',
}
const REC_LABEL: Record<RecStatus, string> = {
  ready: '就绪',
  generating: '生成中',
  failed: '失败',
}

/**
 * MR 报表定义详情（/mr/reports/:id）。
 * 定义经 useReportDefinitionById 加载；该定义的生成记录经 useReportRecords({reportDefinitionId})。
 * 立即生成 / 下载走 useGenerateReport / useDownloadReport。全部 @core 真实数据。
 */
export default function ReportDetail() {
  const navigate = useNavigate()
  const { id = '' } = useParams<{ id: string }>()
  const [recPage, setRecPage] = useState(1)
  const recPageSize = 20

  const def = useReportDefinitionById(id)
  const recParams = useMemo(
    () => ({ page: recPage, pageSize: recPageSize, reportDefinitionId: id }),
    [recPage, id],
  )
  const recs = useReportRecords(recParams)
  const generateReport = useGenerateReport()
  const downloadReport = useDownloadReport()

  const d = def.data
  const records = recs.data?.items ?? []
  const recTotal = recs.data?.total ?? 0
  const recTotalPages = Math.max(1, Math.ceil(recTotal / recPageSize))

  const onGenerate = () => {
    if (!d) return
    generateReport.mutate({ definitionId: d.id }, { onSuccess: () => void recs.refetch() })
  }
  const onDownload = (recId: string) => {
    downloadReport.mutate(recId, {
      onSuccess: (res) => {
        if (res?.url) window.open(res.url, '_blank', 'noopener,noreferrer')
      },
    })
  }

  return (
    <PageShell
      code="F05"
      title="REPORT DEFINITION · 报表定义详情"
      subtitle={d ? d.reportName : 'MR ANALYSIS REPORT DEFINITION'}
      isFetching={def.isFetching || recs.isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/mr/reports')}>
            BACK
          </NeonButton>
          <NeonButton
            icon={<Play />}
            disabled={!d || generateReport.isPending}
            onClick={onGenerate}
          >
            立即生成
          </NeonButton>
          <NeonButton
            icon={<RefreshCcw />}
            onClick={() => {
              void def.refetch()
              void recs.refetch()
            }}
          >
            REFRESH
          </NeonButton>
        </>
      }
    >
      {def.isLoading ? (
        <Centered>
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING DEFINITION…</span>
        </Centered>
      ) : def.isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {def.error instanceof Error ? def.error.message : '加载失败'}
        </div>
      ) : !d ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <FileText className="size-10 text-cyan-300/40" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
            报表定义 {id} 不存在
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-12 gap-3">
          {/* 定义信息 */}
          <div className="col-span-12 lg:col-span-5">
            <GlassPanel
              title="DEFINITION · 定义信息"
              meta={<StatusBadge status={DEF_STATUS_BADGE[d.status]} label={DEF_STATUS_LABEL[d.status]} />}
            >
              <div className="flex flex-col gap-3 p-4">
                <div>
                  <div className="font-display text-lg font-bold text-cyan-100">{d.reportName}</div>
                  <div className="mt-0.5 text-xs text-cyan-300/65">{d.description || '—'}</div>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <KV label="TYPE · 类型" value={TYPE_LABEL[d.reportType]} />
                  <KV label="PERIOD · 周期" value={PERIOD_LABEL[d.period]} />
                  <KV label="FORMAT · 格式" value={(d.format ?? []).join(' / ').toUpperCase() || '—'} />
                  <KV label="CREATOR · 创建者" value={d.creator || '—'} />
                  <KV
                    label="SCHEDULE · 调度"
                    value={d.autoGenerate ? d.cronExpression || 'CRON' : '手动'}
                    icon={d.autoGenerate ? <CalendarClock className="size-3" /> : undefined}
                  />
                  <KV label="LAST GEN · 上次生成" value={d.lastGenTime ? formatTime(d.lastGenTime) : '—'} />
                </div>
                {d.kpiCodes && d.kpiCodes.length > 0 ? (
                  <div>
                    <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/45">
                      KPI CODES · 指标集
                    </div>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {d.kpiCodes.map((c) => (
                        <span key={c} className="chip font-mono text-cyan-300/75">
                          {c}
                        </span>
                      ))}
                    </div>
                  </div>
                ) : null}
              </div>
            </GlassPanel>
          </div>

          {/* 该定义的生成记录 */}
          <div className="col-span-12 lg:col-span-7">
            <GlassPanel title="GENERATED RECORDS · 生成记录" meta={`TOTAL ${recTotal}`}>
              <div className="max-h-[calc(100vh-340px)] min-h-[240px] overflow-auto">
                {recs.isLoading ? (
                  <Centered>
                    <Loader2 className="size-4 animate-spin" />
                    <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING RECORDS…</span>
                  </Centered>
                ) : recs.isError ? (
                  <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
                    RECORD FEED UNAVAILABLE
                  </div>
                ) : records.length === 0 ? (
                  <div className="flex flex-col items-center justify-center gap-3 py-16">
                    <FileText className="size-9 text-cyan-300/40" />
                    <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
                      该定义暂无生成记录
                    </div>
                  </div>
                ) : (
                  records.map((r) => {
                    const st = r.status as RecStatus
                    return (
                      <div
                        key={r.id}
                        className="fleet-row grid grid-cols-[1fr_auto] items-center gap-2 px-3 py-2.5"
                        style={{
                          ['--row-color' as never]:
                            st === 'ready' ? '#00ff88' : st === 'failed' ? '#ff2d6f' : '#ffaa00',
                        }}
                      >
                        <div className="min-w-0">
                          <div className="truncate font-display text-xs font-bold text-cyan-100">
                            {r.reportName}
                          </div>
                          <div className="mt-0.5 flex items-center gap-2 font-mono text-[10px] text-cyan-300/55">
                            <span>{r.period || '—'}</span>
                            <span className="text-cyan-300/30">·</span>
                            <span className="uppercase">{r.format}</span>
                            <span className="text-cyan-300/30">·</span>
                            <span>{r.fileSize ? formatBytes(r.fileSize) : '—'}</span>
                          </div>
                          <div className="font-mono text-[10px] text-cyan-300/40">
                            {formatTime(r.generateTime)}
                          </div>
                        </div>
                        <div className="flex flex-col items-end gap-1.5">
                          <StatusBadge status={REC_BADGE[st] ?? 'unknown'} label={REC_LABEL[st] ?? st} />
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
                  <NeonButton
                    className="px-2 py-0.5"
                    onClick={() => setRecPage((p) => Math.max(1, p - 1))}
                    disabled={recPage <= 1}
                  >
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
      )}
    </PageShell>
  )
}

function KV({
  label,
  value,
  icon,
}: {
  label: string
  value: string
  icon?: React.ReactNode
}) {
  return (
    <div className="rounded-sm border border-cyan-500/12 bg-[#070b16]/45 px-3 py-2">
      <div className="font-mono text-[9px] uppercase tracking-[0.18em] text-cyan-300/45">{label}</div>
      <div className="mt-0.5 flex items-center gap-1 text-sm text-cyan-100/90">
        {icon ? <span className="text-cyan-300/65">{icon}</span> : null}
        {value}
      </div>
    </div>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">{children}</div>
  )
}
