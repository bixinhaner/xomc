import { useMemo, useState } from 'react'
import {
  RefreshCcw,
  Loader2,
  Inbox,
  Server,
  Plug,
  ShieldCheck,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { useFTPConfigs, useTestFTPConnection } from '@core/hooks/api/useBackup'
import type { FTPConfig } from '@core/mock/data/backup'

import { Modal } from './Modal'

const PAGE_SIZE = 20

const PROTOCOL_COLOR: Record<FTPConfig['protocol'], string> = {
  FTP: '#5b9eff',
  SFTP: '#00ff88',
  FTPS: '#a855f7',
}

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return '未知错误'
}

// ---------------------------------------------------------------------------
// FTP 通道 · /backup/ftp
// ---------------------------------------------------------------------------

export default function FTPConfigPage() {
  const [page, setPage] = useState(1)
  const query = useFTPConfigs({ page, pageSize: PAGE_SIZE })
  const testConn = useTestFTPConnection()

  const items = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const stats = useMemo(() => {
    const enabled = items.filter((f) => f.enabled).length
    const secure = items.filter((f) => f.protocol !== 'FTP').length
    return { enabled, secure }
  }, [items])

  const [testResult, setTestResult] = useState<{
    config: FTPConfig
    success?: boolean
    message: string
  } | null>(null)

  const onTest = async (f: FTPConfig) => {
    setTestResult({ config: f, message: '正在测试连接…' })
    try {
      const r = await testConn.mutateAsync(f.id)
      setTestResult({ config: f, success: r.success, message: r.message })
    } catch (e) {
      setTestResult({ config: f, success: false, message: getErrMsg(e) })
    }
  }

  return (
    <PageShell
      code="F06"
      title="FTP CHANNELS · 远端通道"
      subtitle="FTP / SFTP / FTPS BACKUP TRANSPORT"
      isFetching={query.isFetching || undefined}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <MiniStat label="通道总数 · TOTAL" value={total} color="#00f0ff" icon={<Server className="size-4" />} />
        <MiniStat label="启用中 · ENABLED" value={stats.enabled} color="#00ff88" icon={<Plug className="size-4" />} />
        <MiniStat label="加密通道 · SECURE" value={stats.secure} color="#a855f7" icon={<ShieldCheck className="size-4" />} />
      </div>

      {query.isLoading ? (
        <CenterSync />
      ) : query.isError ? (
        <FailureBox error={query.error} />
      ) : items.length === 0 ? (
        <EmptyBox hint="NO FTP CHANNELS · 无 FTP 通道" />
      ) : (
        <>
          <div className="space-y-1.5">
            {items.map((f: FTPConfig) => {
              const color = f.enabled ? PROTOCOL_COLOR[f.protocol] : '#525a78'
              return (
                <div
                  key={f.id}
                  className="fleet-row grid grid-cols-[10px_1.4fr_1.6fr_1fr_120px] items-center gap-3 rounded-sm px-3 py-2.5"
                  style={{ ['--row-color' as never]: color }}
                >
                  <span
                    className="size-2 rounded-full"
                    style={{ background: color, boxShadow: `0 0 8px ${color}` }}
                  />
                  <div className="min-w-0">
                    <div className="truncate font-display text-sm font-bold text-cyan-100">
                      {f.configName}
                    </div>
                    <div className="font-mono text-[10px]" style={{ color }}>
                      {f.protocol} · {f.passive ? 'PASSIVE' : 'ACTIVE'}
                    </div>
                  </div>
                  <div className="font-mono text-[11px] text-cyan-200/85">
                    <div>
                      {f.host}:{f.port}
                    </div>
                    <div className="truncate text-[10px] text-cyan-300/55">{f.remotePath}</div>
                  </div>
                  <div className="font-mono text-[11px] text-cyan-300/75">{f.username}</div>
                  <div className="flex items-center justify-end gap-2">
                    <span className="chip" style={{ color: f.enabled ? '#00ff88' : '#525a78' }}>
                      {f.enabled ? '启用' : '停用'}
                    </span>
                    <NeonButton
                      icon={<Plug />}
                      disabled={testConn.isPending}
                      onClick={() => {
                        void onTest(f)
                      }}
                    >
                      测试
                    </NeonButton>
                  </div>
                </div>
              )
            })}
          </div>
          <Pager page={page} totalPages={totalPages} total={total} onPage={setPage} />
        </>
      )}

      <Modal
        open={Boolean(testResult)}
        title="连接测试结果"
        subtitle="FTP CONNECTIVITY"
        onClose={() => setTestResult(null)}
        width={440}
        footer={<NeonButton onClick={() => setTestResult(null)}>知道了</NeonButton>}
      >
        {testResult ? (
          <div className="space-y-2">
            <div className="font-mono text-xs text-cyan-300/70">
              {testResult.config.configName} · {testResult.config.host}:
              {testResult.config.port}
            </div>
            {testResult.success === undefined ? (
              <div className="flex items-center gap-2 text-sm text-cyan-200">
                <Loader2 className="size-4 animate-spin" />
                {testResult.message}
              </div>
            ) : (
              <div
                className="rounded-sm border px-3 py-2 font-mono text-xs"
                style={{
                  borderColor: testResult.success ? 'rgba(0,255,136,0.4)' : 'rgba(255,45,111,0.4)',
                  background: testResult.success ? 'rgba(0,255,136,0.05)' : 'rgba(255,45,111,0.05)',
                  color: testResult.success ? '#7dffc0' : '#ff9bb6',
                }}
              >
                {testResult.success ? 'OK · ' : 'FAILED · '}
                {testResult.message}
              </div>
            )}
          </div>
        ) : null}
      </Modal>
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 局部组件
// ---------------------------------------------------------------------------

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
