import { useMemo, useState } from 'react'
import {
  RefreshCcw,
  FileStack,
  GitCompare,
  ListChecks,
  Send,
  X,
  CheckCircle2,
  XCircle,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  useConfigTemplates,
  useBaselineConfigs,
  useConfigTasks,
} from '@core/hooks/api/useConfig'
import { useDispatchTemplate } from '@core/hooks/api/useTemplate'
import { useDeviceList } from '@core/hooks/api/useDevices'
import type { ConfigTemplate, ConfigTaskStatus } from '@core/types/config'
import type { DispatchTemplateResponse } from '@core/services/api/templateApi'

// ============================================================
// 配置管理（v2 皮肤）— 适配 v1 业务深度
//   三视图：配置模板 / 配置基线 / 下发任务
//   模板支持显式下发（选设备 → dispatch API → 结果面板）
// ============================================================

type ConfigTab = 'templates' | 'baselines' | 'tasks'

const TABS: { key: ConfigTab; label: string; icon: typeof FileStack }[] = [
  { key: 'templates', label: '配置模板', icon: FileStack },
  { key: 'baselines', label: '配置基线', icon: GitCompare },
  { key: 'tasks', label: '下发任务', icon: ListChecks },
]

// ---- 基线状态 ----
const BASELINE_STATUS: Record<
  string,
  { variant: 'success' | 'warning' | 'muted'; label: string }
> = {
  active: { variant: 'success', label: '生效' },
  draft: { variant: 'warning', label: '草稿' },
  deprecated: { variant: 'muted', label: '废弃' },
}

// ---- 任务状态 ----
const TASK_STATUS: Record<
  ConfigTaskStatus,
  { variant: 'success' | 'warning' | 'destructive' | 'secondary' | 'muted'; label: string }
> = {
  pending: { variant: 'secondary', label: '待执行' },
  running: { variant: 'warning', label: '执行中' },
  success: { variant: 'success', label: '成功' },
  failed: { variant: 'destructive', label: '失败' },
  partial: { variant: 'warning', label: '部分成功' },
  cancelled: { variant: 'muted', label: '已取消' },
}

const TASK_TYPE_LABEL: Record<string, string> = {
  'param-sync': '参数同步',
  'batch-config': '批量配置',
  'baseline-apply': '基线应用',
  'neighbor-update': '邻区更新',
}

export function ConfigPage() {
  const [tab, setTab] = useState<ConfigTab>('templates')

  // 各视图独立分页
  const [tplPage, setTplPage] = useState(1)
  const [blPage, setBlPage] = useState(1)
  const [taskPage, setTaskPage] = useState(1)
  const pageSize = 20

  // 基线筛选
  const [blDeviceType, setBlDeviceType] = useState('')
  const [blStatus, setBlStatus] = useState('')

  // ---- 数据查询（按当前 tab 启用，避免无谓请求）----
  const tplQuery = useConfigTemplates({ page: tplPage, pageSize })
  const blQuery = useBaselineConfigs(
    useMemo(
      () => ({
        page: blPage,
        pageSize,
        ...(blDeviceType ? { deviceType: blDeviceType } : {}),
        ...(blStatus ? { status: blStatus } : {}),
      }),
      [blPage, blStatus, blDeviceType]
    )
  )
  const taskQuery = useConfigTasks({ page: taskPage, pageSize })

  // ---- 下发 Modal 状态 ----
  const [dispatchTarget, setDispatchTarget] = useState<ConfigTemplate | null>(null)
  const [dispatchDeviceIds, setDispatchDeviceIds] = useState<string[]>([])
  const [dispatchResult, setDispatchResult] = useState<DispatchTemplateResponse | null>(null)
  const [dispatchErr, setDispatchErr] = useState<string | null>(null)

  const isFetching =
    (tab === 'templates' && tplQuery.isFetching) ||
    (tab === 'baselines' && blQuery.isFetching) ||
    (tab === 'tasks' && taskQuery.isFetching)

  const refetchCurrent = () => {
    if (tab === 'templates') void tplQuery.refetch()
    else if (tab === 'baselines') void blQuery.refetch()
    else void taskQuery.refetch()
  }

  // ---- 概览统计 ----
  const tplTotal = tplQuery.data?.total ?? 0
  const blTotal = blQuery.data?.total ?? 0
  const taskItems = taskQuery.data?.items ?? []
  const runningTasks = taskItems.filter(
    (t) => t.status === 'running' || t.status === 'pending'
  ).length
  const failedTasks = taskItems.filter(
    (t) => t.status === 'failed' || t.status === 'partial'
  ).length

  return (
    <PageShell
      title="配置管理"
      description="配置模板 · 配置基线 · 下发任务"
      isFetching={isFetching}
      toolbar={
        <Button variant="outline" size="sm" className="ml-auto" onClick={refetchCurrent}>
          <RefreshCcw className="size-4" /> 刷新
        </Button>
      }
    >
      {/* 概览统计 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="模板" value={tplTotal} />
        <Stat label="基线" value={blTotal} />
        <Stat label="进行中任务" value={runningTasks} tone="amber" />
        <Stat label="失败/部分" value={failedTasks} tone={failedTasks > 0 ? 'red' : 'default'} />
      </div>

      {/* Tab 切换 */}
      <div className="mb-4 flex flex-wrap items-center gap-1 border-b">
        {TABS.map((tabDef) => {
          const Icon = tabDef.icon
          const active = tab === tabDef.key
          return (
            <button
              key={tabDef.key}
              type="button"
              onClick={() => setTab(tabDef.key)}
              className={cn(
                'inline-flex items-center gap-1.5 border-b-2 px-3 py-2 text-sm font-medium transition-colors',
                active
                  ? 'border-primary text-foreground'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              )}
            >
              <Icon className="size-4" />
              {tabDef.label}
            </button>
          )
        })}
      </div>

      {tab === 'templates' && (
        <TemplatesView
          query={tplQuery}
          page={tplPage}
          pageSize={pageSize}
          onPageChange={setTplPage}
          onDispatch={(t) => {
            setDispatchTarget(t)
            setDispatchDeviceIds([])
            setDispatchResult(null)
            setDispatchErr(null)
          }}
        />
      )}

      {tab === 'baselines' && (
        <BaselinesView
          query={blQuery}
          page={blPage}
          pageSize={pageSize}
          onPageChange={setBlPage}
          deviceType={blDeviceType}
          status={blStatus}
          onDeviceTypeChange={(v) => {
            setBlDeviceType(v)
            setBlPage(1)
          }}
          onStatusChange={(v) => {
            setBlStatus(v)
            setBlPage(1)
          }}
        />
      )}

      {tab === 'tasks' && (
        <TasksView
          query={taskQuery}
          page={taskPage}
          pageSize={pageSize}
          onPageChange={setTaskPage}
        />
      )}

      {dispatchTarget && (
        <DispatchModal
          target={dispatchTarget}
          deviceIds={dispatchDeviceIds}
          onDeviceIdsChange={setDispatchDeviceIds}
          result={dispatchResult}
          error={dispatchErr}
          onResult={setDispatchResult}
          onError={setDispatchErr}
          onClose={() => {
            setDispatchTarget(null)
            setDispatchDeviceIds([])
            setDispatchResult(null)
            setDispatchErr(null)
          }}
        />
      )}
    </PageShell>
  )
}

// ============================================================
// 配置模板视图
// ============================================================
function TemplatesView({
  query,
  page,
  pageSize,
  onPageChange,
  onDispatch,
}: {
  query: ReturnType<typeof useConfigTemplates>
  page: number
  pageSize: number
  onPageChange: (p: number) => void
  onDispatch: (t: ConfigTemplate) => void
}) {
  const { data, isLoading, isError, error } = query
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const cols = ['模板名称', '参数数量', '创建人', '创建时间', '操作']

  return (
    <>
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
                    <div className="font-medium">{t.templateName}</div>
                    <div className="line-clamp-1 text-xs text-muted-foreground">
                      {t.description || '—'}
                    </div>
                  </TableCell>
                  <TableCell className="tabular-nums">{t.params?.length ?? 0}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {t.creator || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(t.createTime)}
                  </TableCell>
                  <TableCell>
                    <Button variant="outline" size="sm" onClick={() => onDispatch(t)}>
                      <Send className="size-3.5" /> 下发
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={onPageChange} />
    </>
  )
}

// ============================================================
// 配置基线视图
// ============================================================
function BaselinesView({
  query,
  page,
  pageSize,
  onPageChange,
  deviceType,
  status,
  onDeviceTypeChange,
  onStatusChange,
}: {
  query: ReturnType<typeof useBaselineConfigs>
  page: number
  pageSize: number
  onPageChange: (p: number) => void
  deviceType: string
  status: string
  onDeviceTypeChange: (v: string) => void
  onStatusChange: (v: string) => void
}) {
  const { data, isLoading, isError, error } = query
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const cols = ['基线名称', '设备类型', '版本', '参数数量', '状态', '创建人', '更新时间']

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Select
          value={deviceType || 'all'}
          onValueChange={(v) => onDeviceTypeChange(v === 'all' ? '' : v)}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder="设备类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            <SelectItem value="eNB">eNB (4G)</SelectItem>
            <SelectItem value="gNB">gNB (5G)</SelectItem>
            <SelectItem value="CPE">CPE</SelectItem>
          </SelectContent>
        </Select>
        <Select value={status || 'all'} onValueChange={(v) => onStatusChange(v === 'all' ? '' : v)}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="active">生效</SelectItem>
            <SelectItem value="draft">草稿</SelectItem>
            <SelectItem value="deprecated">废弃</SelectItem>
          </SelectContent>
        </Select>
      </div>

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
              <EmptyRow colSpan={cols.length}>暂无配置基线</EmptyRow>
            ) : (
              rows.map((b) => {
                const s = BASELINE_STATUS[b.status] ?? BASELINE_STATUS.draft
                return (
                  <TableRow key={b.id}>
                    <TableCell>
                      <div className="font-medium">{b.baselineName}</div>
                      <div className="line-clamp-1 text-xs text-muted-foreground">
                        {b.description || '—'}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{b.deviceType || '—'}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{b.version || '—'}</TableCell>
                    <TableCell className="tabular-nums">{b.params?.length ?? 0}</TableCell>
                    <TableCell>
                      <Badge variant={s.variant}>{s.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {b.creator || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(b.updateTime)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={onPageChange} />
    </>
  )
}

// ============================================================
// 下发任务视图
// ============================================================
function TasksView({
  query,
  page,
  pageSize,
  onPageChange,
}: {
  query: ReturnType<typeof useConfigTasks>
  page: number
  pageSize: number
  onPageChange: (p: number) => void
}) {
  const { data, isLoading, isError, error } = query
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const cols = ['任务名称', '类型', '设备数', '进度', '状态', '创建人', '创建时间']

  return (
    <>
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
              <EmptyRow colSpan={cols.length}>暂无下发任务</EmptyRow>
            ) : (
              rows.map((t) => {
                const s = TASK_STATUS[t.status] ?? TASK_STATUS.pending
                const pct = Math.max(0, Math.min(100, Math.round(t.progress ?? 0)))
                return (
                  <TableRow key={t.id}>
                    <TableCell>
                      <div className="font-medium">{t.taskName}</div>
                      {t.message && (
                        <div className="line-clamp-1 text-xs text-muted-foreground">
                          {t.message}
                        </div>
                      )}
                    </TableCell>
                    <TableCell className="text-xs">
                      {TASK_TYPE_LABEL[t.taskType] ?? t.taskType}
                    </TableCell>
                    <TableCell className="tabular-nums">{t.totalCount ?? t.deviceSns?.length ?? 0}</TableCell>
                    <TableCell className="w-44">
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
                          <div
                            className={cn(
                              'h-full rounded-full',
                              t.status === 'failed'
                                ? 'bg-destructive'
                                : t.status === 'partial'
                                  ? 'bg-amber-500'
                                  : 'bg-primary'
                            )}
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                        <span className="text-xs tabular-nums text-muted-foreground">
                          {t.successCount ?? 0}/{t.failCount ?? 0}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={s.variant}>{s.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {t.creator || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(t.createdAt)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>
      <Pagination page={page} totalPages={totalPages} pageSize={pageSize} onChange={onPageChange} />
    </>
  )
}

// ============================================================
// 模板下发 Modal（选设备 → dispatch → 结果面板）
// ============================================================
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
        onError: (err: Error) => onError(err.message || '下发失败'),
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
                <Label className="mb-2 block">
                  目标设备（已选 {deviceIds.length} 台）
                </Label>
                {devLoading ? (
                  <div className="py-8 text-center text-sm text-muted-foreground">
                    加载设备列表…
                  </div>
                ) : devices.length === 0 ? (
                  <div className="py-8 text-center text-sm text-muted-foreground">
                    暂无可选设备
                  </div>
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
              <Button
                onClick={handleDispatch}
                disabled={deviceIds.length === 0 || dispatch.isPending}
              >
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

// ============================================================
// 小统计卡片
// ============================================================
function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number
  tone?: 'default' | 'amber' | 'red'
}) {
  const toneClass = {
    default: 'text-foreground',
    amber: 'text-amber-600',
    red: 'text-destructive',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</div>
    </div>
  )
}
