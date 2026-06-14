import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, RefreshCcw, Cpu, ListChecks, Pencil } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { useConfigParams } from '@core/hooks/api/useConfig'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { ConfigParam } from '@core/types/config'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

type RwFilter = 'all' | 'writable' | 'readonly'

// ===========================================================================
// CONFIG · 参数列表
// 选设备 → 真实参数全量平铺列表 + 类别 / 读写 / 关键字三向过滤
// 「编辑」跳转在线参数配置页（list → live edit）。
// ===========================================================================
export default function ParamList() {
  const navigate = useNavigate()
  const [deviceId, setDeviceId] = useState('')
  const [keyword, setKeyword] = useState('')
  const [category, setCategory] = useState('')
  const [rw, setRw] = useState<RwFilter>('all')

  const devicesQ = useDeviceList({ page: 1, pageSize: 200 })
  const devices = devicesQ.data?.items ?? []
  const effectiveDeviceId = deviceId || devices[0]?.id || ''

  const paramsQ = useConfigParams({ deviceId: effectiveDeviceId, page: 1, pageSize: 1000 })
  const params: ConfigParam[] = (paramsQ.data?.items as ConfigParam[] | undefined) ?? []

  const categories = useMemo(
    () => Array.from(new Set(params.map((p) => p.category || '未分类'))),
    [params],
  )

  const kw = keyword.trim().toLowerCase()
  const rows = params.filter((p) => {
    if (category && (p.category || '未分类') !== category) return false
    if (rw === 'writable' && p.readonly) return false
    if (rw === 'readonly' && !p.readonly) return false
    if (kw && !(p.paramName?.toLowerCase().includes(kw) || p.paramCode?.toLowerCase().includes(kw)))
      return false
    return true
  })

  const writable = params.filter((p) => !p.readonly).length

  return (
    <PageShell
      code="F02"
      title="PARAM LIST · 参数列表"
      subtitle="FLAT PARAMETER INVENTORY"
      isFetching={devicesQ.isFetching || paramsQ.isFetching}
      bare
      toolbar={
        <>
          <select
            className="neon-input w-48"
            value={effectiveDeviceId}
            onChange={(e) => setDeviceId(e.target.value)}
          >
            {devices.length === 0 ? <option value="">无设备</option> : null}
            {devices.map((d) => (
              <option key={d.id} value={d.id}>
                {d.sn}
              </option>
            ))}
          </select>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-56 pl-9"
              placeholder="参数名 / 路径"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => void paramsQ.refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="PARAMS · 参数" value={params.length} color="#00f0ff" />
        <StatCard label="WRITABLE · 可写" value={writable} color="#00ff88" />
        <StatCard label="READONLY · 只读" value={params.length - writable} color="#525a78" />
        <StatCard label="MATCHED · 匹配" value={rows.length} color="#a855f7" />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setCategory('')}
          className={`chip ${category === '' ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
        >
          全部类
        </button>
        {categories.map((c) => (
          <button
            key={c}
            type="button"
            onClick={() => setCategory(c)}
            className={`chip ${category === c ? 'text-[#00f0ff] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            {c}
          </button>
        ))}
        <span className="mx-2 h-4 w-px bg-cyan-500/20" />
        {(['all', 'writable', 'readonly'] as RwFilter[]).map((r) => (
          <button
            key={r}
            type="button"
            onClick={() => setRw(r)}
            className={`chip ${rw === r ? 'text-[#a855f7] shadow-[0_0_10px_currentColor]' : 'text-[#6b86b6]'}`}
          >
            {r === 'all' ? '读写全部' : r === 'writable' ? '仅可写' : '仅只读'}
          </button>
        ))}
      </div>

      <GlassPanel title="PARAMETERS" meta={`${rows.length} / ${params.length}`} className="min-h-0">
        {paramsQ.isLoading ? (
          <HudLoading />
        ) : paramsQ.isError ? (
          <HudError error={paramsQ.error} />
        ) : devices.length === 0 ? (
          <HudEmpty icon={Cpu} text="无设备" />
        ) : rows.length === 0 ? (
          <HudEmpty icon={ListChecks} text="无匹配参数" />
        ) : (
          <div className="max-h-[56vh] overflow-auto">
            <div className="grid grid-cols-[1.8fr_1fr_1fr_120px_80px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
              <span>PARAM</span>
              <span>CATEGORY / 类</span>
              <span>VALUE / 当前</span>
              <span>TYPE</span>
              <span className="text-right">EDIT</span>
            </div>
            {rows.map((p) => (
              <div
                key={p.id}
                className="grid grid-cols-[1.8fr_1fr_1fr_120px_80px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
              >
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/90">{p.paramName}</div>
                  <code className="block truncate font-mono text-[10px] text-cyan-300/50">{p.paramCode}</code>
                </div>
                <div className="truncate font-mono text-[11px] text-cyan-300/65">{p.category || '未分类'}</div>
                <div className="truncate font-mono text-[11px] text-cyan-200/85">
                  {String(p.paramValue)}
                  {p.unit ? <span className="text-cyan-300/40"> {p.unit}</span> : null}
                </div>
                <div>
                  <span className="chip text-[#5b9eff]">{p.paramType}</span>
                </div>
                <div className="flex justify-end">
                  {p.readonly ? (
                    <StatusBadge status="off" label="只读" />
                  ) : (
                    <NeonButton icon={<Pencil />} onClick={() => navigate('/config/live-param')}>
                      改
                    </NeonButton>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </GlassPanel>
    </PageShell>
  )
}
