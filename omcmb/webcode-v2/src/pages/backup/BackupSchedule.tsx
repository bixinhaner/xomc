import { useMemo, useState } from 'react'
import { Pencil, Plus, RefreshCcw, Trash2, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useBackupSchedules,
  useCreateBackupSchedule,
  useUpdateBackupSchedule,
  useDeleteBackupSchedules,
} from '@core/hooks/api/useBackup'
import { useAllDeviceGroups } from '@core/hooks/api/useSystem'
import type { BackupSchedule } from '@core/mock/data/backup'

// ============================================================
// 备份调度 — 对齐 v1 webcode/src/pages/backup/BackupSchedule（路由 /backup/schedule）
//   · cron 调度任务列表 + 启用开关 + 删除
//   · 创建/编辑：名称 / cron 预设 / 自定义 cron / 备份方式 / 设备分组 / 启用
// 全部数据走 @core hooks（真实后端），三态完整。
// ============================================================

type BackupType = BackupSchedule['backupType']

const BACKUP_TYPE_LABEL: Record<BackupType, string> = {
  full: '全量',
  incremental: '增量',
  'config-only': '配置',
}

const BACKUP_TYPE_VARIANT: Record<BackupType, 'success' | 'warning' | 'secondary'> = {
  full: 'success',
  incremental: 'warning',
  'config-only': 'secondary',
}

interface CronPreset {
  key: string
  cron: string
  label: string
}

const CRON_PRESETS: readonly CronPreset[] = [
  { key: 'daily-midnight', cron: '0 0 * * *', label: '每天 00:00' },
  { key: 'weekly-mon', cron: '0 0 * * 1', label: '每周一 00:00' },
  { key: 'monthly-1st', cron: '0 0 1 * *', label: '每月 1 号 00:00' },
  { key: 'every-6h', cron: '0 */6 * * *', label: '每 6 小时' },
  { key: 'custom', cron: '', label: '自定义' },
] as const

function isValidCron(s: string): boolean {
  const parts = s.trim().split(/\s+/)
  return parts.length === 5 && parts.every((p) => p.length > 0)
}

function presetForCron(cron: string): string {
  const hit = CRON_PRESETS.find((p) => p.cron === cron)
  return hit ? hit.key : 'custom'
}

interface FormState {
  scheduleName: string
  cronPreset: string
  cronExpression: string
  backupType: BackupType
  deviceGroups: string[]
  enabled: boolean
}

const EMPTY_FORM: FormState = {
  scheduleName: '',
  cronPreset: 'daily-midnight',
  cronExpression: '0 0 * * *',
  backupType: 'full',
  deviceGroups: [],
  enabled: true,
}

function getErrMsg(e: unknown): string {
  return e instanceof Error ? e.message : '操作失败'
}

export default function BackupSchedule() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [editing, setEditing] = useState<BackupSchedule | 'new' | null>(null)
  const [form, setForm] = useState<FormState>(EMPTY_FORM)
  const [formError, setFormError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState<BackupSchedule | null>(null)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(null)

  const { data, isLoading, isError, error, isFetching, refetch } =
    useBackupSchedules({ page, pageSize })
  const { data: groups } = useAllDeviceGroups()
  const createSchedule = useCreateBackupSchedule()
  const updateSchedule = useUpdateBackupSchedule()
  const deleteSchedules = useDeleteBackupSchedules()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const groupOptions = groups ?? []
  const groupNameById = useMemo(() => {
    const m = new Map<string, string>()
    for (const g of groups ?? []) m.set(g.id, g.name)
    return m
  }, [groups])

  const cols = ['名称', 'Cron 表达式', '备份方式', '设备分组', '状态', '创建时间', '操作']

  const flash = (kind: 'ok' | 'err', msg: string) => {
    setToast({ kind, msg })
    window.setTimeout(() => setToast(null), 3000)
  }

  const openCreate = () => {
    setForm(EMPTY_FORM)
    setFormError('')
    setEditing('new')
  }

  const openEdit = (s: BackupSchedule) => {
    setForm({
      scheduleName: s.scheduleName,
      cronPreset: presetForCron(s.cronExpression),
      cronExpression: s.cronExpression,
      backupType: s.backupType,
      deviceGroups: s.deviceGroups ?? [],
      enabled: s.enabled,
    })
    setFormError('')
    setEditing(s)
  }

  const closeModal = () => {
    setEditing(null)
    setFormError('')
  }

  const mutating = createSchedule.isPending || updateSchedule.isPending

  const handleSave = () => {
    if (!form.scheduleName.trim()) {
      setFormError('请输入调度名称')
      return
    }
    if (!isValidCron(form.cronExpression)) {
      setFormError('Cron 表达式需为标准 5 段格式')
      return
    }
    if (form.deviceGroups.length === 0) {
      setFormError('请至少选择一个设备分组')
      return
    }
    setFormError('')

    const payload: Omit<BackupSchedule, 'id' | 'createTime'> = {
      scheduleName: form.scheduleName.trim(),
      cronExpression: form.cronExpression,
      cronDescription: '',
      enabled: form.enabled,
      backupType: form.backupType,
      deviceGroups: form.deviceGroups,
      retentionDays: 0,
      storageLocation: '',
      nextRunTime: '',
      creator: '',
    }

    if (editing && editing !== 'new') {
      updateSchedule.mutate(
        { id: editing.id, data: payload },
        {
          onSuccess: () => {
            flash('ok', '调度已更新')
            closeModal()
          },
          onError: (e: unknown) => setFormError(getErrMsg(e)),
        }
      )
    } else {
      createSchedule.mutate(payload, {
        onSuccess: () => {
          flash('ok', '调度已创建')
          closeModal()
        },
        onError: (e: unknown) => setFormError(getErrMsg(e)),
      })
    }
  }

  const handleToggleEnabled = (s: BackupSchedule) => {
    updateSchedule.mutate(
      { id: s.id, data: { enabled: !s.enabled } },
      { onError: (e: unknown) => flash('err', getErrMsg(e)) }
    )
  }

  const handleDelete = (s: BackupSchedule) => {
    deleteSchedules.mutate([s.id], {
      onSuccess: () => {
        flash('ok', `已删除调度「${s.scheduleName}」`)
        setConfirmDelete(null)
      },
      onError: (e: unknown) => flash('err', getErrMsg(e)),
    })
  }

  return (
    <PageShell
      title="备份调度"
      description="基于 Cron 的定时备份任务管理"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
          <Button size="sm" className="ml-auto" onClick={openCreate}>
            <Plus /> 新建调度
          </Button>
        </div>
      }
    >
      {toast ? (
        <div
          className={cn(
            'mb-3 rounded-md px-3 py-2 text-sm',
            toast.kind === 'ok'
              ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
              : 'bg-destructive/10 text-destructive'
          )}
        >
          {toast.msg}
        </div>
      ) : null}

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无备份调度</EmptyRow>
            ) : (
              rows.map((s) => {
                const groupNames = (s.deviceGroups ?? [])
                  .map((id) => groupNameById.get(id) ?? id)
                  .join('、')
                return (
                  <TableRow key={s.id}>
                    <TableCell className="font-medium">{s.scheduleName}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {s.cronExpression}
                    </TableCell>
                    <TableCell>
                      <Badge variant={BACKUP_TYPE_VARIANT[s.backupType]}>
                        {BACKUP_TYPE_LABEL[s.backupType] ?? s.backupType}
                      </Badge>
                    </TableCell>
                    <TableCell
                      className="max-w-[16rem] truncate text-xs text-muted-foreground"
                      title={groupNames}
                    >
                      {(s.deviceGroups ?? []).length > 0
                        ? `${s.deviceGroups.length} 个分组`
                        : '—'}
                    </TableCell>
                    <TableCell>
                      <button
                        type="button"
                        onClick={() => handleToggleEnabled(s)}
                        disabled={updateSchedule.isPending}
                        className="cursor-pointer"
                      >
                        <Badge variant={s.enabled ? 'success' : 'muted'}>
                          {s.enabled ? '已启用' : '已停用'}
                        </Badge>
                      </button>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(s.createTime)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button variant="ghost" size="sm" onClick={() => openEdit(s)}>
                          <Pencil className="size-4" /> 编辑
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive"
                          onClick={() => setConfirmDelete(s)}
                        >
                          <Trash2 className="size-4" /> 删除
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={pageSize}
        onChange={setPage}
      />

      {/* 创建/编辑弹窗 */}
      {editing ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/40" onClick={closeModal} aria-hidden />
          <div className="relative z-10 w-full max-w-lg rounded-xl border bg-card shadow-lg">
            <div className="flex items-center justify-between border-b px-5 py-3">
              <h2 className="text-base font-semibold">
                {editing === 'new' ? '新建备份调度' : '编辑备份调度'}
              </h2>
              <Button variant="ghost" size="sm" onClick={closeModal}>
                <X className="size-4" />
              </Button>
            </div>

            <div className="max-h-[70vh] space-y-4 overflow-auto px-5 py-4">
              <div className="space-y-1.5">
                <Label htmlFor="sch-name">调度名称</Label>
                <Input
                  id="sch-name"
                  value={form.scheduleName}
                  maxLength={128}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, scheduleName: e.target.value }))
                  }
                  placeholder="例如：每日全量备份"
                />
              </div>

              <div className="space-y-1.5">
                <Label>Cron 预设</Label>
                <Select
                  value={form.cronPreset}
                  onValueChange={(key) => {
                    const preset = CRON_PRESETS.find((p) => p.key === key)
                    setForm((f) => ({
                      ...f,
                      cronPreset: key,
                      cronExpression:
                        preset && preset.cron ? preset.cron : f.cronExpression,
                    }))
                  }}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CRON_PRESETS.map((p) => (
                      <SelectItem key={p.key} value={p.key}>
                        {p.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="sch-cron">Cron 表达式</Label>
                <Input
                  id="sch-cron"
                  className="font-mono"
                  value={form.cronExpression}
                  disabled={form.cronPreset !== 'custom'}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, cronExpression: e.target.value }))
                  }
                  placeholder="* * * * *"
                />
                {form.cronPreset === 'custom' ? (
                  <p className="text-xs text-muted-foreground">
                    标准 5 段：分 时 日 月 周
                  </p>
                ) : null}
              </div>

              <div className="space-y-1.5">
                <Label>备份方式</Label>
                <Select
                  value={form.backupType}
                  onValueChange={(v) =>
                    setForm((f) => ({ ...f, backupType: v as BackupType }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="full">全量</SelectItem>
                    <SelectItem value="incremental">增量</SelectItem>
                    <SelectItem value="config-only">配置</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1.5">
                <Label>设备分组</Label>
                <div className="flex max-h-40 flex-wrap gap-1.5 overflow-auto rounded-md border p-2">
                  {groupOptions.length === 0 ? (
                    <span className="text-xs text-muted-foreground">暂无可选分组</span>
                  ) : (
                    groupOptions.map((g) => {
                      const selected = form.deviceGroups.includes(g.id)
                      return (
                        <button
                          key={g.id}
                          type="button"
                          onClick={() =>
                            setForm((f) => ({
                              ...f,
                              deviceGroups: selected
                                ? f.deviceGroups.filter((id) => id !== g.id)
                                : [...f.deviceGroups, g.id],
                            }))
                          }
                        >
                          <Badge variant={selected ? 'default' : 'outline'}>
                            {g.name}
                          </Badge>
                        </button>
                      )
                    })
                  )}
                </div>
                <p className="text-xs text-muted-foreground">
                  已选 {form.deviceGroups.length} 个分组
                </p>
              </div>

              <div className="flex items-center gap-2">
                <Label>启用调度</Label>
                <button
                  type="button"
                  onClick={() => setForm((f) => ({ ...f, enabled: !f.enabled }))}
                >
                  <Badge variant={form.enabled ? 'success' : 'muted'}>
                    {form.enabled ? '已启用' : '已停用'}
                  </Badge>
                </button>
              </div>

              {formError ? (
                <p className="rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">
                  {formError}
                </p>
              ) : null}
            </div>

            <div className="flex justify-end gap-2 border-t px-5 py-3">
              <Button variant="outline" size="sm" onClick={closeModal}>
                取消
              </Button>
              <Button size="sm" disabled={mutating} onClick={handleSave}>
                {mutating ? '保存中…' : '保存'}
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      {/* 删除确认 */}
      {confirmDelete ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => setConfirmDelete(null)}
            aria-hidden
          />
          <div className="relative z-10 w-full max-w-sm rounded-xl border bg-card shadow-lg">
            <div className="px-5 py-4">
              <h2 className="text-base font-semibold">删除备份调度</h2>
              <p className="mt-2 text-sm text-muted-foreground">
                确认删除调度「{confirmDelete.scheduleName}」？此操作不可恢复。
              </p>
            </div>
            <div className="flex justify-end gap-2 border-t px-5 py-3">
              <Button variant="outline" size="sm" onClick={() => setConfirmDelete(null)}>
                取消
              </Button>
              <Button
                variant="destructive"
                size="sm"
                disabled={deleteSchedules.isPending}
                onClick={() => handleDelete(confirmDelete)}
              >
                {deleteSchedules.isPending ? '处理中…' : '确认删除'}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </PageShell>
  )
}
