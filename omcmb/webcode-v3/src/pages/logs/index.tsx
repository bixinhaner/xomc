import { useState, useMemo } from 'react'
import { Search, RefreshCcw, Loader2 } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useSystemLogs } from '@core/hooks/api/useLogs'

type LogLevel = 'INFO' | 'WARN' | 'ERROR' | 'DEBUG'

const LEVEL_COLOR: Record<string, string> = {
  DEBUG: '#5b9eff',
  INFO: '#00f0ff',
  WARN: '#ffaa00',
  ERROR: '#ff2d6f',
}

export function LogsPage() {
  const [page, setPage] = useState(1)
  const pageSize = 50
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState<LogLevel | ''>('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(level ? { level } : {}),
      ...(keyword ? { keyword } : {}),
    }),
    [page, level, keyword]
  )

  const { data, isLoading, isError, isFetching, refetch } = useSystemLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  return (
    <PageShell
      code="F06"
      title="LOG STREAM · 系统日志"
      subtitle="SYSLOG VIEWER · CRT FEED"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="关键字搜索"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
            />
          </div>
          {(['', 'INFO', 'WARN', 'ERROR'] as const).map((l) => (
            <button
              key={l || 'all'}
              type="button"
              onClick={() => {
                setLevel(l as LogLevel | '')
                setPage(1)
              }}
              className={`chip ${level === l ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60'}`}
              style={{ color: l ? LEVEL_COLOR[l] : '#00f0ff' }}
            >
              {l || 'ALL'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="terminal h-full overflow-auto p-3 text-[12px]">
        {isLoading ? (
          <div className="flex items-center gap-2 text-emerald-300/70">
            <Loader2 className="size-3 animate-spin" /> SYNC…
          </div>
        ) : isError ? (
          <div className="text-rose-300">SYNC FAILED</div>
        ) : rows.length === 0 ? (
          <div className="text-emerald-300/60">// no records</div>
        ) : (
          rows.map((r) => {
            const lvl = r.level
            const color = LEVEL_COLOR[lvl] ?? '#00ff88'
            return (
              <div key={r.id} className="flex gap-3 py-0.5">
                <span className="shrink-0 text-emerald-300/55">{formatTime(r.timestamp)}</span>
                <span
                  className="shrink-0 font-bold uppercase"
                  style={{ color, textShadow: `0 0 6px ${color}` }}
                >
                  {lvl.padEnd(5, ' ')}
                </span>
                <span className="shrink-0 text-emerald-300/65">[{r.source || '-'}]</span>
                <span className="text-emerald-100/85">{r.message}</span>
              </div>
            )
          })
        )}
      </div>

      <div className="mt-3 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} · {pageSize}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton onClick={() => setPage((p) => p + 1)}>NEXT ▸</NeonButton>
        </div>
      </div>
    </PageShell>
  )
}
