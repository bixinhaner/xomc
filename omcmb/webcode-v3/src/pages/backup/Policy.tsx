import { RefreshCcw, Loader2, AlertTriangle } from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { NeonButton } from '@/components/ui/NeonButton'
import { RadialGauge } from '@/components/viz/RadialGauge'
import { useBackupPolicy } from '@core/hooks/api/useBackup'
import { formatSystemTime } from '@core/utils/systemTime'

function getErrMsg(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'object' && e && 'message' in e) {
    return String((e as { message: unknown }).message)
  }
  return '未知错误'
}

function formatTime(iso?: string): string {
  // #459 子单 D：保留后端系统时区钟面，不按浏览器本地二次转换。
  return formatSystemTime(iso, { placeholder: '—' })
}

// ---------------------------------------------------------------------------
// 备份保留策略 · /backup/policy （单例只读概览）
// ---------------------------------------------------------------------------

export default function BackupPolicyPage() {
  const query = useBackupPolicy()
  const p = query.data

  return (
    <PageShell
      code="F06"
      title="BACKUP POLICY · 保留策略"
      subtitle="RETENTION · COMPRESSION · ENCRYPTION · ALERT"
      isFetching={query.isFetching || undefined}
      bare
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => query.refetch()}>
          REFRESH
        </NeonButton>
      }
    >
      {query.isLoading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      ) : query.isError ? (
        <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
          FAILURE · {getErrMsg(query.error)}
        </div>
      ) : !p ? (
        <div className="flex flex-col items-center justify-center gap-3 py-16">
          <AlertTriangle className="size-10 text-amber-400/60" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-amber-300/70">
            NO POLICY · 无保留策略
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
            <div className="glass relative flex items-center gap-4 overflow-hidden rounded-sm px-4 py-4">
              <div className="scanline" />
              <RadialGauge
                value={p.alertThresholdPercent}
                label="告警阈值"
                size={104}
                color={p.alertThresholdPercent >= 90 ? '#ff2d6f' : '#ffaa00'}
              />
              <div>
                <div className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
                  存储上限 · MAX
                </div>
                <div className="font-display text-2xl font-bold text-cyan-100 text-glow">
                  {p.maxStorageGB} GB
                </div>
                <div className="font-mono text-[10px] text-cyan-300/55">
                  {(p.storageBackend || '—').toUpperCase()} · {p.localPath || '—'}
                </div>
              </div>
            </div>

            <PolicyCard
              title="保留 · RETENTION"
              rows={[
                ['保留天数', `${p.retentionDays} 天`],
                ['保留份数', `${p.minBackupCount} ~ ${p.maxBackupCount}`],
                ['至少保留最近', `${p.keepLastN} 份`],
                ['自动清理', p.autoCleanup ? `开 · ${p.cleanupTime}` : '关'],
              ]}
            />

            <PolicyCard
              title="压缩 / 加密 · DATA"
              rows={[
                [
                  '压缩',
                  p.enableCompression
                    ? `${(p.compressionFormat || '').toUpperCase()} L${p.compressionLevel}`
                    : '关',
                ],
                ['加密', p.enableEncryption ? p.encryptionAlgorithm : '关'],
                ['失败告警', p.alertOnFailure ? '开' : '关'],
                ['告警级别', (p.alertSeverity || '—').toUpperCase()],
              ]}
            />
          </div>

          <PolicyCard
            title="告警通知 · ALERT"
            rows={[
              ['告警邮箱', p.alertEmail || '未配置'],
              ['阈值百分比', `${p.alertThresholdPercent}%`],
              ['最近更新', formatTime(p.updatedAt)],
            ]}
            wide
          />

          <div className="font-mono text-[10px] text-cyan-300/40">
            策略字段为只读概览。编辑 / 下发为 v1 表单深度，本皮肤本轮未实现。
          </div>
        </div>
      )}
    </PageShell>
  )
}

function PolicyCard({
  title,
  rows,
  wide,
}: {
  title: string
  rows: [string, string][]
  wide?: boolean
}) {
  return (
    <div className="glass relative overflow-hidden rounded-sm">
      <div className="scanline" />
      <div className="relative border-b border-cyan-500/15 px-4 py-2 font-mono text-[11px] uppercase tracking-[0.2em] text-cyan-300/90">
        {title}
      </div>
      <div
        className={`relative grid gap-x-6 gap-y-2 px-4 py-3 ${
          wide ? 'grid-cols-1 sm:grid-cols-3' : 'grid-cols-1'
        }`}
      >
        {rows.map(([k, v]) => (
          <div key={k} className="flex items-center justify-between gap-3">
            <span className="font-mono text-[11px] text-cyan-300/55">{k}</span>
            <span className="font-display text-sm text-cyan-100">{v}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
