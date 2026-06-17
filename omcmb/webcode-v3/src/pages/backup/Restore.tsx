import { useMemo, useState } from 'react'
import {
  RefreshCcw,
  Loader2,
  Inbox,
  PlusCircle,
  HardDriveDownload,
  Activity,
  CheckCircle2,
  XCircle,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import {
  useBackupRestoreTasks,
  useCreateBackupRestore,
  useCreateBackupRestoreByTaskID,
} from '@core/hooks/api/useBackup'
import type { RestoreTask, RestoreStatus } from '@core/mock/data/backup'
import { formatSystemTime } from '@core/utils/systemTime'

import { Modal } from './Modal'

const PAGE_SIZE = 20

const RESTORE_STATUS_COLOR: Record<RestoreStatus, string> = {
  pending: '#5b9eff',
  running: '#00f0ff',
  completed: '#00ff88',
  failed: '#ff2d6f',
  cancelled: '#ffaa00',
}
const RESTORE_STATUS_LABEL: Record<RestoreStatus, string> = {
  pending: '等待',
  running: '运行中',
  completed: '已完成',
  failed: '失败',
  cancelled: '已取消',
}

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return '未知错误'
}

function parseSnList(raw: string): string[] {
  return raw
    .split(/[\r\n,]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

function formatTime(iso?: string): string {
  // #459 子单 D：保留后端系统时区钟面，不按浏览器本地二次转换。
  return formatSystemTime(iso, { placeholder: '—' })
}

// ---------------------------------------------------------------------------
// 数据恢复 · /backup/restore
// ---------------------------------------------------------------------------

export default function RestoreDataPage() {
  const [page, setPage] = useState(1)
  const query = useBackupRestoreTasks({ page, pageSize: PAGE_SIZE })
  const createByPath = useCreateBackupRestore()
  const createByTask = useCreateBackupRestoreByTaskID()

  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const stats = useMemo(() => {
    let running = 0
    let completed = 0
    let failed = 0
    for (const r of items) {
      if (r.status === 'running' || r.status === 'pending') running += 1
      if (r.status === 'completed') completed += 1
      if (r.status === 'failed') failed += 1
    }
    return { running, completed, failed }
  }, [items])

  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'path' | 'task'>('path')
  const [bucket, setBucket] = useState('config_backup')
  const [objectPath, setObjectPath] = useState('')
  const [backupTaskId, setBackupTaskId] = useState('')
  const [snRaw, setSnRaw] = useState('')
  const [formErr, setFormErr] = useState<string | null>(null)
  const [resultMsg, setResultMsg] = useState<string | null>(null)

  const resetForm = () => {
    setMode('path')
    setBucket('config_backup')
    setObjectPath('')
    setBackupTaskId('')
    setSnRaw('')
    setFormErr(null)
  }

  const submitting = createByPath.isPending || createByTask.isPending

  const onSubmit = async () => {
    setFormErr(null)
    const sns = parseSnList(snRaw)
    if (sns.length === 0) {
      setFormErr('请至少填写一个目标设备 SN')
      return
    }
    try {
      if (mode === 'task') {
        if (!backupTaskId.trim()) {
          setFormErr('请填写备份任务 ID')
          return
        }
        await createByTask.mutateAsync({
          backupTaskId: backupTaskId.trim(),
          targetDeviceSns: sns,
        })
      } else {
        if (!objectPath.trim()) {
          setFormErr('请填写备份对象路径')
          return
        }
        if (objectPath.includes('..') || objectPath.startsWith('/')) {
          setFormErr('对象路径非法（不可含 .. 或以 / 开头）')
          return
        }
        await createByPath.mutateAsync({
          bucket: bucket.trim(),
          objectPath: objectPath.trim(),
          targetDeviceSns: sns,
        })
      }
      setOpen(false)
      resetForm()
      setResultMsg(`已提交恢复任务 · 目标 ${sns.length} 台设备`)
    } catch (e) {
      setFormErr(`提交失败：${getErrMsg(e)}`)
    }
  }

  return (
    <PageShell
      code="F06"
      title="RESTORE · 数据恢复"
      subtitle="CONFIG SNAPSHOT FAN-OUT TO DEVICES"
      isFetching={query.isFetching || undefined}
      bare
      toolbar={
        <div className="flex flex-wrap items-center gap-2">
          <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
            REFRESH
          </NeonButton>
          <NeonButton
            icon={<PlusCircle />}
            onClick={() => {
              resetForm()
              setOpen(true)
            }}
          >
            新建恢复
          </NeonButton>
        </div>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <MiniStat label="进行中 · RUNNING" value={stats.running} color="#00f0ff" icon={<Activity className="size-4" />} />
        <MiniStat label="已完成 · DONE" value={stats.completed} color="#00ff88" icon={<CheckCircle2 className="size-4" />} />
        <MiniStat label="失败 · FAILED" value={stats.failed} color={stats.failed > 0 ? '#ff2d6f' : '#525a78'} icon={<XCircle className="size-4" />} />
      </div>

      <div className="mb-3 font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/55">
        RESTORE FAN-OUT · 5s 自动刷新
      </div>

      {query.isLoading ? (
        <CenterSync />
      ) : query.isError ? (
        <FailureBox error={query.error} />
      ) : items.length === 0 ? (
        <EmptyBox hint="NO RESTORE TASKS · 无恢复任务" />
      ) : (
        <>
          <div className="space-y-1.5">
            {items.map((r: RestoreTask) => {
              const color = RESTORE_STATUS_COLOR[r.status]
              return (
                <div
                  key={r.id}
                  className="fleet-row grid grid-cols-[10px_2.2fr_0.8fr_1.4fr_1.2fr] items-center gap-3 rounded-sm px-3 py-2.5"
                  style={{ ['--row-color' as never]: color }}
                >
                  <span
                    className="size-2 rounded-full"
                    style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                  />
                  <div className="min-w-0">
                    <div className="truncate font-mono text-xs text-cyan-100">
                      {r.sourceObjectPath || '—'}
                    </div>
                    <div className="truncate font-mono text-[10px] text-cyan-300/55">
                      {r.sourceBucket}
                      {r.errorMessage ? ` · ⚠ ${r.errorMessage}` : ''}
                    </div>
                  </div>
                  <div className="text-xs text-cyan-100/85">{r.targetDeviceSns.length} 台</div>
                  <div>
                    <div className="mb-1 flex items-center justify-between">
                      <span className="chip" style={{ color }}>
                        {RESTORE_STATUS_LABEL[r.status]}
                      </span>
                      <span className="font-mono text-[10px] text-cyan-300/65">{r.progress}%</span>
                    </div>
                    <div className="h-1 overflow-hidden rounded-full bg-cyan-500/10">
                      <div
                        className="h-full rounded-full transition-all"
                        style={{
                          width: `${Math.min(100, Math.max(0, r.progress))}%`,
                          background: color,
                          boxShadow: `0 0 6px ${color}`,
                        }}
                      />
                    </div>
                  </div>
                  <div className="font-mono text-[10px] text-cyan-300/65">
                    <div>起 {formatTime(r.startedAt)}</div>
                    <div className="text-cyan-300/45">终 {formatTime(r.completedAt)}</div>
                  </div>
                </div>
              )
            })}
          </div>
          <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
        </>
      )}

      {/* 新建恢复 */}
      <Modal
        open={open}
        title="新建恢复任务"
        subtitle="RESTORE FAN-OUT"
        onClose={() => setOpen(false)}
        footer={
          <>
            <NeonButton onClick={() => setOpen(false)}>取消</NeonButton>
            <NeonButton
              icon={<HardDriveDownload />}
              disabled={submitting}
              onClick={() => {
                void onSubmit()
              }}
            >
              {submitting ? '提交中…' : '提交'}
            </NeonButton>
          </>
        }
      >
        <div className="space-y-4">
          <div className="flex gap-2">
            {(['path', 'task'] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => {
                  setMode(m)
                  setFormErr(null)
                }}
                className={`chip transition-all ${
                  mode === m ? 'shadow-[0_0_10px_currentColor] text-cyan-200' : 'text-cyan-300/55'
                }`}
              >
                {m === 'path' ? '按对象路径' : '按备份任务'}
              </button>
            ))}
          </div>

          {mode === 'task' ? (
            <Field label="备份任务 ID" hint="从备份任务列表复制任务 ID，回放其快照">
              <input
                className="neon-input w-full"
                value={backupTaskId}
                placeholder="bkp-xxxx"
                onChange={(e) => setBackupTaskId(e.target.value)}
              />
            </Field>
          ) : (
            <>
              <Field label="存储桶 Bucket" hint="当前仅支持 config_backup">
                <input
                  className="neon-input w-full"
                  value={bucket}
                  onChange={(e) => setBucket(e.target.value)}
                />
              </Field>
              <Field label="对象路径 Object Path" hint="相对路径，不可含 .. 或以 / 开头">
                <input
                  className="neon-input w-full"
                  value={objectPath}
                  placeholder="backup/2026/04/29/cfg-cmcc-lte-001.xml.gz"
                  onChange={(e) => setObjectPath(e.target.value)}
                />
              </Field>
            </>
          )}

          <Field label="目标设备 SN" hint="每行一个，或逗号分隔">
            <textarea
              className="neon-input min-h-[120px] w-full resize-y"
              value={snRaw}
              placeholder={'SN001\nSN002\nSN003'}
              onChange={(e) => setSnRaw(e.target.value)}
            />
          </Field>

          {formErr ? (
            <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
              {formErr}
            </div>
          ) : null}
        </div>
      </Modal>

      <Modal
        open={Boolean(resultMsg)}
        title="操作结果"
        subtitle="RESULT"
        onClose={() => setResultMsg(null)}
        width={420}
        footer={<NeonButton onClick={() => setResultMsg(null)}>知道了</NeonButton>}
      >
        <p className="text-sm text-cyan-100/85">{resultMsg}</p>
      </Modal>
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 局部组件
// ---------------------------------------------------------------------------

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <label className="block">
      <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/65">
        {label}
      </div>
      {children}
      {hint ? <div className="mt-1 font-mono text-[10px] text-cyan-300/40">{hint}</div> : null}
    </label>
  )
}

function MiniStat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: number
  color: string
  icon: React.ReactNode
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative flex items-center gap-3 p-4">
        <span style={{ color }}>{icon}</span>
        <div>
          <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
            {label}
          </div>
          <div className="font-display text-2xl font-bold text-glow" style={{ color }}>
            {value.toLocaleString()}
          </div>
        </div>
      </div>
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}

function CenterSync() {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
      <Loader2 className="size-4 animate-spin" />
      <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
    </div>
  )
}

function FailureBox({ error }: { error: unknown }) {
  return (
    <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
      FAILURE · {getErrMsg(error)}
    </div>
  )
}

function EmptyBox({ hint }: { hint: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16">
      <Inbox className="size-10 text-cyan-400/50" />
      <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/55">
        {hint}
      </div>
    </div>
  )
}
