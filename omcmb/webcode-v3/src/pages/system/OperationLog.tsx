import { useMemo, useState } from 'react'
import { RefreshCcw, ScrollText } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useOperationLogs } from '@core/hooks/api/useLogs'
import type { OperationResult } from '@core/types/system'

import { StateBlock, MiniStat, RowHeader, KeywordToolbar, Pager } from './_shared'

const RESULT_LABEL: Record<OperationResult, string> = {
  success: '成功',
  failure: '失败',
  partial: '部分成功',
}
const RESULT_BADGE: Record<OperationResult, string> = {
  success: 'ok',
  failure: 'critical',
  partial: 'warning',
}

export default function OperationLog() {
  const [page, setPage] = useState(1)
  const pageSize = 30
  const [keyword, setKeyword] = useState('')
  const [result, setResult] = useState<OperationResult | ''>('')
  const [selectedLogId, setSelectedLogId] = useState<string | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(result ? { result } : {}),
    }),
    [page, keyword, result]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useOperationLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const selectedLog = rows.find((log) => log.id === selectedLogId) ?? null

  const okCount = useMemo(() => rows.filter((l) => l.result === 'success').length, [rows])
  const failCount = useMemo(() => rows.filter((l) => l.result === 'failure').length, [rows])

  return (
    <PageShell
      code="F06"
      title="AUDIT · 审计日志"
      subtitle="操作日志 / 安全日志 / 系统日志共用同一数据源"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <KeywordToolbar
            placeholder="操作人 / 目标 / 内容"
            value={keyword}
            onChange={(v) => {
              setKeyword(v)
              setPage(1)
            }}
          />
          {(['', 'success', 'failure', 'partial'] as const).map((r) => (
            <button
              key={r || 'all'}
              type="button"
              onClick={() => {
                setResult(r)
                setPage(1)
              }}
              className={`chip transition-all ${
                result === r ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60 hover:opacity-100'
              }`}
              style={{
                color: r
                  ? r === 'failure'
                    ? '#ff2d6f'
                    : r === 'partial'
                      ? '#ffaa00'
                      : '#00ff88'
                  : '#00f0ff',
              }}
            >
              {r ? RESULT_LABEL[r] : 'ALL'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="flex h-full flex-col gap-3">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <MiniStat label="本页记录" value={total.toLocaleString()} color="#00f0ff" icon={<ScrollText className="size-3.5" />} />
          <MiniStat label="成功 · OK" value={okCount} color="#00ff88" />
          <MiniStat label="失败 · FAIL" value={failCount} color={failCount > 0 ? '#ff2d6f' : '#525a78'} />
          <MiniStat label="页码" value={`${page}/${totalPages}`} color="#a855f7" />
        </div>

        <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
          <div className="scanline" />
          <div className="relative h-full overflow-auto p-3">
            <StateBlock
              isLoading={isLoading}
              isError={isError}
              error={error}
              isEmpty={rows.length === 0}
              emptyLabel="NO AUDIT RECORDS · 无审计记录"
            >
              <div className="space-y-1.5">
                <RowHeader cols="1.5fr_1fr_1fr_2fr_1fr_1.4fr">
                  <span>操作人 · IP</span>
                  <span>模块</span>
                  <span>类型</span>
                  <span>目标 / 内容</span>
                  <span>结果</span>
                  <span>时间</span>
                </RowHeader>
                {rows.map((l) => {
                  const badge = RESULT_BADGE[l.result] ?? 'unknown'
                  return (
                    <div
                      key={l.id}
                      role="button"
                      tabIndex={0}
                      onClick={() => setSelectedLogId(l.id)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          setSelectedLogId(l.id)
                        }
                      }}
                      className="fleet-row grid grid-cols-[1.5fr_1fr_1fr_2fr_1fr_1.4fr] items-center gap-3 rounded-sm px-3 py-2.5 cursor-pointer"
                      style={{
                        ['--row-color' as never]:
                          l.result === 'failure'
                            ? '#ff2d6f'
                            : l.result === 'partial'
                              ? '#ffaa00'
                              : '#00ff88',
                      }}
                    >
                      <div className="min-w-0">
                        <div className="truncate font-display text-sm font-bold text-cyan-100">
                          {l.operator || '—'}
                        </div>
                        <div className="truncate font-mono text-[10px] text-cyan-300/55">
                          {l.clientIp || '—'}
                        </div>
                      </div>
                      <div className="truncate text-xs text-cyan-100/85">{l.module || '—'}</div>
                      <div className="truncate font-mono text-[11px] uppercase text-cyan-300/70">
                        {l.operationType || '—'}
                      </div>
                      <div className="min-w-0">
                        <div className="truncate text-xs text-cyan-100/85">
                          {l.target || l.content || '—'}
                        </div>
                        {l.message ? (
                          <div className="truncate font-mono text-[10px] text-cyan-300/55">
                            {l.message}
                          </div>
                        ) : null}
                      </div>
                      <div>
                        <StatusBadge status={badge} label={RESULT_LABEL[l.result] ?? l.result} />
                      </div>
                      <div className="font-mono text-[11px] text-cyan-300/75">
                        {formatTime(l.operationTime)}
                      </div>
                    </div>
                  )
                })}
              </div>
            </StateBlock>
          </div>
        </div>

        <div className="glass-strong rounded-sm border border-cyan-500/15 p-4">
          <div className="mb-3 flex items-center justify-between gap-3">
            <div>
              <div className="font-display text-sm font-bold text-cyan-100">DETAIL</div>
              <div className="text-xs text-cyan-300/60">点击一条记录查看完整内容；若后端未返回 details，则展示摘要。</div>
            </div>
            {selectedLog ? (
              <button
                type="button"
                className="rounded-sm border border-cyan-500/25 px-3 py-1 text-xs text-cyan-100/80 transition-colors hover:bg-cyan-500/10"
                onClick={() => setSelectedLogId(null)}
              >
                CLOSE
              </button>
            ) : null}
          </div>

          {selectedLog ? (
            <div className="grid gap-3 md:grid-cols-2">
              <div className="space-y-2 text-sm text-cyan-50/90">
                <div><span className="text-cyan-300/55">操作人：</span>{selectedLog.operator || '—'}</div>
                <div><span className="text-cyan-300/55">IP：</span><span className="font-mono">{selectedLog.clientIp || '—'}</span></div>
                <div><span className="text-cyan-300/55">模块：</span>{selectedLog.module || '—'}</div>
                <div><span className="text-cyan-300/55">类型：</span>{selectedLog.operationType || '—'}</div>
                <div><span className="text-cyan-300/55">目标：</span>{selectedLog.target || '—'}</div>
                <div><span className="text-cyan-300/55">结果：</span>{selectedLog.result || '—'}</div>
                <div><span className="text-cyan-300/55">时间：</span>{formatTime(selectedLog.operationTime)}</div>
              </div>
              <div className="space-y-2 text-sm text-cyan-50/90">
                <div className="text-cyan-300/55">内容</div>
                <pre className="max-h-64 overflow-auto rounded-sm bg-cyan-950/40 p-3 text-xs leading-5 whitespace-pre-wrap break-all text-cyan-100/90">
                  {selectedLog.content || selectedLog.detail || selectedLog.message || '暂无详情内容'}
                </pre>
              </div>
            </div>
          ) : (
            <div className="text-sm text-cyan-300/60">当前没有选中记录。</div>
          )}
        </div>

        <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />
      </div>
    </PageShell>
  )
}
