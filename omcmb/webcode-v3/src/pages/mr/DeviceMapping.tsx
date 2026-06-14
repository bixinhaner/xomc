import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Boxes, ChevronRight, Loader2, RefreshCcw, Search } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { useMRMappings, useToggleMRMapping } from '@core/hooks/api/useMR'
import { formatTime } from '@/lib/format'

const PAGE_SIZE = 20

/**
 * 设备小区映射（对齐 v1 mr/DeviceMapping）。
 * 列表/开关全部走 @core useMRMappings + useToggleMRMapping。点行进入 /mr/device-mapping/:id。
 */
export default function DeviceMapping() {
  const navigate = useNavigate()
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
    [page, deviceSn, enabled],
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useMRMappings(params)
  const toggle = useToggleMRMapping()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <PageShell
      code="F05"
      title="MR DEVICE MAPPING · 设备小区映射"
      subtitle="DEVICE ⇄ CELL MR COLLECTION BINDINGS"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-52 pl-9"
              placeholder="设备 SN"
              value={deviceSn}
              onChange={(e) => {
                setDeviceSn(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {(['', 'true', 'false'] as const).map((s) => (
            <button
              key={s || 'all'}
              type="button"
              onClick={() => {
                setEnabled(s)
                setPage(1)
              }}
              className={`chip transition-all ${
                enabled === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: s ? (s === 'true' ? '#00ff88' : '#525a78') : '#6b86b6' }}
            >
              {s === '' ? 'ALL' : s === 'true' ? '已启用' : '已禁用'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => void refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-20 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING MAPPINGS…</span>
        </div>
      ) : isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {error instanceof Error ? error.message : '加载失败'}
        </div>
      ) : rows.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 py-20">
          <Boxes className="size-10 text-cyan-400/50" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/60">
            NO MAPPINGS · 暂无采集映射
          </div>
        </div>
      ) : (
        <div className="space-y-1.5">
          {rows.map((m) => (
            <div
              key={m.id}
              role="button"
              tabIndex={0}
              onClick={() => navigate(`/mr/device-mapping/${encodeURIComponent(m.id)}`)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  navigate(`/mr/device-mapping/${encodeURIComponent(m.id)}`)
                }
              }}
              className="fleet-row grid cursor-pointer grid-cols-[1.6fr_1.6fr_1fr_1.4fr_90px_24px] items-center gap-3 rounded-sm px-3 py-2.5"
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
                <div className="text-cyan-300/45">
                  {Number(m.totalRecords ?? 0).toLocaleString()} rec
                </div>
              </div>
              <div className="font-mono text-[10px] text-cyan-300/70">
                LAST · {m.lastCollectTime ? formatTime(m.lastCollectTime) : '从未采集'}
              </div>
              <div className="flex items-center justify-end">
                <button
                  type="button"
                  disabled={toggle.isPending}
                  onClick={(e) => {
                    e.stopPropagation()
                    toggle.mutate({ id: m.id, enabled: !m.enabled })
                  }}
                  className={`chip transition-all disabled:opacity-40 ${
                    m.enabled ? 'shadow-[0_0_8px_currentColor]' : 'opacity-70'
                  }`}
                  style={{ color: m.enabled ? '#00ff88' : '#525a78' }}
                >
                  {m.enabled ? 'ON' : 'OFF'}
                </button>
              </div>
              <ChevronRight className="size-3.5 text-cyan-300/45" />
            </div>
          ))}
        </div>
      )}

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
    </PageShell>
  )
}
