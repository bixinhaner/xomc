import { useEffect, useState } from 'react'
import { Play, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useCreateMMLScriptExecution } from '@core/hooks/api/useMML'
import type { MMLExecuteType, MMLScript, MMLScriptExecutionInput, MMLScriptImportValidation } from '@core/types/mml'

export interface ScriptExecutionDialogProps {
  open: boolean
  script: MMLScript | null
  onClose: () => void
  onSuccess?: () => void
}

function validationFromError(error: unknown): MMLScriptImportValidation | undefined {
  if (!error || typeof error !== 'object') return undefined
  const value = (error as { validation?: unknown }).validation
  return value && typeof value === 'object' ? value as MMLScriptImportValidation : undefined
}

/** Executes a stored server snapshot; browser never submits parsed plan/commands. */
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
    setTaskName(script ? `执行脚本: ${script.scriptName}` : '')
    setExecuteType('immediate'); setScheduledAt(''); setPeriodStart(''); setPeriodEnd(''); setPeriodTime('')
    setOfflineRetry(false); setOfflineRetryWait(60); setFailedRetry(false); setFailedRetryCount(3); setFailedRetryInterval(5)
    setValidation(null); setWarningInput(null); setNotice('')
  }, [open, script])

  if (!open || !script) return null

  const execute = async (input: MMLScriptExecutionInput) => {
    try {
      const result = await mutation.mutateAsync({ id: script.id, input })
      const next = result.validation
      const warning = next.summary.warningCount > 0 || next.issues.some((issue) => issue.severity === 'warning')
      if (warning && !input.confirmWarnings) { setValidation(next); setWarningInput(input); return }
      onSuccess?.(); onClose()
    } catch (error) {
      const next = validationFromError(error)
      if (next) setValidation(next)
      const status = typeof error === 'object' && error && 'status' in error ? ` (${String((error as { status?: unknown }).status)})` : ''
      setNotice(`${error instanceof Error ? error.message : '执行校验失败'}${status}`)
    }
  }

  const submit = () => {
    if (!taskName.trim()) { setNotice('请输入任务名称'); return }
    if (executeType === 'scheduled' && !scheduledAt) { setNotice('请选择执行时间'); return }
    if (executeType === 'periodic' && (!periodStart || !periodEnd || !periodTime)) { setNotice('请选择周期日期和时间'); return }
    const input: MMLScriptExecutionInput = { taskName: taskName.trim(), executeType, scheduledAt: scheduledAt || undefined, periodStart: periodStart || undefined, periodEnd: periodEnd || undefined, periodTime: periodTime || undefined, offlineRetry, offlineRetryWait, failedRetry, failedRetryCount, failedRetryInterval }
    void execute(input)
  }

  return <div className="fixed inset-0 z-50 flex items-center justify-center" role="dialog" aria-modal="true" aria-label="执行脚本"><div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden="true" /><div className="relative flex max-h-[90vh] w-[min(620px,calc(100vw-32px))] flex-col overflow-hidden rounded-lg border bg-background shadow-xl"><header className="flex items-center justify-between border-b px-5 py-3"><div><div className="font-semibold">执行脚本</div><div className="text-xs text-muted-foreground">{script.scriptName}</div></div><Button variant="ghost" size="icon" onClick={onClose} aria-label="关闭"><X /></Button></header><div className="min-h-0 flex-1 space-y-4 overflow-auto p-5"><label className="block space-y-1 text-sm">任务名称<Input value={taskName} onChange={(e) => setTaskName(e.target.value)} /></label><label className="block space-y-1 text-sm">执行方式<select aria-label="执行方式" className="h-9 w-full rounded border bg-background px-2" value={executeType} onChange={(e) => setExecuteType(e.target.value as MMLExecuteType)}><option value="immediate">立即</option><option value="suspended">挂起</option><option value="scheduled">定时</option><option value="periodic">周期</option></select></label>{executeType === 'scheduled' ? <label className="block space-y-1 text-sm">执行时间<input aria-label="执行时间" type="datetime-local" className="h-9 w-full rounded border bg-background px-2" value={scheduledAt} onChange={(e) => setScheduledAt(e.target.value)} /></label> : null}{executeType === 'periodic' ? <div className="grid gap-3 sm:grid-cols-3"><label className="space-y-1 text-sm">周期开始<input aria-label="周期开始" type="date" className="h-9 w-full rounded border bg-background px-2" value={periodStart} onChange={(e) => setPeriodStart(e.target.value)} /></label><label className="space-y-1 text-sm">周期结束<input aria-label="周期结束" type="date" className="h-9 w-full rounded border bg-background px-2" value={periodEnd} onChange={(e) => setPeriodEnd(e.target.value)} /></label><label className="space-y-1 text-sm">周期时间<input aria-label="周期时间" type="time" className="h-9 w-full rounded border bg-background px-2" value={periodTime} onChange={(e) => setPeriodTime(e.target.value)} /></label></div> : null}<div className="grid gap-3 sm:grid-cols-2"><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={offlineRetry} onChange={(e) => setOfflineRetry(e.target.checked)} />离线等待重试</label><label className="space-y-1 text-sm">离线等待（秒）<Input type="number" min={1} value={offlineRetryWait} onChange={(e) => setOfflineRetryWait(Number(e.target.value))} /></label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={failedRetry} onChange={(e) => setFailedRetry(e.target.checked)} />失败重试</label><label className="space-y-1 text-sm">失败重试次数<Input type="number" min={0} value={failedRetryCount} onChange={(e) => setFailedRetryCount(Number(e.target.value))} /></label><label className="space-y-1 text-sm">失败重试间隔（秒）<Input type="number" min={1} value={failedRetryInterval} onChange={(e) => setFailedRetryInterval(Number(e.target.value))} /></label></div>{notice ? <div className="rounded border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm text-destructive">{notice}</div> : null}{validation ? <div className="rounded border p-3 text-sm">服务端校验：错误 {validation.summary.errorCount}，警告 {validation.summary.warningCount}</div> : null}</div><footer className="flex justify-end gap-2 border-t px-5 py-3"><Button variant="outline" onClick={onClose}>取消</Button><Button onClick={submit} disabled={mutation.isPending}>{mutation.isPending ? '提交中…' : <><Play />执行</>}</Button></footer>{warningInput ? <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/80 p-6"><div className="w-full max-w-sm space-y-3 rounded-lg border bg-background p-5 shadow-lg"><h2 className="font-semibold">校验发现警告</h2><p className="text-sm text-muted-foreground">脚本包含警告，确认后继续执行。</p><div className="flex justify-end gap-2"><Button variant="outline" onClick={() => setWarningInput(null)}>取消</Button><Button onClick={() => { const next = warningInput; setWarningInput(null); void execute({ ...next, confirmWarnings: true }) }}>确认执行</Button></div></div></div> : null}</div></div>
}
