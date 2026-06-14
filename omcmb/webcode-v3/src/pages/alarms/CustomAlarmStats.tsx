import { useCallback, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ChevronRight,
  Layers,
  Loader2,
  Plus,
  RefreshCcw,
  Trash2,
  X,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { SeverityBar } from '@/components/viz/SeverityBar'
import { formatTime } from '@/lib/format'
import { useAlarmList } from '@core/hooks/api/useAlarms'
import type { AlarmFilter } from '@core/types/alarm'
import type { AlarmSeverity } from '@core/types/common'
import { neTypeLabel } from './AlarmDetailPanel'

const SEV_LABEL: Record<AlarmSeverity, string> = {
  critical: '紧急',
  major: '重要',
  minor: '次要',
  warning: '警告',
}
const SEV_COLOR: Record<AlarmSeverity, string> = {
  critical: '#ff2d6f',
  major: '#ff7a1a',
  minor: '#ffd400',
  warning: '#5b9eff',
}
const SEV_KEYS: AlarmSeverity[] = ['critical', 'major', 'minor', 'warning']

const PAGE_SIZE = 50

interface CustomGroup {
  id: string
  name: string
  keyword: string
  severity: AlarmSeverity | ''
}

// 预置分组（按设备制式 / 等级聚焦真实活动告警）
const DEFAULT_GROUPS: CustomGroup[] = [
  { id: 'all', name: '全部活动告警', keyword: '', severity: '' },
  { id: 'critical', name: '紧急告警视图', keyword: '', severity: 'critical' },
  { id: 'enb', name: 'LTE / eNB', keyword: 'eNB', severity: '' },
  { id: 'gnb', name: '5G / gNB', keyword: 'gNB', severity: '' },
]

export default function CustomAlarmStats() {
  const navigate = useNavigate()
  const [groups, setGroups] = useState<CustomGroup[]>(DEFAULT_GROUPS)
  const [activeId, setActiveId] = useState<string>('all')
  const [page, setPage] = useState(1)

  // 新建分组表单
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [newKeyword, setNewKeyword] = useState('')
  const [newSeverity, setNewSeverity] = useState<AlarmSeverity | ''>('')
  const [createError, setCreateError] = useState<string | null>(null)

  const activeGroup = useMemo(
    () => groups.find((g) => g.id === activeId) ?? groups[0],
    [groups, activeId]
  )

  const params = useMemo<AlarmFilter & { page: number; pageSize: number; isActive?: boolean }>(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      isActive: true,
      ...(activeGroup?.keyword.trim() ? { keyword: activeGroup.keyword.trim() } : {}),
      ...(activeGroup?.severity ? { severity: activeGroup.severity } : {}),
    }),
    [page, activeGroup]
  )

  const { data, isLoading, isError, error, isFetching, refetch } = useAlarmList(params)
  const rows = useMemo(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 当前页严重度统计（基于真实命中行）
  const stats = useMemo(() => {
    const base: Record<AlarmSeverity, number> = { critical: 0, major: 0, minor: 0, warning: 0 }
    let unacked = 0
    let unread = 0
    for (const a of rows) {
      base[a.severity] += 1
      if (a.dealState === '0') unacked += 1
      if (a.unread === '1') unread += 1
    }
    return { ...base, unacked, unread }
  }, [rows])

  const selectGroup = useCallback((id: string) => {
    setActiveId(id)
    setPage(1)
  }, [])

  const addGroup = useCallback(() => {
    const name = newName.trim()
    if (!name) {
      setCreateError('请输入分组名称')
      return
    }
    if (groups.some((g) => g.name === name)) {
      setCreateError('分组名称已存在')
      return
    }
    const id = `g-${Date.now()}`
    setGroups((prev) => [
      ...prev,
      { id, name, keyword: newKeyword.trim(), severity: newSeverity },
    ])
    setActiveId(id)
    setPage(1)
    setCreating(false)
    setNewName('')
    setNewKeyword('')
    setNewSeverity('')
    setCreateError(null)
  }, [newName, newKeyword, newSeverity, groups])

  const removeGroup = useCallback(
    (id: string) => {
      setGroups((prev) => {
        const next = prev.filter((g) => g.id !== id)
        if (id === activeId) {
          setActiveId(next[0]?.id ?? '')
          setPage(1)
        }
        return next
      })
    },
    [activeId]
  )

  return (
    <PageShell
      code="F04"
      title="CUSTOM ALARM STATS · 自定义告警统计"
      subtitle="USER-DEFINED ALARM GROUPS · LIVE SLICE"
      isFetching={isFetching}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="grid h-full grid-cols-12 gap-3 p-4">
        {/* 左侧分组树 */}
        <div className="col-span-12 lg:col-span-3">
          <GlassPanel title="GROUPS · 告警分组" meta={`${groups.length}`}>
            <div className="divide-y divide-cyan-500/8">
              {groups.map((g) => {
                const active = g.id === activeId
                const removable = !DEFAULT_GROUPS.some((d) => d.id === g.id)
                return (
                  <div
                    key={g.id}
                    className={`group flex items-center gap-2 px-3 py-2.5 ${
                      active ? 'bg-cyan-500/10' : 'hover:bg-cyan-500/5'
                    }`}
                  >
                    <button
                      type="button"
                      onClick={() => selectGroup(g.id)}
                      className="flex min-w-0 flex-1 items-center gap-2 text-left"
                    >
                      <Layers
                        className={`size-3.5 shrink-0 ${
                          active ? 'text-cyan-200' : 'text-cyan-300/45'
                        }`}
                      />
                      <span className="min-w-0">
                        <span
                          className={`block truncate font-display text-xs font-bold ${
                            active ? 'text-cyan-100' : 'text-cyan-200/80'
                          }`}
                        >
                          {g.name}
                        </span>
                        <span className="block truncate font-mono text-[9px] uppercase tracking-[0.12em] text-cyan-300/45">
                          {g.severity ? SEV_LABEL[g.severity] : '全部等级'}
                          {g.keyword ? ` · ${g.keyword}` : ''}
                        </span>
                      </span>
                    </button>
                    {removable && (
                      <button
                        type="button"
                        title="删除分组"
                        onClick={() => removeGroup(g.id)}
                        className="flex size-5 shrink-0 items-center justify-center rounded-sm border border-rose-500/25 text-rose-300/70 opacity-0 transition-opacity hover:border-rose-400/60 hover:text-rose-200 group-hover:opacity-100"
                      >
                        <Trash2 className="size-3" />
                      </button>
                    )}
                  </div>
                )
              })}
            </div>

            <div className="border-t border-cyan-500/15 p-3">
              {creating ? (
                <div className="space-y-2">
                  <input
                    className="neon-input w-full"
                    placeholder="分组名称"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                  />
                  <input
                    className="neon-input w-full"
                    placeholder="关键字（设备/告警名，可空）"
                    value={newKeyword}
                    onChange={(e) => setNewKeyword(e.target.value)}
                  />
                  <select
                    className="neon-input w-full"
                    value={newSeverity}
                    onChange={(e) => setNewSeverity(e.target.value as AlarmSeverity | '')}
                  >
                    <option value="" className="bg-[#03050d]">
                      全部等级
                    </option>
                    {SEV_KEYS.map((s) => (
                      <option key={s} value={s} className="bg-[#03050d]">
                        {SEV_LABEL[s]}
                      </option>
                    ))}
                  </select>
                  {createError && (
                    <div className="font-mono text-[10px] text-rose-300">{createError}</div>
                  )}
                  <div className="flex gap-2">
                    <NeonButton icon={<Plus />} onClick={addGroup}>
                      保存
                    </NeonButton>
                    <NeonButton
                      icon={<X />}
                      onClick={() => {
                        setCreating(false)
                        setCreateError(null)
                      }}
                    >
                      取消
                    </NeonButton>
                  </div>
                </div>
              ) : (
                <NeonButton icon={<Plus />} onClick={() => setCreating(true)}>
                  新建分组
                </NeonButton>
              )}
            </div>
          </GlassPanel>
        </div>

        {/* 右侧统计 + 列表 */}
        <div className="col-span-12 space-y-3 lg:col-span-9">
          <div className="grid grid-cols-3 gap-3 lg:grid-cols-6">
            {SEV_KEYS.map((s) => (
              <Stat key={s} label={SEV_LABEL[s]} color={SEV_COLOR[s]} value={stats[s]} />
            ))}
            <Stat label="未确认" color="#00f0ff" value={stats.unacked} />
            <Stat label="未读" color="#a855f7" value={stats.unread} />
          </div>

          <GlassPanel
            title={`${activeGroup?.name ?? '告警分组'} · 命中告警`}
            meta={`TOTAL ${total} · PAGE ${page}/${totalPages}`}
          >
            {rows.length > 0 && (
              <div className="grid grid-cols-[28px_2fr_1.3fr_1fr_1fr] items-center gap-3 border-b border-cyan-500/10 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
                <span />
                <span>ALARM · 告警</span>
                <span>DEVICE · 设备</span>
                <span>NE · 网元</span>
                <span className="text-right">TIME · 时间</span>
              </div>
            )}

            <div>
              {isLoading ? (
                <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
                  <Loader2 className="size-4 animate-spin" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
                </div>
              ) : isError ? (
                <div className="flex flex-col items-center gap-3 px-4 py-10">
                  <div className="font-mono text-sm text-rose-300">
                    FAILURE · {error instanceof Error ? error.message : '未知错误'}
                  </div>
                  <NeonButton tone="danger" icon={<RefreshCcw />} onClick={() => refetch()}>
                    RETRY
                  </NeonButton>
                </div>
              ) : rows.length === 0 ? (
                <div className="flex flex-col items-center justify-center gap-2 py-14">
                  <Layers className="size-9 text-cyan-300/35" />
                  <span className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
                    NO MATCH · 该分组暂无命中告警
                  </span>
                </div>
              ) : (
                rows.map((a) => (
                  <button
                    key={a.id}
                    type="button"
                    onClick={() =>
                      navigate(`/alarms?keyword=${encodeURIComponent(a.deviceSn || a.alarmIdentifier)}`)
                    }
                    className="grid w-full grid-cols-[28px_2fr_1.3fr_1fr_1fr] items-center gap-3 border-b border-cyan-500/8 px-3.5 py-2.5 text-left last:border-b-0 hover:bg-cyan-500/5"
                  >
                    <SeverityBar severity={a.severity} />
                    <div className="min-w-0">
                      <div className="truncate font-display text-sm font-bold text-cyan-100">
                        {a.alarmName || a.alarmIdentifier}
                      </div>
                      <div className="truncate font-mono text-[10px] text-cyan-300/50">
                        CODE {a.alarmIdentifier}
                      </div>
                    </div>
                    <div className="min-w-0">
                      <div className="truncate text-xs text-cyan-100/85">
                        {a.deviceName || '—'}
                      </div>
                      <div className="truncate font-mono text-[10px] text-cyan-300/50">
                        {a.deviceSn}
                      </div>
                    </div>
                    <div className="truncate text-xs text-cyan-100/80">
                      {neTypeLabel(a.neType)}
                    </div>
                    <div className="flex items-center justify-end gap-1.5">
                      <span className="font-mono text-[11px] text-cyan-300/70">
                        {formatTime(a.eventTime)}
                      </span>
                      <ChevronRight className="size-3.5 text-cyan-300/30" />
                    </div>
                  </button>
                ))
              )}
            </div>
          </GlassPanel>

          <div className="flex items-center justify-between">
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
        </div>
      </div>
    </PageShell>
  )
}

function Stat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[9px] uppercase tracking-[0.12em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}
