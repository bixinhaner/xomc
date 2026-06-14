import { useCallback, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  ChevronRight,
  FileText,
  Loader2,
  Mail,
  MessageSquare,
  Plus,
  RefreshCcw,
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
  useNotificationTemplates,
  useCreateNotificationTemplate,
  useDeleteNotificationTemplate,
} from '@core/hooks/api/useNotifications'
import type {
  NotificationChannel,
  NotificationLanguage,
  NotificationTemplate,
  NotificationTemplateCreatePayload,
} from '@core/types/notification'

const PAGE_SIZE = 20

const CHANNEL_META: Record<
  NotificationChannel,
  { label: string; color: string; icon: typeof Mail }
> = {
  email: { label: '邮件', color: '#5b9eff', icon: Mail },
  sms: { label: '短信', color: '#00ff88', icon: MessageSquare },
  webhook: { label: 'Webhook', color: '#b96bff', icon: Webhook },
}

const CHANNEL_FILTER: { v: '' | NotificationChannel; t: string }[] = [
  { v: '', t: '全部渠道' },
  { v: 'email', t: '邮件' },
  { v: 'sms', t: '短信' },
  { v: 'webhook', t: 'Webhook' },
]

const LANGUAGE_FILTER: { v: '' | NotificationLanguage; t: string }[] = [
  { v: '', t: '全部语言' },
  { v: 'zh-CN', t: '简体中文' },
  { v: 'en-US', t: 'English' },
]

export default function TemplateList() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [channel, setChannel] = useState<'' | NotificationChannel>('')
  const [language, setLanguage] = useState<'' | NotificationLanguage>('')
  const [keyword, setKeyword] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [opError, setOpError] = useState<string | null>(null)
  const [createOpen, setCreateOpen] = useState(false)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(channel ? { channel } : {}),
      ...(language ? { language } : {}),
    }),
    [page, channel, language]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useNotificationTemplates(params)
  const createMut = useCreateNotificationTemplate()
  const deleteMut = useDeleteNotificationTemplate()

  const allItems = useMemo(() => data?.items ?? [], [data])
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // keyword 为前端本页内过滤（后端列表接口不带 keyword）
  const items = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return allItems
    return allItems.filter(
      (t) =>
        t.name.toLowerCase().includes(kw) ||
        (t.subject ?? '').toLowerCase().includes(kw)
    )
  }, [allItems, keyword])

  const enabledCount = useMemo(() => allItems.filter((t) => t.enabled).length, [allItems])

  const onDelete = useCallback(
    async (tpl: NotificationTemplate) => {
      setBusyId(tpl.id)
      setOpError(null)
      try {
        await deleteMut.mutateAsync(tpl.id)
        await refetch()
      } catch (e) {
        setOpError(e instanceof Error ? e.message : '删除失败')
      } finally {
        setBusyId(null)
      }
    },
    [deleteMut, refetch]
  )

  const onCreate = useCallback(
    async (payload: NotificationTemplateCreatePayload) => {
      setOpError(null)
      try {
        const created = await createMut.mutateAsync(payload)
        setCreateOpen(false)
        await refetch()
        navigate(`/notifications/templates/${created.id}`)
      } catch (e) {
        setOpError(e instanceof Error ? e.message : '创建失败')
      }
    },
    [createMut, refetch, navigate]
  )

  return (
    <PageShell
      code="F-N"
      title="NOTIFICATION TEMPLATES · 通知模板"
      subtitle="EMAIL · SMS · WEBHOOK PRESETS"
      isFetching={isFetching}
      toolbar={
        <>
          <input
            className="neon-input w-56"
            placeholder="名称 / 主题（本页）"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
          <NeonButton icon={<Plus />} onClick={() => setCreateOpen(true)}>
            新建模板
          </NeonButton>
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
    >
      <div className="mb-3 grid grid-cols-3 gap-3">
        <Stat label="TEMPLATES · 模板总数" color="#00f0ff" value={total} />
        <Stat label="ENABLED · 本页启用" color="#00ff88" value={enabledCount} />
        <Stat label="DISABLED · 本页禁用" color="#525a78" value={allItems.length - enabledCount} />
      </div>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        {CHANNEL_FILTER.map((o) => (
          <button
            key={o.v || 'all-ch'}
            type="button"
            onClick={() => {
              setChannel(o.v)
              setPage(1)
            }}
            className={`chip transition-all ${
              channel === o.v
                ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-300/80'
            }`}
          >
            {o.t}
          </button>
        ))}
        <span className="mx-1 h-4 w-px bg-cyan-500/20" />
        {LANGUAGE_FILTER.map((o) => (
          <button
            key={o.v || 'all-lang'}
            type="button"
            onClick={() => {
              setLanguage(o.v)
              setPage(1)
            }}
            className={`chip transition-all ${
              language === o.v
                ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                : 'text-cyan-300/45 hover:text-cyan-300/80'
            }`}
          >
            {o.t}
          </button>
        ))}
      </div>

      {opError && (
        <div className="mb-2 border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
          OP FAILED · {opError}
        </div>
      )}

      <GlassPanel title="TEMPLATE MATRIX · 模板列表" meta={`PAGE ${page}/${totalPages}`}>
        {items.length > 0 && (
          <div className="grid grid-cols-[2fr_1fr_0.8fr_2fr_1fr_120px] items-center gap-3 border-b border-cyan-500/10 px-3.5 py-2 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/45">
            <span>NAME · 名称</span>
            <span>CHANNEL · 渠道</span>
            <span>LANG · 语言</span>
            <span>SUBJECT · 主题</span>
            <span>STATE · 状态</span>
            <span className="text-right">OPS · 操作</span>
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-12 text-cyan-300/60">
            <Loader2 className="size-4 animate-spin" />
            <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
          </div>
        ) : isError ? (
          <div className="flex flex-col items-center gap-3 px-4 py-10">
            <AlertTriangle className="size-7 text-rose-400/70" />
            <div className="font-mono text-sm text-rose-300">
              FAILURE · {error instanceof Error ? error.message : '未知错误'}
            </div>
            <NeonButton tone="danger" icon={<RefreshCcw />} onClick={() => refetch()}>
              RETRY
            </NeonButton>
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-14">
            <FileText className="size-9 text-cyan-300/40" />
            <div className="font-mono text-xs uppercase tracking-[0.2em] text-cyan-300/45">
              NO TEMPLATES · 暂无模板
            </div>
          </div>
        ) : (
          items.map((tpl) => {
            const meta = CHANNEL_META[tpl.channel]
            const Icon = meta.icon
            const busy = busyId === tpl.id
            return (
              <div
                key={tpl.id}
                className="grid grid-cols-[2fr_1fr_0.8fr_2fr_1fr_120px] items-center gap-3 border-b border-cyan-500/8 px-3.5 py-2.5 last:border-b-0 hover:bg-cyan-500/5"
              >
                <button
                  type="button"
                  onClick={() => navigate(`/notifications/templates/${tpl.id}`)}
                  className="min-w-0 text-left"
                >
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {tpl.name}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/50">
                    {tpl.variables.length} 变量 · {formatTime(tpl.updatedAt)}
                  </div>
                </button>
                <span className="chip" style={{ color: meta.color }}>
                  <Icon className="size-3" />
                  {meta.label}
                </span>
                <span className="font-mono text-[11px] text-cyan-300/70">{tpl.language}</span>
                <span className="truncate font-mono text-[11px] text-cyan-200/80">
                  {tpl.subject || '—'}
                </span>
                <span>
                  <StatusBadge
                    status={tpl.enabled ? 'active' : 'inactive'}
                    label={tpl.enabled ? '启用' : '禁用'}
                  />
                </span>
                <div className="flex items-center justify-end gap-1.5">
                  {busy ? (
                    <Loader2 className="size-4 animate-spin text-cyan-300/70" />
                  ) : (
                    <>
                      <RowAction
                        title="删除"
                        danger
                        onClick={() => void onDelete(tpl)}
                        icon={<Trash2 className="size-3.5" />}
                      />
                      <RowAction
                        title="详情"
                        onClick={() => navigate(`/notifications/templates/${tpl.id}`)}
                        icon={<ChevronRight className="size-3.5" />}
                      />
                    </>
                  )}
                </div>
              </div>
            )
          })
        )}
      </GlassPanel>

      <div className="mt-4 flex items-center justify-between">
        <span className="font-mono text-[11px] text-cyan-300/55">
          PAGE {page} / {totalPages} · {PAGE_SIZE}/PAGE · TOTAL {total}
        </span>
        <div className="flex gap-2">
          <NeonButton onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
            ◂ PREV
          </NeonButton>
          <NeonButton
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page >= totalPages}
          >
            NEXT ▸
          </NeonButton>
        </div>
      </div>

      {createOpen && (
        <CreateTemplateModal
          submitting={createMut.isPending}
          onClose={() => setCreateOpen(false)}
          onSubmit={(p) => void onCreate(p)}
        />
      )}
    </PageShell>
  )
}

function CreateTemplateModal({
  submitting,
  onClose,
  onSubmit,
}: {
  submitting: boolean
  onClose: () => void
  onSubmit: (payload: NotificationTemplateCreatePayload) => void
}) {
  const [name, setName] = useState('')
  const [channel, setChannel] = useState<NotificationChannel>('email')
  const [language, setLanguage] = useState<NotificationLanguage>('zh-CN')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [variables, setVariables] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [localErr, setLocalErr] = useState<string | null>(null)

  const submit = () => {
    if (!name.trim() || !subject.trim() || !body.trim()) {
      setLocalErr('名称 / 主题 / 内容为必填')
      return
    }
    onSubmit({
      name: name.trim(),
      channel,
      language,
      subject: subject.trim(),
      body,
      variables: variables
        .split(',')
        .map((v) => v.trim())
        .filter(Boolean),
      enabled,
    })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-[560px] max-w-[92vw]">
        <GlassPanel strong title="NEW TEMPLATE · 新建模板" meta="CREATE">
          <div className="space-y-3 p-4">
            <ModalField label="名称">
              <input
                className="neon-input w-full"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="模板名称"
              />
            </ModalField>
            <div className="grid grid-cols-2 gap-3">
              <ModalField label="渠道">
                <select
                  className="neon-input w-full"
                  value={channel}
                  onChange={(e) => setChannel(e.target.value as NotificationChannel)}
                >
                  <option value="email">邮件</option>
                  <option value="sms">短信</option>
                  <option value="webhook">Webhook</option>
                </select>
              </ModalField>
              <ModalField label="语言">
                <select
                  className="neon-input w-full"
                  value={language}
                  onChange={(e) => setLanguage(e.target.value as NotificationLanguage)}
                >
                  <option value="zh-CN">简体中文</option>
                  <option value="en-US">English</option>
                </select>
              </ModalField>
            </div>
            <ModalField label="主题">
              <input
                className="neon-input w-full"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="通知主题"
              />
            </ModalField>
            <ModalField label="内容">
              <textarea
                className="neon-input w-full"
                rows={5}
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder="支持 {{变量}} 占位符"
              />
            </ModalField>
            <ModalField label="变量（逗号分隔）">
              <input
                className="neon-input w-full"
                value={variables}
                onChange={(e) => setVariables(e.target.value)}
                placeholder="deviceName, alarmId"
              />
            </ModalField>
            <label className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.12em] text-cyan-300/70">
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
              />
              启用模板
            </label>
            {localErr && (
              <div className="border border-rose-500/40 bg-rose-500/5 px-3 py-2 font-mono text-xs text-rose-300">
                {localErr}
              </div>
            )}
          </div>
          <div className="flex justify-end gap-2 border-t border-cyan-500/15 p-3.5">
            <NeonButton icon={<X />} onClick={onClose}>
              取消
            </NeonButton>
            <NeonButton icon={<Plus />} onClick={submit} disabled={submitting}>
              {submitting ? '创建中…' : '创建'}
            </NeonButton>
          </div>
        </GlassPanel>
      </div>
    </div>
  )
}

function ModalField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/55">
        {label}
      </div>
      {children}
    </div>
  )
}

function Stat({ label, color, value }: { label: string; color: string; value: number }) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-3 py-2.5"
      style={{ borderLeftColor: color }}
    >
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-cyan-300/65">
        {label}
      </div>
      <div
        className="font-display text-2xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

function RowAction({
  title,
  icon,
  danger,
  disabled,
  onClick,
}: {
  title: string
  icon: React.ReactNode
  danger?: boolean
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      disabled={disabled}
      className={`flex size-6 items-center justify-center rounded-sm border transition-colors ${
        disabled
          ? 'cursor-not-allowed border-cyan-500/10 text-cyan-300/25'
          : danger
            ? 'border-rose-500/30 text-rose-300/80 hover:border-rose-400/70 hover:text-rose-200'
            : 'border-cyan-500/25 text-cyan-300/75 hover:border-cyan-400/60 hover:text-cyan-100'
      }`}
    >
      {icon}
    </button>
  )
}
