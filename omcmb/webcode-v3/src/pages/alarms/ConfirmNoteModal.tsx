import { useEffect, useState } from 'react'
import { X } from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'

export function ConfirmNoteModal({
  open,
  title,
  message,
  confirmText = '确认',
  danger,
  loading,
  onConfirm,
  onCancel,
}: {
  open: boolean
  title: string
  message: string
  confirmText?: string
  danger?: boolean
  loading?: boolean
  onConfirm: (note: string) => void
  onCancel: () => void
}) {
  const [note, setNote] = useState('')

  useEffect(() => {
    if (open) setNote('')
  }, [open])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center" role="dialog" aria-modal="true">
      <button
        type="button"
        aria-label="关闭"
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onCancel}
      />
      <div className="glass-strong warp-in relative w-[420px] max-w-[92vw] rounded-sm border border-cyan-500/30">
        <div className="flex items-center justify-between border-b border-cyan-500/20 px-4 py-3">
          <span
            className="font-display text-sm font-bold"
            style={{ color: danger ? '#ff7a1a' : '#00f0ff' }}
          >
            {title}
          </span>
          <button
            type="button"
            onClick={onCancel}
            className="rounded-sm border border-cyan-500/25 p-1 text-cyan-300/70 hover:border-cyan-400/60 hover:text-cyan-200"
          >
            <X className="size-4" />
          </button>
        </div>
        <div className="space-y-3 p-4">
          <div className="text-[13px] text-cyan-100/85">{message}</div>
          <div>
            <div className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/55">
              备注 · NOTE（可选）
            </div>
            <textarea
              className="neon-input h-20 w-full resize-none py-2"
              placeholder="填写处理备注…"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </div>
        </div>
        <div className="flex justify-end gap-2 border-t border-cyan-500/15 px-4 py-3">
          <NeonButton onClick={onCancel} disabled={loading}>
            取消
          </NeonButton>
          <NeonButton
            tone={danger ? 'danger' : 'cyan'}
            disabled={loading}
            onClick={() => onConfirm(note.trim())}
          >
            {loading ? '处理中…' : confirmText}
          </NeonButton>
        </div>
      </div>
    </div>
  )
}
