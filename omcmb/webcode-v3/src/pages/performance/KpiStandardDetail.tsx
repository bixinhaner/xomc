import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, FunctionSquare, Loader2, RefreshCcw } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { cn } from '@/lib/utils'
import { useIndicatorGroupTree, useIndicatorInfo } from '@core/hooks/api/useIndicator'
import type { IndicatorGroup } from '@core/types/indicator'

/**
 * F03 · 指标详情（/performance/kpi-standard/detail/:deviceType/:indicatorId）
 * useParams 取 deviceType + indicatorId → useIndicatorInfo 真实加载单指标。
 */

type DeviceTab = 'ENB' | 'GSM' | 'GNB'

function isDeviceTab(v: string | undefined): v is DeviceTab {
  return v === 'ENB' || v === 'GSM' || v === 'GNB'
}

function findGroupName(groups: IndicatorGroup[], id: string): string {
  for (const g of groups) {
    if (g.id === id) return g.cnName || g.enName || id
    if (g.children?.length) {
      const found = findGroupName(g.children, id)
      if (found) return found
    }
  }
  return ''
}

function fmtTime(v?: string): string {
  if (!v) return '—'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return v
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export default function KpiStandardDetail() {
  const navigate = useNavigate()
  const { deviceType: rawType, indicatorId } = useParams<{ deviceType: string; indicatorId: string }>()
  const deviceType: DeviceTab = isDeviceTab(rawType) ? rawType : 'ENB'
  const isGNB = deviceType === 'GNB'

  const { data: indicator, isLoading, isError, isFetching, refetch } = useIndicatorInfo(
    indicatorId || '',
    deviceType,
  )
  const { data: groupTree = [] } = useIndicatorGroupTree({ deviceType })

  const groupName = useMemo(() => {
    if (!indicator?.catagoryId) return indicator?.catagoryName || '—'
    return findGroupName(groupTree, indicator.catagoryId) || indicator.catagoryName || '—'
  }, [groupTree, indicator])

  const backToList = () => navigate('/performance/kpi-standard')

  return (
    <PageShell
      code="F03"
      title="INDICATOR DETAIL · 指标详情"
      subtitle={`${deviceType} · ${indicatorId ?? ''}`}
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={backToList}>
            返回列表
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {isLoading ? (
        <GlassPanel title="LOADING">
          <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">
            <Loader2 className="size-5 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING INDICATOR…</span>
          </div>
        </GlassPanel>
      ) : isError || !indicator || !indicatorId ? (
        <GlassPanel title="NOT FOUND">
          <div className="flex flex-col items-center justify-center gap-3 py-16">
            <div className="font-mono text-sm uppercase tracking-[0.2em] text-rose-300">
              指标未找到 · ID: {indicatorId || '(空)'}
            </div>
            <NeonButton icon={<ArrowLeft />} onClick={backToList}>
              返回指标库
            </NeonButton>
          </div>
        </GlassPanel>
      ) : (
        <div className="space-y-3">
          {/* 标题卡 */}
          <GlassPanel strong title="OVERVIEW · 概览">
            <div className="flex flex-wrap items-end justify-between gap-4 p-4">
              <div>
                <div className="font-display text-2xl font-bold text-cyan-100 text-glow">
                  {indicator.kpiName || '—'}
                </div>
                <div className="mt-1 font-mono text-[12px] text-cyan-300/55">{indicator.kpiId}</div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <span
                  className="chip"
                  style={{ color: indicator.indicatorType === 'counter' ? '#00ff88' : '#5b9eff' }}
                >
                  {indicator.indicatorType === 'counter' ? 'COUNTER' : 'KPI'}
                </span>
                <StatusBadge
                  status={indicator.isCustomize ? 'warning' : 'online'}
                  label={indicator.isCustomize ? '自定义' : '内置'}
                />
                <StatusBadge
                  status={indicator.isEnable ? 'ok' : 'off'}
                  label={indicator.isEnable ? '测量中' : '未测量'}
                />
              </div>
            </div>
          </GlassPanel>

          {/* 属性栅格 */}
          <GlassPanel title="ATTRIBUTES · 属性">
            <div className="grid grid-cols-2 gap-px bg-cyan-500/8 md:grid-cols-3 lg:grid-cols-4">
              <Field label="指标编号" value={indicator.kpiId} mono />
              <Field label="自定义名" value={indicator.custName || '—'} />
              <Field label="功能集" value={groupName} />
              {!isGNB && <Field label="产品类" value={indicator.productClass || '—'} />}
              {!isGNB && (
                <Field
                  label="等级"
                  value={
                    indicator.indicatorLevel === 'device'
                      ? 'Device'
                      : indicator.indicatorLevel === 'plmn'
                        ? 'PLMN'
                        : '—'
                  }
                />
              )}
              <Field label="单位" value={indicator.unit || '—'} />
              <Field label="统计类型" value={indicator.statisType || '—'} />
              <Field label="制式" value={deviceType} mono />
              <Field label="更新人" value={indicator.updater || '—'} />
              <Field label="更新时间" value={fmtTime(indicator.updateTime)} mono />
            </div>
          </GlassPanel>

          {/* 定义 + 公式 */}
          <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
            <GlassPanel title="DEFINITION · 定义">
              <div className="p-4 text-[13px] leading-relaxed text-cyan-100/85">
                {indicator.definition || '—'}
              </div>
            </GlassPanel>
            {indicator.indicatorType === 'kpi' && (
              <GlassPanel title="FORMULA · 计算公式" meta={<FunctionSquare className="size-3.5" />}>
                <div className="p-4">
                  <pre className="overflow-auto whitespace-pre-wrap break-all rounded-sm border border-cyan-500/15 bg-cyan-500/[0.03] p-3 font-mono text-[12px] text-cyan-200">
                    {indicator.arithmetic || '—'}
                  </pre>
                </div>
              </GlassPanel>
            )}
          </div>
        </div>
      )}
    </PageShell>
  )
}

function Field({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="bg-[#03050d] px-3.5 py-2.5">
      <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">{label}</div>
      <div
        className={cn('mt-0.5 truncate text-[13px] text-cyan-100', mono && 'font-mono text-[12px]')}
        title={value}
      >
        {value}
      </div>
    </div>
  )
}
