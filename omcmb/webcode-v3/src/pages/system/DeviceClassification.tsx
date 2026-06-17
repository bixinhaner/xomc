import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Boxes, Cpu, Wifi, ChevronRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useProductClasses, useDeviceList } from '@core/hooks/api/useDevices'
import type { Device } from '@core/types/device'

import { StateBlock, MiniStat, RowHeader, Pager } from './_shared'

export default function DeviceClassification() {
  const navigate = useNavigate()
  const {
    data: classes,
    isLoading: classesLoading,
    isError: classesError,
    error: classesErr,
    isFetching: classesFetching,
  } = useProductClasses()

  const [selected, setSelected] = useState<string>('')
  const [page, setPage] = useState(1)
  const pageSize = 20

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(selected ? { productModel: selected } : {}),
    }),
    [page, selected]
  )
  const {
    data: list,
    isLoading: listLoading,
    isError: listError,
    error: listErr,
    isFetching: listFetching,
  } = useDeviceList(params)

  const rows = list?.items ?? []
  const total = list?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const onlineCount = list?.stats?.online_count ?? 0
  const offlineCount = list?.stats?.offline_count ?? 0

  const productClasses = classes ?? []

  return (
    <PageShell
      code="F06"
      title="DEVICE CLASS · 设备分类"
      subtitle="PRODUCT CLASS REGISTRY"
      isFetching={classesFetching || listFetching}
      bare
    >
      <div className="flex h-full min-h-0 flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat label="产品类型数" value={productClasses.length} color="#00f0ff" icon={<Boxes className="size-3.5" />} />
          <MiniStat
            label={selected ? `${selected} 设备数` : '全部设备数'}
            value={total.toLocaleString()}
            color="#a855f7"
            icon={<Cpu className="size-3.5" />}
          />
          <MiniStat label="在线 · ONLINE" value={onlineCount} color="#00ff88" icon={<Wifi className="size-3.5" />} />
          <MiniStat label="离线 · OFFLINE" value={offlineCount} color="#525a78" />
        </div>

        <div className="grid min-h-0 flex-1 gap-3 lg:grid-cols-[260px_1fr]">
          {/* 左：产品类型列表 */}
          <div className="glass-strong relative flex min-h-0 flex-col overflow-hidden rounded-sm">
            <div className="scanline" />
            <div className="relative flex-1 overflow-auto p-2">
              <StateBlock
                isLoading={classesLoading}
                isError={classesError}
                error={classesErr}
                isEmpty={productClasses.length === 0}
                emptyLabel="NO PRODUCT CLASS · 无产品类型"
              >
                <div className="space-y-1">
                  <button
                    type="button"
                    onClick={() => {
                      setSelected('')
                      setPage(1)
                    }}
                    className={`flex w-full items-center justify-between rounded-sm px-3 py-2 text-left text-sm transition-all ${
                      selected === ''
                        ? 'bg-cyan-500/15 text-cyan-100 shadow-[inset_0_0_0_1px_rgba(0,240,255,0.35)]'
                        : 'text-cyan-300/70 hover:bg-cyan-500/5'
                    }`}
                  >
                    <span className="font-display">全部设备</span>
                    <span className="font-mono text-[10px] text-cyan-300/55">ALL</span>
                  </button>
                  {productClasses.map((pc) => (
                    <button
                      key={pc}
                      type="button"
                      onClick={() => {
                        setSelected(pc)
                        setPage(1)
                      }}
                      className={`flex w-full items-center justify-between rounded-sm px-3 py-2 text-left text-sm transition-all ${
                        selected === pc
                          ? 'bg-cyan-500/15 text-cyan-100 shadow-[inset_0_0_0_1px_rgba(0,240,255,0.35)]'
                          : 'text-cyan-300/70 hover:bg-cyan-500/5'
                      }`}
                    >
                      <span className="truncate font-mono text-xs">{pc}</span>
                      <ChevronRight className="size-3.5 shrink-0 text-cyan-300/40" />
                    </button>
                  ))}
                </div>
              </StateBlock>
            </div>
          </div>

          {/* 右：该分类下设备 */}
          <div className="flex min-h-0 flex-col gap-3">
            <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
              <div className="scanline" />
              <div className="relative h-full overflow-auto p-3">
                <StateBlock
                  isLoading={listLoading}
                  isError={listError}
                  error={listErr}
                  isEmpty={rows.length === 0}
                  emptyLabel="NO DEVICES · 该分类无设备"
                >
                  <div className="space-y-1.5">
                    <RowHeader cols="1.6fr_1.4fr_1fr_1fr_1fr_1.4fr_0.4fr">
                      <span>设备 · NAME</span>
                      <span>SN</span>
                      <span>产品类型</span>
                      <span>制式</span>
                      <span>状态</span>
                      <span>最后在线</span>
                      <span />
                    </RowHeader>
                    {rows.map((d: Device) => (
                      <button
                        key={d.id}
                        type="button"
                        onClick={() => navigate(`/device/list?sn=${encodeURIComponent(d.sn)}`)}
                        className="fleet-row grid w-full grid-cols-[1.6fr_1.4fr_1fr_1fr_1fr_1.4fr_0.4fr] items-center gap-3 rounded-sm px-3 py-2.5 text-left"
                        style={{ ['--row-color' as never]: d.isOnline ? '#00ff88' : '#525a78' }}
                      >
                        <div className="truncate font-display text-sm font-bold text-cyan-100">
                          {d.name || d.sn}
                        </div>
                        <div className="truncate font-mono text-[11px] text-cyan-100/80">{d.sn}</div>
                        <div className="truncate font-mono text-[11px] text-cyan-300/75">
                          {d.productClass || '—'}
                        </div>
                        <div className="font-mono text-[11px] uppercase text-cyan-300/70">
                          {d.networkType || '—'}
                        </div>
                        <div>
                          <StatusBadge
                            status={d.isOnline ? 'online' : 'offline'}
                            label={d.isOnline ? '在线' : '离线'}
                          />
                        </div>
                        <div className="font-mono text-[11px] text-cyan-300/75">
                          {d.lastOnlineTime ? formatTime(d.lastOnlineTime) : '—'}
                        </div>
                        <ChevronRight className="size-3.5 justify-self-end text-cyan-300/40" />
                      </button>
                    ))}
                  </div>
                </StateBlock>
              </div>
            </div>
            <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />
          </div>
        </div>
      </div>
    </PageShell>
  )
}
