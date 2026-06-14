import { useMemo, useState } from 'react'
import { FileWarning, RefreshCcw, Search } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useOperationLogs } from '@core/hooks/api/useLogs'
import type { OperationLog, OperationResult, OperationType } from '@core/types/system'

import {
  DRow,
  DetailOverlay,
  EmptyFeed,
  FeedError,
  FeedLoading,
  FilterInput,
  LOG_PAGE_SIZE,
  Pager,
  StatCard,
} from './_shared'

// ──────────────────────────────────────────────────────────────────────────
// /logs/operation —— 操作审计（对照 v1 webcode log/operation · OperationLog）
// 真实端点：useOperationLogs → adminApi GET /admin/audit-logs
// HUD 形态：结果统计 + 行列表 + 详情下钻
// ──────────────────────────────────────────────────────────────────────────

const RESULT_COLOR: Record<OperationResult, string> = {
  success: '#00ff88',
  failure: '#ff2d6f',
  partial: '#ffaa00',
}
const RESULT_LABEL: Record<OperationResult, string> = {
  success: '成功',
  failure: '失败',
  partial: '部分',
}
const OP_TYPES: OperationType[] = [
  'create',
  'update',
  'delete',
  'query',
  'export',
  'import',
  'login',
  'logout',
  'execute',
  'deploy',
  'approve',
]

const COLS = 'grid-cols-[150px_110px_120px_1.4fr_90px_140px]'

export default function OperationLogPage() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [operationType, setOperationType] = useState<OperationType | ''>('')
  const [result, setResult] = useState<OperationResult | ''>('')
  const [selected, setSelected] = useState<OperationLog | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: LOG_PAGE_SIZE,
      ...(operationType ? { operationType } : {}),
      ...(result ? { result } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, operationType, result, keyword],
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOperationLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / LOG_PAGE_SIZE))

  const resultCount = useMemo(() => {
    const c: Record<string, number> = { success: 0, failure: 0, partial: 0 }
    rows.forEach((r) => {
      c[r.result] = (c[r.result] ?? 0) + 1
    })
    return c
  }, [rows])

  return (
    <PageShell
      code="F06"
      title="AUDIT LOG · 操作审计"
      subtitle="OPERATOR ACTIONS"
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
          <StatCard label="成功" color={RESULT_COLOR.success} value={resultCount.success} hint="本页" />
          <StatCard label="失败" color={RESULT_COLOR.failure} value={resultCount.failure} hint="本页" />
          <StatCard label="部分" color={RESULT_COLOR.partial} value={resultCount.partial} hint="本页" />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <FilterInput
            value={keyword}
            onChange={(v) => {
              setKeyword(v)
              setPage(1)
            }}
            placeholder="操作员 / 目标 / 内容"
            icon={<Search />}
          />
          <select
            className="neon-input w-36"
            value={operationType}
            onChange={(e) => {
              setOperationType(e.target.value as OperationType | '')
              setPage(1)
            }}
          >
            <option value="">全部类型</option>
            {OP_TYPES.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
          {(['', 'success', 'failure', 'partial'] as const).map((r) => (
            <button
              key={r || 'all'}
              type="button"
              onClick={() => {
                setResult(r as OperationResult | '')
                setPage(1)
              }}
              className={`chip ${result === r ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60'}`}
              style={{ color: r ? RESULT_COLOR[r as OperationResult] : '#00f0ff' }}
            >
              {r ? RESULT_LABEL[r as OperationResult] : 'ALL'}
            </button>
          ))}
        </div>

        <GlassPanel strong className="min-h-0 flex-1 overflow-hidden">
          <div className="h-full overflow-auto">
            <div
              className={`sticky top-0 z-10 grid ${COLS} items-center gap-3 border-b border-cyan-500/15 bg-black/40 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55 backdrop-blur`}
            >
              <span>OPERATOR</span>
              <span>TYPE</span>
              <span>MODULE</span>
              <span>TARGET / 内容</span>
              <span>RESULT</span>
              <span>TIME</span>
            </div>

            {isLoading ? (
              <FeedLoading />
            ) : isError ? (
              <FeedError error={error} />
            ) : rows.length === 0 ? (
              <EmptyFeed icon={<FileWarning className="size-10 text-emerald-400/60" />} text="无审计记录" />
            ) : (
              rows.map((r) => {
                const rc = RESULT_COLOR[r.result] ?? '#6b86b6'
                return (
                  <button
                    type="button"
                    key={r.id}
                    onClick={() => setSelected(r)}
                    className={`fleet-row grid w-full ${COLS} items-center gap-3 px-3 py-2.5 text-left`}
                    style={{ ['--row-color' as never]: rc }}
                  >
                    <span className="truncate font-display text-sm font-bold text-cyan-100">
                      {r.operator || '—'}
                    </span>
                    <span className="chip" style={{ color: '#5b9eff' }}>
                      {r.operationType}
                    </span>
                    <span className="truncate font-mono text-[11px] text-cyan-300/70">
                      {r.module || '—'}
                    </span>
                    <span className="min-w-0">
                      <span className="block truncate text-xs text-cyan-100/85">
                        {r.target || r.content || r.message || '—'}
                      </span>
                      <span className="block truncate font-mono text-[10px] text-cyan-300/45">
                        {r.clientIp || '—'}
                      </span>
                    </span>
                    <span className="chip" style={{ color: rc }}>
                      {RESULT_LABEL[r.result] ?? r.result}
                    </span>
                    <span className="font-mono text-[11px] text-cyan-300/70">
                      {formatTime(r.operationTime || r.startTime)}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        </GlassPanel>

        <Pager page={page} totalPages={totalPages} total={total} onChange={setPage} />

        {selected && (
          <DetailOverlay
            title={`${selected.operationType} · ${selected.operator || '—'}`}
            meta={formatTime(selected.operationTime || selected.startTime)}
            accent={RESULT_COLOR[selected.result] ?? '#00f0ff'}
            onClose={() => setSelected(null)}
          >
            <dl className="grid grid-cols-[110px_1fr] gap-x-4 gap-y-2.5 p-1 text-[12px]">
              <DRow k="操作员" v={selected.operator} />
              <DRow k="客户端 IP" v={selected.clientIp} mono />
              <DRow k="模块" v={selected.module} mono />
              <DRow k="类型" v={selected.operationType} />
              <DRow k="目标" v={selected.target} />
              <DRow
                k="结果"
                v={RESULT_LABEL[selected.result] ?? selected.result}
                color={RESULT_COLOR[selected.result]}
              />
              <DRow k="提示" v={selected.message} />
              {selected.reason && <DRow k="原因" v={selected.reason} />}
              <DRow k="开始" v={formatTime(selected.startTime || selected.operationTime)} mono />
              {selected.endTime && <DRow k="结束" v={formatTime(selected.endTime)} mono />}
            </dl>
            {(selected.content || selected.detail) && (
              <div className="terminal mt-3 max-h-[40vh] overflow-auto whitespace-pre-wrap break-all p-3 text-[11px] text-emerald-100/85">
                {selected.detail || selected.content}
              </div>
            )}
          </DetailOverlay>
        )}
      </div>
    </PageShell>
  )
}
