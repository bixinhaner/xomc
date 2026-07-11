import { useEffect, useRef, useState } from 'react'
import { Download, Loader2, Upload, X } from 'lucide-react'
import { NeonButton } from '@/components/ui/NeonButton'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { useT, type TranslateFn } from '@/hooks/useT'
import { useCreateImportedMMLScript, useReplaceImportedMMLScript, useValidateMMLScriptImport } from '@core/hooks/api/useMML'
import { mmlApi } from '@core/services/api/mmlApi'
import type { MMLScript, MMLScriptImportValidation, MMLScriptIssue, MMLTaskPlanItem } from '@core/types/mml'

export interface ScriptImportDialogProps {
  open: boolean
  onClose: () => void
  script?: MMLScript | null
  onSaved?: (script: MMLScript) => void
}

function validationFromError(error: unknown): MMLScriptImportValidation | undefined {
  if (!error || typeof error !== 'object') return undefined
  const value = (error as { validation?: unknown }).validation
  return value && typeof value === 'object' ? value as MMLScriptImportValidation : undefined
}

function issueText(issue: MMLScriptIssue, t: TranslateFn) {
  return `${issue.lineNo ? t('mml.scriptImport.linePrefix', { line: issue.lineNo }) : ''}${issue.code}${issue.message ? ` — ${issue.message}` : ''}`
}

/** STARFORGE TXT import/re-import. Content and plans are server-authoritative read-only snapshots. */
export default function ScriptImportDialog({ open, onClose, script, onSaved }: ScriptImportDialogProps) {
  const t = useT()
  const inputRef = useRef<HTMLInputElement>(null)
  const [validation, setValidation] = useState<MMLScriptImportValidation | null>(null)
  const [scriptName, setScriptName] = useState('')
  const [description, setDescription] = useState('')
  const [filter, setFilter] = useState<'all' | 'error' | 'warning'>('all')
  const [uploading, setUploading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState('')
  const [confirmWarnings, setConfirmWarnings] = useState(false)
  const validateMutation = useValidateMMLScriptImport()
  const createMutation = useCreateImportedMMLScript()
  const replaceMutation = useReplaceImportedMMLScript()

  useEffect(() => {
    if (!open) return
    setValidation(null)
    setFilter('all')
    setNotice('')
    setConfirmWarnings(false)
    setScriptName(script?.scriptName ?? '')
    setDescription(script?.description ?? '')
    const onKeyDown = (event: KeyboardEvent) => { if (event.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, script, onClose])

  if (!open) return null
  const issues = validation?.issues ?? []
  const hasErrors = Boolean(validation && (validation.summary.errorCount > 0 || issues.some((i) => i.severity === 'error')))
  const hasWarnings = Boolean(validation && (validation.summary.warningCount > 0 || issues.some((i) => i.severity === 'warning')))
  const visibleIssues = filter === 'all' ? issues : issues.filter((i) => i.severity === filter)
  const visiblePlanItems = filter === 'all'
    ? (validation?.planItems ?? [])
    : (validation?.planItems ?? []).filter((item) => visibleIssues.some((issue) => issue.lineNo === item.lineNo))

  const validateFile = async (file: File) => {
    setUploading(true); setValidation(null); setNotice('')
    try {
      const next = script?.id ? await mmlApi.validateScriptReplacement(script.id, file) : await validateMutation.mutateAsync(file)
      setValidation(next)
    } catch (error) {
      setValidation(validationFromError(error) ?? null)
      const status = typeof error === 'object' && error && 'status' in error ? ` (${String((error as { status?: unknown }).status)})` : ''
      setNotice(`${error instanceof Error ? error.message : t('mml.scriptImport.validationFailed')}${status}`)
    } finally { setUploading(false) }
  }

  const doSave = async () => {
    if (!validation?.validationToken || hasErrors) return
    setSaving(true); setNotice('')
    try {
      const result = script?.id
        ? await replaceMutation.mutateAsync({ id: script.id, input: { validationToken: validation.validationToken, scriptName: scriptName.trim(), description: description.trim(), tags: [], expectedUpdatedAt: script.updateTime } })
        : await createMutation.mutateAsync({ validationToken: validation.validationToken, scriptName: scriptName.trim(), description: description.trim(), tags: [] })
      onSaved?.(result as MMLScript); onClose()
    } catch (error) {
      setValidation(validationFromError(error) ?? null)
      const status = typeof error === 'object' && error && 'status' in error ? ` (${String((error as { status?: unknown }).status)})` : ''
      setNotice(`${error instanceof Error ? error.message : t('common.saveFailed')}${status}`)
    } finally { setSaving(false); setConfirmWarnings(false) }
  }

  const save = () => {
    if (!validation?.validationToken || hasErrors || !scriptName.trim()) return
    if (hasWarnings && !confirmWarnings) { setConfirmWarnings(true); return }
    void doSave()
  }

  const downloadTemplate = async () => {
    try {
      const result = await mmlApi.downloadScriptImportTemplate()
      const url = URL.createObjectURL(result.blob); const anchor = document.createElement('a')
      anchor.href = url; anchor.download = result.filename; anchor.click(); URL.revokeObjectURL(url)
    } catch (error) { setNotice(error instanceof Error ? error.message : t('mml.scriptImport.downloadTemplateFailed')) }
  }

  const downloadReport = () => {
    const report = visibleIssues.map((issue) => `${issue.lineNo ?? '-'}\t${issue.severity}\t${issue.code}\t${issue.message ?? ''}`).join('\n')
    const url = URL.createObjectURL(new Blob([report], { type: 'text/plain;charset=utf-8' })); const anchor = document.createElement('a')
    anchor.href = url; anchor.download = 'mml-script-errors.txt'; anchor.click(); URL.revokeObjectURL(url)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label={script ? t('mml.scriptImport.replaceTitle') : t('mml.scriptImport.title')} onClick={onClose}>
      <div className="glass-strong relative flex max-h-[90vh] w-[min(920px,calc(100vw-32px))] flex-col overflow-hidden border border-cyan-500/30" onClick={(event) => event.stopPropagation()}>
        <header className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <div><div className="font-display text-base font-bold text-cyan-100">{script ? t('mml.scriptImport.replaceTitle') : t('mml.scriptImport.title')}</div><div className="font-mono text-[10px] text-cyan-300/50">SERVER VALIDATION · READ-ONLY SNAPSHOT</div></div>
          <button type="button" onClick={onClose} aria-label={t('common.close')} className="rounded-sm border border-cyan-500/25 p-1.5 text-cyan-300/70 hover:text-cyan-100"><X className="size-4" /></button>
        </header>
        <div className="min-h-0 flex-1 space-y-3 overflow-auto px-4 py-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block"><span className="hud-label">{t('mml.scriptName')}</span><input aria-label={t('mml.scriptName')} className="neon-input w-full" value={scriptName} onChange={(e) => setScriptName(e.target.value)} /></label>
            <label className="block"><span className="hud-label">{t('mml.description')}</span><input aria-label={t('mml.description')} className="neon-input w-full" value={description} onChange={(e) => setDescription(e.target.value)} /></label>
          </div>
          <div className="flex flex-wrap gap-2">
            <NeonButton icon={<Upload />} onClick={() => inputRef.current?.click()} disabled={uploading}>{uploading ? t('mml.scriptImport.validating') : t('mml.scriptImport.chooseTxt')}</NeonButton>
            <input ref={inputRef} className="sr-only" type="file" accept=".txt,text/plain" aria-label={t('mml.scriptImport.chooseTxt')} onChange={(event) => { const file = event.target.files?.[0]; if (file) void validateFile(file); event.currentTarget.value = '' }} />
            <NeonButton icon={<Download />} onClick={() => void downloadTemplate()}>{t('mml.scriptImport.downloadTemplate')}</NeonButton>
          </div>
          {uploading ? <div role="status" className="font-mono text-xs text-cyan-200">{t('mml.scriptImport.validating')}</div> : null}
          {notice ? <div role="alert" className="border border-rose-400/40 bg-rose-500/10 px-3 py-2 font-mono text-xs text-rose-200">{notice}</div> : null}
          {validation ? <GlassPanel title={`VALIDATION SNAPSHOT · ${t('mml.scriptImport.readOnlyPreview')}`} meta={validation.originalFilename ?? 'TXT'}>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <MiniStat label={t('mml.scriptImport.validLines')} value={validation.summary.validLines} /><MiniStat label={t('mml.scriptImport.deviceCount')} value={validation.summary.deviceCount} /><MiniStat label={t('mml.scriptImport.errorCount')} value={validation.summary.errorCount} /><MiniStat label={t('mml.scriptImport.warningCount')} value={validation.summary.warningCount} />
            </div>
            <div className="mt-3 flex flex-wrap gap-2">
              {(['all', 'error', 'warning'] as const).map((value) => <NeonButton key={value} onClick={() => setFilter(value)}>{value === 'all' ? t('mml.scriptImport.all') : value === 'error' ? t('mml.scriptImport.onlyErrors') : t('mml.scriptImport.onlyWarnings')}</NeonButton>)}
              <NeonButton onClick={downloadReport} disabled={!visibleIssues.length}>{t('mml.scriptImport.downloadErrorReport')}</NeonButton>
            </div>
            {visibleIssues.length ? <div className="mt-3 space-y-1 border border-cyan-500/15 bg-black/20 p-3">{visibleIssues.map((issue, index) => <div key={`${issue.code}-${issue.lineNo ?? 'x'}-${index}`} className={`font-mono text-xs ${issue.severity === 'error' ? 'text-rose-300' : 'text-amber-200'}`}><span className="mr-2 border border-current px-1 text-[10px] uppercase">{issue.severity}</span>{issueText(issue, t)}</div>)}</div> : null}
            <div className="mt-3 overflow-auto border border-cyan-500/15"><div className="grid grid-cols-[64px_150px_64px_1fr] border-b border-cyan-500/15 px-3 py-2 font-mono text-[10px] uppercase tracking-[.12em] text-cyan-300/55"><span>{t('mml.scriptImport.lineNo')}</span><span>{t('mml.scriptImport.deviceSn')}</span><span>{t('mml.scriptImport.order')}</span><span>{t('mml.scriptImport.command')}</span></div>{visiblePlanItems.map((item) => <PlanRow key={`${item.lineNo}-${item.deviceSn}-${item.order}`} item={item} />)}</div>
          </GlassPanel> : null}
        </div>
        <footer className="flex justify-end gap-2 border-t border-cyan-500/20 px-4 py-3"><NeonButton onClick={onClose}>{t('common.cancel')}</NeonButton><NeonButton onClick={save} disabled={saving || uploading || !validation?.validationToken || hasErrors || !scriptName.trim()} icon={saving ? <Loader2 className="animate-spin" /> : undefined}>{saving ? t('mml.scriptImport.saving') : t('mml.scriptImport.saveConfirm')}</NeonButton></footer>
        {confirmWarnings ? <div className="absolute inset-0 z-10 flex items-center justify-center bg-[#03050d]/90 p-6"><div className="w-full max-w-sm space-y-3 border border-amber-400/40 bg-[#070b18] p-5"><h2 className="font-display text-base font-bold text-amber-200">{t('mml.scriptImport.warningTitle')}</h2><p className="font-mono text-xs text-cyan-100/70">{t('mml.scriptImport.warningSaveContent')}</p><div className="flex justify-end gap-2"><NeonButton onClick={() => setConfirmWarnings(false)}>{t('common.cancel')}</NeonButton><NeonButton onClick={() => void doSave()}>{t('mml.scriptImport.continueSave')}</NeonButton></div></div></div> : null}
      </div>
    </div>
  )
}

function MiniStat({ label, value }: { label: string; value: number }) { return <div className="border border-cyan-500/15 bg-cyan-500/5 px-3 py-2"><div className="hud-label">{label}</div><div className="font-display text-xl font-bold text-cyan-100">{value}</div></div> }
function PlanRow({ item }: { item: MMLTaskPlanItem }) { return <div className="grid grid-cols-[64px_150px_64px_1fr] border-b border-cyan-500/10 px-3 py-2 font-mono text-xs text-cyan-100/85"><span>{item.lineNo}</span><span className="truncate text-cyan-300/75">{item.deviceSn}</span><span>{item.order}</span><span className="truncate">{item.command.commandCode}</span></div> }
