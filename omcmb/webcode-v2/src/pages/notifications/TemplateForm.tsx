import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Loader2, Save } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import {
  useCreateNotificationTemplate,
  useUpdateNotificationTemplate,
} from '@core/hooks/api/useNotifications'
import type {
  NotificationChannel,
  NotificationLanguage,
  NotificationTemplate,
  NotificationTemplateCreatePayload,
} from '@core/types/notification'

// ============================================================
// 通知中心 → 模板表单（新建 / 编辑共用）
// 对照 v1 webcode/src/pages/notifications/TemplateForm（Drawer 表单）。
// v2 改为整页表单，受控字段 + 校验 + 真实 mutation（create / update）。
// 由 TemplateCreate / TemplateEdit 两个路由页复用，提交成功后回模板列表。
// ============================================================

export interface TemplateFormProps {
  // 编辑态传入已加载的模板；新建态为 undefined
  initial?: NotificationTemplate
}

interface FormState {
  name: string
  channel: NotificationChannel
  language: NotificationLanguage
  subject: string
  body: string
  variables: string
  enabled: boolean
}

function toFormState(t?: NotificationTemplate): FormState {
  return {
    name: t?.name ?? '',
    channel: t?.channel ?? 'email',
    language: t?.language ?? 'zh-CN',
    subject: t?.subject ?? '',
    body: t?.body ?? '',
    variables: (t?.variables ?? []).join(', '),
    enabled: t?.enabled ?? true,
  }
}

export default function TemplateForm({ initial }: TemplateFormProps) {
  const navigate = useNavigate()
  const isEdit = Boolean(initial)

  const [form, setForm] = useState<FormState>(() => toFormState(initial))
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({})

  const createMutation = useCreateNotificationTemplate()
  const updateMutation = useUpdateNotificationTemplate()
  const submitting = createMutation.isPending || updateMutation.isPending
  const submitError = isEdit ? updateMutation.error : createMutation.error

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const validate = (): boolean => {
    const next: Partial<Record<keyof FormState, string>> = {}
    if (!form.name.trim()) next.name = '请输入模板名称'
    if (!form.subject.trim()) next.subject = '请输入主题'
    if (!form.body.trim()) next.body = '请输入正文内容'
    setErrors(next)
    return Object.keys(next).length === 0
  }

  const handleSubmit = () => {
    if (!validate()) return
    const variables = form.variables
      .split(',')
      .map((v) => v.trim())
      .filter(Boolean)
    const payload: NotificationTemplateCreatePayload = {
      name: form.name.trim(),
      channel: form.channel,
      language: form.language,
      subject: form.subject.trim(),
      body: form.body,
      variables,
      enabled: form.enabled,
    }

    if (isEdit && initial) {
      updateMutation.mutate(
        { id: initial.id, payload },
        { onSuccess: () => navigate('/notifications/templates') },
      )
    } else {
      createMutation.mutate(payload, {
        onSuccess: () => navigate('/notifications/templates'),
      })
    }
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-5 p-6">
        <Field label="模板名称" required error={errors.name}>
          <Input
            value={form.name}
            placeholder="如：设备离线告警通知"
            onChange={(e) => set('name', e.target.value)}
          />
        </Field>

        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
          <Field label="渠道" required>
            <Select
              value={form.channel}
              onValueChange={(v) => set('channel', v as NotificationChannel)}
            >
              <SelectTrigger>
                <SelectValue placeholder="选择渠道" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="email">邮件</SelectItem>
                <SelectItem value="sms">短信</SelectItem>
                <SelectItem value="webhook">Webhook</SelectItem>
              </SelectContent>
            </Select>
          </Field>

          <Field label="语言" required>
            <Select
              value={form.language}
              onValueChange={(v) => set('language', v as NotificationLanguage)}
            >
              <SelectTrigger>
                <SelectValue placeholder="选择语言" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="zh-CN">简体中文</SelectItem>
                <SelectItem value="en-US">English</SelectItem>
              </SelectContent>
            </Select>
          </Field>
        </div>

        <Field label="主题" required error={errors.subject}>
          <Input
            value={form.subject}
            placeholder="通知主题（支持变量占位）"
            onChange={(e) => set('subject', e.target.value)}
          />
        </Field>

        <Field label="正文内容" required error={errors.body}>
          <textarea
            className="min-h-[160px] w-full rounded-md border bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            value={form.body}
            placeholder="通知正文，可使用 {{变量名}} 占位符"
            onChange={(e) => set('body', e.target.value)}
          />
        </Field>

        <Field
          label="变量"
          hint="以英文逗号分隔，如：deviceName, alarmTime"
        >
          <Input
            value={form.variables}
            placeholder="deviceName, alarmTime"
            onChange={(e) => set('variables', e.target.value)}
          />
        </Field>

        <Field label="启用状态">
          <label className="flex w-fit cursor-pointer items-center gap-2 text-sm">
            <input
              type="checkbox"
              className="size-4 cursor-pointer accent-primary"
              checked={form.enabled}
              onChange={(e) => set('enabled', e.target.checked)}
            />
            {form.enabled ? '已启用' : '已禁用'}
          </label>
        </Field>

        {submitError && (
          <div className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-2 text-sm text-destructive">
            保存失败：
            {submitError instanceof Error ? submitError.message : '未知错误'}
          </div>
        )}

        <div className="flex items-center gap-2 border-t pt-4">
          <Button disabled={submitting} onClick={handleSubmit}>
            {submitting ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Save className="size-4" />
            )}
            保存
          </Button>
          <Button
            variant="outline"
            disabled={submitting}
            onClick={() => navigate('/notifications/templates')}
          >
            取消
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function Field({
  label,
  required,
  error,
  hint,
  children,
}: {
  label: string
  required?: boolean
  error?: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>
        {label}
        {required && <span className="ml-0.5 text-destructive">*</span>}
      </Label>
      {children}
      {hint && !error && (
        <span className="text-xs text-muted-foreground">{hint}</span>
      )}
      {error && <span className="text-xs text-destructive">{error}</span>}
    </div>
  )
}
