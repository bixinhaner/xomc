import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CheckCircle2, RefreshCcw, Send, X, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
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

import { useConfigTemplates } from '@core/hooks/api/useConfig'
import { useDispatchTemplate } from '@core/hooks/api/useTemplate'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { ConfigTemplate } from '@core/types/config'
import type { DispatchTemplateResponse } from '@core/services/api/templateApi'

// ============================================================
// 模板批量配置 — 对照 v1 config/BatchParamTemplate。
// 真实数据：配置模板列表（useConfigTemplates）；行内「下发」走
// dispatch API（Path A）；模板名 → 详情子路由 /config/batch-template/:id。
// ============================================================

export default function BatchParamTemplate() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const pageSize = 20

  const [target, setTarget] = useState<ConfigTemplate | null>(null)
  const [deviceIds, setDeviceIds] = useState<string[]>([])
  const [result, setResult] = useState<DispatchTemplateResponse | null>(null)
  const [err, setErr] = useState<string | null>(null)

  const { data, isLoading, isError, error, isFetching, refetch } = useConfigTemplates({
    page,
    pageSize,
  })
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const cols = ['模板名称', '参数数量', '创建人', '创建时间', '操作']

  const openDispatch = (t: ConfigTemplate) => {
    setTarget(t)
    setDeviceIds([])
    setResult(null)
    setErr(null)
  }
  const closeDispatch = () => {
    setTarget(null)
    setDeviceIds([])
    setResult(null)
    setErr(null)
  }

  return (
    <PageShell
      title="模板批量配置"
      description="配置模板管理 — 选模板下发到设备（Path A）"
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" className="ml-auto" onClick={() => void refetch()}>
          <RefreshCcw className="size-4" /> 刷新
        </Button>
      }
    >
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
              <EmptyRow colSpan={cols.length}>暂无配置模板</EmptyRow>
            ) : (
              rows.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left font-medium text-primary hover:underline"
                      onClick={() => navigate(`/config/batch-template`)}
                    >
                      {t.templateName}
                    </button>
                    <div className="line-clamp-1 text-xs text-muted-foreground">
                      {t.description || '—'}
                    </div>
                  </TableCell>
                  <TableCell className="tabular-nums">{t.params?.length ?? 0}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{t.creator || '—'}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.createTime)}
                  </TableCell>
                  <TableCell>
                    <Button variant="outline" size="sm" onClick={() => openDispatch(t)}>
                      <Send className="size-3.5" /> 下发
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={setPage} />

      {target && (
        <DispatchModal
          target={target}
          deviceIds={deviceIds}
          onDeviceIdsChange={setDeviceIds}
          result={result}
          error={err}
          onResult={setResult}
          onError={setErr}
          onClose={closeDispatch}
        />
      )}
    </PageShell>
  )
}

function DispatchModal({
  target,
  deviceIds,
  onDeviceIdsChange,
  result,
  error,
  onResult,
  onError,
  onClose,
}: {
  target: ConfigTemplate
  deviceIds: string[]
  onDeviceIdsChange: (ids: string[]) => void
  result: DispatchTemplateResponse | null
  error: string | null
  onResult: (r: DispatchTemplateResponse) => void
  onError: (e: string | null) => void
  onClose: () => void
}) {
  const dispatch = useDispatchTemplate()
  const { data: devicePage, isLoading: devLoading } = useDeviceList({ page: 1, pageSize: 200 })
  const devices = devicePage?.items ?? []

  const toggleDevice = (id: string) => {
    onDeviceIdsChange(
      deviceIds.includes(id) ? deviceIds.filter((d) => d !== id) : [...deviceIds, id]
    )
  }

  const handleDispatch = () => {
    if (deviceIds.length === 0) return
    onError(null)
    dispatch.mutate(
      { templateId: target.id, deviceIds },
      {
        onSuccess: (resp) => onResult(resp),
        onError: (e: Error) => onError(e.message || '下发失败'),
      }
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="flex max-h-[85vh] w-full max-w-2xl flex-col overflow-hidden rounded-lg border bg-card shadow-lg">
        <div className="flex items-center justify-between border-b px-5 py-3">
          <div className="font-semibold">下发模板：{target.templateName}</div>
          <Button variant="ghost" size="sm" onClick={onClose}>
            <X className="size-4" />
          </Button>
        </div>

        <div className="flex-1 overflow-auto p-5">
          {result ? (
            <DispatchResultPanel result={result} />
          ) : (
            <div className="space-y-4">
              {error && (
                <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  下发失败：{error}
                </div>
              )}
              <div>
                <Label className="mb-2 block">目标设备（已选 {deviceIds.length} 台）</Label>
                {devLoading ? (
                  <div className="py-8 text-center text-sm text-muted-foreground">加载设备列表…</div>
                ) : devices.length === 0 ? (
                  <div className="py-8 text-center text-sm text-muted-foreground">暂无可选设备</div>
                ) : (
                  <div className="max-h-64 overflow-auto rounded-md border">
                    {devices.map((d) => {
                      const checked = deviceIds.includes(d.id)
                      return (
                        <button
                          key={d.id}
                          type="button"
                          onClick={() => toggleDevice(d.id)}
                          className={cn(
                            'flex w-full items-center justify-between border-b px-3 py-2 text-left text-sm last:border-b-0 hover:bg-muted/50',
                            checked && 'bg-primary/5'
                          )}
                        >
                          <span className="font-mono text-xs">
                            {d.sn}
                            {d.name && d.name !== d.sn ? (
                              <span className="ml-2 text-muted-foreground">{d.name}</span>
                            ) : null}
                          </span>
                          {checked && <CheckCircle2 className="size-4 text-primary" />}
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
              <p className="text-xs text-muted-foreground">
                说明：本次下发将强制走 Path A（模板 standardPath → privatePath 翻译 →
                SetParameterValues），不受全局 auto_configure 开关影响。每台设备独立成败。
              </p>
            </div>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 border-t px-5 py-3">
          {result ? (
            <Button onClick={onClose}>关闭</Button>
          ) : (
            <>
              <Button variant="outline" onClick={onClose}>
                取消
              </Button>
              <Button onClick={handleDispatch} disabled={deviceIds.length === 0 || dispatch.isPending}>
                <Send className="size-4" />
                {dispatch.isPending ? '下发中…' : `确认下发到 ${deviceIds.length} 台`}
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function DispatchResultPanel({ result }: { result: DispatchTemplateResponse }) {
  const rows = [
    ...result.dispatched.map((r) => ({ ...r, ok: true, key: `s-${r.deviceId}` })),
    ...result.failed.map((r) => ({ ...r, ok: false, key: `f-${r.deviceId}` })),
  ]
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Badge variant="secondary">总计 {result.totalDevices}</Badge>
        <Badge variant="success">成功 {result.dispatched.length}</Badge>
        <Badge variant={result.failed.length > 0 ? 'destructive' : 'muted'}>
          失败 {result.failed.length}
        </Badge>
      </div>
      <div className="overflow-hidden rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">状态</TableHead>
              <TableHead>设备 ID</TableHead>
              <TableHead>Task ID / 错误</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((r) => (
              <TableRow key={r.key}>
                <TableCell>
                  {r.ok ? (
                    <CheckCircle2 className="size-4 text-emerald-600" />
                  ) : (
                    <XCircle className="size-4 text-destructive" />
                  )}
                </TableCell>
                <TableCell className="font-mono text-xs">{r.deviceId}</TableCell>
                <TableCell className="text-xs">
                  {r.ok ? (
                    <code className="text-muted-foreground">{r.taskId}</code>
                  ) : (
                    <span className="text-destructive">{r.error}</span>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
