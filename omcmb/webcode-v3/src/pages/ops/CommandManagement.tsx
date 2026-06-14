import { useMemo, useState } from 'react'
import { Eye } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { formatTime } from '@/lib/format'
import type { OpsCommandRecord } from '@core/mock/data/opsTools'
import { useOpsCommandRecords } from '@core/hooks/api/useOpsTools'

import { IconBtn, Pager, StatCard, StateBlock, Toolbar } from './_shared'
import { CommandDetailDrawer } from './CommandDetailDrawer'

const PAGE_SIZE = 20

function durationText(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms}ms`
}

export default function CommandManagement() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [deviceSn, setDeviceSn] = useState('')
  const [operator, setOperator] = useState('')
  const [result, setResult] = useState<'' | 'success' | 'failed'>('')

  const [detail, setDetail] = useState<OpsCommandRecord | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(operator.trim() ? { operator: operator.trim() } : {}),
      ...(result === 'success' ? { success: true } : result === 'failed' ? { success: false } : {}),
    }),
    [page, deviceSn, operator, result],
  )

  const { data, isLoading, isError, isFetching, refetch } = useOpsCommandRecords(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0

  // keyword 后端不支持，对当前页结果叠加前端模糊匹配
  const filtered = useMemo(() => {
    if (!keyword.trim()) return rows
    const kw = keyword.trim().toLowerCase()
    return rows.filter(
      (r) => r.commandText.toLowerCase().includes(kw) || r.deviceSn.toLowerCase().includes(kw),
    )
  }, [rows, keyword])

  const okCount = useMemo(() => rows.filter((r) => r.success).length, [rows])
  const failCount = rows.length - okCount

  return (
    <PageShell
      code="F06"
      title="OPS · 指令流水"
      subtitle="COMMAND LOG · EXECUTED MML / RPC RECORDS"
      isFetching={isFetching}
      bare
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <StatCard label="本页指令 · PAGE" value={rows.length} color="#00f0ff" />
        <StatCard label="成功 · SUCCESS" value={okCount} color="#00ff88" />
        <StatCard label="失败 · FAILED" value={failCount} color={failCount > 0 ? '#ff2d6f' : '#525a78'} />
      </div>

      <Toolbar
        keyword={keyword}
        keywordPlaceholder="指令 / 设备"
        onKeyword={(v) => setKeyword(v)}
        isFetching={isFetching}
        onRefresh={() => refetch()}
      >
        <input
          className="neon-input w-36"
          placeholder="设备 SN"
          value={deviceSn}
          onChange={(e) => {
            setDeviceSn(e.target.value)
            setPage(1)
          }}
        />
        <input
          className="neon-input w-32"
          placeholder="操作人"
          value={operator}
          onChange={(e) => {
            setOperator(e.target.value)
            setPage(1)
          }}
        />
        {(['', 'success', 'failed'] as const).map((r) => (
          <button
            key={r || 'all'}
            type="button"
            onClick={() => {
              setResult(r)
              setPage(1)
            }}
            className={`chip ${result === r ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55'}`}
            style={{ color: r === 'success' ? '#00ff88' : r === 'failed' ? '#ff2d6f' : '#00f0ff' }}
          >
            {r === 'success' ? '成功' : r === 'failed' ? '失败' : 'ALL'}
          </button>
        ))}
      </Toolbar>

      <StateBlock
        loading={isLoading}
        error={isError}
        empty={filtered.length === 0}
        emptyText="NO RECORDS · 无指令记录"
      >
        <div className="overflow-hidden rounded-sm border border-cyan-500/12">
          <div className="grid grid-cols-[150px_2fr_1.4fr_100px_80px_70px_60px] gap-3 border-b border-cyan-500/15 bg-cyan-500/[0.05] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/70">
            <span>时间</span>
            <span>指令 · COMMAND</span>
            <span>设备 · DEVICE</span>
            <span>操作人</span>
            <span>耗时</span>
            <span>结果</span>
            <span className="text-right">操作</span>
          </div>
          {filtered.map((r) => {
            const ok = r.success
            const durColor = r.duration > 5000 ? '#ff2d6f' : r.duration > 2000 ? '#ffaa00' : '#00ff88'
            return (
              <div
                key={r.id}
                className="fleet-row grid grid-cols-[150px_2fr_1.4fr_100px_80px_70px_60px] items-center gap-3 px-3 py-2"
                style={{ ['--row-color' as never]: ok ? '#00ff88' : '#ff2d6f' }}
              >
                <span className="font-mono text-[11px] text-cyan-300/70">{formatTime(r.executeTime)}</span>
                <span className="truncate rounded-sm bg-black/30 px-2 py-0.5 font-mono text-[11px] text-emerald-300/85">
                  {r.commandText}
                </span>
                <span className="min-w-0">
                  <span className="block truncate text-[11px] text-cyan-100/85">{r.deviceName}</span>
                  <span className="block truncate font-mono text-[10px] text-cyan-300/55">{r.deviceSn}</span>
                </span>
                <span className="truncate text-[11px] text-cyan-300/75">{r.operator}</span>
                <span className="font-mono text-[11px]" style={{ color: durColor }}>
                  {durationText(r.duration)}
                </span>
                <span>
                  <span className="chip" style={{ color: ok ? '#00ff88' : '#ff2d6f' }}>
                    {ok ? '成功' : '失败'}
                  </span>
                </span>
                <span className="text-right">
                  <IconBtn title="详情" color="#00f0ff" onClick={() => setDetail(r)}>
                    <Eye className="size-3.5" />
                  </IconBtn>
                </span>
              </div>
            )
          })}
        </div>
        <Pager page={page} total={total} pageSize={PAGE_SIZE} onPage={setPage} />
      </StateBlock>

      <CommandDetailDrawer record={detail} open={detail !== null} onClose={() => setDetail(null)} />
    </PageShell>
  )
}
