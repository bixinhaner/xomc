import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  AlertTriangle,
  ArrowLeft,
  FileQuestion,
  Loader2,
  Mail,
  MessageSquare,
  Pencil,
  RefreshCcw,
  Save,
  Trash2,
  Webhook,
  X,
} from 'lucide-react'

import { PageShell } from '@/components/shell/PageShell'
import { GlassPanel } from '@/components/ui/GlassPanel'
import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import {
  useNotificationTemplate,
  useUpdateNotificationTemplate,
  useDeleteNotificationTemplate,
} from '@core/hooks/api/useNotifications'
import type {
  NotificationChannel,
  NotificationLanguage,
  NotificationTemplateUpdatePayload,
} from '@core/types/notification'

const CHANNEL_META: Record<
  NotificationChannel,
  { label: string; color: string; icon: typeof Mail }
> = {
  email: { label: '邮件', color: '#5b9eff', icon: Mail },
  sms: { label: '短信', color: '#00ff88', icon: MessageSquare },
  webhook: { label: 'Webhook', color: '#b96bff', icon: Webhook },
}

interface EditState {
  name: string
  channel: NotificationChannel
  language: NotificationLanguage
  subject: string
  body: string
  variables: string
  enabled: boolean
}

export default function TemplateDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching, refetch } = useNotificationTemplate(
    id ?? ''
  )
  const updateMut = useUpdateNotificationTemplate()
  const deleteMut = useDeleteNotificationTemplate()

  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState<EditState | null>(null)
  const [opError, setOpError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (data && editing && form === null) {
      setForm({
        name: data.name,
        channel: data.channel,
        language: data.language,
        subject: data.subject,
        body: data.body,
        variables: data.variables.join(', '),
        enabled: data.enabled,
      })
    }
  }, [data, editing, form])

  const startEdit = () => {
    if (!data) return
    setForm({
      name: data.name,
      channel: data.channel,
      language: data.language,
      subject: data.subject,
      body: data.body,
      variables: data.variables.join(', '),
      enabled: data.enabled,
    })
    setOpError(null)
    setEditing(true)
  }

  const cancelEdit = () => {
    setEditing(false)
    setForm(null)
    setOpError(null)
  }

  const onSave = useCallback(async () => {
    if (!data || !form) return
    if (!form.name.trim() || !form.subject.trim() || !form.body.trim()) {
      setOpError('名称 / 主题 / 内容为必填')
      return
    }
    const payload: NotificationTemplateUpdatePayload = {
      name: form.name.trim(),
      channel: form.channel,
      language: form.language,
      subject: form.subject.trim(),
      body: form.body,
      variables: form.variables
        .split(',')
        .map((v) => v.trim())
        .filter(Boolean),
      enabled: form.enabled,
    }
    setBusy(true)
    setOpError(null)
    try {
      await updateMut.mutateAsync({ id: data.id, payload })
      await refetch()
      setEditing(false)
      setForm(null)
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '保存失败')
    } finally {
      setBusy(false)
    }
  }, [data, form, updateMut, refetch])

  const onDelete = useCallback(async () => {
    if (!data) return
    setBusy(true)
    setOpError(null)
    try {
      await deleteMut.mutateAsync(data.id)
      navigate('/notifications/templates')
    } catch (e) {
      setOpError(e instanceof Error ? e.message : '删除失败')
      setBusy(false)
    }
  }, [data, deleteMut, navigate])

  const meta = data ? CHANNEL_META[data.channel] : null

  return (
    <PageShell
      code="F-N"
      title="TEMPLATE DETAIL · 模板详情"
      subtitle={id ? `TPL-ID · ${id}` : 'NO TEMPLATE SELECTED'}
      isFetching={isFetching}
      toolbar={
        <>
          <NeonButton icon={<ArrowLeft />} onClick={() => navigate('/notifications/templates')}>
            BACK
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      {opError && (
        <div className="mb-2 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
          OP FAILED · {opError}
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
          <Loader2 className="size-4 animate-spin" />
          <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
        </div>
      ) : isError ? (
        <div className="flex flex-col items-center gap-3 py-14">
          <AlertTriangle className="size-9 text-rose-400/70" />
          <div className="font-mono text-sm text-rose-300">
            FAILURE · {error instanceof Error ? error.message : '未知错误'}
          </div>
          <NeonButton tone="danger" icon={<RefreshCcw />} onClick={() => refetch()}>
            RETRY
          </NeonButton>
        </div>
      ) : !data ? (
        <div className="flex flex-col items-center justify-center gap-3 py-16">
          <FileQuestion className="size-10 text-cyan-300/40" />
          <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
            TEMPLATE NOT FOUND · 未找到模板 {id}
          </div>
          <NeonButton
            icon={<ArrowLeft />}
            onClick={() => navigate('/notifications/templates')}
          >
            返回模板列表
          </NeonButton>
        </div>
      ) : editing && form ? (
        <div className="grid grid-cols-12 gap-3">
          <GlassPanel title="EDIT TEMPLATE · 编辑模板" meta="UPDATE" className="col-span-12">
            <div className="space-y-3 p-4">
              <EditField label="名称">
                <input
                  className="neon-input w-full"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </EditField>
              <div className="grid grid-cols-2 gap-3">
                <EditField label="渠道">
                  <select
                    className="neon-input w-full"
                    value={form.channel}
                    onChange={(e) =>
                      setForm({ ...form, channel: e.target.value as NotificationChannel })
                    }
                  >
                    <option value="email">邮件</option>
                    <option value="sms">短信</option>
                    <option value="webhook">Webhook</option>
                  </select>
                </EditField>
                <EditField label="语言">
                  <select
                    className="neon-input w-full"
                    value={form.language}
                    onChange={(e) =>
                      setForm({ ...form, language: e.target.value as NotificationLanguage })
                    }
                  >
                    <option value="zh-CN">简体中文</option>
                    <option value="en-US">English</option>
                  </select>
                </EditField>
              </div>
              <EditField label="主题">
                <input
                  className="neon-input w-full"
                  value={form.subject}
                  onChange={(e) => setForm({ ...form, subject: e.target.value })}
                />
              </EditField>
              <EditField label="内容">
                <textarea
                  className="neon-input w-full"
                  rows={8}
                  value={form.body}
                  onChange={(e) => setForm({ ...form, body: e.target.value })}
                />
              </EditField>
              <EditField label="变量（逗号分隔）">
                <input
                  className="neon-input w-full"
                  value={form.variables}
                  onChange={(e) => setForm({ ...form, variables: e.target.value })}
                />
              </EditField>
              <label className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.12em] text-cyan-300/70">
                <input
                  type="checkbox"
                  checked={form.enabled}
                  onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
                />
                启用模板
              </label>
            </div>
            <div className="flex justify-end gap-2 border-t border-cyan-500/15 p-3.5">
              <NeonButton icon={<X />} onClick={cancelEdit}>
                取消
              </NeonButton>
              <NeonButton icon={<Save />} onClick={() => void onSave()} disabled={busy}>
                {busy ? '保存中…' : '保存'}
              </NeonButton>
            </div>
          </GlassPanel>
        </div>
      ) : (
        <div className="grid grid-cols-12 gap-3">
          {/* 概要 */}
          <GlassPanel
            title="OVERVIEW · 概要"
            meta={meta?.label}
            className="col-span-12 lg:col-span-5"
          >
            <div className="space-y-0">
              <Field label="模板名称">
                <span className="font-display text-sm font-bold text-cyan-100">{data.name}</span>
              </Field>
              <Field label="渠道">
                <span className="chip" style={{ color: meta?.color }}>
                  {meta?.label}
                </span>
              </Field>
              <Field label="语言">{data.language}</Field>
              <Field label="启用状态">
                <StatusBadge
                  status={data.enabled ? 'active' : 'inactive'}
                  label={data.enabled ? '已启用' : '已禁用'}
                />
              </Field>
              <Field label="变量">
                {data.variables.length === 0 ? (
                  '—'
                ) : (
                  <span className="flex flex-wrap gap-1.5">
                    {data.variables.map((v) => (
                      <span key={v} className="chip text-cyan-300/80">
                        {`{{${v}}}`}
                      </span>
                    ))}
                  </span>
                )}
              </Field>
              <Field label="创建时间">{formatTime(data.createdAt)}</Field>
              <Field label="更新时间">{formatTime(data.updatedAt)}</Field>
            </div>

            <div className="flex flex-wrap gap-2 border-t border-cyan-500/15 p-3.5">
              {busy ? (
                <span className="flex items-center gap-2 font-mono text-xs text-cyan-300/70">
                  <Loader2 className="size-4 animate-spin" /> 处理中…
                </span>
              ) : (
                <>
                  <NeonButton icon={<Pencil />} onClick={startEdit}>
                    编辑
                  </NeonButton>
                  <NeonButton icon={<Trash2 />} tone="danger" onClick={() => void onDelete()}>
                    删除
                  </NeonButton>
                </>
              )}
            </div>
          </GlassPanel>

          {/* 主题 + 内容 */}
          <div className="col-span-12 space-y-3 lg:col-span-7">
            <GlassPanel title="SUBJECT · 主题">
              <div className="px-3.5 py-3 font-mono text-sm text-cyan-100/90">
                {data.subject || '—'}
              </div>
            </GlassPanel>
            <GlassPanel title="BODY · 内容">
              <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap break-words px-3.5 py-3 font-mono text-[12px] leading-relaxed text-cyan-200/85">
                {data.body || '—'}
              </pre>
            </GlassPanel>
          </div>
        </div>
      )}
    </PageShell>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[100px_1fr] items-center gap-3 border-b border-cyan-500/10 px-3.5 py-2 last:border-b-0">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </div>
      <div className="break-words text-[12px] text-cyan-100/90">{children ?? '—'}</div>
    </div>
  )
}

function EditField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </div>
      {children}
    </div>
  )
}
