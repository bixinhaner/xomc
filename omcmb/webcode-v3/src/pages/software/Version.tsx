import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, RefreshCcw, Loader2, Package, Star, Activity, ChevronRight } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { formatTime } from '@/lib/format'
import { useSoftwareVersions } from '@core/hooks/api/useSoftware'
import type { SoftwareVersion, VersionStatus } from '@core/mock/data/software'

import { NEON, VERSION_STATUS, StatCard, Syncing, ErrorBlock, EmptyBlock, Pager, formatFileSize } from './_shared'

// ---------------------------------------------------------------------------
// 版本查询 · software/version
// 固件版本注册表 —— 只读，行 → 详情 software/version/:id
// ---------------------------------------------------------------------------

const STATUS_FILTERS = ['', 'current', 'beta', 'deprecated', 'archived'] as const

export default function Version() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 24
  const [keyword, setKeyword] = useState('')
  const [vendor, setVendor] = useState('')
  const [deviceType, setDeviceType] = useState('')
  const [status, setStatus] = useState<VersionStatus | ''>('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(vendor ? { vendor } : {}),
      ...(deviceType ? { deviceType } : {}),
      ...(status ? { status } : {}),
    }),
    [page, vendor, deviceType, status]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useSoftwareVersions(params)
  const allItems = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const items = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return allItems
    return allItems.filter(
      (v) =>
        v.versionCode.toLowerCase().includes(k) ||
        v.versionName.toLowerCase().includes(k) ||
        (v.fileName ?? '').toLowerCase().includes(k)
    )
  }, [allItems, keyword])

  const stat = useMemo(() => {
    const recommended = allItems.filter((v) => v.recommend).length
    const current = allItems.filter((v) => v.status === 'current').length
    const totalBytes = allItems.reduce((acc, v) => acc + (v.fileSize || 0), 0)
    return { recommended, current, totalBytes }
  }, [allItems])

  return (
    <PageShell
      code="F06"
      title="VERSION REGISTRY · 版本查询"
      subtitle="FIRMWARE VERSION CATALOG · READ-ONLY"
      isFetching={isFetching}
      bare
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
            <input
              className="neon-input w-64 pl-9"
              placeholder="版本号 / 名称 / 文件名"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {/* 统计 */}
      <div className="mb-3 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatCard label="版本总数 · REGISTRY" value={total.toLocaleString()} color={NEON.cyan} icon={<Package className="size-4" />} />
        <StatCard label="推荐版本 · RECOMMENDED" value={stat.recommended.toLocaleString()} color={NEON.gold} icon={<Star className="size-4" />} />
        <StatCard label="当前在用 · CURRENT" value={stat.current.toLocaleString()} color={NEON.green} icon={<Activity className="size-4" />} />
        <StatCard label="本页容量 · SIZE" value={formatFileSize(stat.totalBytes)} color={NEON.violet} icon={<Package className="size-4" />} />
      </div>

      {/* 筛选 */}
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <FilterSelect value={vendor} onChange={(v) => { setVendor(v); setPage(1) }} placeholder="厂商" options={['华为', '中兴', '爱立信', '诺基亚', 'Baicells']} />
        <FilterSelect value={deviceType} onChange={(v) => { setDeviceType(v); setPage(1) }} placeholder="制式" options={['eNB', 'gNB', 'RRU', 'AAU', 'CPE']} />
        {STATUS_FILTERS.map((s) => (
          <button
            key={s || 'all'}
            type="button"
            onClick={() => { setStatus(s); setPage(1) }}
            className={`chip transition-all ${status === s ? 'shadow-[0_0_10px_currentColor]' : 'opacity-55 hover:opacity-100'}`}
            style={{ color: s ? VERSION_STATUS[s].color : NEON.cyan }}
          >
            {s ? VERSION_STATUS[s].label : 'ALL'}
          </button>
        ))}
        {isFetching ? (
          <span className="flex items-center gap-1.5 text-[11px] text-cyan-300/60">
            <Loader2 className="size-3 animate-spin" /> SYNC
          </span>
        ) : null}
      </div>

      {/* 版本网格 */}
      {isLoading ? (
        <Syncing label="SCANNING REGISTRY…" />
      ) : isError ? (
        <ErrorBlock msg={error instanceof Error ? error.message : '未知错误'} />
      ) : items.length === 0 ? (
        <EmptyBlock label="NO FIRMWARE VERSIONS · 无固件版本" />
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {items.map((v) => (
            <VersionCard key={v.id} v={v} onOpen={() => navigate(`/software/version/${v.id}`)} />
          ))}
        </div>
      )}

      <Pager page={page} totalPages={totalPages} pageSize={pageSize} total={total} onPage={setPage} />
    </PageShell>
  )
}

function VersionCard({ v, onOpen }: { v: SoftwareVersion; onOpen: () => void }) {
  const sc = VERSION_STATUS[v.status] ?? { label: v.status, color: NEON.dim }
  return (
    <div
      className="glass group relative cursor-pointer overflow-hidden rounded-sm border-l-2 p-3 transition-all hover:bg-cyan-500/[0.04]"
      style={{ borderLeftColor: sc.color }}
      onClick={onOpen}
    >
      <div className="scanline" />
      <div className="relative">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="truncate font-mono text-sm font-bold text-cyan-100">{v.versionCode}</div>
            <div className="truncate font-mono text-[10px] text-cyan-300/55">{v.versionName}</div>
          </div>
          {v.recommend ? (
            <Star className="size-4 shrink-0" style={{ color: NEON.gold, fill: NEON.gold, filter: `drop-shadow(0 0 5px ${NEON.gold})` }} />
          ) : null}
        </div>

        <div className="mt-2 flex flex-wrap items-center gap-1.5">
          <span className="chip" style={{ color: sc.color }}>
            <span className="size-1.5 rounded-full bg-current shadow-[0_0_6px_currentColor]" />
            {sc.label}
          </span>
          <span className="chip" style={{ color: NEON.blue }}>{v.deviceType || '—'}</span>
          <span className="chip" style={{ color: NEON.cyan }}>{v.vendor || v.manufacturer || '—'}</span>
        </div>

        <div className="mt-2 grid grid-cols-2 gap-1 font-mono text-[10px] text-cyan-300/65">
          <div>SIZE <span className="text-cyan-100/85">{formatFileSize(v.fileSize)}</span></div>
          <div>DATE <span className="text-cyan-100/85">{v.releaseDate ? formatTime(v.releaseDate).slice(0, 10) : '—'}</span></div>
        </div>

        {v.releaseNotes ? (
          <div className="mt-1.5 line-clamp-2 text-[11px] leading-snug text-cyan-100/55">{v.releaseNotes}</div>
        ) : null}

        <div className="mt-2 flex items-center justify-between border-t border-cyan-500/10 pt-2">
          <span className="truncate font-mono text-[9px] text-cyan-300/40">
            {v.checksum ? `MD5 ${v.checksum.slice(0, 12)}…` : v.fileName || '—'}
          </span>
          <span className="flex items-center gap-0.5 font-mono text-[10px] text-cyan-300/55 opacity-60 transition-opacity group-hover:opacity-100">
            DETAIL <ChevronRight className="size-3" />
          </span>
        </div>
      </div>
    </div>
  )
}

function FilterSelect({
  value,
  onChange,
  placeholder,
  options,
}: {
  value: string
  onChange: (v: string) => void
  placeholder: string
  options: string[]
}) {
  return (
    <select className="neon-input w-32 cursor-pointer" value={value} onChange={(e) => onChange(e.target.value)}>
      <option value="">{placeholder} · 全部</option>
      {options.map((o) => (
        <option key={o} value={o}>{o}</option>
      ))}
    </select>
  )
}
