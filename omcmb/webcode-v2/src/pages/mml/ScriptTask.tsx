import { useMemo, useState } from 'react'
import { Loader2, Play, RefreshCcw, Search, ScrollText, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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

import { useCreateMMLTask, useMMLScriptById, useMMLScripts } from '@core/hooks/api/useMML'
import type { MMLScript, MMLScriptStatus } from '@core/types/mml'
import { parseMmlScriptPlan } from '@core/utils/mmlScriptPlanParser'

// ============================================================
// MML 脚本库 — mml_scripts 批量脚本（对齐 v1 mml/ScriptTask，参照 v3 mml/script）
// real：useMMLScripts。列表 + 详情抽屉（只读：名称/描述/标签/脚本内容）。
// ============================================================

const STATUS_META: Record<
  MMLScriptStatus,
  { label: string; variant: 'default' | 'destructive' | 'warning' | 'muted' }
> = {
  active: { label: '活跃', variant: 'default' },
  archived: { label: '已归档', variant: 'muted' },
  pending: { label: '待执行', variant: 'muted' },
  running: { label: '执行中', variant: 'warning' },
  paused: { label: '已暂停', variant: 'warning' },
  completed: { label: '已完成', variant: 'default' },
  failed: { label: '失败', variant: 'destructive' },
  cancelled: { label: '已取消', variant: 'muted' },
}

function statusMeta(status: MMLScriptStatus) {
  return STATUS_META[status] ?? { label: status, variant: 'muted' as const }
}

const PAGE_SIZE = 20

export default function ScriptTask() {
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [viewing, setViewing] = useState<MMLScript | null>(null)
  const [executing, setExecuting] = useState<MMLScript | null>(null)

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(search.trim() ? { search: search.trim() } : {}),
    }),
    [page, search]
  )

  const { data, isLoading, isError, error, isFetching, refetch } =
    useMMLScripts(params)
  const scripts = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  // 当前页状态分桶
  const statusCounts = useMemo(() => {
    const m = new Map<string, number>()
    for (const s of scripts) m.set(s.status, (m.get(s.status) ?? 0) + 1)
    return m
  }, [scripts])

  const applySearch = () => {
    setSearch(searchInput)
    setPage(1)
  }

  const cols = ['脚本名', '描述 / 标签', '创建人', '状态', '更新时间', '操作']

  return (
    <PageShell
      title="MML 脚本库"
      description="批量执行脚本管理 · mml_scripts"
      isFetching={isFetching}
      toolbar={
        <>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder="脚本名称"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applySearch()
              }}
            />
          </div>
          <Button variant="outline" size="sm" onClick={applySearch}>
            查询
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => refetch()}
          >
            <RefreshCcw /> 刷新
          </Button>
        </>
      }
    >
      {/* 统计卡片 */}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-5">
        <Stat label="总计" value={total} />
        <Stat label="活跃" value={statusCounts.get('active') ?? 0} tone="emerald" />
        <Stat label="执行中" value={statusCounts.get('running') ?? 0} tone="amber" />
        <Stat label="已完成" value={statusCounts.get('completed') ?? 0} tone="emerald" />
        <Stat label="失败" value={statusCounts.get('failed') ?? 0} tone="destructive" />
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
            ) : scripts.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无脚本</EmptyRow>
            ) : (
              scripts.map((s) => {
                const meta = statusMeta(s.status)
                return (
                  <TableRow key={s.id} className="align-top">
                    <TableCell>
                      <button
                        type="button"
                        className="flex items-center gap-1.5 text-left hover:underline"
                        onClick={() => setViewing(s)}
                      >
                        <ScrollText className="size-3.5 shrink-0 text-muted-foreground" />
                        <span className="font-medium">{s.scriptName}</span>
                      </button>
                      <div className="font-mono text-[11px] text-muted-foreground">
                        {s.id}
                      </div>
                    </TableCell>
                    <TableCell className="max-w-[280px]">
                      <div className="truncate text-xs" title={s.description}>
                        {s.description || '—'}
                      </div>
                      {s.tags && s.tags.length > 0 ? (
                        <div className="mt-1 flex flex-wrap gap-1">
                          {s.tags.slice(0, 4).map((tag) => (
                            <span
                              key={tag}
                              className="rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground"
                            >
                              {tag}
                            </span>
                          ))}
                        </div>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-xs">{s.creator || '—'}</TableCell>
                    <TableCell>
                      <Badge variant={meta.variant}>{meta.label}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(s.updateTime)}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 px-2 text-xs"
                          onClick={() => setViewing(s)}
                        >
                          详情
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-7 px-2 text-xs"
                          onClick={() => setExecuting(s)}
                        >
                          <Play className="size-3" /> 执行
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
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />

      {viewing ? (
        <ScriptDetailDrawer script={viewing} onClose={() => setViewing(null)} />
      ) : null}
      {executing ? (
        <ScriptExecuteDialog script={executing} onClose={() => setExecuting(null)} />
      ) : null}
    </PageShell>
  )
}

function ScriptExecuteDialog({
  script,
  onClose,
}: {
  script: MMLScript
  onClose: () => void
}) {
  const { data, isFetching } = useMMLScriptById(script.id)
  const detailScript = data ?? script
  const [taskName, setTaskName] = useState(`执行脚本: ${script.scriptName}`)
  const [deviceInput, setDeviceInput] = useState('')
  const [error, setError] = useState('')
  const createTask = useCreateMMLTask()
  const parsed = useMemo(
    () => parseMmlScriptPlan(detailScript.content ?? '', { format: 'auto' }),
    [detailScript.content]
  )
  const isDeviceBound = parsed.executeMode === 'device_bound'

  const submit = async () => {
    setError('')
    const deviceSns = isDeviceBound ? parsed.deviceSns : parseDeviceInput(deviceInput)
    if (!isDeviceBound && deviceSns.length === 0) {
      setError('请输入设备 SN')
      return
    }
    if (parsed.commands.length === 0) {
      setError('脚本内容为空')
      return
    }
    await createTask.mutateAsync({
      taskName: taskName.trim() || `执行脚本: ${detailScript.scriptName}`,
      scriptId: detailScript.id,
      deviceSns,
      commands: parsed.commands,
      executeMode: parsed.executeMode,
      planItems: isDeviceBound ? parsed.planItems : undefined,
      creator: '',
      executeType: 'immediate',
      offlineRetry: false,
      offlineRetryWait: 60,
      failedRetry: false,
      failedRetryCount: 3,
      failedRetryInterval: 5,
    })
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />
      <div className="relative flex max-h-[86vh] w-[min(720px,calc(100vw-32px))] flex-col overflow-hidden rounded border bg-background shadow-xl">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div className="min-w-0">
            <div className="truncate text-base font-semibold">执行脚本</div>
            <div className="truncate text-xs text-muted-foreground">{detailScript.scriptName}</div>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="关闭">
            <X />
          </Button>
        </div>
        <div className="min-h-0 flex-1 space-y-3 overflow-auto px-4 py-3">
          <Field label="任务名" value={<Input value={taskName} onChange={(e) => setTaskName(e.target.value)} />} />
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div className="rounded border bg-muted/30 px-3 py-2.5">
              <div className="text-[11px] text-muted-foreground">脚本执行方式</div>
              <div className="mt-1 text-sm font-semibold">
                {isDeviceBound ? '按设备编排执行' : '统一脚本批量执行'}
              </div>
              <div className="mt-1 text-xs leading-5 text-muted-foreground">
                {isDeviceBound
                  ? '每行命令绑定设备 SN；同一设备按脚本从上到下执行。'
                  : '选择多台设备，每台设备执行同一套脚本。'}
              </div>
            </div>
            <MiniStat label="解析结果" value={isDeviceBound ? `${parsed.planItems.length} 行 / ${parsed.deviceSns.length} 台` : `${parsed.commands.length} 条命令`} />
          </div>
          {!isDeviceBound ? (
            <Field
              label="设备 SN"
              value={
                <Input
                  value={deviceInput}
                  placeholder="逗号、空格或换行分隔"
                  onChange={(e) => setDeviceInput(e.target.value)}
                />
              }
            />
          ) : (
            <div className="rounded border bg-muted/30">
              <div className="border-b px-3 py-2 text-xs font-medium text-muted-foreground">计划行预览</div>
              <div className="max-h-56 overflow-auto">
                {parsed.planItems.slice(0, 20).map((item) => (
                  <div key={`${item.lineNo}-${item.deviceSn}-${item.order}`} className="grid grid-cols-[72px_150px_1fr] gap-2 border-b px-3 py-1.5 text-xs last:border-b-0">
                    <span className="font-mono">#{item.lineNo}/{item.order}</span>
                    <span className="truncate font-mono">{item.deviceSn}</span>
                    <span className="truncate">{item.command.commandCode}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
          {error ? <div className="text-sm text-destructive">{error}</div> : null}
          {isFetching ? <div className="text-xs text-muted-foreground">加载脚本内容中...</div> : null}
        </div>
        <div className="flex justify-end gap-2 border-t px-4 py-3">
          <Button variant="outline" onClick={onClose}>取消</Button>
          <Button onClick={() => void submit()} disabled={createTask.isPending}>
            {createTask.isPending ? <Loader2 className="size-4 animate-spin" /> : <Play className="size-4" />}
            执行
          </Button>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// 详情抽屉
// ---------------------------------------------------------------------------

function ScriptDetailDrawer({
  script,
  onClose,
}: {
  script: MMLScript
  onClose: () => void
}) {
  const { data, isFetching } = useMMLScriptById(script.id)
  const detailScript = data ?? script
  const meta = statusMeta(detailScript.status)
  return (
    <div className="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} aria-hidden />
      <div className="relative flex h-full w-full max-w-[640px] flex-col overflow-hidden border-l bg-background shadow-xl">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div className="min-w-0">
            <div className="truncate text-base font-semibold">
              {detailScript.scriptName}
            </div>
            <div className="truncate font-mono text-[11px] text-muted-foreground">
              {detailScript.id}
            </div>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="关闭">
            <X />
          </Button>
        </div>

        <div className="grid grid-cols-2 gap-3 border-b px-4 py-3">
          <Field label="状态" value={<Badge variant={meta.variant}>{meta.label}</Badge>} />
          <Field label="类型" value={detailScript.type} />
          <Field label="创建人" value={detailScript.creator || '—'} />
          <Field label="进度" value={`${detailScript.progress ?? 0}%`} />
          <Field label="创建时间" value={formatTime(detailScript.createTime)} />
          <Field label="更新时间" value={formatTime(detailScript.updateTime)} />
        </div>

        {detailScript.description ? (
          <div className="border-b px-4 py-3">
            <div className="mb-1 text-xs uppercase tracking-wider text-muted-foreground">
              描述
            </div>
            <div className="text-sm">{detailScript.description}</div>
          </div>
        ) : null}

        {detailScript.tags && detailScript.tags.length > 0 ? (
          <div className="flex flex-wrap items-center gap-1.5 border-b px-4 py-3">
            {detailScript.tags.map((tag) => (
              <span
                key={tag}
                className="rounded border bg-muted px-2 py-0.5 font-mono text-[11px] text-muted-foreground"
              >
                {tag}
              </span>
            ))}
          </div>
        ) : null}

        <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
          <div className="mb-1.5 flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
            <span>脚本内容</span>
            {isFetching ? <span>加载中...</span> : null}
          </div>
          {detailScript.content?.trim() ? (
            <pre className="overflow-auto whitespace-pre-wrap rounded border bg-muted/40 p-3 font-mono text-xs leading-relaxed">
              {detailScript.content}
            </pre>
          ) : (
            <div className="rounded border border-dashed bg-muted/20 px-3 py-8 text-center text-sm text-muted-foreground">
              暂无脚本内容
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function parseDeviceInput(value: string): string[] {
  return Array.from(new Set(value.split(/[,\s;]+/).map((v) => v.trim()).filter(Boolean)))
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded border bg-muted/30 px-3 py-2">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">{label}</div>
      <div className="mt-0.5 truncate text-sm font-medium">{value}</div>
    </div>
  )
}

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-0.5 truncate text-sm">{value}</div>
    </div>
  )
}

function Stat({
  label,
  value,
  tone = 'default',
}: {
  label: string
  value: number
  tone?: 'default' | 'emerald' | 'amber' | 'destructive'
}) {
  const toneClass = {
    default: 'text-foreground',
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    destructive: 'text-destructive',
  }[tone]
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className={`mt-1 text-2xl font-semibold tabular-nums ${toneClass}`}>
        {value}
      </div>
    </div>
  )
}
