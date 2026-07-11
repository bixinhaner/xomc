import { useEffect, useRef, useState } from 'react'
import { Download, Upload, X } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useT, type TranslateFn } from '@/hooks/useT'
import { useCreateImportedMMLScript, useReplaceImportedMMLScript, useValidateMMLScriptImport } from '@core/hooks/api/useMML'
import { mmlApi } from '@core/services/api/mmlApi'
import type { MMLScript, MMLScriptImportValidation, MMLScriptIssue } from '@core/types/mml'

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

function issueMessage(issue: MMLScriptIssue, t: TranslateFn) {
  return `${issue.lineNo ? t('mml.scriptImport.linePrefix', { line: issue.lineNo }) : ''}${issue.code}${issue.message ? ` — ${issue.message}` : ''}`
}

/** TXT import/re-import dialog. The content and plan are always read-only server snapshots. */
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
  const visiblePlanItems = filter === 'all' ? (validation?.planItems ?? []) : (validation?.planItems ?? []).filter((item) => visibleIssues.some((issue) => issue.lineNo === item.lineNo))

  const validateFile = async (file: File) => {
    setUploading(true)
    setValidation(null)
    setNotice('')
    try {
      const next = script?.id ? await mmlApi.validateScriptReplacement(script.id, file) : await validateMutation.mutateAsync(file)
      setValidation(next)
    } catch (error) {
      setValidation(validationFromError(error) ?? null)
      const status = typeof error === 'object' && error && 'status' in error ? ` (${String((error as { status?: unknown }).status)})` : ''
      setNotice(`${error instanceof Error ? error.message : t('mml.scriptImport.validationFailed')}${status}`)
    } finally {
      setUploading(false)
    }
  }

  const doSave = async () => {
    if (!validation?.validationToken || hasErrors) return
    setSaving(true)
    setNotice('')
    try {
      const result = script?.id
        ? await replaceMutation.mutateAsync({ id: script.id, input: { validationToken: validation.validationToken, scriptName: scriptName.trim(), description: description.trim(), tags: [], expectedUpdatedAt: script.updateTime } })
        : await createMutation.mutateAsync({ validationToken: validation.validationToken, scriptName: scriptName.trim(), description: description.trim(), tags: [] })
      onSaved?.(result as MMLScript)
      onClose()
    } catch (error) {
      setValidation(validationFromError(error) ?? null)
      const status = typeof error === 'object' && error && 'status' in error ? ` (${String((error as { status?: unknown }).status)})` : ''
      setNotice(`${error instanceof Error ? error.message : t('common.saveFailed')}${status}`)
    } finally {
      setSaving(false)
      setConfirmWarnings(false)
    }
  }

  const save = () => {
    if (hasWarnings && !confirmWarnings) { setConfirmWarnings(true); return }
    void doSave()
  }

  const downloadTemplate = async () => {
    try {
      const result = await mmlApi.downloadScriptImportTemplate()
      const url = URL.createObjectURL(result.blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = result.filename
      anchor.click()
      URL.revokeObjectURL(url)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : t('mml.scriptImport.downloadTemplateFailed'))
    }
  }

  const downloadReport = () => {
    const report = visibleIssues.map((issue) => `${issue.lineNo ?? '-'}\t${issue.severity}\t${issue.code}\t${issue.message ?? ''}`).join('\n')
    const url = URL.createObjectURL(new Blob([report], { type: 'text/plain;charset=utf-8' }))
    const anchor = document.createElement('a'); anchor.href = url; anchor.download = 'mml-script-errors.txt'; anchor.click(); URL.revokeObjectURL(url)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" role="dialog" aria-modal="true" aria-label={script ? t('mml.scriptImport.replaceTitle') : t('mml.scriptImport.title')}>
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden="true" />
      <div className="relative flex max-h-[90vh] w-[min(900px,calc(100vw-32px))] flex-col overflow-hidden rounded-lg border bg-background shadow-xl">
        <header className="flex items-center justify-between border-b px-5 py-3"><div className="font-semibold">{script ? t('mml.scriptImport.replaceTitle') : t('mml.scriptImport.title')}</div><Button variant="ghost" size="icon" onClick={onClose} aria-label={t('common.close')}><X /></Button></header>
        <div className="min-h-0 flex-1 space-y-4 overflow-auto p-5">
          <div className="grid gap-3 sm:grid-cols-2"><label className="space-y-1 text-sm">{t('mml.scriptName')}<Input aria-label={t('mml.scriptName')} value={scriptName} onChange={(e) => setScriptName(e.target.value)} /></label><label className="space-y-1 text-sm">{t('mml.description')}<Input aria-label={t('mml.description')} value={description} onChange={(e) => setDescription(e.target.value)} /></label></div>
          <div className="flex flex-wrap gap-2"><Button variant="outline" onClick={() => inputRef.current?.click()} disabled={uploading}><Upload />{t('mml.scriptImport.chooseTxt')}</Button><input ref={inputRef} className="sr-only" type="file" accept=".txt,text/plain" aria-label={t('mml.scriptImport.chooseTxt')} onChange={(e) => { const file = e.target.files?.[0]; if (file) void validateFile(file); e.currentTarget.value = '' }} /><Button variant="outline" onClick={() => void downloadTemplate()}><Download />{t('mml.scriptImport.downloadTemplate')}</Button></div>
          {uploading ? <div className="text-sm text-muted-foreground">{t('mml.scriptImport.validating')}</div> : null}
          {notice ? <div className="rounded border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm text-destructive">{notice}</div> : null}
          {validation ? <div className="sr-only">{t('mml.scriptImport.readOnlyPreview')} {issues.map((issue) => issue.code).join(' ')}</div> : null}
          {validation ? <section className="space-y-3"><div className="text-xs text-muted-foreground">{t('mml.scriptImport.fileLabel')}：<span className="font-mono">{validation.originalFilename ?? '—'}</span> · {t('mml.scriptImport.readOnlyPreview')}</div><div className="grid grid-cols-2 gap-2 sm:grid-cols-4">{[[t('mml.scriptImport.validLines'), validation.summary.validLines], [t('mml.scriptImport.deviceCount'), validation.summary.deviceCount], [t('mml.scriptImport.errorCount'), validation.summary.errorCount], [t('mml.scriptImport.warningCount'), validation.summary.warningCount]].map(([label, value]) => <div key={String(label)} className="rounded border bg-muted/30 px-3 py-2"><div className="text-xs text-muted-foreground">{label}</div><div className="text-lg font-semibold">{value}</div></div>)}</div><div className="flex flex-wrap gap-2"><Button size="sm" variant={filter === 'all' ? 'default' : 'outline'} onClick={() => setFilter('all')}>{t('mml.scriptImport.all')}</Button><Button size="sm" variant={filter === 'error' ? 'default' : 'outline'} onClick={() => setFilter('error')}>{t('mml.scriptImport.onlyErrors')}</Button><Button size="sm" variant={filter === 'warning' ? 'default' : 'outline'} onClick={() => setFilter('warning')}>{t('mml.scriptImport.onlyWarnings')}</Button><Button size="sm" variant="outline" disabled={!visibleIssues.length} onClick={downloadReport}>{t('mml.scriptImport.downloadErrorReport')}</Button></div>{visibleIssues.length ? <div className="space-y-1 rounded border p-3">{visibleIssues.map((issue, index) => <div key={`${issue.code}-${issue.lineNo ?? 'x'}-${index}`} className={issue.severity === 'error' ? 'text-sm text-destructive' : 'text-sm text-amber-600'}><Badge variant={issue.severity === 'error' ? 'destructive' : 'warning'}>{issue.severity}</Badge> <span>{issueMessage(issue, t)}</span></div>)}</div> : null}<div className="rounded border"><Table className="text-left text-xs"><TableHeader className="border-b bg-muted/30"><TableRow><TableHead className="px-3 py-2">{t('mml.scriptImport.lineNo')}</TableHead><TableHead className="px-3 py-2">{t('mml.scriptImport.deviceSn')}</TableHead><TableHead className="px-3 py-2">{t('mml.scriptImport.order')}</TableHead><TableHead className="px-3 py-2">{t('mml.scriptImport.command')}</TableHead></TableRow></TableHeader><TableBody>{visiblePlanItems.map((item) => <TableRow key={`${item.lineNo}-${item.deviceSn}-${item.order}`}><TableCell className="px-3 py-2">{item.lineNo}</TableCell><TableCell className="px-3 py-2 font-mono">{item.deviceSn}</TableCell><TableCell className="px-3 py-2">{item.order}</TableCell><TableCell className="px-3 py-2 font-mono">{item.command.commandCode}</TableCell></TableRow>)}</TableBody></Table></div></section> : null}
        </div>
        <footer className="flex justify-end gap-2 border-t px-5 py-3"><Button variant="outline" onClick={onClose}>{t('common.cancel')}</Button><Button onClick={save} disabled={saving || uploading || !validation?.validationToken || hasErrors || !scriptName.trim()}>{saving ? t('mml.scriptImport.saving') : t('mml.scriptImport.saveConfirm')}</Button></footer>
        {confirmWarnings ? <div className="absolute inset-0 z-10 flex items-center justify-center bg-background/80 p-6"><div className="w-full max-w-sm space-y-3 rounded-lg border bg-background p-5 shadow-lg"><h2 className="font-semibold">{t('mml.scriptImport.warningTitle')}</h2><p className="text-sm text-muted-foreground">{t('mml.scriptImport.warningSaveContent')}</p><div className="flex justify-end gap-2"><Button variant="outline" onClick={() => setConfirmWarnings(false)}>{t('common.cancel')}</Button><Button onClick={() => void doSave()}>{t('mml.scriptImport.continueSave')}</Button></div></div></div> : null}
      </div>
    </div>
  )
}
