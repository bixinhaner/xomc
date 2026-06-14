import { useState } from 'react'
import { Pencil, Plug, Plus, RefreshCcw, Trash2, X } from 'lucide-react'

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
  useFTPConfigs,
  useCreateFTPConfig,
  useUpdateFTPConfig,
  useDeleteFTPConfigs,
  useTestFTPConnection,
} from '@core/hooks/api/useBackup'
import type { FTPConfig as FTPConfigModel } from '@core/mock/data/backup'

// ============================================================
// FTP 配置 — 对齐 v1 webcode/src/pages/backup/FTPConfig（路由 /backup/ftp）
//   · FTP/SFTP/FTPS 服务器配置列表 + 启用开关 + 删除
//   · 创建/编辑：名称 / 协议 / 主机 / 端口 / 账号 / 密码 / 远端路径 / 被动模式
//   · 连接测试（useTestFTPConnection）
// 全部数据走 @core hooks（真实后端），三态完整。
// ============================================================

type Protocol = FTPConfigModel['protocol']

const PROTOCOL_VARIANT: Record<Protocol, 'success' | 'warning' | 'secondary'> = {
  FTP: 'success',
  SFTP: 'warning',
  FTPS: 'secondary',
}

const DEFAULT_PORTS: Record<Protocol, number> = {
  FTP: 21,
  SFTP: 22,
  FTPS: 990,
}

interface FormState {
  configName: string
  protocol: Protocol
  host: string
  port: number
  username: string
  password: string
  remotePath: string
  passive: boolean
  enabled: boolean
}

const EMPTY_FORM: FormState = {
  configName: '',
  protocol: 'FTP',
  host: '',
  port: 21,
  username: '',
  password: '',
  remotePath: '/backup/omc',
  passive: true,
  enabled: true,
}

function getErrMsg(e: unknown): string {
  return e instanceof Error ? e.message : '操作失败'
}

export default function FTPConfig() {
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [editing, setEditing] = useState<FTPConfigModel | 'new' | null>(null)
  const [form, setForm] = useState<FormState>(EMPTY_FORM)
  const [formError, setFormError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState<FTPConfigModel | null>(null)
  const [toast, setToast] = useState<{ kind: 'ok' | 'err'; msg: string } | null>(null)

  const { data, isLoading, isError, error, isFetching, refetch } = useFTPConfigs({
    page,
    pageSize,
  })
  const createConfig = useCreateFTPConfig()
  const updateConfig = useUpdateFTPConfig()
  const deleteConfigs = useDeleteFTPConfigs()
  const testConnection = useTestFTPConnection()

  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['名称', '协议', '主机', '端口', '账号', '远端路径', '被动模式', '状态', '创建时间', '操作']

  const flash = (kind: 'ok' | 'err', msg: string) => {
    setToast({ kind, msg })
    window.setTimeout(() => setToast(null), 3000)
  }

  const openCreate = () => {
    setForm(EMPTY_FORM)
    setFormError('')
    setEditing('new')
  }

  const openEdit = (c: FTPConfigModel) => {
    setForm({
      configName: c.configName,
      protocol: c.protocol,
      host: c.host,
      port: c.port,
      username: c.username,
      password: '',
      remotePath: c.remotePath,
      passive: c.passive,
      enabled: c.enabled,
    })
    setFormError('')
    setEditing(c)
  }

  const closeModal = () => {
    setEditing(null)
    setFormError('')
  }

  const mutating = createConfig.isPending || updateConfig.isPending

  const handleSave = () => {
    if (!form.configName.trim()) return setFormError('请输入配置名称')
    if (!form.host.trim()) return setFormError('请输入主机地址')
    if (!form.username.trim()) return setFormError('请输入账号')
    if (editing === 'new' && !form.password.trim())
      return setFormError('请输入密码')
    if (!form.remotePath.trim()) return setFormError('请输入远端路径')
    setFormError('')

    const base = {
      configName: form.configName.trim(),
      protocol: form.protocol,
      host: form.host.trim(),
      port: form.port,
      username: form.username.trim(),
      remotePath: form.remotePath.trim(),
      passive: form.passive,
      enabled: form.enabled,
    }

    if (editing && editing !== 'new') {
      const patch: Partial<FTPConfigModel> & { password?: string } = { ...base }
      if (form.password.trim()) patch.password = form.password
      updateConfig.mutate(
        { id: editing.id, data: patch },
        {
          onSuccess: () => {
            flash('ok', '配置已更新')
            closeModal()
          },
          onError: (e: unknown) => setFormError(getErrMsg(e)),
        }
      )
    } else {
      const payload = { ...base, password: form.password } as Omit<
        FTPConfigModel,
        'id' | 'createTime'
      >
      createConfig.mutate(payload, {
        onSuccess: () => {
          flash('ok', '配置已创建')
          closeModal()
        },
        onError: (e: unknown) => setFormError(getErrMsg(e)),
      })
    }
  }

  const handleToggleEnabled = (c: FTPConfigModel) => {
    updateConfig.mutate(
      { id: c.id, data: { enabled: !c.enabled } },
      { onError: (e: unknown) => flash('err', getErrMsg(e)) }
    )
  }

  const handleTest = (c: FTPConfigModel) => {
    testConnection.mutate(c.id, {
      onSuccess: (result) => {
        const r = result as { success?: boolean; message?: string }
        if (r?.success === false) {
          flash('err', `连接失败${r.message ? `：${r.message}` : ''}`)
        } else {
          flash('ok', `「${c.configName}」连接正常`)
        }
      },
      onError: (e: unknown) => flash('err', `连接测试失败：${getErrMsg(e)}`),
    })
  }

  const handleDelete = (c: FTPConfigModel) => {
    deleteConfigs.mutate([c.id], {
      onSuccess: () => {
        flash('ok', `已删除配置「${c.configName}」`)
        setConfirmDelete(null)
      },
      onError: (e: unknown) => flash('err', getErrMsg(e)),
    })
  }

  return (
    <PageShell
      title="FTP 配置"
      description="备份文件外传的 FTP / SFTP / FTPS 服务器配置"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
          <Button size="sm" className="ml-auto" onClick={openCreate}>
            <Plus /> 新建配置
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
              <EmptyRow colSpan={cols.length}>暂无 FTP 配置</EmptyRow>
            ) : (
              rows.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-medium">{c.configName}</TableCell>
                  <TableCell>
                    <Badge variant={PROTOCOL_VARIANT[c.protocol]}>{c.protocol}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{c.host}</TableCell>
                  <TableCell className="text-xs tabular-nums text-muted-foreground">
                    {c.port}
                  </TableCell>
                  <TableCell className="text-xs">{c.username}</TableCell>
                  <TableCell
                    className="max-w-[12rem] truncate font-mono text-xs text-muted-foreground"
                    title={c.remotePath}
                  >
                    {c.remotePath}
                  </TableCell>
                  <TableCell>
                    <Badge variant={c.passive ? 'secondary' : 'muted'}>
                      {c.passive ? '被动' : '主动'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <button
                      type="button"
                      onClick={() => handleToggleEnabled(c)}
                      disabled={updateConfig.isPending}
                      className="cursor-pointer"
                    >
                      <Badge variant={c.enabled ? 'success' : 'muted'}>
                        {c.enabled ? '已启用' : '已停用'}
                      </Badge>
                    </button>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(c.createTime)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={testConnection.isPending}
                        onClick={() => handleTest(c)}
                      >
                        <Plug className="size-4" /> 测试
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => openEdit(c)}>
                        <Pencil className="size-4" /> 编辑
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive"
                        onClick={() => setConfirmDelete(c)}
                      >
                        <Trash2 className="size-4" /> 删除
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
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
                {editing === 'new' ? '新建 FTP 配置' : '编辑 FTP 配置'}
              </h2>
              <Button variant="ghost" size="sm" onClick={closeModal}>
                <X className="size-4" />
              </Button>
            </div>

            <div className="max-h-[70vh] space-y-4 overflow-auto px-5 py-4">
              <div className="space-y-1.5">
                <Label htmlFor="ftp-name">配置名称</Label>
                <Input
                  id="ftp-name"
                  value={form.configName}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, configName: e.target.value }))
                  }
                  placeholder="例如：主 FTP 服务器"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label>协议</Label>
                  <Select
                    value={form.protocol}
                    onValueChange={(v) => {
                      const p = v as Protocol
                      setForm((f) => ({ ...f, protocol: p, port: DEFAULT_PORTS[p] }))
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="FTP">FTP (21)</SelectItem>
                      <SelectItem value="SFTP">SFTP (22)</SelectItem>
                      <SelectItem value="FTPS">FTPS (990)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="ftp-port">端口</Label>
                  <Input
                    id="ftp-port"
                    type="number"
                    min={1}
                    max={65535}
                    value={form.port}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, port: Number(e.target.value) || 0 }))
                    }
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="ftp-host">主机地址</Label>
                <Input
                  id="ftp-host"
                  value={form.host}
                  onChange={(e) => setForm((f) => ({ ...f, host: e.target.value }))}
                  placeholder="192.168.1.10"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="ftp-user">账号</Label>
                  <Input
                    id="ftp-user"
                    value={form.username}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, username: e.target.value }))
                    }
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="ftp-pass">
                    密码{editing !== 'new' ? '（留空不修改）' : ''}
                  </Label>
                  <Input
                    id="ftp-pass"
                    type="password"
                    value={form.password}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, password: e.target.value }))
                    }
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="ftp-path">远端路径</Label>
                <Input
                  id="ftp-path"
                  className="font-mono"
                  value={form.remotePath}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, remotePath: e.target.value }))
                  }
                  placeholder="/backup/omc"
                />
              </div>

              <div className="flex items-center gap-6">
                <div className="flex items-center gap-2">
                  <Label>被动模式</Label>
                  <button
                    type="button"
                    onClick={() => setForm((f) => ({ ...f, passive: !f.passive }))}
                  >
                    <Badge variant={form.passive ? 'secondary' : 'muted'}>
                      {form.passive ? '开' : '关'}
                    </Badge>
                  </button>
                </div>
                <div className="flex items-center gap-2">
                  <Label>启用</Label>
                  <button
                    type="button"
                    onClick={() => setForm((f) => ({ ...f, enabled: !f.enabled }))}
                  >
                    <Badge variant={form.enabled ? 'success' : 'muted'}>
                      {form.enabled ? '已启用' : '已停用'}
                    </Badge>
                  </button>
                </div>
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
              <h2 className="text-base font-semibold">删除 FTP 配置</h2>
              <p className="mt-2 text-sm text-muted-foreground">
                确认删除配置「{confirmDelete.configName}」？此操作不可恢复。
              </p>
            </div>
            <div className="flex justify-end gap-2 border-t px-5 py-3">
              <Button variant="outline" size="sm" onClick={() => setConfirmDelete(null)}>
                取消
              </Button>
              <Button
                variant="destructive"
                size="sm"
                disabled={deleteConfigs.isPending}
                onClick={() => handleDelete(confirmDelete)}
              >
                {deleteConfigs.isPending ? '处理中…' : '确认删除'}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </PageShell>
  )
}
