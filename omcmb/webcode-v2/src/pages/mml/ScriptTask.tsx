import { useMemo, useState } from 'react'
import { Play, RefreshCcw, Search, ScrollText, X } from 'lucide-react'

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
import { useT } from '@/hooks/useT'

import { useMMLScriptById, useMMLScripts } from '@core/hooks/api/useMML'
import type { MMLScript, MMLScriptStatus } from '@core/types/mml'
import ScriptExecutionDialog from './components/ScriptExecutionDialog'
import ScriptImportDialog from './components/ScriptImportDialog'

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
  const tr = useT()
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [viewing, setViewing] = useState<MMLScript | null>(null)
  const [executing, setExecuting] = useState<MMLScript | null>(null)
  const [importing, setImporting] = useState<MMLScript | null>(null)

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

  const cols = [tr('mml.scriptName'), '描述 / 标签', tr('mml.creator'), '状态', tr('mml.updateTime'), tr('table.operation')]

  return (
    <PageShell
      title="MML 脚本库"
      description="批量执行脚本管理 · mml_scripts"
      isFetching={isFetching}
      toolbar={
        <>
          <Button variant="default" size="sm" onClick={() => setImporting({} as MMLScript)}>{tr('mml.script.action.importTxt')}</Button>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="w-64 pl-9"
              placeholder={tr('mml.scriptName')}
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') applySearch()
              }}
            />
          </div>
          <Button variant="outline" size="sm" onClick={applySearch}>
            {tr('common.search')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="ml-auto"
            onClick={() => refetch()}
          >
            <RefreshCcw /> {tr('common.refresh')}
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
                          {tr('mml.detail')}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-7 px-2 text-xs"
                          onClick={() => setExecuting(s)}
                        >
                          <Play className="size-3" /> {tr('mml.script.action.execute')}
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
        <ScriptDetailDrawer script={viewing} onClose={() => setViewing(null)} onReimport={() => setImporting(viewing)} />
      ) : null}
      {executing ? <ScriptExecutionDialog open script={executing} onClose={() => setExecuting(null)} /> : null}
      <ScriptImportDialog open={Boolean(importing)} script={importing?.id ? importing : null} onClose={() => setImporting(null)} />
    </PageShell>
  )
}

// ---------------------------------------------------------------------------
// 详情抽屉
// ---------------------------------------------------------------------------

function ScriptDetailDrawer({
  script,
  onClose,
  onReimport,
}: {
  script: MMLScript
  onClose: () => void
  onReimport?: () => void
}) {
  const tr = useT()
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
          <Button variant="ghost" size="icon" onClick={onClose} aria-label={tr('common.close')}>
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
              {tr('mml.script.noContent')}
            </div>
          )}
        </div>
        {onReimport ? <div className="border-t px-4 py-3"><Button variant="outline" size="sm" onClick={onReimport}>{tr('mml.script.action.reimport')}</Button></div> : null}
      </div>
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
