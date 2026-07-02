import { useCallback, useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import {
  CheckCircle2,
  CloudDownload,
  Loader2,
  RadioTower,
  RefreshCcw,
  Send,
  XCircle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatHHMMSS } from '@/lib/format'
import { deviceTaskApi, isAbortError } from '@core/services/api/deviceTaskApi'
import { useAlarmCount, useTriggerAlarmSync } from '@core/hooks/api/useAlarms'

type SyncStatus = 'running' | 'success' | 'failed'

interface SyncRecord {
  id: string
  deviceSn: string
  startTs: string
  endTs: string | null
  status: SyncStatus
  message?: string
}

export default function AlarmSync() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const triggerSync = useTriggerAlarmSync()
  const { data: cnt, isFetching: cntFetching, refetch: refetchCount } = useAlarmCount()

  const [sn, setSn] = useState('')
  const [records, setRecords] = useState<SyncRecord[]>([])
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const syncAbortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    return () => {
      syncAbortRef.current?.abort()
    }
  }, [])

  const submit = useCallback(() => {
    if (submitting) return
    const deviceSn = sn.trim()
    if (!deviceSn) {
      setFormError('请输入设备 SN')
      return
    }
    setFormError(null)
    const id = `${Date.now()}-${deviceSn}`
    setRecords((prev) => [
      { id, deviceSn, startTs: formatHHMMSS(), endTs: null, status: 'running' },
      ...prev,
    ])
    setSn('')
    setSubmitting(true)
    syncAbortRef.current?.abort()
    const abortController = new AbortController()
    syncAbortRef.current = abortController
    void (async () => {
      try {
        const triggerResult = await triggerSync.mutateAsync(deviceSn)
        if (!triggerResult.taskId) {
          throw new Error('告警同步任务不可用，请稍后重试。')
        }
        const task = await deviceTaskApi.waitForTerminal(triggerResult.taskId, {
          signal: abortController.signal,
        })
        if (task.status !== 'completed') {
          throw new Error(task.errorMessage || task.status)
        }
        await queryClient.invalidateQueries({ queryKey: ['alarms'] })
        setRecords((prev) =>
          prev.map((r) =>
            r.id === id
              ? { ...r, status: 'success', endTs: formatHHMMSS(), message: '同步完成并已入库' }
              : r
          )
        )
      } catch (e) {
        if (isAbortError(e)) {
          return
        }
        setRecords((prev) =>
          prev.map((r) =>
            r.id === id
              ? {
                  ...r,
                  status: 'failed',
                  endTs: formatHHMMSS(),
                  message: e instanceof Error ? e.message : '同步失败',
                }
              : r
          )
        )
      } finally {
        if (syncAbortRef.current === abortController) {
          syncAbortRef.current = null
        }
        if (!abortController.signal.aborted) {
          setSubmitting(false)
        }
      }
    })()
  }, [queryClient, sn, submitting, triggerSync])

  const runningCount = records.filter((r) => r.status === 'running').length
  const successCount = records.filter((r) => r.status === 'success').length
  const failedCount = records.filter((r) => r.status === 'failed').length

  return (
    <PageShell
      code="F04"
      title="ALARM SYNC · 告警同步"
      subtitle="ON-DEMAND ALARM RESYNC FROM NETWORK ELEMENT"
      isFetching={cntFetching || submitting}
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetchCount()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="grid grid-cols-12 gap-3">
        {/* 触发卡 */}
        <GlassPanel
          title="TRIGGER SYNC · 发起同步"
          meta="REAL-TIME"
          className="col-span-12 lg:col-span-5"
        >
          <div className="space-y-3 p-4">
            <p className="font-mono text-[11px] leading-relaxed text-cyan-300/60">
              输入设备 SN，向 ACS 下发该网元的告警全量同步请求。后端将拉取该设备当前告警快照并入库，可在
              <button
                type="button"
                onClick={() => navigate('/alarm/current')}
                className="mx-1 text-cyan-200 underline decoration-cyan-500/40 underline-offset-2 hover:text-cyan-100"
              >
                警报阵列
              </button>
              查看结果。
            </p>
            <label className="flex flex-col gap-1">
              <span className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
                DEVICE SN · 设备序列号
              </span>
              <input
                className="neon-input"
                placeholder="例如 ENB-08F311"
                value={sn}
                onChange={(e) => setSn(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') submit()
                }}
              />
            </label>
            {formError && (
              <div className="font-mono text-[11px] text-rose-300">{formError}</div>
            )}
            <NeonButton
              icon={submitting ? <Loader2 className="animate-spin" /> : <Send />}
              onClick={submit}
              disabled={submitting}
            >
              {submitting ? 'SYNCING…' : 'DISPATCH SYNC · 下发同步'}
            </NeonButton>
          </div>
        </GlassPanel>

        {/* 实时计数 + 会话统计 */}
        <div className="col-span-12 space-y-3 lg:col-span-7">
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <Stat label="ACTIVE · 当前告警" color="#00f0ff" value={cnt?.total_active ?? 0} />
            <Stat label="RUNNING · 进行中" color="#ffaa00" value={runningCount} />
            <Stat label="SUCCESS · 成功" color="#00ff88" value={successCount} />
            <Stat label="FAILED · 失败" color="#ff2d6f" value={failedCount} />
          </div>

          <GlassPanel
            title="SYNC HISTORY · 本次会话同步记录"
            meta={`${records.length} 条`}
          >
            {records.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-2 px-4 py-12">
                <CloudDownload className="size-9 text-cyan-300/35" />
                <span className="font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/45">
                  NO SYNC YET · 暂无同步任务
                </span>
                <span className="font-mono text-[10px] text-cyan-300/30">
                  从左侧发起一次同步
                </span>
              </div>
            ) : (
              <div className="divide-y divide-cyan-500/8">
                {records.map((r) => (
                  <div
                    key={r.id}
                    className="grid grid-cols-[28px_1.6fr_1fr_1fr_120px] items-center gap-3 px-3.5 py-2.5"
                  >
                    <SyncIcon status={r.status} />
                    <div className="min-w-0">
                      <div className="truncate font-mono text-xs text-cyan-100">{r.deviceSn}</div>
                      {r.message && (
                        <div className="truncate font-mono text-[10px] text-cyan-300/50">
                          {r.message}
                        </div>
                      )}
                    </div>
                    <span className="font-mono text-[11px] text-cyan-300/70">{r.startTs}</span>
                    <span className="font-mono text-[11px] text-cyan-300/70">
                      {r.endTs ?? '—'}
                    </span>
                    <span className="flex justify-end">
                      <StatusBadge
                        status={
                          r.status === 'running'
                            ? 'warn'
                            : r.status === 'success'
                              ? 'ok'
                              : 'error'
                        }
                        label={
                          r.status === 'running'
                            ? '进行中'
                            : r.status === 'success'
                              ? '成功'
                              : '失败'
                        }
                      />
                    </span>
                  </div>
                ))}
              </div>
            )}
          </GlassPanel>
        </div>
      </div>
    </PageShell>
  )
}

function SyncIcon({ status }: { status: SyncStatus }) {
  if (status === 'running')
    return <Loader2 className="size-4 animate-spin text-amber-300" />
  if (status === 'success') return <CheckCircle2 className="size-4 text-emerald-400" />
  if (status === 'failed') return <XCircle className="size-4 text-rose-400" />
  return <RadioTower className="size-4 text-cyan-300/60" />
}

function Stat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.14em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}
