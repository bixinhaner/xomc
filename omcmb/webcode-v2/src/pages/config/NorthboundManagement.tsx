import { useState } from 'react'
import { Loader2, Plus, RefreshCcw, Trash2, UploadCloud, X } from 'lucide-react'

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
  TableCard,
} from '@/components/layout/PageShell'

import {
  usePushTargets,
  useAddPushTarget,
  useRemovePushTarget,
  useFullSync,
} from '@core/hooks/api/useNorthbound'
import type { PushTarget } from '@core/services/api/northboundApi'

// ============================================================
// 北向/OSS 管理 — 对照 v1 config/NorthboundManagement。
// 真实数据：usePushTargets 推送目标；增删 + 全量同步（useFullSync）。
// ============================================================

const SYNC_TYPES = ['device', 'alarm', 'pm', 'config']
const DATA_TYPES = ['alarm', 'pm', 'config', 'device']

export default function NorthboundManagement() {
  const { data, isLoading, isError, error, isFetching, refetch } = usePushTargets()
  const addTarget = useAddPushTarget()
  const removeTarget = useRemovePushTarget()
  const fullSync = useFullSync()

  const targets: PushTarget[] = data ?? []

  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState({ id: '', url: '', dataTypes: ['device'] as string[] })
  const [addErr, setAddErr] = useState<string | null>(null)

  const [syncType, setSyncType] = useState('device')
  const [syncMsg, setSyncMsg] = useState<string | null>(null)

  const cols = ['ID', 'URL', '认证', '数据类型', '格式', '批量', '重试', '启用', '操作']

  const submitAdd = () => {
    if (!form.id.trim() || !form.url.trim()) {
      setAddErr('ID 和 URL 必填')
      return
    }
    setAddErr(null)
    addTarget.mutate(
      {
        id: form.id.trim(),
        url: form.url.trim(),
        dataTypes: form.dataTypes,
        enabled: true,
      },
      {
        onSuccess: () => {
          setAdding(false)
          setForm({ id: '', url: '', dataTypes: ['device'] })
        },
        onError: (e: Error) => setAddErr(e.message || '添加失败'),
      }
    )
  }

  const handleSync = () => {
    setSyncMsg(null)
    fullSync.mutate(syncType, {
      onSuccess: (res) => setSyncMsg(`全量同步完成：${res.total} 条 ${res.dataType}`),
      onError: (e: Error) => setSyncMsg(`同步失败：${e.message || '未知错误'}`),
    })
  }

  return (
    <PageShell
      title="北向 / OSS 管理"
      description="推送目标管理与数据同步（F08）"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select value={syncType} onValueChange={setSyncType}>
            <SelectTrigger className="w-36">
              <SelectValue placeholder="同步类型" />
            </SelectTrigger>
            <SelectContent>
              {SYNC_TYPES.map((t) => (
                <SelectItem key={t} value={t}>
                  {t}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button size="sm" disabled={fullSync.isPending} onClick={handleSync}>
            {fullSync.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <UploadCloud className="size-4" />
            )}
            全量同步
          </Button>
          <Button variant="outline" size="sm" onClick={() => setAdding((v) => !v)}>
            <Plus className="size-4" /> 新增推送目标
          </Button>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => void refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
      {syncMsg && (
        <div className="mb-3 rounded-md border bg-muted/40 px-4 py-2 text-sm">{syncMsg}</div>
      )}

      {adding && (
        <div className="mb-4 rounded-lg border bg-card p-4">
          <div className="mb-3 flex items-center justify-between">
            <div className="font-medium">新增推送目标</div>
            <Button variant="ghost" size="sm" onClick={() => setAdding(false)}>
              <X className="size-4" />
            </Button>
          </div>
          {addErr && (
            <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {addErr}
            </div>
          )}
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div>
              <Label className="mb-1 block text-xs">目标 ID</Label>
              <Input
                value={form.id}
                onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))}
                placeholder="oss-primary"
              />
            </div>
            <div>
              <Label className="mb-1 block text-xs">URL</Label>
              <Input
                value={form.url}
                onChange={(e) => setForm((f) => ({ ...f, url: e.target.value }))}
                placeholder="https://oss.example.com/api/v1/push"
              />
            </div>
            <div>
              <Label className="mb-1 block text-xs">数据类型</Label>
              <div className="flex flex-wrap gap-1.5">
                {DATA_TYPES.map((t) => {
                  const checked = form.dataTypes.includes(t)
                  return (
                    <button
                      key={t}
                      type="button"
                      onClick={() =>
                        setForm((f) => ({
                          ...f,
                          dataTypes: checked
                            ? f.dataTypes.filter((x) => x !== t)
                            : [...f.dataTypes, t],
                        }))
                      }
                      className="rounded-md border px-2 py-1 text-xs"
                    >
                      <Badge variant={checked ? 'default' : 'muted'}>{t}</Badge>
                    </button>
                  )
                })}
              </div>
            </div>
          </div>
          <div className="mt-4 flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setAdding(false)}>
              取消
            </Button>
            <Button
              size="sm"
              disabled={addTarget.isPending || form.dataTypes.length === 0}
              onClick={submitAdd}
            >
              {addTarget.isPending ? <Loader2 className="size-4 animate-spin" /> : null}
              保存
            </Button>
          </div>
        </div>
      )}

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
            ) : targets.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无推送目标</EmptyRow>
            ) : (
              targets.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="font-mono text-xs">{t.id}</TableCell>
                  <TableCell className="max-w-xs truncate font-mono text-xs">{t.url}</TableCell>
                  <TableCell>
                    <Badge variant="muted">{t.authType}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {t.dataTypes.map((dt) => (
                        <Badge key={dt} variant="secondary">
                          {dt}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs uppercase">{t.format}</TableCell>
                  <TableCell className="tabular-nums">{t.batchSize}</TableCell>
                  <TableCell className="tabular-nums">{t.retryCount}</TableCell>
                  <TableCell>
                    {t.enabled ? (
                      <Badge variant="success">启用</Badge>
                    ) : (
                      <Badge variant="muted">停用</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={removeTarget.isPending}
                      onClick={() => removeTarget.mutate(t.id)}
                    >
                      <Trash2 className="size-3.5 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
  )
}
