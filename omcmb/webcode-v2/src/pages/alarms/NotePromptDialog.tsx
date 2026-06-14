import { useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export function NotePromptDialog({
  open,
  title,
  description,
  confirmText = '确定',
  loading,
  onConfirm,
  onCancel,
}: {
  open: boolean
  title: string
  description?: string
  confirmText?: string
  loading?: boolean
  onConfirm: (note: string) => void
  onCancel: () => void
}) {
  const [note, setNote] = useState('')

  // 每次打开时清空备注
  useEffect(() => {
    if (open) setNote('')
  }, [open])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onCancel} aria-hidden />
      <div className="relative w-full max-w-md rounded-lg border bg-background p-5 shadow-xl">
        <h2 className="text-base font-semibold">{title}</h2>
        {description ? (
          <p className="mt-1 text-sm text-muted-foreground">{description}</p>
        ) : null}
        <div className="mt-4 space-y-1.5">
          <Label htmlFor="alarm-note">备注（可选）</Label>
          <Input
            id="alarm-note"
            value={note}
            placeholder="输入处理备注"
            onChange={(e) => setNote(e.target.value)}
            autoFocus
          />
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={loading}>
            取消
          </Button>
          <Button size="sm" onClick={() => onConfirm(note.trim())} disabled={loading}>
            {confirmText}
          </Button>
        </div>
      </div>
    </div>
  )
}
