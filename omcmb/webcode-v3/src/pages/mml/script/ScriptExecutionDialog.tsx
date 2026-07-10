import { useEffect, useState } from 'react'
import { Loader2, Play, X } from 'lucide-react'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { useCreateMMLScriptExecution } from '@core/hooks/api/useMML'
import type { MMLExecuteType, MMLScript, MMLScriptExecutionInput, MMLScriptImportValidation } from '@core/types/mml'

export interface ScriptExecutionDialogProps { open: boolean; script: MMLScript | null; onClose: () => void; onSuccess?: () => void }

function validationFromError(error: unknown): MMLScriptImportValidation | undefined {
  if (!error || typeof error !== 'object') return undefined
  const value = (error as { validation?: unknown }).validation
  return value && typeof value === 'object' ? value as MMLScriptImportValidation : undefined
}
function hasWarnings(validation: MMLScriptImportValidation) { return validation.summary.warningCount > 0 || validation.issues.some((issue) => issue.severity === 'warning') }
function hasErrors(validation: MMLScriptImportValidation) { return validation.summary.errorCount > 0 || validation.issues.some((issue) => issue.severity === 'error') }

/** Execute a stored server snapshot. Browser submits strategy fields only, never parsed commands. */
export default function ScriptExecutionDialog({ open, script, onClose, onSuccess }: ScriptExecutionDialogProps) {
  const mutation = useCreateMMLScriptExecution()
  const [taskName, setTaskName] = useState('')
  const [executeType, setExecuteType] = useState<MMLExecuteType>('immediate')
  const [scheduledAt, setScheduledAt] = useState('')
  const [periodStart, setPeriodStart] = useState('')
  const [periodEnd, setPeriodEnd] = useState('')
  const [periodTime, setPeriodTime] = useState('')
  const [offlineRetry, setOfflineRetry] = useState(false)
  const [offlineRetryWait, setOfflineRetryWait] = useState(60)
  const [failedRetry, setFailedRetry] = useState(false)
  const [failedRetryCount, setFailedRetryCount] = useState(3)
  const [failedRetryInterval, setFailedRetryInterval] = useState(5)
  const [validation, setValidation] = useState<MMLScriptImportValidation | null>(null)
  const [warningInput, setWarningInput] = useState<MMLScriptExecutionInput | null>(null)
  const [notice, setNotice] = useState('')

  useEffect(() => {
    if (!open) return
    setTaskName(script ? `执行脚本: ${script.scriptName}` : ''); setExecuteType('immediate'); setScheduledAt(''); setPeriodStart(''); setPeriodEnd(''); setPeriodTime('')
    setOfflineRetry(false); setOfflineRetryWait(60); setFailedRetry(false); setFailedRetryCount(3); setFailedRetryInterval(5)
    setValidation(null); setWarningInput(null); setNotice('')
    const onKeyDown = (event: KeyboardEvent) => { if (event.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKeyDown); return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, script, onClose])

  if (!open || !script) return null
  const execute = async (input: MMLScriptExecutionInput) => {
    setNotice('')
    try {
      const result = await mutation.mutateAsync({ id: script.id, input })
      if (hasWarnings(result.validation) && !input.confirmWarnings) { setValidation(result.validation); setWarningInput(input); return }
      onSuccess?.(); onClose()
    } catch (error) {
      const next = validationFromError(error)
      if (next) {
        setValidation(next)
        if (!hasErrors(next) && hasWarnings(next)) { setWarningInput(input); return }
      }
      const status = typeof error === 'object' && error && 'status' in error ? ` (${String((error as { status?: unknown }).status)})` : ''
      setNotice(`${error instanceof Error ? error.message : '执行校验失败'}${status}`)
    }
  }

  const submit = () => {
    setNotice('')
    if (!taskName.trim()) { setNotice('请输入任务名称'); return }
    if (executeType === 'scheduled' && !scheduledAt) { setNotice('请选择执行时间'); return }
    if (executeType === 'periodic' && (!periodStart || !periodEnd || !periodTime)) { setNotice('请选择周期日期和时间'); return }
    const normalizedScheduledAt = scheduledAt ? (scheduledAt.length === 16 ? `${scheduledAt}:00` : scheduledAt) : undefined
    const normalizedPeriodTime = periodTime ? (periodTime.length === 5 ? `${periodTime}:00` : periodTime) : undefined
    const normalizedPeriodStart = periodStart ? (periodStart.length === 10 ? `${periodStart}T00:00:00` : periodStart) : undefined
    const normalizedPeriodEnd = periodEnd ? (periodEnd.length === 10 ? `${periodEnd}T23:59:59` : periodEnd) : undefined
    void execute({ taskName: taskName.trim(), executeType, scheduledAt: normalizedScheduledAt, periodStart: normalizedPeriodStart, periodEnd: normalizedPeriodEnd, periodTime: normalizedPeriodTime, offlineRetry, offlineRetryWait, failedRetry, failedRetryCount, failedRetryInterval })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label="执行脚本" onClick={onClose}>
      <div className="glass-strong relative flex max-h-[90vh] w-[min(680px,calc(100vw-32px))] flex-col overflow-hidden border border-cyan-500/30" onClick={(event) => event.stopPropagation()}>
        <header className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3"><div><div className="font-display text-base font-bold text-cyan-100">EXEC SCRIPT</div><div className="font-mono text-[10px] text-cyan-300/50">{script.scriptName}</div></div><button type="button" onClick={onClose} aria-label="关闭" className="rounded-sm border border-cyan-500/25 p-1.5 text-cyan-300/70"><X className="size-4" /></button></header>
        <div className="min-h-0 flex-1 space-y-3 overflow-auto px-4 py-3">
          <label className="block"><span className="hud-label">任务名称</span><input aria-label="任务名称" className="neon-input w-full" value={taskName} onChange={(e) => setTaskName(e.target.value)} /></label>
          <label className="block"><span className="hud-label">执行方式</span><select aria-label="执行方式" className="neon-input w-full" value={executeType} onChange={(e) => setExecuteType(e.target.value as MMLExecuteType)}><option value="immediate">立即</option><option value="suspended">挂起</option><option value="scheduled">定时</option><option value="periodic">周期</option></select></label>
          {executeType === 'scheduled' ? <label className="block"><span className="hud-label">执行时间</span><input aria-label="执行时间" type="datetime-local" className="neon-input w-full" value={scheduledAt} onChange={(e) => setScheduledAt(e.target.value)} /></label> : null}
          {executeType === 'periodic' ? <div className="grid gap-3 sm:grid-cols-3"><label><span className="hud-label">周期开始</span><input aria-label="周期开始" type="date" className="neon-input w-full" value={periodStart} onChange={(e) => setPeriodStart(e.target.value)} /></label><label><span className="hud-label">周期结束</span><input aria-label="周期结束" type="date" className="neon-input w-full" value={periodEnd} onChange={(e) => setPeriodEnd(e.target.value)} /></label><label><span className="hud-label">周期时间</span><input aria-label="周期时间" type="time" className="neon-input w-full" value={periodTime} onChange={(e) => setPeriodTime(e.target.value)} /></label></div> : null}
          <GlassPanel title="EXECUTION STRATEGY" meta="SERVER SNAPSHOT"><div className="grid gap-3 sm:grid-cols-2"><label className="flex items-center gap-2 font-mono text-xs text-cyan-100/80"><input aria-label="离线等待重试" type="checkbox" checked={offlineRetry} onChange={(e) => setOfflineRetry(e.target.checked)} />离线等待重试</label><label><span className="hud-label">离线等待（秒）</span><input aria-label="离线等待（秒）" type="number" min={1} className="neon-input w-full" value={offlineRetryWait} onChange={(e) => setOfflineRetryWait(Number(e.target.value))} /></label><label className="flex items-center gap-2 font-mono text-xs text-cyan-100/80"><input aria-label="失败重试" type="checkbox" checked={failedRetry} onChange={(e) => setFailedRetry(e.target.checked)} />失败重试</label><label><span className="hud-label">失败重试次数</span><input aria-label="失败重试次数" type="number" min={0} className="neon-input w-full" value={failedRetryCount} onChange={(e) => setFailedRetryCount(Number(e.target.value))} /></label><label><span className="hud-label">失败重试间隔（秒）</span><input aria-label="失败重试间隔（秒）" type="number" min={1} className="neon-input w-full" value={failedRetryInterval} onChange={(e) => setFailedRetryInterval(Number(e.target.value))} /></label></div></GlassPanel>
          {mutation.isPending ? <div role="status" className="font-mono text-xs text-cyan-200">正在提交执行…</div> : null}
          {notice ? <div role="alert" className="border border-rose-400/40 bg-rose-500/10 px-3 py-2 font-mono text-xs text-rose-200">{notice}</div> : null}
          {validation ? <GlassPanel title="SERVER VALIDATION" meta={`错误 ${validation.summary.errorCount} · 警告 ${validation.summary.warningCount}`}><div className="space-y-1 font-mono text-xs">{validation.issues.map((issue, index) => <div key={`${issue.code}-${index}`} className={issue.severity === 'error' ? 'text-rose-300' : 'text-amber-200'}>{issue.lineNo ? `第 ${issue.lineNo} 行：` : ''}{issue.code}{issue.message ? ` — ${issue.message}` : ''}</div>)}</div></GlassPanel> : null}
        </div>
        <footer className="flex justify-end gap-2 border-t border-cyan-500/20 px-4 py-3"><NeonButton onClick={onClose}>取消</NeonButton><NeonButton onClick={submit} disabled={mutation.isPending} icon={mutation.isPending ? <Loader2 className="animate-spin" /> : <Play />}>{mutation.isPending ? '提交中…' : '执行'}</NeonButton></footer>
        {warningInput ? <div className="absolute inset-0 z-10 flex items-center justify-center bg-[#03050d]/90 p-6"><div className="w-full max-w-sm space-y-3 border border-amber-400/40 bg-[#070b18] p-5"><h2 className="font-display text-base font-bold text-amber-200">校验发现警告</h2><p className="font-mono text-xs text-cyan-100/70">脚本包含警告，确认后继续执行。</p><div className="flex justify-end gap-2"><NeonButton onClick={() => setWarningInput(null)}>取消</NeonButton><NeonButton onClick={() => { const next = warningInput; setWarningInput(null); void execute({ ...next, confirmWarnings: true }) }}>确认执行</NeonButton></div></div></div> : null}
      </div>
    </div>
  )
}
