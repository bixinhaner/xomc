import { useMemo, useState } from 'react'
import { RefreshCcw, Search, Terminal } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useSystemLogs } from '@core/hooks/api/useLogs'
import type { SystemLog } from '@core/mock/data/logs'

import {
  DetailOverlay,
  FeedError,
  FeedLoading,
  FilterInput,
  LOG_PAGE_SIZE,
  Pager,
  StatCard,
} from './_shared'

// ──────────────────────────────────────────────────────────────────────────
// /logs/system —— 运行期系统日志（对照 v1 webcode log/system · SystemLog）
// 真实端点：useSystemLogs → logApi GET /logs/system（30s 轮询）
// HUD 形态：CRT 终端流 + 级别配色 + ERROR/details 详情下钻
// ──────────────────────────────────────────────────────────────────────────

type LogLevel = SystemLog['level']

const LEVEL_COLOR: Record<string, string> = {
  DEBUG: '#5b9eff',
  INFO: '#00f0ff',
  WARN: '#ffaa00',
  ERROR: '#ff2d6f',
}

const SYSTEM_SOURCES = [
  'auth',
  'device',
  'alarm',
  'performance',
  'file',
  'scheduler',
  'database',
  'gateway',
]

export default function SystemLogPage() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState<LogLevel | ''>('')
  const [source, setSource] = useState('')
  const [selected, setSelected] = useState<SystemLog | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: LOG_PAGE_SIZE,
      ...(level ? { level } : {}),
      ...(source ? { source } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, level, source, keyword],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSystemLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / LOG_PAGE_SIZE))

  const levelCount = useMemo(() => {
    const c: Record<string, number> = { DEBUG: 0, INFO: 0, WARN: 0, ERROR: 0 }
    rows.forEach((r) => {
      c[r.level] = (c[r.level] ?? 0) + 1
    })
    return c
  }, [rows])

  return (
    <PageShell
      code="F06"
      title="SYSTEM LOG · 系统日志"
      subtitle="RUNTIME SYSLOG · 30s POLL"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <StatCard label="TOTAL" color="#00f0ff" value={total} hint="后端总量" />
          {(['ERROR', 'WARN', 'INFO'] as const).map((lv) => (
            <StatCard key={lv} label={lv} color={LEVEL_COLOR[lv]} value={levelCount[lv] ?? 0} hint="本页" />
          ))}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <FilterInput
            value={keyword}
            onChange={(v) => {
              setKeyword(v)
              setPage(1)
            }}
            placeholder="关键字搜索"
            icon={<Search />}
          />
          <select
            className="neon-input w-40"
            value={source}
            onChange={(e) => {
              setSource(e.target.value)
              setPage(1)
            }}
          >
            <option value="">全部来源</option>
            {SYSTEM_SOURCES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
          {(['', 'INFO', 'WARN', 'ERROR', 'DEBUG'] as const).map((l) => (
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
        </div>

        <div className="terminal min-h-0 flex-1 overflow-auto p-3 text-[12px]">
          {isLoading ? (
            <FeedLoading />
          ) : isError ? (
            <FeedError error={error} />
          ) : rows.length === 0 ? (
            <div className="flex items-center gap-2 text-emerald-300/55">
              <Terminal className="size-4" />
              <span>// no records</span>
            </div>
          ) : (
            rows.map((r) => {
              const color = LEVEL_COLOR[r.level] ?? '#00ff88'
              const drillable = r.level === 'ERROR' || Boolean(r.details)
              return (
                <button
                  type="button"
                  key={r.id}
                  onClick={() => drillable && setSelected(r)}
                  disabled={!drillable}
                  className={`flex w-full gap-3 py-0.5 text-left ${
                    drillable ? 'cursor-pointer hover:bg-cyan-400/5' : 'cursor-default'
                  }`}
                >
                  <span className="shrink-0 text-emerald-300/55">{formatTime(r.timestamp)}</span>
                  <span
                    className="shrink-0 font-bold uppercase"
                    style={{ color, textShadow: `0 0 6px ${color}` }}
                  >
                    {r.level.padEnd(5, ' ')}
                  </span>
                  <span className="shrink-0 text-emerald-300/65">[{r.source || '-'}]</span>
                  <span className="flex-1 text-emerald-100/85">{r.message}</span>
                  {drillable && <span className="shrink-0 text-cyan-300/45">›</span>}
                </button>
              )
            })
          )}
        </div>

        <Pager page={page} totalPages={totalPages} total={total} onChange={setPage} />

        {selected && (
          <DetailOverlay
            title={`${selected.level} · ${selected.source}`}
            meta={formatTime(selected.timestamp)}
            accent={LEVEL_COLOR[selected.level] ?? '#00f0ff'}
            onClose={() => setSelected(null)}
          >
            <div className="terminal max-h-[55vh] overflow-auto whitespace-pre-wrap break-all p-3 text-[12px]">
              <span className="text-emerald-100/90">{selected.message}</span>
              {selected.details && (
                <>
                  {'\n\n'}
                  <span className="text-rose-300/90">{selected.details}</span>
                </>
              )}
            </div>
          </DetailOverlay>
        )}
      </div>
    </PageShell>
  )
}
