import { useMemo, useState } from 'react'
import { Search, RefreshCcw, Loader2, X, FileWarning, ArrowLeftRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { formatTime } from '@/lib/format'
import { useSystemLogs, useOperationLogs, useNEMessageLogs } from '@core/hooks/api/useLogs'
import type { SystemLog } from '@core/mock/data/logs'
import type { NEMessageLog } from '@core/mock/data/logs'
import type { OperationLog, OperationResult, OperationType } from '@core/types/system'

// ──────────────────────────────────────────────────────────────────────────
// v3 STARFORGE 皮肤 · 日志中枢 (F06 · LOG NEXUS)
// 对照 v1 webcode/src/pages/log/{SystemLog,OperationLog,NEMessageLog}（真实后端数据的三页），
// 把主列表 / 概览统计 / 筛选 / 详情下钻 补齐到 v1 业务深度，并维持沉浸式 HUD 审美。
// v1 中 DeviceLog / HeartbeatLog / LogConfig / AlarmLog 为纯 mock（无 @core 真实端点），
// 本轮不引入伪数据页（见末尾 notes）。
// ──────────────────────────────────────────────────────────────────────────

type TabKey = 'system' | 'operation' | 'ne'

const TABS: { key: TabKey; label: string; sub: string }[] = [
  { key: 'system', label: 'SYSTEM · 系统日志', sub: 'RUNTIME SYSLOG · 30s' },
  { key: 'operation', label: 'AUDIT · 操作审计', sub: 'OPERATOR ACTIONS' },
  { key: 'ne', label: 'NE-MSG · 网元报文', sub: 'SOUTHBOUND/NORTHBOUND · 10s' },
]

const PAGE_SIZE = 50

// ── 系统日志级别配色 ──
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

// ── 操作审计：结果 / 类型配色 ──
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

// ── 网元报文类型配色 ──
type NEMsgType = NEMessageLog['messageType']
const NE_TYPE_COLOR: Record<NEMsgType, string> = {
  notification: '#5b9eff',
  alarm: '#ff2d6f',
  heartbeat: '#6b86b6',
  config_response: '#00f0ff',
  perf_data: '#00ff88',
}
const NE_TYPE_LABEL: Record<NEMsgType, string> = {
  notification: '通知',
  alarm: '告警上报',
  heartbeat: '心跳',
  config_response: '配置响应',
  perf_data: '性能上报',
}
const NE_TYPES = Object.keys(NE_TYPE_LABEL) as NEMsgType[]

// ──────────────────────────────────────────────────────────────────────────

export function LogsPage() {
  const [tab, setTab] = useState<TabKey>('system')

  return (
    <PageShell
      code="F06"
      title="LOG NEXUS · 日志中枢"
      subtitle={TABS.find((t) => t.key === tab)?.sub ?? ''}
      bare
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          {TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              onClick={() => setTab(t.key)}
              className={`chip transition-all ${
                tab === t.key ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'
              }`}
              style={{ color: '#00f0ff' }}
            >
              {t.label}
            </button>
          ))}
        </div>
      }
    >
      {tab === 'system' && <SystemLogView />}
      {tab === 'operation' && <OperationLogView />}
      {tab === 'ne' && <NEMessageView />}
    </PageShell>
  )
}

// ════════════════════════════════════════════════════════════════════════
// 1) 系统日志 —— CRT 终端流 (保留并增强 v1 webcode-v3 原有形态 + 详情下钻)
// ════════════════════════════════════════════════════════════════════════
function SystemLogView() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [level, setLevel] = useState<LogLevel | ''>('')
  const [source, setSource] = useState('')
  const [selected, setSelected] = useState<SystemLog | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(level ? { level } : {}),
      ...(source ? { source } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, level, source, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSystemLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 本页级别分布（后端不返回聚合，按当前页派生）
  const levelCount = useMemo(() => {
    const c: Record<string, number> = { DEBUG: 0, INFO: 0, WARN: 0, ERROR: 0 }
    rows.forEach((r) => {
      c[r.level] = (c[r.level] ?? 0) + 1
    })
    return c
  }, [rows])

  return (
    <div className="flex h-full flex-col gap-3">
      {/* 概览统计 */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatCard label="TOTAL" color="#00f0ff" value={total} hint="后端总量" />
        {(['ERROR', 'WARN', 'INFO', 'DEBUG'] as const).slice(0, 3).map((lv) => (
          <StatCard
            key={lv}
            label={lv}
            color={LEVEL_COLOR[lv]}
            value={levelCount[lv] ?? 0}
            hint="本页"
          />
        ))}
      </div>

      {/* 筛选条 */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-60 pl-9"
            placeholder="关键字搜索"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
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
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
      </div>

      {/* 终端流 */}
      <div className="terminal min-h-0 flex-1 overflow-auto p-3 text-[12px]">
        {isLoading ? (
          <FeedLoading />
        ) : isError ? (
          <FeedError error={error} />
        ) : rows.length === 0 ? (
          <div className="text-emerald-300/60">// no records</div>
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

      {/* 详情下钻 */}
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
  )
}

// ════════════════════════════════════════════════════════════════════════
// 2) 操作审计 —— 行列表 + 结果统计 + 详情下钻
// ════════════════════════════════════════════════════════════════════════
function OperationLogView() {
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [operationType, setOperationType] = useState<OperationType | ''>('')
  const [result, setResult] = useState<OperationResult | ''>('')
  const [selected, setSelected] = useState<OperationLog | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(operationType ? { operationType } : {}),
      ...(result ? { result } : {}),
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, operationType, result, keyword]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useOperationLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const resultCount = useMemo(() => {
    const c: Record<string, number> = { success: 0, failure: 0, partial: 0 }
    rows.forEach((r) => {
      c[r.result] = (c[r.result] ?? 0) + 1
    })
    return c
  }, [rows])

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatCard label="TOTAL" color="#00f0ff" value={total} hint="后端总量" />
        <StatCard label="成功" color={RESULT_COLOR.success} value={resultCount.success} hint="本页" />
        <StatCard label="失败" color={RESULT_COLOR.failure} value={resultCount.failure} hint="本页" />
        <StatCard label="部分" color={RESULT_COLOR.partial} value={resultCount.partial} hint="本页" />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-60 pl-9"
            placeholder="操作员 / 目标 / 内容"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value)
              setPage(1)
            }}
          />
        </div>
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
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
      </div>

      <GlassPanel strong className="min-h-0 flex-1 overflow-hidden">
        <div className="h-full overflow-auto">
          {/* 表头 */}
          <div className="sticky top-0 z-10 grid grid-cols-[150px_110px_120px_1.4fr_90px_140px] items-center gap-3 border-b border-cyan-500/15 bg-black/40 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55 backdrop-blur">
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
                  className="fleet-row grid w-full grid-cols-[150px_110px_120px_1.4fr_90px_140px] items-center gap-3 px-3 py-2.5 text-left"
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
  )
}

// ════════════════════════════════════════════════════════════════════════
// 3) 网元报文 —— 行列表 + 方向/成功统计 + 报文内容下钻
// ════════════════════════════════════════════════════════════════════════
function NEMessageView() {
  const [page, setPage] = useState(1)
  const [deviceSn, setDeviceSn] = useState('')
  const [messageType, setMessageType] = useState<NEMsgType | ''>('')
  const [selected, setSelected] = useState<NEMessageLog | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
      ...(messageType ? { messageType } : {}),
    }),
    [page, deviceSn, messageType]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useNEMessageLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const stats = useMemo(() => {
    let ok = 0
    let fail = 0
    let south = 0
    rows.forEach((r) => {
      if (r.success) ok += 1
      else fail += 1
      if (r.direction === 'southbound') south += 1
    })
    return { ok, fail, south, north: rows.length - south }
  }, [rows])

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatCard label="TOTAL" color="#00f0ff" value={total} hint="后端总量" />
        <StatCard label="SUCCESS" color="#00ff88" value={stats.ok} hint="本页" />
        <StatCard label="FAILURE" color="#ff2d6f" value={stats.fail} hint="本页" />
        <StatCard label="↓SOUTH / ↑NORTH" color="#a855f7" value={`${stats.south}/${stats.north}`} hint="本页方向" />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
          <input
            className="neon-input w-56 pl-9"
            placeholder="设备 SN"
            value={deviceSn}
            onChange={(e) => {
              setDeviceSn(e.target.value)
              setPage(1)
            }}
          />
        </div>
        {(['', ...NE_TYPES] as const).map((m) => (
          <button
            key={m || 'all'}
            type="button"
            onClick={() => {
              setMessageType(m as NEMsgType | '')
              setPage(1)
            }}
            className={`chip ${messageType === m ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60'}`}
            style={{ color: m ? NE_TYPE_COLOR[m as NEMsgType] : '#00f0ff' }}
          >
            {m ? NE_TYPE_LABEL[m as NEMsgType] : 'ALL'}
          </button>
        ))}
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          {isFetching ? 'SYNC…' : 'REFRESH'}
        </NeonButton>
      </div>

      <GlassPanel strong className="min-h-0 flex-1 overflow-hidden">
        <div className="h-full overflow-auto">
          <div className="sticky top-0 z-10 grid grid-cols-[160px_120px_90px_90px_1.6fr_140px] items-center gap-3 border-b border-cyan-500/15 bg-black/40 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55 backdrop-blur">
            <span>DEVICE</span>
            <span>TYPE</span>
            <span>DIR</span>
            <span>PROTO</span>
            <span>CONTENT</span>
            <span>TIME</span>
          </div>

          {isLoading ? (
            <FeedLoading />
          ) : isError ? (
            <FeedError error={error} />
          ) : rows.length === 0 ? (
            <EmptyFeed icon={<ArrowLeftRight className="size-10 text-emerald-400/60" />} text="无网元报文" />
          ) : (
            rows.map((r) => {
              const tc = NE_TYPE_COLOR[r.messageType] ?? '#6b86b6'
              const rowColor = r.success ? tc : '#ff2d6f'
              return (
                <button
                  type="button"
                  key={r.id}
                  onClick={() => setSelected(r)}
                  className="fleet-row grid w-full grid-cols-[160px_120px_90px_90px_1.6fr_140px] items-center gap-3 px-3 py-2.5 text-left"
                  style={{ ['--row-color' as never]: rowColor }}
                >
                  <span className="min-w-0">
                    <span className="block truncate font-display text-sm font-bold text-cyan-100">
                      {r.deviceName || '—'}
                    </span>
                    <span className="block truncate font-mono text-[10px] text-cyan-300/50">
                      {r.deviceSn}
                    </span>
                  </span>
                  <span className="chip" style={{ color: tc }}>
                    {NE_TYPE_LABEL[r.messageType] ?? r.messageType}
                  </span>
                  <span className="font-mono text-[11px] text-cyan-300/70">
                    {r.direction === 'southbound' ? '↓ S' : '↑ N'}
                  </span>
                  <span className="font-mono text-[11px] text-cyan-300/70">{r.protocol}</span>
                  <span className="flex min-w-0 items-center gap-2">
                    {!r.success && (
                      <span className="chip shrink-0" style={{ color: '#ff2d6f' }}>
                        FAIL
                      </span>
                    )}
                    <span className="truncate font-mono text-[11px] text-emerald-100/75">
                      {r.content || '—'}
                    </span>
                  </span>
                  <span className="font-mono text-[11px] text-cyan-300/70">
                    {formatTime(r.timestamp)}
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
          title={`${NE_TYPE_LABEL[selected.messageType] ?? selected.messageType} · ${selected.deviceSn}`}
          meta={formatTime(selected.timestamp)}
          accent={selected.success ? NE_TYPE_COLOR[selected.messageType] ?? '#00f0ff' : '#ff2d6f'}
          onClose={() => setSelected(null)}
        >
          <dl className="grid grid-cols-[110px_1fr] gap-x-4 gap-y-2.5 p-1 text-[12px]">
            <DRow k="设备名" v={selected.deviceName} />
            <DRow k="设备 SN" v={selected.deviceSn} mono />
            <DRow k="报文类型" v={NE_TYPE_LABEL[selected.messageType] ?? selected.messageType} />
            <DRow k="方向" v={selected.direction === 'southbound' ? '南向 ↓' : '北向 ↑'} />
            <DRow k="协议" v={selected.protocol} mono />
            <DRow
              k="结果"
              v={selected.success ? '成功' : '失败'}
              color={selected.success ? '#00ff88' : '#ff2d6f'}
            />
          </dl>
          <div className="mt-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
            报文内容
          </div>
          <div className="terminal mt-1 max-h-[45vh] overflow-auto whitespace-pre-wrap break-all p-3 text-[11px] text-emerald-100/85">
            {selected.content || '// empty'}
          </div>
        </DetailOverlay>
      )}
    </div>
  )
}

// ════════════════════════════════════════════════════════════════════════
// 共享小组件（本模块内，避免触碰共享组件）
// ════════════════════════════════════════════════════════════════════════
function StatCard({
  label,
  color,
  value,
  hint,
}: {
  label: string
  color: string
  value: number | string
  hint?: string
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="flex items-center justify-between">
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
          {label}
        </span>
        {hint && <span className="font-mono text-[9px] text-cyan-300/35">{hint}</span>}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight tabular-nums"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  onChange,
}: {
  page: number
  totalPages: number
  total: number
  onChange: (updater: (p: number) => number) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onChange((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onChange((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

function FeedLoading() {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function FeedError({ error }: { error: unknown }) {
  return (
    <div className="m-3 border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      SYNC FAILED · {error instanceof Error ? error.message : '未知错误'}
    </div>
  )
}

function EmptyFeed({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      {icon}
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-emerald-300/70">{text}</div>
    </div>
  )
}

function DetailOverlay({
  title,
  meta,
  accent,
  onClose,
  children,
}: {
  title: string
  meta: string
  accent: string
  onClose: () => void
  children: React.ReactNode
}) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="w-full max-w-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <GlassPanel strong className="overflow-hidden">
          <div
            className="flex items-center justify-between border-b px-4 py-3"
            style={{ borderColor: `${accent}40` }}
          >
            <div className="flex items-center gap-2">
              <span
                className="size-1.5 rounded-full"
                style={{ background: accent, boxShadow: `0 0 8px ${accent}` }}
              />
              <span className="font-display text-sm font-bold text-cyan-100">{title}</span>
              <span className="font-mono text-[10px] text-cyan-300/50">{meta}</span>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="text-cyan-300/60 transition-colors hover:text-cyan-100"
            >
              <X className="size-4" />
            </button>
          </div>
          <div className="p-4">{children}</div>
        </GlassPanel>
      </div>
    </div>
  )
}

function DRow({
  k,
  v,
  mono,
  color,
}: {
  k: string
  v?: string | null
  mono?: boolean
  color?: string
}) {
  return (
    <>
      <dt className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/50">{k}</dt>
      <dd
        className={`break-all ${mono ? 'font-mono text-[11px]' : 'text-[12px]'}`}
        style={{ color: color ?? '#cfe6ff' }}
      >
        {v || '—'}
      </dd>
    </>
  )
}
