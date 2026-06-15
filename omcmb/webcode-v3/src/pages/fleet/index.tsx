import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, RefreshCcw, Power, Loader2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { Sparkline } from '@/components/viz/Sparkline'
import { formatTime } from '@/lib/format'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { Device } from '@core/types/device'

const STATUS_COLOR: Record<string, string> = {
  online: '#00ff88',
  offline: '#525a78',
}

export function FleetPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { searchText: keyword.trim() } : {}),
    }),
    [page, keyword]
  )

  const { data, isFetching, isLoading, isError, error, refetch } = useDeviceList(params)
  const items: Device[] = data?.items ?? []
  const total = data?.total ?? 0

  return (
    <PageShell
      code="F02"
      title="FLEET · 舰队管理"
      subtitle="DEVICE INVENTORY · LIVE TR069 SESSIONS"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-72 pl-9"
              placeholder="SN / 名称 / IP / MAC"
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
      {/* 状态摘要 */}
      <div className="mb-3 flex flex-wrap items-center gap-3 text-[11px] uppercase tracking-[0.18em]">
        <span className="text-cyan-300/60">
          TOTAL <span className="ml-1 font-display text-base text-cyan-200 text-glow">{total}</span>
        </span>
        <span className="text-emerald-300/80">
          ONLINE
          <span className="ml-1 font-display text-base text-emerald-200 text-glow">
            {data?.stats?.online ?? '—'}
          </span>
        </span>
        <span className="text-zinc-300/70">
          OFFLINE
          <span className="ml-1 font-display text-base text-zinc-100">
            {data?.stats?.offline ?? '—'}
          </span>
        </span>
        <span className="text-rose-300/80">
          ALARMED
          <span className="ml-1 font-display text-base text-rose-200 text-glow">
            {data?.stats?.alarmed ?? '—'}
          </span>
        </span>
      </div>

      {/* 数据矩阵 */}
      <div className="space-y-2">
        {isLoading ? (
          <Skeleton />
        ) : isError ? (
          <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
        ) : items.length === 0 ? (
          <EmptyBlock />
        ) : (
          items.map((d) => (
            <div
              key={d.id}
              className="fleet-row group grid grid-cols-[12px_1.6fr_1fr_1fr_1fr_120px_120px] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{
                ['--row-color' as never]:
                  d.connStatus === 'online'
                    ? '#00ff88'
                    : d.alarmLevel === 'critical'
                      ? '#ff2d6f'
                      : d.alarmLevel === 'major'
                        ? '#ff7a1a'
                        : '#525a78',
              }}
            >
              {/* 状态圆灯 */}
              <span
                className="size-2.5 rounded-full"
                style={{
                  background: STATUS_COLOR[d.connStatus] ?? '#525a78',
                  boxShadow: `0 0 8px ${STATUS_COLOR[d.connStatus] ?? '#525a78'}`,
                }}
              />

              {/* 名称 + SN */}
              <div className="min-w-0">
                <div className="truncate font-display text-sm font-bold text-cyan-100">
                  {d.name || d.sn}
                </div>
                <div className="truncate font-mono text-[10px] text-cyan-300/55">
                  SN {d.sn} · {d.vendor} · {d.networkType || d.deviceModel}
                </div>
              </div>

              {/* 状态徽 */}
              <div className="flex flex-col gap-1">
                <StatusBadge status={d.connStatus} />
                {d.alarmLevel !== 'none' ? (
                  // #361: 把活动告警数拼进 HUD 徽标（如「ALM · major · 3」）。
                  <StatusBadge
                    status={d.alarmLevel}
                    label={`ALM · ${d.alarmLevel}${(d.activeAlarmCount ?? 0) > 0 ? ` · ${d.activeAlarmCount}` : ''}`}
                  />
                ) : (
                  <span className="font-mono text-[10px] text-cyan-300/40">NO ALARM</span>
                )}
              </div>

              {/* 位置 */}
              <div className="min-w-0">
                <div className="truncate text-xs text-cyan-100/85">{d.region || '—'}</div>
                <div className="truncate font-mono text-[10px] text-cyan-300/55">
                  {d.subnet || '—'} · {d.ipAddress || '—'}
                </div>
              </div>

              {/* 最后在线 */}
              <div className="font-mono text-[10px] text-cyan-300/65">
                <div className="text-cyan-100/85">{formatTime(d.lastOnlineTime)}</div>
                <div>{d.softwareVersion ? `FW ${d.softwareVersion}` : '—'}</div>
              </div>

              {/* 信号 sparkline */}
              <div className="hidden md:block">
                <Sparkline
                  data={fakeSpark(d.id)}
                  color={d.connStatus === 'online' ? '#00ff88' : '#525a78'}
                  width={110}
                  height={28}
                  fill
                />
              </div>

              {/* 动作 */}
              <div className="flex items-center justify-end gap-2 opacity-60 transition-opacity group-hover:opacity-100">
                <NeonButton
                  tone="cyan"
                  className="!py-1 !px-2"
                  onClick={() => navigate(`/fleet/detail/${d.sn}`)}
                >
                  DETAIL
                </NeonButton>
                <NeonButton tone="danger" icon={<Power />} className="!py-1 !px-2">
                  RBT
                </NeonButton>
              </div>
            </div>
          ))
        )}
      </div>

      {/* 分页 */}
      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {Math.max(1, Math.ceil(total / pageSize))} · {pageSize}/PAGE
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => p + 1)}
            disabled={page >= Math.ceil(total / pageSize)}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>
    </PageShell>
  )
}

function Skeleton() {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING FLEET…</span>
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

function EmptyBlock() {
  return (
    <div className="border border-cyan-500/15 px-4 py-12 text-center font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/40">
      NO DEVICES IN ORBIT
    </div>
  )
}

/** 基于 id 生成稳定的伪随机 sparkline */
function fakeSpark(seed: string): number[] {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  const out: number[] = []
  for (let i = 0; i < 24; i++) {
    h = (h * 1664525 + 1013904223) >>> 0
    out.push((h % 100) / 100)
  }
  return out
}
