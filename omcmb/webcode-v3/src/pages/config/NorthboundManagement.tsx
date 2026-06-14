import { useState } from 'react'
import {
  RefreshCcw,
  Trash2,
  Loader2,
  Cloud,
  Server,
  Radar,
  CheckCircle2,
  DownloadCloud,
  Power,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import {
  usePushTargets,
  useRemovePushTarget,
  useNorthboundServers,
  useSwitchActiveNorthboundServer,
  useFullSync,
} from '@core/hooks/api/useNorthbound'
import type { PushTarget, NorthboundServer, SyncResult } from '@core/services/api/northboundApi'
import { formatTime } from '@/lib/format'
import { StatCard, HudLoading, HudError, HudEmpty } from './_hud'

const SYNC_TYPES = ['device', 'alarm', 'pm', 'config']

// ===========================================================================
// CONFIG · 北向接口管理
// 真实北向推送目标（usePushTargets）+ 主备服务器（useNorthboundServers，可切换）
// + 全量同步（useFullSync）触发 + 结果回显。
// ===========================================================================
export default function NorthboundManagement() {
  const targetsQ = usePushTargets()
  const serversQ = useNorthboundServers()
  const remove = useRemovePushTarget()
  const switchActive = useSwitchActiveNorthboundServer()
  const fullSync = useFullSync()

  const targets: PushTarget[] = targetsQ.data ?? []
  const servers: NorthboundServer[] = serversQ.data ?? []

  const [syncResult, setSyncResult] = useState<SyncResult | null>(null)
  const [syncErr, setSyncErr] = useState('')
  const [syncing, setSyncing] = useState<string>('')

  const triggerSync = (dataType: string) => {
    setSyncErr('')
    setSyncResult(null)
    setSyncing(dataType)
    fullSync.mutate(dataType, {
      onSuccess: (r) => {
        setSyncResult(r)
        setSyncing('')
      },
      onError: (e: Error) => {
        setSyncErr(e.message || '同步失败')
        setSyncing('')
      },
    })
  }

  const enabledTargets = targets.filter((t) => t.enabled).length

  return (
    <PageShell
      code="F08"
      title="NORTHBOUND · 北向接口"
      subtitle="OSS PUSH / SERVERS / SYNC"
      isFetching={targetsQ.isFetching || serversQ.isFetching}
      bare
      toolbar={
        <NeonButton
          icon={<RefreshCcw />}
          onClick={() => {
            void targetsQ.refetch()
            void serversQ.refetch()
          }}
        >
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-3 grid grid-cols-4 gap-3">
        <StatCard label="PUSH TARGETS · 推送目标" value={targets.length} color="#00f0ff" />
        <StatCard label="ENABLED · 启用" value={enabledTargets} color="#00ff88" />
        <StatCard label="SERVERS · 主备服务器" value={servers.length} color="#a855f7" />
        <StatCard
          label="ACTIVE · 激活组"
          value={servers.find((s) => s.isActive)?.role.toUpperCase() ?? '—'}
          color="#ffaa00"
        />
      </div>

      <div className="grid grid-cols-[1fr_1fr] gap-3">
        {/* 主备服务器 */}
        <GlassPanel title="OSS SERVERS · 主备服务器" className="min-h-0">
          {serversQ.isLoading ? (
            <HudLoading />
          ) : serversQ.isError ? (
            <HudError error={serversQ.error} />
          ) : servers.length === 0 ? (
            <HudEmpty icon={Server} text="无主备配置" />
          ) : (
            <div className="space-y-2 p-3">
              {servers.map((s) => (
                <div
                  key={s.id}
                  className="rounded-sm border px-3 py-2.5"
                  style={{ borderColor: s.isActive ? 'rgba(0,255,136,0.4)' : 'rgba(0,240,255,0.1)' }}
                >
                  <div className="mb-1.5 flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <Server className="size-3.5 text-cyan-300/60" />
                      <span className="font-display text-sm font-bold text-cyan-100">
                        {s.role === 'primary' ? '主用 PRIMARY' : '备用 STANDBY'}
                      </span>
                      {s.isActive ? <StatusBadge status="active" label="激活" /> : null}
                    </div>
                    {!s.isActive ? (
                      <NeonButton
                        icon={switchActive.isPending ? <Loader2 className="animate-spin" /> : <Power />}
                        onClick={() => switchActive.mutate(s.role)}
                        disabled={switchActive.isPending}
                      >
                        切为激活
                      </NeonButton>
                    ) : null}
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/70">
                    {s.host}:{s.port}
                  </div>
                  {s.description ? (
                    <div className="mt-0.5 text-[10px] text-cyan-300/50">{s.description}</div>
                  ) : null}
                  <div className="mt-1 font-mono text-[10px] text-cyan-300/40">更新 {formatTime(s.updatedAt)}</div>
                </div>
              ))}
            </div>
          )}
        </GlassPanel>

        {/* 数据同步 */}
        <GlassPanel title="DATA SYNC · 全量同步" className="min-h-0">
          <div className="p-3">
            <div className="mb-3 grid grid-cols-2 gap-2">
              {SYNC_TYPES.map((dt) => (
                <NeonButton
                  key={dt}
                  icon={
                    syncing === dt ? <Loader2 className="animate-spin" /> : <DownloadCloud />
                  }
                  onClick={() => triggerSync(dt)}
                  disabled={fullSync.isPending}
                >
                  同步 {dt.toUpperCase()}
                </NeonButton>
              ))}
            </div>

            {syncErr ? (
              <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
                {syncErr}
              </div>
            ) : null}

            {syncResult ? (
              <div className="rounded-sm border border-emerald-500/30 bg-emerald-500/5 px-3 py-2.5">
                <div className="mb-1 flex items-center gap-2 font-mono text-xs text-emerald-300">
                  <CheckCircle2 className="size-4" />
                  {syncResult.dataType.toUpperCase()} 同步完成
                </div>
                <div className="flex flex-wrap gap-2 font-mono text-[11px]">
                  <span className="chip text-[#00f0ff]">条目 {syncResult.total}</span>
                  <span className="chip text-[#a855f7]">返回 {syncResult.items.length}</span>
                  {syncResult.truncated ? <span className="chip text-[#ffaa00]">已截断</span> : null}
                  <span className="chip text-[#5b9eff]">{formatTime(syncResult.syncedAt)}</span>
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center gap-2 py-8 text-cyan-300/45">
                <Radar className="size-8 text-cyan-400/40" />
                <span className="font-mono text-[11px] uppercase tracking-[0.18em]">选择数据类型触发全量同步</span>
              </div>
            )}
          </div>
        </GlassPanel>
      </div>

      {/* 推送目标 */}
      <GlassPanel title="PUSH TARGETS · 推送目标" meta={`${targets.length}`} className="mt-3 min-h-0">
        {targetsQ.isLoading ? (
          <HudLoading />
        ) : targetsQ.isError ? (
          <HudError error={targetsQ.error} />
        ) : targets.length === 0 ? (
          <HudEmpty icon={Cloud} text="无推送目标" />
        ) : (
          <div className="max-h-[34vh] overflow-auto">
            <div className="grid grid-cols-[2fr_1fr_1.2fr_90px_90px_70px] gap-3 border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45">
              <span>URL</span>
              <span>AUTH</span>
              <span>DATA TYPES</span>
              <span>FORMAT</span>
              <span>STATUS</span>
              <span className="text-right">DEL</span>
            </div>
            {targets.map((t) => (
              <div
                key={t.id}
                className="grid grid-cols-[2fr_1fr_1.2fr_90px_90px_70px] items-center gap-3 border-b border-cyan-500/8 px-3 py-2 hover:bg-cyan-500/5"
              >
                <code className="truncate font-mono text-[11px] text-cyan-100/85">{t.url}</code>
                <span className="font-mono text-[11px] text-cyan-300/65">{t.authType}</span>
                <span className="truncate font-mono text-[10px] text-cyan-300/55">
                  {t.dataTypes.join(', ') || '—'}
                </span>
                <span className="chip text-[#5b9eff]">{t.format}</span>
                <span>
                  <StatusBadge status={t.enabled ? 'active' : 'off'} label={t.enabled ? '启用' : '停用'} />
                </span>
                <div className="flex justify-end">
                  <NeonButton
                    icon={remove.isPending ? <Loader2 className="animate-spin" /> : <Trash2 />}
                    onClick={() => remove.mutate(t.id)}
                    disabled={remove.isPending}
                  >
                    删
                  </NeonButton>
                </div>
              </div>
            ))}
          </div>
        )}
      </GlassPanel>
    </PageShell>
  )
}
